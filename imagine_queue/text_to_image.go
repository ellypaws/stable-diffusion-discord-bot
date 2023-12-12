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

func (q *queueImplementation) processImagineGrid(c *entities.QueueItem) error {
	newGeneration := c.ImageGenerationRequest
	config, err := q.stableDiffusionAPI.GetConfig()
	originalConfig := config
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return err
	} else {
		config, err = q.updateModels(newGeneration, c, config)
		if err != nil {
			return err
		}
	}

	log.Printf("Processing imagine #%s: %v\n", c.DiscordInteraction.ID, newGeneration.Prompt)

	newContent := imagineMessageSimple(newGeneration, c.DiscordInteraction.Member.User, 0)

	embed := generationEmbedDetails(&discordgo.MessageEmbed{}, newGeneration, c, c.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Embeds:     &[]*discordgo.MessageEmbed{embed},
	}

	message, err := q.botSession.InteractionResponseEdit(c.DiscordInteraction, webhook)
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
		return err
	}

	// store message ID in c.DiscordInteraction.Message
	if c.DiscordInteraction != nil && c.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting c.DiscordInteraction.Message to message: %v", message)
		c.DiscordInteraction.Message = message
	}

	newGeneration.InteractionID = c.DiscordInteraction.ID
	newGeneration.MessageID = message.ID
	newGeneration.MemberID = c.DiscordInteraction.Member.User.ID
	newGeneration.SortOrder = 0
	newGeneration.Processed = true

	var ok bool
	if newGeneration.Prompt, ok = strings.CutSuffix(newGeneration.Prompt, "{DEBUG}"); ok {
		byteArr, _ := newGeneration.TextToImageRequest.Marshal()
		log.Printf("{DEBUG} TextToImageRequest: %v", string(byteArr))
	}

	// return newGeneration from image_generations.Create as we need newGeneration.CreatedAt later on
	newGeneration, err = q.imageGenerationRepo.Create(context.Background(), newGeneration)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
		return err
	}

	generationDone := make(chan bool)

	go func() {
		for {
			select {
			case c.DiscordInteraction = <-c.Interrupt:
				err := q.stableDiffusionAPI.Interrupt()
				if err != nil {
					handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
					return
				}
				message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, c.DiscordInteraction, "Generation Interrupted", webhook, handlers.Components[handlers.DeleteGeneration])
				if c.DiscordInteraction.Message == nil && message != nil {
					log.Printf("Setting c.DiscordInteraction.Message to message from channel c.Interrupt: %v", message)
					c.DiscordInteraction.Message = message
				}
			case <-generationDone:
				err := q.revertModels(config, originalConfig)
				if err != nil {
					handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, fmt.Sprintf("Error reverting models: %v", err))
				}
				return
			case <-time.After(1 * time.Second):
				progress, progressErr := q.stableDiffusionAPI.GetCurrentProgress()
				if progressErr != nil {
					log.Printf("Error getting current progress: %v", progressErr)
					handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, fmt.Sprintf("Error getting current progress: %v", progressErr))

					return
				}

				if progress.Progress == 0 {
					continue
				}

				progressContent := imagineMessageSimple(newGeneration, c.DiscordInteraction.Member.User, progress.Progress)

				_, progressErr = q.botSession.InteractionResponseEdit(c.DiscordInteraction, &discordgo.WebhookEdit{
					Content: &progressContent,
				})
				if progressErr != nil {
					log.Printf("Error editing interaction: %v", err)
				}
			}
		}
	}()

	switch c.Type {
	case ItemTypeImagine, ItemTypeReroll, ItemTypeVariation, ItemTypeRaw:
		var resp *stable_diffusion_api.TextToImageResponse
		var err error
		switch c.Type {
		case ItemTypeRaw:
			if q.currentImagine.Raw.Unsafe {
				resp, err = q.stableDiffusionAPI.TextToImageRaw(q.currentImagine.Raw.Blob)
			} else {
				marshal, marshalErr := q.currentImagine.Raw.Marshal()
				if marshalErr != nil {
					log.Printf("Error marshalling raw: %v", marshalErr)
					return marshalErr
				}
				resp, err = q.stableDiffusionAPI.TextToImageRaw(marshal)
			}
		default:
			resp, err = q.stableDiffusionAPI.TextToImageRequest(newGeneration.TextToImageRequest)
		}

		generationDone <- true

		if err != nil || resp == nil {
			log.Printf("Error processing image: %v\n", err)
			return err
		}

		// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt and interrupted
		embed = generationEmbedDetails(embed, newGeneration, c, c.Interrupt != nil)

		log.Printf("Seeds: %v Subseeds:%v", resp.Seeds, resp.Subseeds)

		imageBufs := make([]*bytes.Buffer, len(resp.Images))

		for idx, image := range resp.Images {
			decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
			if decodeErr != nil {
				log.Printf("Error decoding image: %v\n", decodeErr)
			}

			imageBufs[idx] = bytes.NewBuffer(decodedImage)
		}

		for idx := range resp.Seeds {
			subGeneration := newGeneration
			subGeneration.SortOrder = idx + 1
			subGeneration.Seed = resp.Seeds[idx]
			subGeneration.Subseed = int64(resp.Subseeds[idx])
			subGeneration.Checkpoint = config.SDModelCheckpoint
			subGeneration.VAE = config.SDVae
			subGeneration.Hypernetwork = config.SDHypernetwork

			_, createErr := q.imageGenerationRepo.Create(context.Background(), subGeneration)
			if createErr != nil {
				log.Printf("Error creating image generation record: %v\n", createErr)
			}
		}

		var thumbnailBuffers []*bytes.Buffer

		if c.ControlnetItem.MessageAttachment != nil {
			decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(c.ControlnetItem.MessageAttachment.Image))
			if err != nil {
				log.Printf("Error decoding image: %v\n", err)
			}
			thumbnailBuffers = append(thumbnailBuffers, bytes.NewBuffer(decodedBytes))
		}

		const maxImages = 4
		if newGeneration.BatchSize == 0 {
			log.Printf("Warning: newGeneration.Batchsize == 0")
			newGeneration.BatchSize = between(newGeneration.BatchSize, 1, maxImages)
		}
		if newGeneration.NIter == 0 {
			log.Printf("Warning: newGeneration.NIter == 0")
			newGeneration.NIter = between(newGeneration.NIter, 1, maxImages/newGeneration.BatchSize)
		}

		totalImages := newGeneration.NIter * newGeneration.BatchSize

		if len(imageBufs) > totalImages {
			log.Printf("received extra images: len(imageBufs): %v, controlnet: %v", len(imageBufs), c.ControlnetItem.Enabled)
			thumbnailBuffers = append(thumbnailBuffers, imageBufs[totalImages:]...)
		}

		mention := fmt.Sprintf("<@%v>", c.DiscordInteraction.Member.User.ID)

		webhook = &discordgo.WebhookEdit{
			Content:    &mention,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: rerollVariationComponents(min(len(imageBufs), totalImages), c.Type == ItemTypeImg2Img),
		}

		if err := imageEmbedFromBuffers(webhook, embed, imageBufs[:min(len(imageBufs), totalImages)], thumbnailBuffers); err != nil {
			log.Printf("Error creating image embed: %v\n", err)
			return err
		}

		_, err = q.botSession.InteractionResponseEdit(c.DiscordInteraction, webhook)
		if err != nil {
			log.Printf("Error editing interaction: %v\n", err)
			return err
		}
	case ItemTypeImg2Img:
		err, done := q.imageToImage(newGeneration, c, generationDone)
		if done {
			return err
		}
	}

	return nil
}

