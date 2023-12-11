package imagine_queue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/sahilm/fuzzy"
	"log"
	"os"
	"os/signal"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
	p "stable_diffusion_bot/gui/progress"
	"stable_diffusion_bot/repositories"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"
	"stable_diffusion_bot/stable_diffusion_api"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	botID = "bot"

	initializedWidth      = 512
	initializedHeight     = 512
	initializedBatchCount = 4
	initializedBatchSize  = 1
)

type queueImplementation struct {
	botSession          *discordgo.Session
	stableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	queue               chan *entities.QueueItem
	currentImagine      *entities.QueueItem
	mu                  sync.Mutex
	imageGenerationRepo image_generations.Repository
	compositeRenderer   composite_renderer.Renderer
	defaultSettingsRepo default_settings.Repository
	botDefaultSettings  *entities.DefaultSettings
	cancelledItems      map[string]bool
}

type Config struct {
	StableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	ImageGenerationRepo image_generations.Repository
	DefaultSettingsRepo default_settings.Repository
}

func New(cfg Config) (Queue, error) {
	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("missing stable diffusion API")
	}

	if cfg.ImageGenerationRepo == nil {
		return nil, errors.New("missing image generation repository")
	}

	if cfg.DefaultSettingsRepo == nil {
		return nil, errors.New("missing default settings repository")
	}

	return &queueImplementation{
		stableDiffusionAPI:  cfg.StableDiffusionAPI,
		imageGenerationRepo: cfg.ImageGenerationRepo,
		queue:               make(chan *entities.QueueItem, 100),
		compositeRenderer:   composite_renderer.Compositor(),
		defaultSettingsRepo: cfg.DefaultSettingsRepo,
		cancelledItems:      make(map[string]bool),
	}, nil
}

const (
	ItemTypeImagine entities.ItemType = iota
	ItemTypeReroll
	ItemTypeUpscale
	ItemTypeVariation
	ItemTypeImg2Img
	ItemTypeRaw // raw JSON input
)

func (q *queueImplementation) DefaultQueueItem() *entities.QueueItem {
	defaultBatchCount, err := q.defaultBatchCount()
	if err != nil {
		log.Printf("Error getting default batch count: %v", err)
		defaultBatchCount = 1
	}

	defaultBatchSize, err := q.defaultBatchSize()
	if err != nil {
		log.Printf("Error getting default batch size: %v", err)
		defaultBatchSize = 4
	}
	return &entities.QueueItem{
		Type: ItemTypeImagine,

		ImageGenerationRequest: entities.ImageGenerationRequest{
			GenerationInfo: entities.GenerationInfo{
				CreatedAt: time.Now(),
			},
			TextToImageRequest: &entities.TextToImageRequest{
				NegativePrompt:    DefaultNegative,
				Steps:             20,
				Seed:              -1,
				SamplerName:       "Euler a",
				EnableHr:          false,
				HrUpscaler:        "R-ESRGAN 2x+",
				HrSecondPassSteps: 20,
				HrScale:           1.0,
				DenoisingStrength: 0.7,
				CFGScale:          7.0,
				NIter:             defaultBatchCount,
				BatchSize:         defaultBatchSize,
			},
		},

		Img2ImgItem: entities.Img2ImgItem{
			DenoisingStrength: 0.7,
		},
		ControlnetItem: entities.ControlnetItem{
			ControlMode: entities.ControlModeBalanced,
			ResizeMode:  entities.ResizeModeScaleToFit,
		},
	}
}

func (q *queueImplementation) NewQueueItem(options ...func(*entities.QueueItem)) *entities.QueueItem {
	queue := q.DefaultQueueItem()

	for _, option := range options {
		option(queue)
	}

	return queue
}

func WithPrompt(prompt string) func(*entities.QueueItem) {
	return func(q *entities.QueueItem) {
		q.Prompt = prompt
	}
}

