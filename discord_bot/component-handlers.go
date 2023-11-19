package discord_bot

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"regexp"
	"slices"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/stable_diffusion_api"
	"strconv"
	"strings"
	"time"
)

var componentHandlers = map[string]func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	handlers.DeleteButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			handlers.ErrorEphemeralResponse(s, i.Interaction, err)
		}
	},

	handlers.DeleteGeneration: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		handlers.Responses[handlers.EphemeralThink].(handlers.NewResponseType)(s, i)
		msg, _ := bot.botSession.ChannelMessage(i.ChannelID, i.Message.ID)

		content := msg.Content
		userRegex := regexp.MustCompile("<@!?(\\d+)>")
		userID := userRegex.FindStringSubmatch(content)[1]

		if userID != i.Member.User.ID {
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You can only delete your own generations")
			return
		}
		err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
		if err != nil {
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, err)
			return
		}

		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, "Generation deleted")
	},

	handlers.DeleteAboveButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		// delete the original interaction response
		msg, err := s.InteractionResponse(i.Interaction)
		if i == nil || err != nil {
			handlers.ErrorEphemeralResponse(s, i.Interaction, fmt.Errorf("failed to retrieve interaction response: %v, %v", i, err))
			return
		}

		err = s.ChannelMessageDelete(i.ChannelID, msg.ID)

		if err != nil {
			handlers.ErrorEphemeralResponse(s, i.Interaction, err)
		}
	},

	handlers.DimensionSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(i.MessageComponentData().Values) == 0 {
			log.Printf("No values for imagine dimension setting menu")

			return
		}

		sizes := strings.Split(i.MessageComponentData().Values[0], "_")

		width := sizes[0]
		height := sizes[1]

		widthInt, err := strconv.Atoi(width)
		if err != nil {
			log.Printf("Error parsing width: %v", err)

			return
		}

		heightInt, err := strconv.Atoi(height)
		if err != nil {
			log.Printf("Error parsing height: %v", err)

			return
		}

		bot.processImagineDimensionSetting(s, i, widthInt, heightInt)
	},

	handlers.CheckpointSelect: (*botImpl).processImagineSDModelNameSetting,
	//handlers.VAESelect:          (*botImpl).processImagineVAESetting,
	//handlers.HypernetworkSelect: (*botImpl).processImagineHypernetworkSetting,

	handlers.BatchCountSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(i.MessageComponentData().Values) == 0 {
			log.Printf("No values for imagine batch count setting menu")

			return
		}

		batchCount := i.MessageComponentData().Values[0]

		batchCountInt, err := strconv.Atoi(batchCount)
		if err != nil {
			log.Printf("Error parsing batch count: %v", err)

			return
		}

		var batchSizeInt int

		// calculate the corresponding batch size
		switch batchCountInt {
		case 1:
			batchSizeInt = 4
		case 2:
			batchSizeInt = 2
		case 4:
			batchSizeInt = 1
		default:
			log.Printf("Unknown batch count: %v", batchCountInt)

			return
		}

		bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
	},

	handlers.BatchSizeSelect: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		if len(i.MessageComponentData().Values) == 0 {
			log.Printf("No values for imagine batch count setting menu")

			return
		}

		batchSize := i.MessageComponentData().Values[0]

		batchSizeInt, err := strconv.Atoi(batchSize)
		if err != nil {
			log.Printf("Error parsing batch count: %v", err)

			return
		}

		var batchCountInt int

		// calculate the corresponding batch count
		switch batchSizeInt {
		case 1:
			batchCountInt = 4
		case 2:
			batchCountInt = 2
		case 4:
			batchCountInt = 1
		default:
			log.Printf("Unknown batch size: %v", batchSizeInt)

			return
		}

		bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
	},

	handlers.RerollButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		bot.processImagineReroll(s, i)
	},

	handlers.UpscaleButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		interactionIndex := strings.TrimPrefix(customID, handlers.UpscaleButton+"_")

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			log.Printf("Error parsing interaction index: %v", err)

			return
		}

		bot.processImagineUpscale(s, i, interactionIndexInt)
	},

	handlers.VariantButton: func(bot *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		interactionIndex := strings.TrimPrefix(customID, "imagine_variation_")

		interactionIndexInt, err := strconv.Atoi(interactionIndex)
		if err != nil {
			log.Printf("Error parsing interaction index: %v", err)

			return
		}

		bot.processImagineVariation(s, i, interactionIndexInt)
	},
}

