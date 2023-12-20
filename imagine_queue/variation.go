package imagine_queue

import (
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"time"
)

func (q *queueImplementation) processVariation() {
	defer q.done()
	c, err := q.currentImagine, error(nil)
	c.ImageGenerationRequest, err = q.getPreviousGeneration(c)
	request := c.ImageGenerationRequest
	if err != nil {
		log.Printf("Error getting prompt for reroll: %v", err)
		handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
		return
	}

	message := handlers.Responses[handlers.EditInteractionResponse].(handlers.MsgReturnType)(q.botSession, c.DiscordInteraction, "Found previous generation...")
	// store the new message to record the correct message ID in the database
	c.DiscordInteraction.Message = message

	err = q.storeMessageInteraction(c, message)
	if err != nil {
		log.Printf("Error storing message interaction: %v", err)
		handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
		return
	}

	// for variations, we need random subseeds
	request.Subseed = -1

	if c.Type == ItemTypeReroll {
		request.Seed = -1
	}

	// for variations, the subseed strength determines how much variation we get
	if c.Type == ItemTypeVariation {
		request.SubseedStrength = 0.15
	}

	// set the time to now since time from database is from the past
	request.CreatedAt = time.Now()

	fillBlankModels(q, request)

	err = q.processImagineGrid(c)
	if err != nil {
		log.Printf("Error processing imagine grid: %v", err)
		handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
		return
	}
}
