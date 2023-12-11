// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    imageToImageRequest, err := UnmarshalImageToImageRequest(bytes)
//    bytes, err = imageToImageRequest.Marshal()

package entities

import (
	"encoding/json"
	"github.com/bwmarrin/discordgo"
)

type MessageAttachment struct {
	discordgo.MessageAttachment
	Image *string `json:"image"`
}

func UnmarshalImageToImageRequest(data []byte) (ImageToImageRequest, error) {
	var r ImageToImageRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ImageToImageRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ImageToImageRequest struct {
	Scripts                           `json:"alwayson_scripts,omitempty"`
	BatchSize                         int                    `json:"batch_size,omitempty"`
	CFGScale                          *float64               `json:"cfg_scale,omitempty"`
	Comments                          map[string]interface{} `json:"comments,omitempty"`
	DenoisingStrength                 *float64               `json:"denoising_strength,omitempty"`
	DisableExtraNetworks              *bool                  `json:"disable_extra_networks,omitempty"`
	DoNotSaveGrid                     *bool                  `json:"do_not_save_grid,omitempty"`
	DoNotSaveSamples                  *bool                  `json:"do_not_save_samples,omitempty"`
	Eta                               *float64               `json:"eta,omitempty"`
	Height                            *int                   `json:"height,omitempty"`
	ImageCFGScale                     *float64               `json:"image_cfg_scale,omitempty"`
	IncludeInitImages                 *bool                  `json:"include_init_images,omitempty"`
	InitImages                        []string               `json:"init_images,omitempty"`
	InitialNoiseMultiplier            *float64               `json:"initial_noise_multiplier,omitempty"`
	InpaintFullRes                    *bool                  `json:"inpaint_full_res,omitempty"`
	InpaintFullResPadding             *int64                 `json:"inpaint_full_res_padding,omitempty"`
	InpaintingFill                    *int64                 `json:"inpainting_fill,omitempty"`
	InpaintingMaskInvert              *int64                 `json:"inpainting_mask_invert,omitempty"`
	LatentMask                        *string                `json:"latent_mask,omitempty"`
	Mask                              *string                `json:"mask,omitempty"`
	MaskBlur                          *int64                 `json:"mask_blur,omitempty"`
	MaskBlurX                         *int64                 `json:"mask_blur_x,omitempty"`
	MaskBlurY                         *int64                 `json:"mask_blur_y,omitempty"`
	NIter                             int                    `json:"n_iter,omitempty"`
	NegativePrompt                    *string                `json:"negative_prompt,omitempty"`
	OverrideSettings                  Config                 `json:"override_settings,omitempty"`
	OverrideSettingsRestoreAfterwards *bool                  `json:"override_settings_restore_afterwards,omitempty"`
	Prompt                            string                 `json:"prompt"`
	RefinerCheckpoint                 *string                `json:"refiner_checkpoint,omitempty"`
	RefinerSwitchAt                   *float64               `json:"refiner_switch_at,omitempty"`
	ResizeMode                        *int64                 `json:"resize_mode,omitempty"`
	RestoreFaces                      *bool                  `json:"restore_faces,omitempty"`
	SChurn                            *float64               `json:"s_churn,omitempty"`
	SMinUncond                        *float64               `json:"s_min_uncond,omitempty"`
	SNoise                            *float64               `json:"s_noise,omitempty"`
	STmax                             *float64               `json:"s_tmax,omitempty"`
	STmin                             *float64               `json:"s_tmin,omitempty"`
	SamplerIndex                      *string                `json:"sampler_index,omitempty"`
	SamplerName                       *string                `json:"sampler_name,omitempty"`
	SaveImages                        *bool                  `json:"save_images,omitempty"`
	ScriptArgs                        []string               `json:"script_args,omitempty"`
	ScriptName                        *string                `json:"script_name,omitempty"`
	Seed                              *int64                 `json:"seed,omitempty"`
	SeedResizeFromH                   *int64                 `json:"seed_resize_from_h,omitempty"`
	SeedResizeFromW                   *int64                 `json:"seed_resize_from_w,omitempty"`
	SendImages                        *bool                  `json:"send_images,omitempty"`
	Steps                             *int                   `json:"steps,omitempty"`
	Styles                            []string               `json:"styles,omitempty"`
	Subseed                           *int64                 `json:"subseed,omitempty"`
	SubseedStrength                   *float64               `json:"subseed_strength,omitempty"`
	Tiling                            *bool                  `json:"tiling,omitempty"`
	Width                             *int                   `json:"width,omitempty"`
}

func UnmarshalImageToImageResponse(data []byte) (*ImageToImageResponse, error) {
	var r ImageToImageResponse
	err := json.Unmarshal(data, &r)
	return &r, err
}

func (r *ImageToImageResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ImageToImageResponse struct {
	// The generated image in base64 format.
	Images     []string       `json:"images,omitempty"`
	Info       string         `json:"info"`
	Parameters map[string]any `json:"parameters"`
}
