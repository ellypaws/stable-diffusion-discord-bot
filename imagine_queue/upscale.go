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

func (q *queueImplementation) processUpscaleImagine(imagine *entities.QueueItem) {
	defer q.done()
	interactionID := imagine.DiscordInteraction.ID
	messageID := ""

	if imagine.DiscordInteraction.Message != nil {
		messageID = imagine.DiscordInteraction.Message.ID
	}

	log.Printf("Upscaling image: %v, Message: %v, Upscale Index: %d",
		interactionID, messageID, imagine.InteractionIndex)

	generation, err := q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, imagine.InteractionIndex)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return
	}

	log.Printf("Found generation: %v", generation)

	config, err := q.stableDiffusionAPI.GetConfig()
	originalConfig := config
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return
	} else {
		config, err = q.updateModels(generation, imagine, config)
		if err != nil {
			return
		}
	}

	newContent := upscaleMessageContent(imagine.DiscordInteraction.Member.User, 0, 0)

	_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
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
			case imagine.DiscordInteraction = <-imagine.Interrupt:
				err := q.stableDiffusionAPI.Interrupt()
				if err != nil {
					handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
					return
				}
				message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, imagine.DiscordInteraction, "Generation Interrupted", handlers.Components[handlers.DeleteGeneration])
				if imagine.DiscordInteraction.Message == nil && message != nil {
					log.Printf("Setting c.DiscordInteraction.Message to message from channel c.Interrupt: %v", message)
					imagine.DiscordInteraction.Message = message
				}
			case <-generationDone:
				err := q.revertModels(config, originalConfig)
				if err != nil {
					handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, fmt.Sprintf("Error reverting models: %v", err))
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

					_, _ = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
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

				progressContent := upscaleMessageContent(imagine.DiscordInteraction.Member.User, fetchProgress, upscaleProgress)

				_, progressErr = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
					Content: &progressContent,
				})
				if progressErr != nil {
					log.Printf("Error editing interaction: %v", err)
				}
			}
		}
	}()

	// Use face segm model if we're upscaling but there's no ADetailer models
	if generation.Scripts.ADetailer == nil {
		generation.Scripts.NewADetailerWithArgs()
		generation.Scripts.ADetailer.AppendSegModelByString("face_yolov8n.pt", generation)
	}

	t2iRequest := generation.TextToImageRequest
	t2iRequest.BatchSize = 1
	t2iRequest.NIter = 1

	resp, err := q.stableDiffusionAPI.UpscaleImage(&stable_diffusion_api.UpscaleRequest{
		ResizeMode:         0,
		UpscalingResize:    2,
		Upscaler1:          "R-ESRGAN 2x+",
		TextToImageRequest: t2iRequest,
	})

	generationDone <- true

	if err != nil {
		log.Printf("Error processing image upscale: %v\n", err)

		handlers.ErrorHandler(q.botSession, imagine.DiscordInteraction, fmt.Sprint("I'm sorry, but I had a problem upscaling your image. ", err))
		return
	}

	decodedImage, decodeErr := base64.StdEncoding.DecodeString(resp.Image)
	if decodeErr != nil {
		log.Printf("Error decoding image: %v\n", decodeErr)
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, decodeErr)
		return
	}
	if len(decodedImage) == 0 {
		log.Printf("Error decoding image: %v\n", "empty image")
		handlers.Errors[handlers.ErrorResponse](q.botSession, imagine.DiscordInteraction, fmt.Errorf("empty image"))
		return
	}

	log.Printf("Successfully upscaled image: %v, Message: %v, Upscale Index: %d",
		interactionID, messageID, imagine.InteractionIndex)

	var scriptsString string

	if generation.Scripts.ADetailer != nil && len(generation.Scripts.ADetailer.Args) > 0 {
		var models []string
		for _, v := range generation.Scripts.ADetailer.Args {
			models = append(models, v.AdModel)
		}
		scriptsString = fmt.Sprintf("\n**ADetailer**: [%v]", strings.Join(models, ", "))
	}
	if generation.Scripts.ControlNet != nil && len(generation.Scripts.ControlNet.Args) > 0 {
		var preprocessor []string
		var model []string
		for _, v := range generation.Scripts.ControlNet.Args {
			preprocessor = append(preprocessor, v.Module)
			model = append(model, v.Model)
		}
		scriptsString = fmt.Sprintf("\n**ControlNet**: [%v]\n**Preprocessor**: [%v]", strings.Join(model, ", "), strings.Join(preprocessor, ", "))
	}

	finishedContent := fmt.Sprintf("<@%s> asked me to upscale their image. (seed: %d) Here's the result:\n\n Scripts: ```json\n%v\n```",
		imagine.DiscordInteraction.Member.User.ID,
		generation.Seed,
		scriptsString,
	)

	if len(finishedContent) > 2000 {
		finishedContent = finishedContent[:2000]
	}

	_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
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
