// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    textToImageRequest, err := UnmarshalTextToImageRequest(bytes)
//    bytes, err = textToImageRequest.Marshal()

package entities

import "encoding/json"

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
