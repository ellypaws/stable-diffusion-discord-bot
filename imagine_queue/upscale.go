package imagine_queue

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/stable_diffusion_api"
	"strings"
	"time"
)

func (q *queueImplementation) processUpscaleImagine() {
	defer q.done()
	queue := q.currentImagine
	var err error
	queue.ImageGenerationRequest, err = q.getPreviousGeneration(queue)
	if err != nil {
		log.Printf("Error getting prompt for upscale: %v", err)
		handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, err)
		return
	}

	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest
	if textToImage == nil {
		handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction,
			fmt.Sprintf("TextToImageRequest of type %v is nil", queue.Type),
		)
		return
	}

	newContent := upscaleMessageContent(queue.DiscordInteraction.Member.User, 0, 0)

	_, err = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
	}

	generationDone := make(chan bool)

	go q.updateUpscaleProgress(queue, generationDone)

	resp, err := q.upscale(request)
	generationDone <- true
	if err != nil {
		log.Printf("Error processing image upscale: %v\n", err)
		handlers.ErrorHandler(q.botSession, queue.DiscordInteraction, fmt.Sprint("I'm sorry, but I had a problem upscaling your image. ", err))
		return
	}

	log.Printf("Successfully upscaled image: %v, Message: %v, Upscale Index: %d", queue.DiscordInteraction.ID, queue.DiscordInteraction.Message.ID, queue.InteractionIndex)

	if err := q.finalUpscaleMessage(queue, resp); err != nil {
		log.Printf("Error finalizing upscale message: %v\n", err)
		handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, err)
		return
	}
}

func (q *queueImplementation) upscale(request *entities.ImageGenerationRequest) (*stable_diffusion_api.UpscaleResponse, error) {
	textToImage := request.TextToImageRequest
	// Use face segm model if we're upscaling but there's no ADetailer models
	if textToImage.Scripts.ADetailer == nil {
		textToImage.Scripts.NewADetailerWithArgs()
		textToImage.Scripts.ADetailer.AppendSegModelByString("face_yolov8n.pt", request)
	}

	textToImage.BatchSize = 1
	textToImage.NIter = 1

	return q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
		ResizeMode:         0,
		UpscalingResize:    2,
		Upscaler1:          "R-ESRGAN 2x+",
		TextToImageRequest: textToImage,
	})
}

func (q *queueImplementation) finalUpscaleMessage(queue *entities.QueueItem, resp *stable_diffusion_api.UpscaleResponse) error {
	textToImage := queue.ImageGenerationRequest.TextToImageRequest

	decodedImage, decodeErr := base64.StdEncoding.DecodeString(resp.Image)
	if decodeErr != nil {
		return fmt.Errorf("error decoding image: %v", decodeErr)
	}
	if len(decodedImage) == 0 {
		return fmt.Errorf("decoded image is empty")
	}

	var scriptsString string
	var scripts []string

	if queue.Type != ItemTypeRaw {
		if textToImage.Scripts.ADetailer != nil {
			scripts = append(scripts, "ADetailer")
		}
		if textToImage.Scripts.ControlNet != nil {
			scripts = append(scripts, "ControlNet")
		}
		if textToImage.Scripts.CFGRescale != nil {
			scripts = append(scripts, "CFGRescale")
		}
	} else {
		for script := range queue.Raw.RawScripts {
			scripts = append(scripts, script)
		}
	}

	if len(scripts) > 0 {
		scriptsString = fmt.Sprintf("\n**Scripts**: [`%v`]", strings.Join(scripts, ", "))
	}

	finishedContent := fmt.Sprintf("<@%s> asked me to upscale their image. (seed: %d) Here's the result:%v",
		queue.DiscordInteraction.Member.User.ID,
		textToImage.Seed,
		scriptsString,
	)

	if len(finishedContent) > 2000 {
		finishedContent = finishedContent[:2000]
	}

	_, err := q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				// add timestamp to output file
				Name:   "upscale_" + time.Now().Format("20060102150405") + ".png",
				Reader: bytes.NewBuffer(decodedImage),
			},
		},
		Components: &[]discordgo.MessageComponent{
			handlers.Components[handlers.DeleteGeneration],
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (q *queueImplementation) updateUpscaleProgress(queue *entities.QueueItem, generationDone chan bool) {
	config, err := q.stableDiffusionAPI.GetConfig()
	originalConfig := config
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return
	} else {
		config, err = q.updateModels(queue.ImageGenerationRequest, queue, config)
		if err != nil {
			return
		}
	}
	lastProgress := float64(0)
	fetchProgress := float64(0)
	upscaleProgress := float64(0)
	for {
		select {
		case queue.DiscordInteraction = <-queue.Interrupt:
			err := q.stableDiffusionAPI.Interrupt()
			if err != nil {
				handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
				return
			}
			message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, queue.DiscordInteraction, "Generation Interrupted", handlers.Components[handlers.DeleteGeneration])
			if queue.DiscordInteraction.Message == nil && message != nil {
				log.Printf("Setting c.DiscordInteraction.Message to message from channel c.Interrupt: %v", message)
				queue.DiscordInteraction.Message = message
			}
		case <-generationDone:
			err := q.revertModels(config, originalConfig)
			if err != nil {
				handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error reverting models: %v", err))
			}
			return
		case <-time.After(1 * time.Second):
			progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
			if progressErr != nil {
				log.Printf("Error getting current progress: %v", progressErr)
				return
			}

			if progress.Progress == 0 {
				continue
			}

			if progress.Progress < lastProgress || upscaleProgress > 0 {
				upscaleProgress = progress.Progress
				fetchProgress = 1
			} else {
				fetchProgress = progress.Progress
			}

			lastProgress = progress.Progress

			progressContent := upscaleMessageContent(queue.DiscordInteraction.Member.User, fetchProgress, upscaleProgress)

			_, progressErr = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
				Content: &progressContent,
			})
			if progressErr != nil {
				log.Printf("Error editing interaction: %v", progressErr)
				return
			}
		}
	}
}