func (q *queueImplementation) AddImagine(item *entities.QueueItem) (int, error) {
	q.queue <- item

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *queueImplementation) StartPolling(botSession *discordgo.Session) {
	q.botSession = botSession

	botDefaultSettings, err := q.initializeOrGetBotDefaults()
	if err != nil {
		log.Printf("Error getting/initializing bot default settings: %v", err)

		return
	}

	q.botDefaultSettings = botDefaultSettings

	log.Println("Press Ctrl+C to exit")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	var wait bool

Polling:
	for {
		select {
		case <-stop:
			break Polling
		case <-time.After(1 * time.Second):
			if q.currentImagine == nil {
				q.pullNextInQueue()
				wait = false
			} else if !wait {
				log.Printf("Waiting for current imagine to finish...\n")
				wait = true
			}
		}
	}

	log.Printf("Polling stopped...\n")
}

func (q *queueImplementation) pullNextInQueue() {
	for len(q.queue) > 0 {
		// Peek at the next item without blocking
		if q.currentImagine != nil {
			log.Printf("WARNING: we're trying to pull the next item in the queue, but currentImagine is not yet nil")
			return // Already processing an item
		}
		select {
		case q.currentImagine = <-q.queue:
			if q.currentImagine.DiscordInteraction == nil {
				// If the interaction is nil, we can't respond. Make sure to set the implementation before adding to the queue.
				// Example: queue.DiscordInteraction = i.Interaction
				log.Panicf("DiscordInteraction is nil! Make sure to set it before adding to the queue. Example: queue.DiscordInteraction = i.Interaction\n%v", q.currentImagine)
				return
			}
			if interaction := q.currentImagine.DiscordInteraction; interaction != nil && q.cancelledItems[q.currentImagine.DiscordInteraction.ID] {
				// If the item is cancelled, skip it
				delete(q.cancelledItems, interaction.ID)
				q.done()
				return
			}
			switch q.currentImagine.Type {
			case ItemTypeImagine, ItemTypeReroll, ItemTypeVariation, ItemTypeRaw:
				go q.processCurrentImagine()
			case ItemTypeImg2Img:
				go q.processImg2ImgImagine()
			case ItemTypeUpscale:
				go q.processUpscaleImagine(q.currentImagine)
			default:
				handlers.Errors[handlers.ErrorResponse](q.botSession, q.currentImagine.DiscordInteraction, fmt.Errorf("unknown item type: %v", q.currentImagine.Type))
				log.Printf("Unknown item type: %v", q.currentImagine.Type)
				q.done()
			}
		default:
			log.Printf("WARNING: we're trying to pull the next item in the queue, but the queue is empty")
			return // Queue is empty
		}
	}
}

func (q *queueImplementation) done() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentImagine = nil
}

func (q *queueImplementation) RemoveFromQueue(messageInteraction *discordgo.MessageInteraction) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Mark the item as cancelled
	q.cancelledItems[messageInteraction.ID] = true

	return nil
}

func (q *queueImplementation) Interrupt(i *discordgo.InteractionCreate) error {

	if q.currentImagine == nil {
		return errors.New("there is no generation currently in progress")
	}

	// Mark the item as cancelled
	log.Printf("Interrupting generation #%s\n", q.currentImagine.DiscordInteraction.ID)
	if q.currentImagine.Interrupt == nil {
		q.currentImagine.Interrupt = make(chan *discordgo.Interaction)
	}
	q.currentImagine.Interrupt <- i.Interaction

	return nil
}

func (q *queueImplementation) fillInBotDefaults(settings *entities.DefaultSettings) (*entities.DefaultSettings, bool) {
	updated := false

	if settings == nil {
		settings = &entities.DefaultSettings{
			MemberID: botID,
		}
	}

	if settings.Width == 0 {
		settings.Width = initializedWidth
		updated = true
	}

	if settings.Height == 0 {
		settings.Height = initializedHeight
		updated = true
	}

	if settings.BatchCount == 0 {
		settings.BatchCount = initializedBatchCount
		updated = true
	}

	if settings.BatchSize == 0 {
		settings.BatchSize = initializedBatchSize
		updated = true
	}

	return settings, updated
}

