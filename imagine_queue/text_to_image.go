package imagine_queue

import (
	"encoding/json"
	"github.com/SpenserCai/sd-webui-discord/utils"
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	"strconv"
)

func (q *queueImplementation) processCurrentImagine() {
	go func() {
		defer q.done()

		if q.currentImagine.Type == ItemTypeUpscale {
			q.processUpscaleImagine(q.currentImagine)
			return
		}

		c := q.currentImagine

		newGeneration, err := &entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				Processed:    false,
				Checkpoint:   c.Checkpoint,
				VAE:          c.VAE,
				Hypernetwork: c.Hypernetwork,
			},
			TextToImageRequest: &entities.TextToImageRequest{
				Prompt:            c.Prompt,
				NegativePrompt:    c.NegativePrompt,
				Width:             initializedWidth,
				Height:            initializedHeight,
				RestoreFaces:      c.RestoreFaces,
				EnableHr:          c.UseHiresFix,
				HrScale:           between(c.HiresUpscaleRate, 1.0, 2.0),
				HrUpscaler:        "R-ESRGAN 2x+",
				HrSecondPassSteps: c.HiresSteps,
				HrResizeX:         initializedWidth,
				HrResizeY:         initializedHeight,
				DenoisingStrength: c.DenoisingStrength,
				Seed:              c.Seed,
				Subseed:           -1,
				SubseedStrength:   0,
				SamplerName:       c.SamplerName1,
				CFGScale:          c.CfgScale,
				Steps:             c.Steps,
			},
		}, error(nil)

		newGeneration.Width, err = q.defaultWidth()
		if err != nil {
			log.Printf("Error getting default width: %v", err)
		}

		newGeneration.Height, err = q.defaultHeight()
		if err != nil {
			log.Printf("Error getting default height: %v", err)
		}

		// add optional parameter: Negative prompt
		if c.NegativePrompt == "" {
			newGeneration.NegativePrompt = defaultNegative
		}

		// add optional parameter: sampler
		if c.SamplerName1 == "" {
			newGeneration.SamplerName = "Euler a"
		}

		// extract key value pairs from prompt
		var parameters map[string]string
		parameters, newGeneration.Prompt = extractKeyValuePairsFromPrompt(newGeneration.Prompt)

		defaultWidth := newGeneration.Width
		defaultHeight := newGeneration.Height
		if c.AspectRatio != "" && c.AspectRatio != "1:1" {
			newGeneration.Width, newGeneration.Height = aspectRatioCalculation(c.AspectRatio, defaultWidth, defaultHeight)
		} else {
			if aspectRatio, ok := parameters["ar"]; ok {
				newGeneration.Width, newGeneration.Height = aspectRatioCalculation(aspectRatio, defaultWidth, defaultHeight)
			}
		}

		// extract --zoom parameter
		adjustedWidth := newGeneration.Width
		adjustedHeight := newGeneration.Height
		if newGeneration.EnableHr && newGeneration.HrScale > 1.0 {
			newGeneration.HrResizeX = int(float64(adjustedWidth) * newGeneration.HrScale)
			newGeneration.HrResizeY = int(float64(adjustedHeight) * newGeneration.HrScale)
		} else {
			newGeneration.EnableHr = false
			newGeneration.HrResizeX = adjustedWidth
			newGeneration.HrResizeY = adjustedHeight
		}

		if zoom, ok := parameters["zoom"]; ok {
			zoomScale, err := strconv.ParseFloat(zoom, 64)
			if err != nil {
				log.Printf("Error extracting zoom scale from prompt: %v", err)
			} else {
				newGeneration.EnableHr = true
				newGeneration.HrScale = between(zoomScale, 1.0, 2.0)
				newGeneration.HrResizeX = int(float64(adjustedWidth) * newGeneration.HrScale)
				newGeneration.HrResizeY = int(float64(adjustedHeight) * newGeneration.HrScale)
			}
		}

		if step, ok := parameters["step"]; ok {
			stepInt, err := strconv.Atoi(step)
			if err != nil {
				log.Printf("Error extracting step from prompt: %v", err)
			} else {
				newGeneration.Steps = stepInt
			}
		}

		if cfgscale, ok := parameters["cfgscale"]; ok {
			cfgScaleFloat, err := strconv.ParseFloat(cfgscale, 64)
			if err != nil {
				log.Printf("Error extracting cfg scale from prompt: %v", err)
			} else {
				newGeneration.CFGScale = cfgScaleFloat
			}
		}

		if seed, ok := parameters["seed"]; ok {
			seedInt, err := strconv.ParseInt(seed, 10, 64)
			if err != nil {
				log.Printf("Error extracting seed from prompt: %v", err)
			} else {
				newGeneration.Seed = seedInt
			}
		}

		config, err := q.stableDiffusionAPI.GetConfig()
		if err != nil {
			log.Printf("Error getting config: %v", err)
		} else {
			if !ptrStringNotBlank(newGeneration.Checkpoint) {
				newGeneration.Checkpoint = config.SDModelCheckpoint
			}
			if !ptrStringNotBlank(newGeneration.VAE) {
				newGeneration.VAE = config.SDVae
			}
			if !ptrStringNotBlank(newGeneration.Hypernetwork) {
				newGeneration.Hypernetwork = config.SDHypernetwork
			}
		}

		if c.ADetailerString != "" {
			log.Printf("q.currentImagine.ADetailerString: %v", c.ADetailerString)

			newGeneration.NewADetailer()

			newGeneration.AlwaysonScripts.ADetailer.AppendSegModelByString(c.ADetailerString, newGeneration)
		}

		if c.ControlnetItem.Enabled {
			log.Printf("q.currentImagine.ControlnetItem.Enabled: %v", c.ControlnetItem.Enabled)

			if newGeneration.AlwaysonScripts == nil {
				newGeneration.NewScripts()
			}
			var controlnetImage *string
			switch {
			case c.ControlnetItem.MessageAttachment != nil && c.ControlnetItem.Image != nil:
				controlnetImage = c.ControlnetItem.Image
			case c.Img2ImgItem.MessageAttachment != nil && c.Img2ImgItem.Image != nil:
				// not needed for Img2Img as it automatically uses it if InputImage is null, only used for width/height
				controlnetImage = c.Img2ImgItem.Image
			default:
				c.Enabled = false
			}
			width, height, err := utils.GetImageSizeFromBase64(safeDereference(controlnetImage))
			var controlnetResolution int
			if err != nil {
				log.Printf("Error getting image size: %v", err)
			} else {
				controlnetResolution = between(max(width, height), min(newGeneration.Width, newGeneration.Height), 1024)
			}

			newGeneration.AlwaysonScripts.ControlNet = &entities.ControlNet{
				Args: []*entities.ControlNetParameters{
					{
						InputImage:   controlnetImage,
						Module:       c.ControlnetItem.Preprocessor,
						Model:        c.ControlnetItem.Model,
						Weight:       1.0,
						ResizeMode:   c.ControlnetItem.ResizeMode,
						ProcessorRes: controlnetResolution,
						ControlMode:  c.ControlnetItem.ControlMode,
						PixelPerfect: false,
					},
				},
			}
			if c.Type == ItemTypeImg2Img && c.ControlnetItem.MessageAttachment == nil {
				// controlnet will automatically use img2img if it is null
				newGeneration.AlwaysonScripts.ControlNet.Args[0].InputImage = nil
			}

			if !c.Enabled {
				newGeneration.AlwaysonScripts.ControlNet = nil
			}
		}

		if newGeneration.AlwaysonScripts != nil {
			// check if both inner scripts are nil, if so, set AlwaysonScripts to nil
			if newGeneration.AlwaysonScripts.ControlNet == nil && newGeneration.AlwaysonScripts.ADetailer == nil {
				newGeneration.AlwaysonScripts = nil
			}
		}

		if newGeneration.AlwaysonScripts != nil && newGeneration.AlwaysonScripts.ADetailer != nil {
			jsonMarshalScripts, err := json.MarshalIndent(&newGeneration.AlwaysonScripts.ADetailer, "", "  ")
			if err != nil {
				log.Printf("Error marshalling scripts: %v", err)
			} else {
				log.Println("Final scripts (Adetailer): ", string(jsonMarshalScripts))
			}
		}

		switch c.Type {
		case ItemTypeReroll, ItemTypeVariation:
			foundGeneration, err := q.getPreviousGeneration(c, c.InteractionIndex)
			if err != nil {
				log.Printf("Error getting prompt for reroll: %v", err)
				handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
				return
			}

			// if we are rerolling, or generating variations, we simply replace some defaults
			newGeneration = foundGeneration

			// for variations, we need random subseeds
			newGeneration.Subseed = -1

			// for variations, the subseed strength determines how much variation we get
			if c.Type == ItemTypeVariation {
				newGeneration.SubseedStrength = 0.15
			}
		}

		err = q.processImagineGrid(newGeneration, c)
		if err != nil {
			log.Printf("Error processing imagine grid: %v", err)
			handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
			return
		}
	}()
}
