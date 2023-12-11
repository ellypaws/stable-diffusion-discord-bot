package image_generations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"stable_diffusion_bot/clock"
	"stable_diffusion_bot/entities"
)

const insertGenerationQuery string = `
INSERT INTO image_generations (interaction_id, message_id, member_id, sort_order, prompt, 
                               negative_prompt, width, height, restore_faces, 
                               enable_hr, hr_scale, hr_upscaler, hires_width, hires_height, denoising_strength, 
                               batch_count, batch_size, seed, subseed, 
                               subseed_strength, sampler_name, cfg_scale, steps, processed, created_at, 
                               always_on_scripts, 
                               checkpoint, vae, hypernetwork) VALUES
                            (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`

const getGenerationByMessageID string = `
SELECT id, interaction_id, message_id, member_id, sort_order, prompt,
       negative_prompt, width, height, restore_faces, 
       enable_hr, hr_scale, hr_upscaler, hires_width, hires_height, 
       denoising_strength, batch_count, batch_size, seed, subseed, 
       subseed_strength, sampler_name, cfg_scale, steps, processed, created_at, 
       always_on_scripts, 
       checkpoint, vae, hypernetwork FROM image_generations WHERE message_id = ?;
`

const getGenerationByMessageIDAndSortOrder string = `
SELECT id, interaction_id, message_id, member_id, sort_order, prompt,
       negative_prompt, width, height, restore_faces, 
       enable_hr, hr_scale, hr_upscaler, hires_width, hires_height, 
       denoising_strength, batch_count, batch_size, seed, subseed, 
       subseed_strength, sampler_name, cfg_scale, steps, processed, created_at, 
       always_on_scripts, 
       checkpoint, vae, hypernetwork FROM image_generations WHERE message_id = ? AND sort_order = ?;
`

type sqliteRepo struct {
	dbConn *sql.DB
	clock  clock.Clock
}

type Config struct {
	DB *sql.DB
}

func NewRepository(cfg *Config) (Repository, error) {
	if cfg.DB == nil {
		return nil, errors.New("missing DB parameter")
	}

	newRepo := &sqliteRepo{
		dbConn: cfg.DB,
		clock:  clock.NewClock(),
	}

	return newRepo, nil
}

func (repo *sqliteRepo) Create(ctx context.Context, generation *entities.ImageGenerationRequest) (*entities.ImageGenerationRequest, error) {
	if generation.CreatedAt.IsZero() {
		generation.CreatedAt = repo.clock.Now()
	}

	marshalAlwaysonScripts, err := json.Marshal(generation.Scripts)
	if err != nil {
		marshalAlwaysonScripts = []byte("{}")
	}
	marshalAlwaysonScriptstoString := string(marshalAlwaysonScripts)
	res, err := repo.dbConn.ExecContext(ctx, insertGenerationQuery,
		generation.InteractionID, generation.MessageID, generation.MemberID, generation.SortOrder, generation.Prompt,
		generation.NegativePrompt, generation.Width, generation.Height, generation.RestoreFaces,
		generation.EnableHr, generation.HrScale, generation.HrUpscaler, generation.HrResizeX, generation.HrResizeY, generation.DenoisingStrength,
		generation.NIter, generation.BatchSize, generation.Seed, generation.Subseed,
		generation.SubseedStrength, generation.SamplerName, generation.CFGScale, generation.Steps, generation.Processed, generation.CreatedAt,
		marshalAlwaysonScriptstoString,
		generation.Checkpoint, generation.VAE, generation.Hypernetwork,
	)
	if err != nil {
		return nil, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	generation.ID = lastID

	return generation, nil
}

func (repo *sqliteRepo) GetByMessage(ctx context.Context, messageID string) (*entities.ImageGenerationRequest, error) {
	var generation = entities.ImageGenerationRequest{TextToImageRequest: &entities.TextToImageRequest{}}
	var alwaysonScriptsString string

	err := repo.dbConn.QueryRowContext(ctx, getGenerationByMessageID, messageID).Scan(
		&generation.ID, &generation.InteractionID, &generation.MessageID, &generation.MemberID, &generation.SortOrder, &generation.Prompt,
		&generation.NegativePrompt, &generation.Width, &generation.Height, &generation.RestoreFaces,
		&generation.EnableHr, &generation.HrScale, &generation.HrUpscaler, &generation.HrResizeX, &generation.HrResizeY, &generation.DenoisingStrength,
		&generation.NIter, &generation.BatchSize, &generation.Seed, &generation.Subseed,
		&generation.SubseedStrength, &generation.SamplerName, &generation.CFGScale, &generation.Steps, &generation.Processed, &generation.CreatedAt,
		&alwaysonScriptsString,
		&generation.Checkpoint, &generation.VAE, &generation.Hypernetwork,
	)
	if err != nil {
		return nil, err
	}

	generation.NewADetailer()
	err = json.Unmarshal([]byte(alwaysonScriptsString), &generation.Scripts)
	if err != nil {
		return nil, err
	}

	return &generation, nil
}

func (repo *sqliteRepo) GetByMessageAndSort(ctx context.Context, messageID string, sortOrder int) (*entities.ImageGenerationRequest, error) {
	var generation = entities.ImageGenerationRequest{TextToImageRequest: &entities.TextToImageRequest{}}
	var alwaysonScriptsString string

	err := repo.dbConn.QueryRowContext(ctx, getGenerationByMessageIDAndSortOrder, messageID, sortOrder).Scan(
		&generation.ID, &generation.InteractionID, &generation.MessageID, &generation.MemberID, &generation.SortOrder, &generation.Prompt,
		&generation.NegativePrompt, &generation.Width, &generation.Height, &generation.RestoreFaces,
		&generation.EnableHr, &generation.HrScale, &generation.HrUpscaler, &generation.HrResizeX, &generation.HrResizeY, &generation.DenoisingStrength,
		&generation.NIter, &generation.BatchSize, &generation.Seed, &generation.Subseed,
		&generation.SubseedStrength, &generation.SamplerName, &generation.CFGScale, &generation.Steps, &generation.Processed, &generation.CreatedAt,
		&alwaysonScriptsString,
		&generation.Checkpoint, &generation.VAE, &generation.Hypernetwork,
	)

	if err != nil {
		return nil, err
	}

	// generation.NewADetailer()
	err = json.Unmarshal([]byte(alwaysonScriptsString), &generation.Scripts)
	if err != nil {
		return nil, err
	}

	return &generation, nil
}
