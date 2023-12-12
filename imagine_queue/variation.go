package imagine_queue

import (
	"log"
	"stable_diffusion_bot/discord_bot/handlers"
	"time"
)

func (q *queueImplementation) processVariation() {
	defer q.done()
	c := q.currentImagine
	request, err := q.getPreviousGeneration(c)
	if err != nil {
		log.Printf("Error getting prompt for reroll: %v", err)
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

	c.ImageGenerationRequest = request

	err = q.processImagineGrid(c)
	if err != nil {
		log.Printf("Error processing imagine grid: %v", err)
		handlers.Errors[handlers.ErrorResponse](q.botSession, c.DiscordInteraction, err)
		return
	}
}
