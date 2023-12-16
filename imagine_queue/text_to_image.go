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

func (q *queueImplementation) processImagineGrid(queue *entities.QueueItem) error {
	request := queue.ImageGenerationRequest
	textToImage := request.TextToImageRequest
	config, err := q.stableDiffusionAPI.GetConfig()
	originalConfig := config
	if err != nil {
		log.Printf("Error getting config: %v", err)
		return err
	} else {
		config, err = q.updateModels(request, queue, config)
		if err != nil {
			return err
		}
	}

	log.Printf("Processing imagine #%s: %v\n", queue.DiscordInteraction.ID, textToImage.Prompt)

	embed, webhook, err := showInitialMessage(queue, q)
	if err != nil {
		return err
	}

	request, err = q.recordToRepository(request, err)
	if err != nil {
		return err
	}

	generationDone := make(chan bool)

	go q.updateProgressBar(queue, generationDone, config, originalConfig, webhook)

	switch queue.Type {
	case ItemTypeImagine, ItemTypeReroll, ItemTypeVariation, ItemTypeRaw:
		response, err := q.textInference(queue)
		generationDone <- true
		if err != nil {
			return err
		}

		if response == nil {
			log.Printf("Response of type %v is nil! Returned error:%v", queue.Type, err)
			return err
		}

		q.recordSeeds(response, request, config)

		err = q.showFinalMessage(queue, response, embed, webhook)
		if err != nil {
			return err
		}
	case ItemTypeImg2Img:
		err := q.imageToImage(generationDone, embed, webhook)
		if err != nil {
			return err
		}
	}

	return nil
}

func showInitialMessage(queue *entities.QueueItem, q *queueImplementation) (*discordgo.MessageEmbed, *discordgo.WebhookEdit, error) {
	request := queue.ImageGenerationRequest
	newContent := imagineMessageSimple(request, queue.DiscordInteraction.Member.User, 0)

	embed := generationEmbedDetails(&discordgo.MessageEmbed{}, request, queue, queue.Interrupt != nil)

	webhook := &discordgo.WebhookEdit{
		Content:    &newContent,
		Components: &[]discordgo.MessageComponent{handlers.Components[handlers.Interrupt]},
		Embeds:     &[]*discordgo.MessageEmbed{embed},
	}

	message, err := q.botSession.InteractionResponseEdit(queue.DiscordInteraction, webhook)
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
		return nil, nil, err
	}

	// store message ID in c.DiscordInteraction.Message
	if queue.DiscordInteraction != nil && queue.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting c.DiscordInteraction.Message to message: %v", message)
		queue.DiscordInteraction.Message = message
	}

	request.InteractionID = queue.DiscordInteraction.ID
	request.MessageID = message.ID
	request.MemberID = queue.DiscordInteraction.Member.User.ID
	request.SortOrder = 0
	request.Processed = true
	return embed, webhook, nil
}

func (q *queueImplementation) showFinalMessage(queue *entities.QueueItem, response *stable_diffusion_api.TextToImageResponse, embed *discordgo.MessageEmbed, webhook *discordgo.WebhookEdit) error {
	request := queue.ImageGenerationRequest
	totalImages := totalImageCount(request)

	imageBuffers, thumbnailBuffers := retrieveImagesFromResponse(response, queue)

	mention := fmt.Sprintf("<@%v>", queue.DiscordInteraction.Member.User.ID)
	// get new embed from generationEmbedDetails as q.imageGenerationRepo.Create has filled in newGeneration.CreatedAt and interrupted
	embed = generationEmbedDetails(embed, request, queue, queue.Interrupt != nil)

	webhook = &discordgo.WebhookEdit{
		Content:    &mention,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: rerollVariationComponents(min(len(imageBuffers), totalImages), queue.Type == ItemTypeImg2Img),
	}

	if queue.Type != ItemTypeImg2Img || len(thumbnailBuffers) > 0 {
		if err := imageEmbedFromBuffers(webhook, embed, imageBuffers[:min(len(imageBuffers), totalImages)], thumbnailBuffers); err != nil {
			log.Printf("Error creating image embed: %v\n", err)
			return err
		}
	} else {
		// because we don't have the original webhook that contains the image file
		var primaryImage *bytes.Reader
		if len(imageBuffers) > 0 {
			primaryImage = bytes.NewReader(imageBuffers[0].Bytes())
		}
		err := imageAttachmentAsThumbnail(webhook, embed, primaryImage, queue.Img2ImgItem.MessageAttachment, true)
		if err != nil {
			log.Printf("Error attaching image as thumbnail: %v", err)
			return err
		}
	}

	_, err := q.botSession.InteractionResponseEdit(queue.DiscordInteraction, webhook)
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)
		return err
	}
	return nil
}

