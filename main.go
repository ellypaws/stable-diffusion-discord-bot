package main

import (
	"context"
	"flag"
	"github.com/joho/godotenv"
	"log"
	"os"
	"stable_diffusion_bot/databases/sqlite"
	"stable_diffusion_bot/discord_bot"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"
	"stable_diffusion_bot/stable_diffusion_api"
	"strings"
)

// Bot parameters
var (
	guildID            = flag.String("guild", "", "Guild ID. If not passed - bot registers commands globally")
	botToken           = flag.String("token", "", "Bot access token")
	apiHost            = flag.String("host", "", "Host for the Automatic1111 API")
	imagineCommand     = flag.String("imagine", "imagine", "Imagine command name. Default is \"imagine\"")
	removeCommandsFlag = flag.Bool("remove", false, "Delete all commands when bot exits")
	devModeFlag        = flag.Bool("dev", false, "Start in development mode, using \"dev_\" prefixed commands instead")
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	log.Println(".env file loaded successfully")

	if botToken == nil || *botToken == "" {
		tokenEnv := os.Getenv("BOT_TOKEN")
		if tokenEnv == "YOUR_BOT_TOKEN_HERE" {
			log.Fatalf("Invalid bot token: %v\n"+
				"Did you edit the .env or run the program with -token ?", tokenEnv)
		}
		if tokenEnv != "" {
			botToken = &tokenEnv
		}
	}

	if apiHost == nil || *apiHost == "" {
		hostEnv := os.Getenv("API_HOST")
		if hostEnv != "" {
			apiHost = &hostEnv
		}
	}

	if apiHost != nil && *apiHost != "" {
		sanitized := strings.TrimSuffix(*apiHost, "/")
		apiHost = &sanitized
	}

	if guildID == nil || *guildID == "" {
		guildEnv := os.Getenv("GUILD_ID")
		if guildEnv != "" {
			guildID = &guildEnv
		}
	}

	if imagineCommand == nil || *imagineCommand == "" {
		imagineEnv := os.Getenv("IMAGINE_COMMAND")
		if imagineEnv != "" {
			*imagineCommand = imagineEnv
		}
	}

	if devModeFlag == nil || !*devModeFlag {
		devModeEnv := os.Getenv("DEV_MODE")
		if devModeEnv != "" {
			devModeFlag = new(bool)
			*devModeFlag = devModeEnv == "true"
		}
	}

	if removeCommandsFlag == nil || !*removeCommandsFlag {
		removeCommandsEnv := os.Getenv("REMOVE_COMMANDS")
		if removeCommandsEnv != "" {
			removeCommandsFlag = new(bool)
			*removeCommandsFlag = removeCommandsEnv == "true"
		}
	}
}

func main() {
	flag.Parse()

	//if guildID == nil || *guildID == "" {
	//	log.Fatalf("Guild ID flag is required")
	//}

	if botToken == nil || *botToken == "" {
		log.Fatalf("Bot token flag is required")
	}

	if apiHost == nil || *apiHost == "" {
		log.Fatalf("API host flag is required")
	}

	alive := handlers.CheckAPIAlive(*apiHost)
	if !alive {
		log.Printf("API (%v) is not running! Continuing anyway...", *apiHost)
	}

	if imagineCommand == nil || *imagineCommand == "" {
		log.Fatalf("Imagine command flag is required")
	}

	devMode := false

	if devModeFlag != nil && *devModeFlag {
		devMode = *devModeFlag
		log.Printf("Starting in development mode.. all commands prefixed with \"dev_\"")
	}

	var removeCommands bool

	if removeCommandsFlag != nil && *removeCommandsFlag {
		removeCommands = *removeCommandsFlag
	}

	stableDiffusionAPI, err := stable_diffusion_api.New(stable_diffusion_api.Config{
		Host: *apiHost,
	})
	if err != nil {
		log.Fatalf("Failed to create Stable Diffusion API: %v", err)
	}

	errors := stableDiffusionAPI.PopulateCache()
	if errors != nil {
		log.Printf("Failed to populate cache: %v", err)
	}

	ctx := context.Background()

	sqliteDB, err := sqlite.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create sqlite database: %v", err)
	}

	generationRepo, err := image_generations.NewRepository(&image_generations.Config{DB: sqliteDB})
	if err != nil {
		log.Fatalf("Failed to create image generation repository: %v", err)
	}

	defaultSettingsRepo, err := default_settings.NewRepository(&default_settings.Config{DB: sqliteDB})
	if err != nil {
		log.Fatalf("Failed to create default settings repository: %v", err)
	}

	imagineQueue, err := imagine_queue.New(imagine_queue.Config{
		StableDiffusionAPI:  stableDiffusionAPI,
		ImageGenerationRepo: generationRepo,
		DefaultSettingsRepo: defaultSettingsRepo,
	})
	if err != nil {
		log.Fatalf("Failed to create imagine queue: %v", err)
	}

	bot, err := discord_bot.New(&discord_bot.Config{
		DevelopmentMode:    devMode,
		BotToken:           *botToken,
		GuildID:            *guildID,
		ImagineQueue:       imagineQueue,
		ImagineCommand:     (*discord_bot.Command)(imagineCommand),
		RemoveCommands:     removeCommands,
		StableDiffusionApi: stableDiffusionAPI,
	})
	if err != nil {
		log.Fatalf("Error creating Discord bot: %v", err)
	}

	bot.Start()

	log.Println("Gracefully shutting down.")
}
