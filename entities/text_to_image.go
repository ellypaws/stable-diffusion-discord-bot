// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    textToImageRequest, err := UnmarshalTextToImageRequest(bytes)
//    bytes, err = textToImageRequest.Marshal()

package entities

import (
	"encoding/json"
	"fmt"
)

func UnmarshalTextToImageRequest(data []byte) (TextToImageRequest, error) {
	var r TextToImageRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TextToImageRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalTextToImageRaw(data []byte) (TextToImageRaw, error) {
	var r TextToImageRaw
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TextToImageRaw) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TextToImageRaw struct {
	RawScripts map[string]any `json:"alwayson_scripts,omitempty"`
	*TextToImageRequest
	RawParams `json:"-,omitempty"`
}

type RawParams struct {
	UseDefault bool
	Unsafe     bool
	Debug      bool
	Blob       []byte
}

type TextToImageRequest struct {
	Scripts                           `json:"alwayson_scripts,omitempty"`
	BatchSize                         int               `json:"batch_size,omitempty"`
	CFGScale                          float64           `json:"cfg_scale,omitempty"`
	Comments                          map[string]string `json:"comments,omitempty"`
	DenoisingStrength                 float64           `json:"denoising_strength,omitempty"`
	DisableExtraNetworks              *bool             `json:"disable_extra_networks,omitempty"`
	DoNotSaveGrid                     *bool             `json:"do_not_save_grid,omitempty"`
	DoNotSaveSamples                  *bool             `json:"do_not_save_samples,omitempty"`
	EnableHr                          bool              `json:"enable_hr,omitempty"`
	Eta                               *float64          `json:"eta,omitempty"`
	FirstphaseHeight                  *int64            `json:"firstphase_height,omitempty"`
	FirstphaseWidth                   *int64            `json:"firstphase_width,omitempty"`
	Height                            int               `json:"height,omitempty"`
	HrCheckpointName                  *string           `json:"hr_checkpoint_name,omitempty"`
	HrNegativePrompt                  *string           `json:"hr_negative_prompt,omitempty"`
	HrPrompt                          *string           `json:"hr_prompt,omitempty"`
	HrResizeX                         int               `json:"hr_resize_x,omitempty"` // Hires width
	HrResizeY                         int               `json:"hr_resize_y,omitempty"` // Hires height
	HrSamplerName                     *string           `json:"hr_sampler_name,omitempty"`
	HrScale                           float64           `json:"hr_scale,omitempty"`
	HrSecondPassSteps                 int64             `json:"hr_second_pass_steps,omitempty"`
	HrUpscaler                        string            `json:"hr_upscaler,omitempty"`
	NIter                             int               `json:"n_iter,omitempty"` // Batch count
	NegativePrompt                    string            `json:"negative_prompt,omitempty"`
	OverrideSettings                  Config            `json:"override_settings,omitempty"`
	OverrideSettingsRestoreAfterwards *bool             `json:"override_settings_restore_afterwards,omitempty"`
	Prompt                            string            `json:"prompt,omitempty"`
	RefinerCheckpoint                 *string           `json:"refiner_checkpoint,omitempty"`
	RefinerSwitchAt                   *float64          `json:"refiner_switch_at,omitempty"`
	RestoreFaces                      bool              `json:"restore_faces,omitempty"`
	SChurn                            *float64          `json:"s_churn,omitempty"`
	SMinUncond                        *float64          `json:"s_min_uncond,omitempty"`
	SNoise                            *float64          `json:"s_noise,omitempty"`
	STmax                             *float64          `json:"s_tmax,omitempty"`
	STmin                             *float64          `json:"s_tmin,omitempty"`
	SamplerIndex                      *string           `json:"sampler_index,omitempty"`
	SamplerName                       string            `json:"sampler_name,omitempty"`
	SaveImages                        *bool             `json:"save_images,omitempty"`
	ScriptArgs                        []string          `json:"script_args,omitempty"`
	ScriptName                        *string           `json:"script_name,omitempty"`
	Seed                              int64             `json:"seed,omitempty"`
	SeedResizeFromH                   *int64            `json:"seed_resize_from_h,omitempty"`
	SeedResizeFromW                   *int64            `json:"seed_resize_from_w,omitempty"`
	SendImages                        *bool             `json:"send_images,omitempty"`
	Steps                             int               `json:"steps,omitempty"`
	Styles                            []string          `json:"styles,omitempty"`
	Subseed                           int64             `json:"subseed,omitempty"`
	SubseedStrength                   float64           `json:"subseed_strength,omitempty"`
	Tiling                            *bool             `json:"tiling,omitempty"`
	Width                             int               `json:"width,omitempty"`
}

func UnmarshalTextToImageResponse(data []byte) (TextToImageResponse, error) {
	var r TextToImageResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TextToImageResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalTextToImageJSONResponse(data []byte) (TextToImageJSONResponse, error) {
	var r TextToImageJSONResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func JSONToTextToImageResponse(data []byte) (*TextToImageResponse, error) {
	r, err := UnmarshalTextToImageJSONResponse(data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling TextToImageJSONResponse: %w", err)
	}
	var info Info
	err = json.Unmarshal([]byte(r.Info), &info)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling Info: %w", err)
	}
	return &TextToImageResponse{
		Images:     r.Images,
		Seeds:      info.AllSeeds,
		Subseeds:   info.AllSubseeds,
		Parameters: r.Parameters,
		Info:       info,
	}, err
}

type TextToImageJSONResponse struct {
	Images     []string       `json:"images"`
	Parameters TextToImageRaw `json:"parameters"`
	Info       string         `json:"info"`
}

type TextToImageResponse struct {
	Images     []string       `json:"images"`
	Seeds      []int64        `json:"seeds"`
	Subseeds   []int64        `json:"subseeds"`
	Parameters TextToImageRaw `json:"parameters"`
	Info       Info           `json:"info"`
}

type Info struct {
	Prompt                        string                 `json:"prompt"`
	AllPrompts                    []string               `json:"all_prompts"`
	NegativePrompt                string                 `json:"negative_prompt"`
	AllNegativePrompts            []string               `json:"all_negative_prompts"`
	Seed                          int64                  `json:"seed"`
	AllSeeds                      []int64                `json:"all_seeds"`
	Subseed                       int64                  `json:"subseed"`
	AllSubseeds                   []int64                `json:"all_subseeds"`
	SubseedStrength               float64                `json:"subseed_strength"`
	Width                         int                    `json:"width"`
	Height                        int                    `json:"height"`
	SamplerName                   string                 `json:"sampler_name"`
	CFGScale                      float64                `json:"cfg_scale"`
	Steps                         int                    `json:"steps"`
	BatchSize                     int                    `json:"batch_size"`
	RestoreFaces                  bool                   `json:"restore_faces"`
	FaceRestorationModel          any                    `json:"face_restoration_model"`
	SDModelName                   *string                `json:"sd_model_name"`
	SDModelHash                   *string                `json:"sd_model_hash"`
	SDVaeName                     *string                `json:"sd_vae_name"`
	SDVaeHash                     *string                `json:"sd_vae_hash"`
	SeedResizeFromW               *int64                 `json:"seed_resize_from_w"`
	SeedResizeFromH               *int64                 `json:"seed_resize_from_h"`
	DenoisingStrength             float64                `json:"denoising_strength"`
	ExtraGenerationParams         *ExtraGenerationParams `json:"extra_generation_params"`
	IndexOfFirstImage             *int64                 `json:"index_of_first_image"`
	Infotexts                     []string               `json:"infotexts"`
	Styles                        []string               `json:"styles"`
	JobTimestamp                  *string                `json:"job_timestamp"`
	ClipSkip                      *int64                 `json:"clip_skip"`
	IsUsingInpaintingConditioning *bool                  `json:"is_using_inpainting_conditioning"`
	Version                       *string                `json:"version"`
}

type ExtraGenerationParams struct {
	LoraHashes string `json:"Lora hashes"`
}