func (b *botImpl) processImagineReroll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeReroll,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm reimagining that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineUpscale(s *discordgo.Session, i *discordgo.InteractionCreate, upscaleIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeUpscale,
		InteractionIndex:   upscaleIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm upscaling that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineVariation(s *discordgo.Session, i *discordgo.InteractionCreate, variationIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeVariation,
		InteractionIndex:   variationIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm imagining more variations for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

// patch from upstream
func (b *botImpl) settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	populateOption(b, handlers.CheckpointSelect, stable_diffusion_api.CheckpointCache)
	populateOption(b, handlers.VAESelect, stable_diffusion_api.VAECache)
	populateOption(b, handlers.HypernetworkSelect, stable_diffusion_api.HypernetworkCache)

	// set default dimension from config
	dimensions := handlers.Components[handlers.DimensionSelect].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	dimensions.Options[0].Default = settings.Width == 512 && settings.Height == 512
	dimensions.Options[1].Default = settings.Width == 768 && settings.Height == 768
	handlers.Components[handlers.DimensionSelect].(discordgo.ActionsRow).Components[0] = dimensions

	batchSlice := []int{1, 2, 4}
	// set default batch count from config
	batchCount := handlers.Components[handlers.BatchCountSelect].(discordgo.ActionsRow)
	for i, option := range batchCount.Components[0].(discordgo.SelectMenu).Options {
		if batchSlice[i] == settings.BatchCount {
			option.Default = true
		} else {
			option.Default = false
		}
		batchCount.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	handlers.Components[handlers.BatchCountSelect] = batchCount

	// set the default batch size from config
	batchSize := handlers.Components[handlers.BatchSizeSelect].(discordgo.ActionsRow)
	for i, option := range batchSize.Components[0].(discordgo.SelectMenu).Options {
		if batchSlice[i] == settings.BatchSize {
			option.Default = true
		} else {
			option.Default = false
		}
		batchSize.Components[0].(discordgo.SelectMenu).Options[i] = option
	}
	handlers.Components[handlers.BatchSizeSelect] = batchSize

	return []discordgo.MessageComponent{
		handlers.Components[handlers.CheckpointSelect],
		handlers.Components[handlers.VAESelect],
		handlers.Components[handlers.HypernetworkSelect],
		handlers.Components[handlers.DimensionSelect],
		//handlers.Components[handlers.BatchCountSelect],
		handlers.Components[handlers.BatchSizeSelect],
	}
}

// populateOption will fill in the options for a given dropdown component that implements stable_diffusion_api.Cacheable
func populateOption(b *botImpl, handler string, cache stable_diffusion_api.Cacheable) {
	checkpointDropdown := handlers.Components[handler].(discordgo.ActionsRow)
	var modelOptions []discordgo.SelectMenuOption

	models, err := cache.GetCache(b.StableDiffusionApi)
	if err != nil {
		fmt.Printf("Failed to retrieve list of models: %v\n", err)
		return
	} else {
		var modelNames []string
		var currentModel string

		switch toRange := models.(type) {
		case *stable_diffusion_api.SDModels:
			currentModel, err = b.StableDiffusionApi.GetCheckpoint()
			if err != nil {
				log.Printf("Failed to retrieve current model: %v", err)
				break
			}
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label:   shortenString(model.ModelName),
					Value:   shortenString(model.Title),
					Default: strings.Contains(currentModel, model.ModelName),
				})
				modelNames = append(modelNames, model.ModelName)
			}
		case *stable_diffusion_api.VAEModels:
			currentModel, err = b.StableDiffusionApi.GetVAE()
			if err != nil {
				log.Printf("Failed to retrieve current model: %v", err)
				break
			}
			for i, model := range *toRange {
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label:   shortenString(model.ModelName),
					Value:   shortenString(model.ModelName),
					Default: strings.Contains(currentModel, model.ModelName),
				})
				modelNames = append(modelNames, model.ModelName)
			}
		case *stable_diffusion_api.HypernetworkModels:
			for i, model := range *toRange {
				currentModel, err = b.StableDiffusionApi.GetHypernetwork()
				if err != nil {
					log.Printf("Failed to retrieve current model: %v", err)
					break
				}
				if i > 20 {
					break
				}
				modelOptions = append(modelOptions, discordgo.SelectMenuOption{
					Label:   shortenString(model.Name),
					Value:   shortenString(model.Name),
					Default: strings.Contains(currentModel, model.Name),
				})
				modelNames = append(modelNames, model.Name)
			}
		}

		if currentModel != "" && !slices.ContainsFunc(modelNames,
			func(model string) bool {
				return strings.Contains(currentModel, model)
			}) {
			modelOptions = append(modelOptions, discordgo.SelectMenuOption{
				Label:   shortenString(currentModel),
				Value:   shortenString(currentModel),
				Default: true,
			})
		}

		if len(modelOptions) == 0 {
			modelOptions = append(modelOptions, discordgo.SelectMenuOption{
				Label:       "No models found",
				Value:       "None",
				Description: "Are you sure you have the right API URL?",
				Default:     false,
			})
		} else {
			modelOptions = append([]discordgo.SelectMenuOption{{
				Label:       "None",
				Value:       "None",
				Description: "Unset the model",
			}}, modelOptions...)
		}
		component := checkpointDropdown.Components[0].(discordgo.SelectMenu)
		component.Options = modelOptions

		handlers.Components[handler].(discordgo.ActionsRow).Components[0] = component
	}
}

