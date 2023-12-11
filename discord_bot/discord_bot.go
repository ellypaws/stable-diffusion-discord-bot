package discord_bot

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"sort"
	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/stable_diffusion_api"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	developmentMode    bool
	botSession         *discordgo.Session
	guildID            string
	imagineQueue       imagine_queue.Queue
	registeredCommands map[Command]*discordgo.ApplicationCommand
	imagineCommand     *Command
	removeCommands     bool
	StableDiffusionApi stable_diffusion_api.StableDiffusionAPI
	config             *Config
}

type Config struct {
	DevelopmentMode    bool
	BotToken           string
	GuildID            string
	ImagineQueue       imagine_queue.Queue
	ImagineCommand     *Command
	RemoveCommands     bool
	StableDiffusionApi stable_diffusion_api.StableDiffusionAPI
}

func (b *botImpl) imagineCommandString() Command {
	if b.developmentMode && !strings.HasPrefix(string(imagineCommand), "dev_") {
		imagineCommand = Command(fmt.Sprintf("dev_%v", strings.TrimPrefix(string(*b.config.ImagineCommand), "dev_")))
		return imagineCommand
	}

	return *b.config.ImagineCommand
}

func (b *botImpl) imagineSettingsCommandString() Command {
	if b.developmentMode && !strings.HasPrefix(string(imagineSettingsCommand), "dev_") {
		imagineSettingsCommand = Command(fmt.Sprintf("dev_%v_settings", strings.TrimPrefix(string(*b.config.ImagineCommand), "dev_")))
		return imagineSettingsCommand
	}

	return Command(fmt.Sprintf("%v_settings", *b.config.ImagineCommand))
}

func New(cfg *Config) (Bot, error) {
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	handlers.Token = &cfg.BotToken

	if cfg.GuildID == "" {
		//return nil, errors.New("missing guild ID")
		log.Printf("Guild ID not provided, commands will be registered globally")
	}

	if cfg.ImagineQueue == nil {
		return nil, errors.New("missing imagine queue")
	}

	if cfg.ImagineCommand == nil || *cfg.ImagineCommand == "" {
		return nil, errors.New("missing imagine command")
	}

	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = botSession.Open()
	if err != nil {
		return nil, err
	}

	bot := &botImpl{
		developmentMode:    cfg.DevelopmentMode,
		botSession:         botSession,
		imagineQueue:       cfg.ImagineQueue,
		registeredCommands: make(map[Command]*discordgo.ApplicationCommand),
		imagineCommand:     cfg.ImagineCommand,
		removeCommands:     cfg.RemoveCommands,
		StableDiffusionApi: cfg.StableDiffusionApi,
		config:             cfg,
	}

	//Read the imagineCommand from the config and remake the maps
	bot.customImagineCommand()

	err = bot.registerCommands()
	if err != nil {
		return nil, err
	}

	bot.registerHandlers(botSession)

	return bot, nil
}

func (b *botImpl) registerHandlers(session *discordgo.Session) {
	session.AddHandler(func(session *discordgo.Session, i *discordgo.InteractionCreate) {
		var h func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate)
		var ok bool
		switch i.Type {
		// commands
		case discordgo.InteractionApplicationCommand:
			h, ok = commandHandlers[Command(i.ApplicationCommandData().Name)]
			//If we're using *Command, we have to range through the map to dereference the Command string
			//for key, command := range commandHandlers {
			//	if string(*key) == i.ApplicationCommandData().Name {
			//		h = command
			//		ok = true
			//	}
			//}
		// buttons
		case discordgo.InteractionMessageComponent:
			log.Printf("Component with customID `%v` was pressed, attempting to respond\n", i.MessageComponentData().CustomID)
			h, ok = componentHandlers[handlers.Component(i.MessageComponentData().CustomID)]
			//bot.p.Send(logger.Message(fmt.Sprintf(
			//	"Handler found, executing on message `%v`\nRan by: <@%v>\nUsername: %v",
			//	i.Message.ID,
			//	i.Member.User.ID,
			//	i.Member.User.Username,
			//)))
			//bot.p.Send(logger.Message(fmt.Sprintf("https://discord.com/channels/%v/%v/%v", i.GuildID, i.ChannelID, i.Message.ID)))

			if !ok {
				switch customID := i.MessageComponentData().CustomID; {
				case strings.HasPrefix(customID, string(handlers.UpscaleButton)):
					h, ok = componentHandlers[handlers.UpscaleButton]
				case strings.HasPrefix(customID, string(handlers.VariantButton)):
					h, ok = componentHandlers[handlers.VariantButton]
				default:
					log.Printf("Unknown message component '%v'", i.MessageComponentData().CustomID)
				}
			}
		// autocomplete
		case discordgo.InteractionApplicationCommandAutocomplete:
			h, ok = autocompleteHandlers[Command(i.ApplicationCommandData().Name)]
		// modals
		case discordgo.InteractionModalSubmit:
			h, ok = modalHandlers[Command(i.ModalSubmitData().CustomID)]
		}

		if !ok {
			var interactionType string
			var interactionName string
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				interactionType = "command"
				interactionName = i.ApplicationCommandData().Name
			case discordgo.InteractionMessageComponent:
				interactionType = "component"
				interactionName = i.MessageComponentData().CustomID
			case discordgo.InteractionApplicationCommandAutocomplete:
				interactionType = "autocomplete"

				data := i.ApplicationCommandData()
				for _, opt := range data.Options {
					if !opt.Focused {
						continue
					}
					interactionName = fmt.Sprintf("command: /%v option: %v (%v)", data.Name, opt.Name)
					break
				}
			case discordgo.InteractionModalSubmit:
				interactionType = "modal"
				interactionName = i.ModalSubmitData().CustomID
			}
			log.Printf("WARNING: Cannot find handler for interaction [%v] '%v'", interactionType, interactionName)
			return
		}

		h(b, session, i)
	})
	//currentProgress = len(commandHandlers) + len(componentHandlers) + len(components)
	//bot.p.Send(load.Goal{
	//	Current: currentProgress,
	//	Total:   totalProgress,
	//	Show:    true,
	//})
	//session.AddHandler(func(session *discordgo.Session, r *discordgo.Ready) {
	//	bot.p.Send(logger.Message(fmt.Sprintf("Logged in as: %v#%v", session.State.User.Username, session.State.User.Discriminator)))
	//})
}