func (q *queueImplementation) recordSeeds(response *stable_diffusion_api.TextToImageResponse, generation *entities.ImageGenerationRequest, config *entities.Config) {
	log.Printf("Seeds: %v Subseeds:%v", response.Seeds, response.Subseeds)
	for idx := range response.Seeds {
		subGeneration := generation
		subGeneration.SortOrder = idx + 1
		subGeneration.Seed = response.Seeds[idx]
		subGeneration.Subseed = int64(response.Subseeds[idx])
		subGeneration.Checkpoint = config.SDModelCheckpoint
		subGeneration.VAE = config.SDVae
		subGeneration.Hypernetwork = config.SDHypernetwork

		_, createErr := q.imageGenerationRepo.Create(context.Background(), subGeneration)
		if createErr != nil {
			log.Printf("Error creating image generation record: %v\n", createErr)
		}
	}
}

func totalImageCount(generation *entities.ImageGenerationRequest) int {
	const maxImages = 4
	if generation.BatchSize == 0 {
		log.Printf("Warning: newGeneration.Batchsize == 0")
		generation.BatchSize = between(generation.BatchSize, 1, maxImages)
	}
	if generation.NIter == 0 {
		log.Printf("Warning: newGeneration.NIter == 0")
		generation.NIter = between(generation.NIter, 1, maxImages/generation.BatchSize)
	}

	totalImages := generation.NIter * generation.BatchSize
	return totalImages
}

func retrieveImagesFromResponse(response *stable_diffusion_api.TextToImageResponse, queue *entities.QueueItem) (images, thumbnails []*bytes.Buffer) {
	images = make([]*bytes.Buffer, len(response.Images))

	for idx, image := range response.Images {
		decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
		if decodeErr != nil {
			log.Printf("Error decoding image: %v\n", decodeErr)
		}

		images[idx] = bytes.NewBuffer(decodedImage)
	}

	if queue.ControlnetItem.MessageAttachment != nil {
		decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(queue.ControlnetItem.MessageAttachment.Image))
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnails = append(thumbnails, bytes.NewBuffer(decodedBytes))
	}

	if queue.Img2ImgItem.MessageAttachment != nil {
		decodedBytes, err := base64.StdEncoding.DecodeString(safeDereference(queue.Img2ImgItem.MessageAttachment.Image))
		if err != nil {
			log.Printf("Error decoding image: %v\n", err)
		}
		thumbnails = append(thumbnails, bytes.NewBuffer(decodedBytes))
	}

	generation := queue.ImageGenerationRequest
	totalImages := totalImageCount(generation)
	if len(images) > totalImages {
		log.Printf("received extra images: len(imageBufs): %v, controlnet: %v", len(images), queue.ControlnetItem.Enabled)
		thumbnails = append(thumbnails, images[totalImages:]...)
	}

	return images, thumbnails
}

func (q *queueImplementation) textInference(queue *entities.QueueItem) (response *stable_diffusion_api.TextToImageResponse, err error) {
	generation := queue.ImageGenerationRequest
	switch queue.Type {
	case ItemTypeRaw:
		if q.currentImagine.Raw.Unsafe {
			response, err = q.stableDiffusionAPI.TextToImageRaw(q.currentImagine.Raw.Blob)
		} else {
			marshal, marshalErr := q.currentImagine.Raw.Marshal()
			if marshalErr != nil {
				log.Printf("Error marshalling raw: %v", marshalErr)
				return nil, marshalErr
			}
			response, err = q.stableDiffusionAPI.TextToImageRaw(marshal)
		}
	default:
		response, err = q.stableDiffusionAPI.TextToImageRequest(generation.TextToImageRequest)
	}
	return response, err
}

func (q *queueImplementation) recordToRepository(generation *entities.ImageGenerationRequest, err error) (*entities.ImageGenerationRequest, error) {
	var ok bool
	if generation.Prompt, ok = strings.CutSuffix(generation.Prompt, "{DEBUG}"); ok {
		byteArr, _ := generation.TextToImageRequest.Marshal()
		log.Printf("{DEBUG} TextToImageRequest: %v", string(byteArr))
	}

	// return newGeneration from image_generations.Create as we need newGeneration.CreatedAt later on
	generation, err = q.imageGenerationRepo.Create(context.Background(), generation)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
		return nil, err
	}
	return generation, nil
}

func (q *queueImplementation) updateProgressBar(queue *entities.QueueItem, generationDone chan bool, config, originalConfig *entities.Config, webhook *discordgo.WebhookEdit) {
	request := queue.ImageGenerationRequest
	for {
		select {
		case queue.DiscordInteraction = <-queue.Interrupt:
			err := q.stableDiffusionAPI.Interrupt()
			if err != nil {
				handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error interrupting: %v", err))
				return
			}
			message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, queue.DiscordInteraction, "Generation Interrupted", webhook, handlers.Components[handlers.DeleteGeneration])
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
				handlers.Errors[handlers.ErrorResponse](q.botSession, queue.DiscordInteraction, fmt.Sprintf("Error getting current progress: %v", progressErr))
				return
			}

			if progress.Progress == 0 {
				continue
			}

			progressContent := imagineMessageSimple(request, queue.DiscordInteraction.Member.User, progress.Progress)

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