func (q *queueImplementation) initializeOrGetBotDefaults() (*entities.DefaultSettings, error) {
	botDefaultSettings, err := q.GetBotDefaultSettings()
	if err != nil && !errors.Is(err, &repositories.NotFoundError{}) {
		return nil, err
	}

	botDefaultSettings, updated := q.fillInBotDefaults(botDefaultSettings)
	if updated {
		botDefaultSettings, err = q.defaultSettingsRepo.Upsert(context.Background(), botDefaultSettings)
		if err != nil {
			return nil, err
		}

		log.Printf("Initialized bot default settings: %+v\n", botDefaultSettings)
	} else {
		log.Printf("Retrieved bot default settings: %+v\n", botDefaultSettings)
	}

	return botDefaultSettings, nil
}

func (q *queueImplementation) GetBotDefaultSettings() (*entities.DefaultSettings, error) {
	if q.botDefaultSettings != nil {
		return q.botDefaultSettings, nil
	}

	defaultSettings, err := q.defaultSettingsRepo.GetByMemberID(context.Background(), botID)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = defaultSettings

	return defaultSettings, nil
}

func (q *queueImplementation) defaultWidth() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Width, nil
}

func (q *queueImplementation) defaultHeight() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.Height, nil
}

func (q *queueImplementation) defaultBatchCount() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchCount, nil
}

func (q *queueImplementation) defaultBatchSize() (int, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return 0, err
	}

	return defaultSettings.BatchSize, nil
}

func (q *queueImplementation) UpdateDefaultDimensions(width, height int) (*entities.DefaultSettings, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return nil, err
	}

	defaultSettings.Width = width
	defaultSettings.Height = height

	newDefaultSettings, err := q.defaultSettingsRepo.Upsert(context.Background(), defaultSettings)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = newDefaultSettings

	log.Printf("Updated default dimensions to: %dx%d\n", width, height)

	return newDefaultSettings, nil
}

func (q *queueImplementation) UpdateDefaultBatch(batchCount, batchSize int) (*entities.DefaultSettings, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return nil, err
	}

	defaultSettings.BatchCount = batchCount
	defaultSettings.BatchSize = batchSize

	newDefaultSettings, err := q.defaultSettingsRepo.Upsert(context.Background(), defaultSettings)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = newDefaultSettings

	log.Printf("Updated default batch count/size to: %d/%d\n", batchCount, batchSize)

	return newDefaultSettings, nil
}

// Deprecated: No longer store the SDModelName to DefaultSettings struct, use stable_diffusion_api.GetConfig instead
func (q *queueImplementation) UpdateModelName(modelName string) (*entities.DefaultSettings, error) {
	defaultSettings, err := q.GetBotDefaultSettings()
	if err != nil {
		return nil, err
	}

	//defaultSettings.SDModelName = modelName

	newDefaultSettings, err := q.defaultSettingsRepo.Upsert(context.Background(), defaultSettings)
	if err != nil {
		return nil, err
	}

	q.botDefaultSettings = newDefaultSettings

	log.Printf("Updated model to: %s\n", modelName)
	return newDefaultSettings, nil
}

// input is 2:3 for example, without the `--ar` part
func aspectRatioCalculation(aspectRatio string, w, h int) (width, height int) {
	// split
	aspectRatioSplit := strings.Split(aspectRatio, ":")
	if len(aspectRatioSplit) != 2 {
		return w, h
	}

	// convert to int
	widthRatio, err := strconv.Atoi(aspectRatioSplit[0])
	if err != nil {
		return w, h
	}
	heightRatio, err := strconv.Atoi(aspectRatioSplit[1])
	if err != nil {
		return w, h
	}

	// calculate
	if widthRatio > heightRatio {
		scaledWidth := float64(h) * (float64(widthRatio) / float64(heightRatio))

		// Round up to the nearest 8
		width = (int(scaledWidth) + 7) & (-8)
		height = h
	} else if heightRatio > widthRatio {
		scaledHeight := float64(w) * (float64(heightRatio) / float64(widthRatio))

		// Round up to the nearest 8
		height = (int(scaledHeight) + 7) & (-8)
		width = w
	} else {
		width = w
		height = h
	}

	return width, height
}

const DefaultNegative = "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
	"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
	"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy"

func between[T cmp.Ordered](value, minimum, maximum T) T {
	return min(max(minimum, value), maximum)
}