func (b *botImpl) processImagineDimensionSetting(s *discordgo.Session, i *discordgo.InteractionCreate, height, width int) {
	botSettings, err := b.imagineQueue.UpdateDefaultDimensions(width, height)
	if err != nil {
		log.Printf("error updating default dimensions: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating default dimensions...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
		return
	}
}

func (b *botImpl) processImagineBatchSetting(s *discordgo.Session, i *discordgo.InteractionCreate, batchCount, batchSize int) {
	botSettings, err := b.imagineQueue.UpdateDefaultBatch(batchCount, batchSize)
	if err != nil {
		log.Printf("error updating batch settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating batch settings...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose default settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineSDModelNameSetting(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(i.MessageComponentData().Values) == 0 {
		log.Printf("No values for %v", i.MessageComponentData().CustomID)
		return
	}
	newModelName := i.MessageComponentData().Values[0]

	botSettings, err := b.imagineQueue.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error retrieving bot settings: %v", err)
		handlers.Responses[handlers.ErrorEphemeral].(handlers.ErrorResponseType)(s, i.Interaction, "Error retrieving bot settings...")
		return
	}
	handlers.Responses[handlers.UpdateFromComponent].(handlers.MsgResponseType)(s, i.Interaction,
		fmt.Sprintf("Updating model to %v...", newModelName),
		b.settingsMessageComponents(botSettings),
	)
	err = b.StableDiffusionApi.UpdateConfiguration(stable_diffusion_api.APIConfig{SDModelCheckpoint: newModelName})
	if err != nil {
		log.Printf("error updating sd model name settings: %v", err)
		handlers.Responses[handlers.ErrorFollowupEphemeral].(handlers.ErrorResponseType)(s, i.Interaction, "Error updating sd model name settings...")

		return
	}

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction,
		fmt.Sprintf("Updated model to %v", newModelName),
		b.settingsMessageComponents(botSettings),
	)

	time.AfterFunc(5*time.Second, func() {
		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction,
			"Choose default settings for the imagine command:",
			b.settingsMessageComponents(botSettings),
		)
	})
}
