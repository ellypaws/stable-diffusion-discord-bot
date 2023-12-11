package discord_bot

import (
	"cmp"
	"encoding/json"
	"fmt"
	"github.com/SpenserCai/sd-webui-discord/utils"
	"github.com/bwmarrin/discordgo"
	"github.com/sahilm/fuzzy"
	"io"
	"log"
	"net/http"
	"regexp"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/stable_diffusion_api"
	"strconv"
	"strings"
)

var commandHandlers = map[Command]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	helloCommand: func(b *botImpl, bot *discordgo.Session, i *discordgo.InteractionCreate) {
		handlers.Responses[handlers.HelloResponse].(handlers.NewResponseType)(bot, i)
	},
	imagineCommand:         (*botImpl).processImagineCommand,
	imagineSettingsCommand: (*botImpl).processImagineSettingsCommand,
	refreshCommand:         (*botImpl).processRefreshCommand,
	rawCommand:             (*botImpl).processRawCommand,
}

var autocompleteHandlers = map[Command]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	imagineCommand: (*botImpl).processImagineAutocomplete,
}

var modalHandlers = map[Command]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate){
	rawCommand: (*botImpl).processRawModal,
}

func getOpts(data discordgo.ApplicationCommandInteractionData) map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption {
	options := data.Options
	optionMap := make(map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[CommandOption(opt.Name)] = opt
	}
	return optionMap
}

// keyValue matches --key value, --key=value, or --key "value with spaces"
var keyValue = regexp.MustCompile(`\B(?:--|—)+(\w+)(?:[ =]([\w./\\:]+|"[^"]+"))?`)

func extractKeyValuePairsFromPrompt(prompt string) (parameters map[CommandOption]string, sanitized string) {
	parameters = make(map[CommandOption]string)
	sanitized = keyValue.ReplaceAllString(prompt, "")
	sanitized = strings.TrimSpace(sanitized)
	for _, match := range keyValue.FindAllStringSubmatch(prompt, -1) {
		parameters[CommandOption(match[1])] = match[2]
	}
	return
}

