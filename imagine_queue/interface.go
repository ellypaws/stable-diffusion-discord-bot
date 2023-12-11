package imagine_queue

import (
	"github.com/bwmarrin/discordgo"
	"stable_diffusion_bot/entities"
)

type Queue interface {
	AddImagine(item *entities.QueueItem) (int, error)
	StartPolling(botSession *discordgo.Session)
	GetBotDefaultSettings() (*entities.DefaultSettings, error)
	UpdateDefaultDimensions(width, height int) (*entities.DefaultSettings, error)
	UpdateDefaultBatch(batchCount, batchSize int) (*entities.DefaultSettings, error)
	UpdateModelName(modelName string) (*entities.DefaultSettings, error) // Deprecated: No longer store the SDModelName to DefaultSettings struct, use stable_diffusion_api.GetConfig instead
	RemoveFromQueue(messageInteraction *discordgo.MessageInteraction) error
	Interrupt(messageInteraction *discordgo.InteractionCreate) error
	NewQueueItem(options ...func(*entities.QueueItem)) *entities.QueueItem
}