func betweenPtr[T cmp.Ordered](value, minimum, maximum T) *T {
	out := min(max(minimum, value), maximum)
	return &out
}

func (q *queueImplementation) getPreviousGeneration(imagine *entities.QueueItem, sortOrder int) (*entities.ImageGenerationRequest, error) {
	interactionID := imagine.DiscordInteraction.ID
	messageID := ""

	if imagine.DiscordInteraction.Message != nil {
		messageID = imagine.DiscordInteraction.Message.ID
	}

	log.Printf("Reimagining interaction: %v, Message: %v", interactionID, messageID)

	generation, err := q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, sortOrder)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return nil, err
	}

	log.Printf("Found generation: %v", generation)

	return generation, nil
}

// Deprecated: use imagineMessageSimple instead
func imagineMessageContent(generation *entities.ImageGenerationRequest, user *discordgo.User, progress float64) string {
	var out = strings.Builder{}

	seedString := fmt.Sprintf("%d", generation.Seed)
	if seedString == "-1" {
		seedString = "at random(-1)"
	}

	out.WriteString(fmt.Sprintf("<@%s> asked me to imagine with step: `%d` cfg: `%s` seed: `%s` sampler: `%s`",
		user.ID,
		generation.Steps,
		strconv.FormatFloat(generation.CFGScale, 'f', 1, 64),
		seedString,
		generation.SamplerName,
	))

	out.WriteString(fmt.Sprintf(" `%d x %d`", generation.Width, generation.Height))

	if generation.EnableHr == true {
		// " -> (x %x) = %d x %d"
		out.WriteString(fmt.Sprintf(" -> (x `%s` by hires.fix) = `%d x %d`",
			strconv.FormatFloat(generation.HrScale, 'f', 1, 64),
			generation.HrResizeX,
			generation.HrResizeY),
		)
	}

	if ptrStringNotBlank(generation.Checkpoint) {
		out.WriteString(fmt.Sprintf("\n**Checkpoint**: `%v`", *generation.Checkpoint))
	}

	if ptrStringNotBlank(generation.VAE) {
		out.WriteString(fmt.Sprintf("\n**VAE**: `%v`", *generation.VAE))
	}

	if ptrStringNotBlank(generation.Hypernetwork) {
		if ptrStringNotBlank(generation.VAE) {
			out.WriteString(" ")
		} else {
			out.WriteString("\n")
		}
		out.WriteString(fmt.Sprintf("**Hypernetwork**: `%v`", *generation.Hypernetwork))
	}

	if progress >= 0 && progress < 1 {
		out.WriteString(fmt.Sprintf("\n**Progress**:\n```ansi\n%v\n```", p.Get().ViewAs(progress)))
	}

	out.WriteString(fmt.Sprintf("\n```\n%s\n```", generation.Prompt))

	if generation.Scripts.ADetailer != nil && len(generation.Scripts.ADetailer.Args) > 0 {
		var models []string
		for _, v := range generation.Scripts.ADetailer.Args {
			models = append(models, v.AdModel)
		}
		out.WriteString(fmt.Sprintf("\n**ADetailer**: [%v]", strings.Join(models, ", ")))
	}

	if generation.Scripts.ControlNet != nil && len(generation.Scripts.ControlNet.Args) > 0 {
		var preprocessor []string
		var model []string
		for _, v := range generation.Scripts.ControlNet.Args {
			preprocessor = append(preprocessor, v.Module)
			model = append(model, v.Model)
		}
		out.WriteString(fmt.Sprintf("\n**ControlNet**: [%v]\n**Preprocessor**: [%v]", strings.Join(preprocessor, ", "), strings.Join(model, ", ")))
	}

	if out.Len() > 2000 {
		return out.String()[:2000]
	}
	return out.String()
}