// If FieldType and ValueType are the same, then we attempt to assert FieldType to value.Value
// Otherwise, we return the interface conversion to the caller to do manual type conversion
//
// Example:
//
//	if int64Val, ok := interfaceConvertAuto[int, int64](&queue.Steps, stepOption, optionMap, parameters); ok {
//		queue.Steps = int(*int64Val)
//	}
//
// (*discordgo.ApplicationCommandInteractionDataOption).IntValue() actually uses float64 for the interface conversion, so use float64 for integers, numbers, etc.
// and then convert to the desired type.
func interfaceConvertAuto[F any, V string | float64](field *F, option CommandOption, optionMap map[CommandOption]*discordgo.ApplicationCommandInteractionDataOption, parameters map[CommandOption]string) (*V, bool) {
	if field == nil {
		log.Printf("WARNING: field %T is nil", field)
	}
	if value, ok := optionMap[option]; ok {
		vToField, ok := value.Value.(F)
		if ok && field != nil {
			*field = vToField
		}
		valueType, ok := value.Value.(V)
		return &valueType, ok
	}
	if value, ok := parameters[option]; ok {
		if field != nil {
			_, err := fmt.Sscanf(value, "%v", field)
			if err != nil {
				return nil, false
			}
		}
		var out V
		_, err := fmt.Sscanf(value, "%v", &out)
		if err != nil {
			return nil, false
		}
		return &out, true
	}
	return nil, false
}

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)

	optionMap := getOpts(i.ApplicationCommandData())

	var position int

	var queue *entities.QueueItem

	if option, ok := optionMap[promptOption]; !ok {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide a prompt.")
		return
	} else {
		parameters, sanitized := extractKeyValuePairsFromPrompt(option.StringValue())
		queue = b.imagineQueue.NewQueueItem(imagine_queue.WithPrompt(sanitized))

		queue.Type = imagine_queue.ItemTypeImagine
		queue.DiscordInteraction = i.Interaction

		if _, ok := interfaceConvertAuto[string, string](&queue.NegativePrompt, negativeOption, optionMap, parameters); ok {
			queue.NegativePrompt = strings.ReplaceAll(queue.NegativePrompt, "{DEFAULT}", imagine_queue.DefaultNegative)
		}

		interfaceConvertAuto[string, string](&queue.SamplerName, samplerOption, optionMap, parameters)

		if floatVal, ok := interfaceConvertAuto[int, float64](&queue.Steps, stepOption, optionMap, parameters); ok {
			queue.Steps = int(*floatVal)
		}

		if floatVal, ok := interfaceConvertAuto[int64, float64](&queue.Seed, seedOption, optionMap, parameters); ok {
			queue.Seed = int64(*floatVal)
		}

		if boolVal, ok := interfaceConvertAuto[bool, string](&queue.RestoreFaces, restoreFacesOption, optionMap, parameters); ok {
			boolean, err := strconv.ParseBool(*boolVal)
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			} else {
				queue.RestoreFaces = boolean
			}
		}

		interfaceConvertAuto[string, string](&queue.ADetailerString, adModelOption, optionMap, parameters)

		if config, err := b.StableDiffusionApi.GetConfig(); err != nil {
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error retrieving config.", err)
		} else {
			queue.Checkpoint = config.SDModelCheckpoint
			queue.VAE = config.SDVae
			queue.Hypernetwork = config.SDHypernetwork
		}

		interfaceConvertAuto[string, string](queue.Checkpoint, checkpointOption, optionMap, parameters)
		interfaceConvertAuto[string, string](queue.VAE, vaeOption, optionMap, parameters)
		interfaceConvertAuto[string, string](queue.Hypernetwork, hypernetworkOption, optionMap, parameters)

		if option, ok := optionMap[embeddingOption]; ok {
			queue.Prompt += " " + option.StringValue()
			log.Printf("Adding embedding: %v", option.StringValue())
		}

		for i := 0; i < extraLoras+1; i++ {
			loraKey := loraOption
			if i != 0 {
				loraKey += CommandOption(fmt.Sprintf("%d", i+1))
			}

			if option, ok := optionMap[loraKey]; ok {
				loraValue := option.StringValue()
				if loraValue != "" {

					loraValue = sanitizeTooltip(loraValue)

					// add :1 if no strength is specified
					strength := regexp.MustCompile(`:([\d.]+)$`)
					if !strength.MatchString(loraValue) {
						loraValue += ":1"
					}
					re := regexp.MustCompile(`.+\\|\.safetensors`)
					loraValue = re.ReplaceAllString(loraValue, "")
					lora := ", <lora:" + loraValue + ">"
					log.Println("Adding lora: ", lora)
					queue.Prompt += lora
				}
			}
		}

		interfaceConvertAuto[string, string](&queue.AspectRatio, aspectRatio, optionMap, parameters)

		if floatVal, ok := interfaceConvertAuto[float64, string](&queue.HrScale, hiresFixSize, optionMap, parameters); ok {
			float, err := strconv.ParseFloat(*floatVal, 64)
			if err != nil {
				log.Printf("Error parsing hiresUpscaleRate: %v", err)
			} else {
				queue.HrScale = between(float, 1.0, 4.0)
				queue.EnableHr = true
			}
		}

		if boolVal, ok := interfaceConvertAuto[bool, string](&queue.EnableHr, hiresFixOption, optionMap, parameters); ok {
			boolean, err := strconv.ParseBool(*boolVal)
			if err != nil {
				log.Printf("Error parsing hiresFix value: %v.", err)
			} else {
				queue.EnableHr = boolean
			}
		}

		interfaceConvertAuto[float64, float64](&queue.CFGScale, cfgScaleOption, optionMap, parameters)

		// calculate batch count and batch size. prefer batch size to be the bigger number, both numbers should add up to 4.
		// if batch size is 4, then batch count should be 1. if both are 4, set batch size to 4 and batch count to 1.
		// if batch size is 1, then batch count *can* be 4, but it can also be 1.

		if floatVal, ok := interfaceConvertAuto[int, float64](&queue.NIter, batchCountOption, optionMap, parameters); ok {
			queue.NIter = int(*floatVal)
		}

		if intVal, ok := interfaceConvertAuto[int, float64](&queue.BatchSize, batchSizeOption, optionMap, parameters); ok {
			queue.BatchSize = int(*intVal)
		}

		const maxImages = 4
		queue.BatchSize = between(queue.BatchSize, 1, maxImages)
		queue.NIter = min(maxImages/queue.BatchSize, queue.NIter)

		if boolVal, ok := interfaceConvertAuto[bool, string](&queue.RestoreFaces, restoreFacesOption, optionMap, parameters); ok {
			boolean, err := strconv.ParseBool(*boolVal)
			if err != nil {
				log.Printf("Error parsing restoreFaces value: %v.", err)
			} else {
				queue.RestoreFaces = boolean
			}
		}

		if i.ApplicationCommandData().Resolved != nil {
			if attachments := i.ApplicationCommandData().Resolved.Attachments; attachments != nil {
				if queue.Attachments == nil {
					queue.Attachments = make(map[string]*entities.MessageAttachment, len(attachments))
				}
				for snowflake, attachment := range attachments {
					queue.Attachments[snowflake] = &entities.MessageAttachment{
						MessageAttachment: *attachment,
					}
					log.Printf("Attachment[%v]: %#v", snowflake, attachment.URL)
					if !strings.HasPrefix(attachment.ContentType, "image") {
						log.Printf("Attachment[%v] is not an image, removing from queue.", snowflake)
						delete(queue.Attachments, snowflake)
					}

					image, err := utils.GetImageBase64(attachment.URL)
					if err != nil {
						log.Printf("Error getting image from URL: %v", err)
						handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error getting image from URL.", err)
						return
					}
					queue.Attachments[snowflake].Image = &image
				}
			}
		}

		if option, ok := optionMap[img2imgOption]; ok {
			if attachment, ok := queue.Attachments[option.Value.(string)]; !ok {
				handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide an image to img2img.")
				return
			} else {
				queue.Type = imagine_queue.ItemTypeImg2Img

				queue.Img2ImgItem.MessageAttachment = attachment

				if option, ok := optionMap[denoisingOption]; ok {
					queue.TextToImageRequest.DenoisingStrength = option.FloatValue()
					queue.Img2ImgItem.DenoisingStrength = option.FloatValue()
				}
			}
		}

		if option, ok := optionMap[controlnetImage]; ok {
			if attachment, ok := queue.Attachments[option.Value.(string)]; ok {
				queue.ControlnetItem.MessageAttachment = attachment
			} else {
				handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide an image to controlnet.")
				return
			}
			queue.ControlnetItem.Enabled = true
		}

		if controlVal, ok := interfaceConvertAuto[entities.ControlMode, string](&queue.ControlnetItem.ControlMode, controlnetControlMode, optionMap, parameters); ok {
			queue.ControlnetItem.ControlMode = entities.ControlMode(*controlVal)
			queue.ControlnetItem.Enabled = true
		}

		if resizeVal, ok := interfaceConvertAuto[entities.ResizeMode, string](&queue.ControlnetItem.ResizeMode, controlnetResizeMode, optionMap, parameters); ok {
			queue.ControlnetItem.ResizeMode = entities.ResizeMode(*resizeVal)
			queue.ControlnetItem.Enabled = true
		}

		if _, ok := interfaceConvertAuto[string, string](&queue.ControlnetItem.Type, controlnetType, optionMap, parameters); ok {
			log.Printf("Controlnet type: %v", queue.ControlnetItem.Type)
			cache, err := stable_diffusion_api.ControlnetTypesCache.GetCache(b.StableDiffusionApi)
			if err != nil {
				log.Printf("Error retrieving controlnet types cache: %v", err)
			} else {
				// set default preprocessor and model
				if types, ok := cache.(*stable_diffusion_api.ControlnetTypes).ControlTypes[queue.ControlnetItem.Type]; ok {
					queue.ControlnetItem.Preprocessor = types.DefaultOption
					queue.ControlnetItem.Model = types.DefaultModel
				}
			}
			queue.ControlnetItem.Enabled = true
		}

		if _, ok := interfaceConvertAuto[string, string](&queue.ControlnetItem.Preprocessor, controlnetPreprocessor, optionMap, parameters); ok {
			//queue.ControlnetItem.Preprocessor = *preprocessor
			queue.ControlnetItem.Enabled = true
		}

		if _, ok := interfaceConvertAuto[string, string](&queue.ControlnetItem.Model, controlnetModel, optionMap, parameters); ok {
			//queue.ControlnetItem.Model = *model
			queue.ControlnetItem.Enabled = true
		}

		interfaceConvertAuto[float64, float64](&queue.OverrideSettings.CLIPStopAtLastLayers, clipSkipOption, optionMap, parameters)

		if floatVal, ok := interfaceConvertAuto[float64, float64](nil, cfgRescaleOption, optionMap, parameters); ok {
			queue.CFGRescale = &entities.CFGRescale{
				Args: entities.CFGRescaleParameters{
					CfgRescale:   *floatVal,
					AutoColorFix: false,
					FixStrength:  0,
					KeepOriginal: false,
				},
			}
		}

		var err error
		position, err = b.imagineQueue.AddImagine(queue)
		if err != nil {
			log.Printf("Error adding imagine to queue: %v\n", err)
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error adding imagine to queue.", err)
		}
	}

	var snowflake string

	switch {
	case i.Member != nil:
		snowflake = i.Member.User.ID
	case i.User != nil:
		snowflake = i.User.ID
	}

	queueString := fmt.Sprintf(
		"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \n```\n%s\n```",
		position,
		snowflake,
		queue.Prompt,
	)

	message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, queueString, handlers.Components[handlers.Cancel])
	if queue.DiscordInteraction != nil && queue.DiscordInteraction.Message == nil && message != nil {
		log.Printf("Setting message ID for interaction %v", queue.DiscordInteraction.ID)
		queue.DiscordInteraction.Message = message
	}
}