func (q *queueImplementation) revertModels(config *entities.Config, originalConfig *entities.Config) error {
	if !ptrStringCompare(config.SDModelCheckpoint, originalConfig.SDModelCheckpoint) ||
		!ptrStringCompare(config.SDVae, originalConfig.SDVae) ||
		!ptrStringCompare(config.SDHypernetwork, originalConfig.SDHypernetwork) {
		log.Printf("Switching back to original models: %v, %v, %v", originalConfig.SDModelCheckpoint, originalConfig.SDVae, originalConfig.SDHypernetwork)
		return q.stableDiffusionAPI.UpdateConfiguration(entities.Config{
			SDModelCheckpoint: originalConfig.SDModelCheckpoint,
			SDVae:             originalConfig.SDVae,
			SDHypernetwork:    originalConfig.SDHypernetwork,
		})
	}
	return nil
}

func (q *queueImplementation) updateModels(newGeneration *entities.ImageGenerationRequest, c *entities.QueueItem, config *entities.Config) (*entities.Config, error) {
	if !ptrStringCompare(newGeneration.Checkpoint, config.SDModelCheckpoint) ||
		!ptrStringCompare(newGeneration.VAE, config.SDVae) ||
		!ptrStringCompare(newGeneration.Hypernetwork, config.SDHypernetwork) {
		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, c.DiscordInteraction,
			fmt.Sprintf("Changing models to: \n**Checkpoint**: `%v` -> `%v`\n**VAE**: `%v` -> `%v`\n**Hypernetwork**: `%v` -> `%v`",
				safeDereference(config.SDModelCheckpoint), safeDereference(newGeneration.Checkpoint),
				safeDereference(config.SDVae), safeDereference(newGeneration.VAE),
				safeDereference(config.SDHypernetwork), safeDereference(newGeneration.Hypernetwork),
			),
			handlers.Components[handlers.CancelDisabled])

		// Insert code to update the configuration here
		err := q.stableDiffusionAPI.UpdateConfiguration(
			q.lookupModel(newGeneration, config,
				[]stable_diffusion_api.Cacheable{
					stable_diffusion_api.CheckpointCache,
					stable_diffusion_api.VAECache,
					stable_diffusion_api.HypernetworkCache,
				}))
		if err != nil {
			log.Printf("Error updating configuration: %v", err)
			return nil, err
		}
		config, err = q.stableDiffusionAPI.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
			return nil, err
		}
		newGeneration.Checkpoint = config.SDModelCheckpoint
		newGeneration.VAE = config.SDVae
		newGeneration.Hypernetwork = config.SDHypernetwork
	}
	return config, nil
}