func imagineMessageSimple(generation *entities.ImageGenerationRequest, user *discordgo.User, progress float64) string {
	var out = strings.Builder{}

	seedString := fmt.Sprintf("%d", generation.Seed)
	if seedString == "-1" {
		seedString = "at random(-1)"
	}

	out.WriteString(fmt.Sprintf("<@%s> asked me to imagine", user.ID))

	out.WriteString(fmt.Sprintf(" `%d x %d`", generation.Width, generation.Height))

	if generation.EnableHr == true {
		// " -> (x %x) = %d x %d"
		out.WriteString(fmt.Sprintf(" -> (x `%s` by hires.fix) = `%d x %d`",
			strconv.FormatFloat(generation.HrScale, 'f', 1, 64),
			generation.HrResizeX,
			generation.HrResizeY),
		)
	}

	if progress >= 0 && progress < 1 {
		out.WriteString(fmt.Sprintf("\n**Progress**:\n```ansi\n%v\n```", p.Get().ViewAs(progress)))
	}

	if out.Len() > 2000 {
		return out.String()[:2000]
	}
	return out.String()
}

// lookupModel searches through []stable_diffusion_api.Cacheable models to find the model to load
func (q *queueImplementation) lookupModel(generation *entities.ImageGenerationRequest, config *entities.Config, c []stable_diffusion_api.Cacheable) (POST entities.Config) {
	for _, c := range c {
		var toLoad *string
		var loadedModel *string
		switch c.(type) {
		case *stable_diffusion_api.SDModels:
			toLoad = generation.Checkpoint
			loadedModel = config.SDModelCheckpoint
			//log.Printf("Checkpoint: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		case *stable_diffusion_api.VAEModels:
			toLoad = generation.VAE
			loadedModel = config.SDVae
			//log.Printf("VAE: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		case *stable_diffusion_api.HypernetworkModels:
			toLoad = generation.Hypernetwork
			loadedModel = config.SDHypernetwork
			//log.Printf("Hypernetwork: %v, loaded: %v", safeDereference(toLoad), safeDereference(loadedModel))
		}

		if ptrStringCompare(toLoad, loadedModel) {
			log.Printf("Model %T \"%v\" already loaded as \"%v\"", toLoad, safeDereference(toLoad), safeDereference(loadedModel))
		}

		if toLoad != nil {
			switch safeDereference(toLoad) {
			case "":
				// set to nil if empty string
				toLoad = nil
			case "None":
				// keep "None" to unload the model
			default:
				// lookup from the list of models
				cache, err := c.GetCache(q.stableDiffusionAPI)
				if err != nil {
					log.Println("Failed to get cached models:", err)
					continue
				}

				results := fuzzy.FindFrom(*toLoad, cache)

				if len(results) > 0 {
					firstResult := cache.String(results[0].Index)
					toLoad = &firstResult
				} else {
					log.Printf("Couldn't find model %v", safeDereference(toLoad))
					//log.Printf("Available models: %v", cache)
				}
			}
		}

		switch c.(type) {
		case *stable_diffusion_api.SDModels:
			POST.SDModelCheckpoint = toLoad
		case *stable_diffusion_api.VAEModels:
			POST.SDVae = toLoad
		case *stable_diffusion_api.HypernetworkModels:
			POST.SDHypernetwork = toLoad
		}
	}

	if POST.SDModelCheckpoint != nil || POST.SDVae != nil || POST.SDHypernetwork != nil {
		marshal, _ := POST.Marshal()
		log.Printf("Switching models to %#v", string(marshal))
	}
	return
}

func upscaleMessageContent(user *discordgo.User, fetchProgress, upscaleProgress float64) string {
	if fetchProgress >= 0 && fetchProgress <= 1 && upscaleProgress < 1 {
		if upscaleProgress == 0 {
			return fmt.Sprintf("Currently upscaling the image for you... Fetch progress: %.0f%%", fetchProgress*100)
		} else {
			return fmt.Sprintf("Currently upscaling the image for you... Fetch progress: %.0f%% Upscale progress: %.0f%%",
				fetchProgress*100, upscaleProgress*100)
		}
	} else {
		return fmt.Sprintf("<@%s> asked me to upscale their image. Here's the result:",
			user.ID)
	}
}

func ptrStringCompare(s1 *string, s2 *string) bool {
	if s1 == nil || s2 == nil {
		return s1 == s2
	}
	return *s1 == *s2
}

func ptrStringNotBlank(s *string) bool {
	if s == nil {
		return false
	}
	return *s != ""
}

func safeDereference(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
