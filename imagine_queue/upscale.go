package imagine_queue

import (
	"bytes"
	"context"
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

func (q *queueImplementation) processUpscaleImagine(queue *entities.QueueItem) {
	defer q.done()
	interactionID := queue.DiscordInteraction.ID
	messageID := ""

	if queue.DiscordInteraction.Message != nil {
		messageID = queue.DiscordInteraction.Message.ID
	}

	log.Printf("Upscaling image: %v, Message: %v, Upscale Index: %d",
		interactionID, messageID, queue.InteractionIndex)

	request, err := q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, queue.InteractionIndex)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return
	}

	log.Printf("Found generation: %v", request)

	textToImage := request.TextToImageRequest
	if textToImage == nil {
		handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction,
			fmt.Sprintf("TextToImageRequest of type %v is nil", queue.Type),
		)
		return
	}

	config, err := q.stableDiffusionAPI.GetConfig()
	originalConfig := config
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return
	} else {
		config, err = q.updateModels(request, queue, config)
		if err != nil {
			return
		}
	}

	newContent := upscaleMessageContent(queue.DiscordInteraction.Member.User, 0, 0)

	_, err = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
	}

	generationDone := make(chan bool)

	go func() {
		lastProgress := float64(0)
		fetchProgress := float64(0)
		upscaleProgress := float64(0)
		elapsedTime := 0

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
				elapsedTime += 1

				if elapsedTime > 60 {
					msg := "Upscale timed out after 60 seconds"
					log.Printf(msg)

					_, _ = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
						Content: &msg,
					})

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
					log.Printf("Error editing interaction: %v", err)
				}
			}
		}
	}()

	// Use face segm model if we're upscaling but there's no ADetailer models
	if textToImage.Scripts.ADetailer == nil {
		textToImage.Scripts.NewADetailerWithArgs()
		textToImage.Scripts.ADetailer.AppendSegModelByString("face_yolov8n.pt", request)
	}

	textToImage.BatchSize = 1
	textToImage.NIter = 1

	resp, err := q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
		ResizeMode:         0,
		UpscalingResize:    2,
		Upscaler1:          "R-ESRGAN 2x+",
		TextToImageRequest: textToImage,
	})

	generationDone <- true

	if err != nil {
		log.Printf("Error processing image upscale: %v\n", err)

		handlers.ErrorHandler(q.botSession, queue.DiscordInteraction, fmt.Sprint("I'm sorry, but I had a problem upscaling your image. ", err))
		return
	}

	decodedImage, decodeErr := base64.StdEncoding.DecodeString(resp.Image)
	if decodeErr != nil {
		log.Printf("Error decoding image: %v\n", decodeErr)
		handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, decodeErr)
		return
	}
	if len(decodedImage) == 0 {
		log.Printf("Error decoding image: %v\n", "empty image")
		handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, fmt.Errorf("empty image"))
		return
	}

	log.Printf("Successfully upscaled image: %v, Message: %v, Upscale Index: %d",
		interactionID, messageID, queue.InteractionIndex)

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
		request.Seed,
		scriptsString,
	)

	if len(finishedContent) > 2000 {
		finishedContent = finishedContent[:2000]
	}

	_, err = q.botSession.InteractionResponseEdit(queue.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				// add timestamp to output file
				Name:   "imagine_" + time.Now().Format("20060102150405") + ".png",
				Reader: bytes.NewBuffer(decodedImage),
			},
		},
		Components: &[]discordgo.MessageComponent{
			handlers.Components[handlers.DeleteGeneration],
		},
	})
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)
		return
	}
}