func (b *botImpl) registerCommands() error {
	b.registeredCommands = make(map[Command]*discordgo.ApplicationCommand, len(commands))
	for key, command := range commands {
		if command.Name == "" {
			// clean the key because it might be a description of some sort
			// only get the first word, and clean to only alphanumeric characters or -
			sanitized := strings.ReplaceAll(string(key), " ", "-")
			sanitized = strings.ToLower(sanitized)

			// remove all non valid characters
			for _, c := range sanitized {
				if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
					sanitized = strings.ReplaceAll(sanitized, string(c), "")
				}
			}
			command.Name = sanitized
		}
		//b.controlnetTypes()
		cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, command)
		if err != nil {
			return errors.New(fmt.Sprintf("Cannot create '%v' command: %v", command.Name, err))
		}
		b.registeredCommands[key] = cmd
		log.Printf("Registered %v command as: /%v", key, cmd.Name)
		//bot.p.Send(logger.Message(fmt.Sprintf("Registered command: %v", cmd.Name)))
		//currentProgress++
		//bot.p.Send(load.Goal{
		//	Current: currentProgress,
		//	Total:   totalProgress,
		//	Show:    true,
		//})
	}

	return nil
}

// customImagineCommand is used to read the imagineCommand from the config and remake the maps
// the keys are copied from the commands and commandHandlers map, deleted, and then re-added with the new command
func (b *botImpl) customImagineCommand() {
	//imagine
	b.rebuildMap((*botImpl).imagineCommandString, &imagineCommand, commands, commandHandlers)

	//imagine_settings
	b.rebuildMap((*botImpl).imagineSettingsCommandString, &imagineSettingsCommand, commands, commandHandlers)
}

func (b *botImpl) rebuildMap(
	f func(*botImpl) Command,
	key *Command,
	m map[Command]*discordgo.ApplicationCommand,
	h map[Command]func(b *botImpl, s *discordgo.Session, i *discordgo.InteractionCreate,
	)) {
	oldKey := *key

	*key = f(b)
	if *key == oldKey {
		return
	}
	log.Printf("Rebuilding map for '%v' to '%v'", oldKey, *key)

	m[*key] = m[oldKey]
	m[*key].Name = string(*key)
	h[*key] = h[oldKey]
	delete(m, oldKey)
	delete(h, oldKey)
}

func (b *botImpl) Start() {
	b.imagineQueue.StartPolling(b.botSession)

	err := b.teardown()
	if err != nil {
		log.Printf("Error tearing down bot: %v", err)
	}
}

func (b *botImpl) teardown() error {
	// Delete all commands added by the bot
	if b.removeCommands {
		log.Printf("Removing all commands added by bot...")

		for key, v := range b.registeredCommands {
			log.Printf("Removing command [key:%v], '%v'...", key, v.Name)

			err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.guildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	return b.botSession.Close()
}

// Deprecated: If we want to dynamically update the controlnet types, we can do it here
func (b *botImpl) controlnetTypes() {
	if false {
		controlnet, err := stable_diffusion_api.ControlnetTypesCache.GetCache(b.StableDiffusionApi)
		if err != nil {
			log.Printf("Error getting controlnet types: %v", err)
			panic(err)
		}
		// modify the choices of controlnetType by using the controlnetTypes cache
		var keys map[string]bool = make(map[string]bool)
		for key := range controlnet.(*stable_diffusion_api.ControlnetTypes).ControlTypes {
			if keys[key] {
				continue
			}
			keys[key] = true

			commandOptions[controlnetType].Choices = append(commandOptions[controlnetType].Choices,
				&discordgo.ApplicationCommandOptionChoice{
					Name:  key,
					Value: key,
				})
			if len(commandOptions[controlnetType].Choices) >= 25 {
				break
			}
		}
		sort.Slice(commandOptions[controlnetType].Choices, func(i, j int) bool {
			return cmp.Less(commandOptions[controlnetType].Choices[i].Name, commandOptions[controlnetType].Choices[j].Name)
		})
	}
}

func shortenString(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}