func between[T cmp.Ordered](value, minimum, maximum T) T {
	return min(max(minimum, value), maximum)
}

var weightRegex = regexp.MustCompile(`.+\\|\.(?:safetensors|ckpt|pth?)|(:[\d.]+$)`)

func (b *botImpl) processImagineAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	log.Printf("running autocomplete handler")
	for optionIndex, opt := range data.Options {
		if !opt.Focused {
			continue
		}
		input := opt.StringValue()
		switch {
		case strings.HasPrefix(opt.Name, string(loraOption)):
			log.Printf("Focused option (%v): %v", optionIndex, opt.Name)

			var choices []*discordgo.ApplicationCommandOptionChoice

			if input != "" {
				log.Printf("Autocompleting '%v'", input)

				input = sanitizeTooltip(input)

				cache, err := b.StableDiffusionApi.SDLorasCache()
				if err != nil {
					log.Printf("Error retrieving loras cache: %v", err)
				}

				sanitized := weightRegex.ReplaceAllString(input, "")

				log.Printf("looking up lora: %v", sanitized)
				results := fuzzy.FindFrom(sanitized, cache)

				for index, result := range results {
					if index > 25 {
						break
					}
					regExp := regexp.MustCompile(`(?:models\\)?Lora\\(.*)`)

					alias := regExp.FindStringSubmatch((*cache)[result.Index].Path)

					var nameToUse string
					switch {
					case alias != nil && alias[1] != "":
						// replace double slash with single slash
						regExp := regexp.MustCompile(`\\{2,}`)
						nameToUse = regExp.ReplaceAllString(alias[1], `\`)
					default:
						nameToUse = (*cache)[result.Index].Name
					}

					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  nameToUse,
						Value: (*cache)[result.Index].Name,
					})
				}

				weightMatches := weightRegex.FindAllStringSubmatch(input, -1)
				log.Printf("weightMatches: %v", weightMatches)

				var tooltip string
				if len(results) > 0 {
					input = (*cache)[results[0].Index].Name
					tooltip = fmt.Sprintf("✨%v", input)
				} else {
					input = sanitized
					tooltip = fmt.Sprintf("❌%v", input)
				}

				if weightMatches != nil && weightMatches[len(weightMatches)-1][1] != "" {
					weight := weightMatches[len(weightMatches)-1][1]
					input += weight
					tooltip += fmt.Sprintf(" 🪄%v", weight)
				} else {
					tooltip += " 🪄1 (𝗱𝗲𝗳𝗮𝘂𝗹𝘁)"
				}

				log.Printf("Name: (tooltip) %v\nValue: (input) %v", tooltip, input)
				choices = append(choices[:min(24, len(choices))], &discordgo.ApplicationCommandOptionChoice{
					Name:  tooltip,
					Value: input,
				})

				//choices = append(choices[:min(24, len(choices))], &discordgo.ApplicationCommandOptionChoice{
				//	Name:  input,
				//	Value: input,
				//})
			} else {
				choices = []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Type a lora name. Add a colon after to specify the strength. (e.g. \"clay:0.5\")",
						Value: "placeholder",
					},
				}
			}

			// make sure we're under 100 char limit and under 25 choices
			for i, choice := range choices {
				if len(choice.Name) > 100 {
					// TODO: check if discord counts bytes or chars
					choices[i].Name = choice.Name[:100]
				}
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: choices[:min(25, len(choices))], // This is basically the whole purpose of autocomplete interaction - return custom options to the user.
				},
			})
		default:
			switch CommandOption(opt.Name) {
			case checkpointOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.CheckpointCache)
			case vaeOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.VAECache)
			case hypernetworkOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.HypernetworkCache)
			case embeddingOption:
				b.autocompleteModels(s, i, optionIndex, opt, input, stable_diffusion_api.EmbeddingCache)
			case controlnetPreprocessor:
				b.autocompleteControlnet(s, i, optionIndex, opt, input, stable_diffusion_api.ControlnetModulesCache)
			case controlnetModel:
				b.autocompleteControlnet(s, i, optionIndex, opt, input, stable_diffusion_api.ControlnetModelsCache)
			}
		}
		break
	}
}

func (b *botImpl) autocompleteModels(s *discordgo.Session, i *discordgo.InteractionCreate, index int, opt *discordgo.ApplicationCommandInteractionDataOption, input string, c stable_diffusion_api.Cacheable) {
	log.Printf("Focused option (%v): %v", index, opt.Name)
	input = opt.StringValue()

	var choices []*discordgo.ApplicationCommandOptionChoice

	if input != "" {
		if c == nil {
			log.Printf("Cacheable interface is nil")
		}
		log.Printf("Autocompleting '%v'", input)

		cache, err := c.GetCache(b.StableDiffusionApi)
		if err != nil {
			log.Printf("Error retrieving %v cache: %v", opt.Name, err)
		}
		results := fuzzy.FindFrom(input, cache)
		//log.Printf("Finding from %v: %v", input, cache)
		//log.Printf("Cache: %v, cache.len(): %v", cache, cache.Len())
		//log.Printf("Results: %v", results)

		for index, result := range results {
			// Match against String() method according to fuzzy docs
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  cache.String(result.Index),
				Value: cache.String(result.Index),
			})
			if index >= 25 {
				break
			}
		}
	} else {
		choices = []*discordgo.ApplicationCommandOptionChoice{
			{
				Name:  fmt.Sprintf("Type the %[1]v name. You can also attempt to fuzzy match the %[1]v.", opt.Name),
				Value: "placeholder",
			},
		}
	}

	// make sure we're under 100 char limit and under 25 choices
	for i, choice := range choices {
		if len(choice.Name) > 100 {
			// TODO: check if discord counts bytes or chars
			choices[i].Name = choice.Name[:100]
		}
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))],
		},
	})
}

func (b *botImpl) autocompleteControlnet(s *discordgo.Session, i *discordgo.InteractionCreate, index int, opt *discordgo.ApplicationCommandInteractionDataOption, input string, c stable_diffusion_api.Cacheable) {
	input = opt.StringValue()

	// check the Type first
	optionMap := getOpts(i.ApplicationCommandData())

	cache, err := stable_diffusion_api.ControlnetTypesCache.GetCache(b.StableDiffusionApi)
	if err != nil {
		log.Printf("Error retrieving %v cache: %v", opt.Name, err)
		return
	}
	controlnets := cache.(*stable_diffusion_api.ControlnetTypes)

	log.Printf("Focused option (%v): %v", index, opt.Name)

	var toSearch []string
	var controlType = "All"
	option, ok := optionMap[controlnetType]
	if ok {
		controlType = option.StringValue()
	}

	if types, ok := controlnets.ControlTypes[controlType]; ok {
		switch c.(type) {
		case *stable_diffusion_api.ControlnetModules:
			toSearch = types.ModuleList
		case *stable_diffusion_api.ControlnetModels:
			toSearch = types.ModelList
		}
	} else {
		log.Printf("No controlnet types found for %v: %v", opt.Name, option.StringValue())
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	if input != "" {
		if len(toSearch) == 0 {
			log.Printf("No controlnet types found for %v", opt.Name)
		}
		log.Printf("Autocompleting '%v'", input)

		results := fuzzy.Find(input, toSearch)
		//log.Printf("Finding from %v: %v", input, cache)
		//log.Printf("Cache: %v, cache.len(): %v", cache, cache.Len())
		//log.Printf("Results: %v", results)

		for index, result := range results {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  toSearch[result.Index],
				Value: toSearch[result.Index],
			})
			if index >= 25 {
				break
			}
		}
	} else {
		for index, item := range toSearch {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  item,
				Value: item,
			})
			if index >= 25 {
				break
			}
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices[:min(25, len(choices))],
		},
	})
}

func sanitizeTooltip(input string) string {
	tooltipRegex := regexp.MustCompile(`[✨❌](.+) 🪄:([\d.]+)$|[✨❌](.+)`)
	sanitizedTooltip := tooltipRegex.FindStringSubmatch(input)

	if sanitizedTooltip != nil {
		log.Printf("Removing tooltip: %#v", sanitizedTooltip)

		switch {
		case sanitizedTooltip[1] != "":
			input = sanitizedTooltip[1] + ":" + sanitizedTooltip[2]
		case sanitizedTooltip[3] != "":
			input = sanitizedTooltip[3]
		}
		log.Printf("Sanitized input: %v", input)
	}
	return input
}

func (b *botImpl) processImagineSettingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)
	botSettings, err := b.imagineQueue.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error getting default settings for settings command: %v", err)

		return
	}

	messageComponents := b.settingsMessageComponents(botSettings)

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction,
		"Choose default settings for the imagine command:",
		messageComponents,
	)
	//err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	//	Type: discordgo.InteractionResponseChannelMessageWithSource,
	//	Data: &discordgo.InteractionResponseData{
	//		Title:      "Settings",
	//		Content:    "Choose default settings for the imagine command:",
	//		Components: messageComponents,
	//	},
	//})
	//if err != nil {
	//	log.Printf("Error responding to interaction: %v", err)
	//}
}

func (b *botImpl) processRefreshCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)

	var errors []error
	var content = strings.Builder{}

	var toRefresh []stable_diffusion_api.Cacheable

	switch CommandOption("refresh_" + i.ApplicationCommandData().Options[0].Name) {
	case refreshLoraOption:
		toRefresh = []stable_diffusion_api.Cacheable{stable_diffusion_api.LoraCache}
	case refreshCheckpoint:
		toRefresh = []stable_diffusion_api.Cacheable{stable_diffusion_api.CheckpointCache}
	case refreshVAEOption:
		toRefresh = []stable_diffusion_api.Cacheable{stable_diffusion_api.VAECache}
	case refreshAllOption:
		toRefresh = []stable_diffusion_api.Cacheable{
			stable_diffusion_api.LoraCache,
			stable_diffusion_api.CheckpointCache,
			stable_diffusion_api.VAECache,
		}
	}

	for _, cache := range toRefresh {
		newCache, err := b.StableDiffusionApi.RefreshCache(cache)
		if err != nil || newCache == nil {
			errors = append(errors, err)
			content.WriteString(fmt.Sprintf("`%T` cache refresh failed.\n", cache))
			continue
		}
		content.WriteString(fmt.Sprintf("`%T` cache refreshed. %v items loaded.\n", newCache, newCache.Len()))
		handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, content.String())
	}

	if errors != nil {
		log.Printf("Error refreshing cache: %v", errors)
		handlers.Errors[handlers.ErrorFollowup](s, i.Interaction, "Error refreshing cache.", errors)
	}

	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(s, i.Interaction, content.String())
}

// processRawCommand responds with a Modal to receive a json blob from the user to pass to the api
func (b *botImpl) processRawCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	optionMap := getOpts(i.ApplicationCommandData())

	params := entities.RawParams{
		UseDefault: true,
		Unsafe:     false,
	}
	if option, ok := optionMap[useDefaults]; ok {
		params.UseDefault = option.BoolValue()
	}

	if option, ok := optionMap[unsafeOption]; ok {
		params.Unsafe = option.BoolValue()
	}

	interactionBytes, _ := json.Marshal(i.Interaction)
	log.Printf("Interaction: %v", string(interactionBytes))

	var snowflake string
	if option, ok := optionMap[jsonFile]; !ok {
		// if no json file is provided, we need to respond with a modal to get the json blob from the user
		modalDefault[i.ID] = params
		log.Printf("modalDefault: %v", modalDefault)
		interactionResponse := discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: string(rawCommand),
				Title:    "Raw JSON",
				Components: []discordgo.MessageComponent{
					handlers.Components[handlers.JSONInput],
				},
			},
		}
		err := s.InteractionRespond(i.Interaction, &interactionResponse)
		if err != nil {
			delete(modalDefault, i.ID)
			log.Printf("Error responding to interaction: %v", err)

			byteArr, err := json.Marshal(interactionResponse)
			if err != nil {
				log.Printf("Error marshalling interaction response data: %v", err)
			}
			log.Printf("Raw JSON: %v", string(byteArr))
		}
		return
	} else {
		snowflake = option.Value.(string)
	}

	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)
	attachments := i.ApplicationCommandData().Resolved.Attachments
	if attachments == nil {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide a JSON file.")
		return
	}

	for snowflake, attachment := range attachments {
		log.Printf("Attachment[%v]: %#v", snowflake, attachment.URL)
		if !strings.HasPrefix(attachment.ContentType, "application/json") {
			log.Printf("Attachment[%v] is not a json file, removing from queue.", snowflake)
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide a JSON file.")
			return
		}
	}

	attachment, ok := attachments[snowflake]
	if !ok {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide a JSON file.")
		return
	}

	// download attachment url using http and convert to []byte
	resp, err := http.Get(attachment.URL)
	if err != nil {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error downloading attachment.", err)
		return
	}
	defer resp.Body.Close()
	if params.Blob, err = io.ReadAll(resp.Body); err != nil {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error reading attachment.", err)
		return
	}

	params.Debug = strings.Contains(attachment.Filename, "DEBUG")
	if err := b.jsonToQueue(i, params); err != nil {
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error adding imagine to queue.", err)
		return
	}
}

var modalDefault = make(map[string]entities.RawParams)

//	type TextInput struct {
//	   CustomID    string         `json:"custom_id"`
//	   Label       string         `json:"label"`
//	   Style       TextInputStyle `json:"style"`
//	   Placeholder string         `json:"placeholder,omitempty"`
//	   Value       string         `json:"value,omitempty"`
//	   Required    bool           `json:"required"`
//	   MinLength   int            `json:"min_length,omitempty"`
//	   MaxLength   int            `json:"max_length,omitempty"`
//	}
func getModalData(data discordgo.ModalSubmitInteractionData) map[handlers.Component]*discordgo.TextInput {
	var options = make(map[handlers.Component]*discordgo.TextInput)
	for _, actionRow := range data.Components {
		for _, c := range actionRow.(*discordgo.ActionsRow).Components {
			switch c := c.(type) {
			case *discordgo.TextInput:
				options[handlers.Component(c.CustomID)] = c
			default:
				log.Fatalf("Wrong component type: %T, skipping...", c)
			}
		}
	}
	return options
}

func (b *botImpl) processRawModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlers.Responses[handlers.ThinkResponse].(handlers.NewResponseType)(s, i)

	modalData := getModalData(i.ModalSubmitData())

	var params entities.RawParams
	if message, err := b.botSession.InteractionResponse(i.Interaction); err != nil {
		if p, ok := modalDefault[message.Interaction.ID]; ok {
			params = p
			delete(modalDefault, message.Interaction.ID)
		}
	}

	if data, ok := modalData[handlers.JSONInput]; !ok || data == nil || data.Value == "" {
		log.Printf("modalData: %#v\n", modalData)
		log.Printf("i.ModalSubmitData(): %#v\n", i.ModalSubmitData())
		handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "You need to provide a JSON blob.")
		return
	} else {
		params.Debug = strings.Contains(data.Value, "{DEBUG}")
		params.Blob = []byte(strings.ReplaceAll(data.Value, "{DEBUG}", ""))
		if err := b.jsonToQueue(i, params); err != nil {
			handlers.Errors[handlers.ErrorResponse](s, i.Interaction, "Error adding imagine to queue.", err)
		}
	}
}

func (b *botImpl) jsonToQueue(i *discordgo.InteractionCreate, params entities.RawParams) error {
	queue := &entities.QueueItem{}
	if params.UseDefault {
		queue = b.imagineQueue.NewQueueItem()
	}

	queue.Type = imagine_queue.ItemTypeRaw
	queue.DiscordInteraction = i.Interaction

	queue.Raw = &entities.TextToImageRaw{TextToImageRequest: queue.TextToImageRequest, RawParams: params}

	// Override Scripts by unmarshalling to Raw
	err := json.Unmarshal(params.Blob, &queue.Raw)
	if err != nil {
		return err
	}

	position, err := b.imagineQueue.AddImagine(queue)
	if err != nil {
		return err
	}
	handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(b.botSession, i.Interaction,
		fmt.Sprintf("I'm dreaming something up for you. You are currently #%d in line.", position),
	)
	return nil
}
