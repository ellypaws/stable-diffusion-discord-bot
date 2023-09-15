package imagine_queue

import (
	"github.com/bwmarrin/discordgo"
	"stable_diffusion_bot/entities"
)

type Queue interface {
	AddImagine(item *QueueItem) (int, error)
	StartPolling(botSession *discordgo.Session)
	GetBotDefaultSettings() (*entities.DefaultSettings, error)
	UpdateDefaultDimensions(width, height int) (*entities.DefaultSettings, error)
	UpdateDefaultBatch(batchCount, batchSize int) (*entities.DefaultSettings, error)
	UpdateModelName(modelName string) (*entities.DefaultSettings, error)
}
