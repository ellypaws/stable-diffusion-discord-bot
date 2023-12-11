package entities

import (
	"strings"
)

type ADetailerParameters struct {
	AdModel                    string  `json:"ad_model,omitempty"`
	AdPrompt                   string  `json:"ad_prompt,omitempty"`
	AdNegativePrompt           string  `json:"ad_negative_prompt,omitempty"`
	AdConfidence               float64 `json:"ad_confidence,omitempty"`
	AdMaskKLargest             int     `json:"ad_mask_k_largest,omitempty"`
	AdMaskMinRatio             float64 `json:"ad_mask_min_ratio,omitempty"`
	AdMaskMaxRatio             float64 `json:"ad_mask_max_ratio,omitempty"`
	AdDilateErode              int     `json:"ad_dilate_erode,omitempty"`
	AdXOffset                  int     `json:"ad_x_offset,omitempty"`
	AdYOffset                  int     `json:"ad_y_offset,omitempty"`
	AdMaskMergeInvert          string  `json:"ad_mask_merge_invert,omitempty"`
	AdMaskBlur                 int     `json:"ad_mask_blur,omitempty"`
	AdDenoisingStrength        float64 `json:"ad_denoising_strength,omitempty"`
	AdInpaintOnlyMasked        bool    `json:"ad_inpaint_only_masked,omitempty"`
	AdInpaintOnlyMaskedPadding int     `json:"ad_inpaint_only_masked_padding,omitempty"`
	AdUseInpaintWidthHeight    bool    `json:"ad_use_inpaint_width_height,omitempty"`
	AdInpaintWidth             int     `json:"ad_inpaint_width,omitempty"`
	AdInpaintHeight            int     `json:"ad_inpaint_height,omitempty"`
	AdUseSteps                 bool    `json:"ad_use_steps,omitempty"`
	AdSteps                    int     `json:"ad_steps,omitempty"`
	AdUseCfgScale              bool    `json:"ad_use_cfg_scale,omitempty"`
	AdCfgScale                 float64 `json:"ad_cfg_scale,omitempty"`
	AdUseSampler               bool    `json:"ad_use_sampler,omitempty"`
	AdSampler                  string  `json:"ad_sampler,omitempty"`
	AdUseNoiseMultiplier       bool    `json:"ad_use_noise_multiplier,omitempty"`
	AdNoiseMultiplier          float64 `json:"ad_noise_multiplier,omitempty"`
	AdUseClipSkip              bool    `json:"ad_use_clip_skip,omitempty"`
	AdClipSkip                 int     `json:"ad_clip_skip,omitempty"`
	AdRestoreFace              bool    `json:"ad_restore_face,omitempty"`
	AdControlnetModel          string  `json:"ad_controlnet_model,omitempty"`
	AdControlnetModule         *string `json:"ad_controlnet_module,omitempty"`
	AdControlnetWeight         float64 `json:"ad_controlnet_weight,omitempty"`
	AdControlnetGuidanceStart  float64 `json:"ad_controlnet_guidance_start,omitempty"`
	AdControlnetGuidanceEnd    float64 `json:"ad_controlnet_guidance_end,omitempty"`
}

func (detailer *ADetailer) CreateArgs() ADetailerParameters {
	detailer.Args = []*ADetailerParameters{}
	return ADetailerParameters{}
}

func (detailer *ADetailer) InsertArgs(params ...*ADetailerParameters) ADetailerParameters {
	if len(params) > 0 {
		detailer.Args = append(detailer.Args, params...)
	}
	detailer.Args = []*ADetailerParameters{}
	return ADetailerParameters{}
}

// AppendSegModel is a function that adds a new segmentation model to the ADetailer's current list of models.
func (detailer *ADetailer) AppendSegModel(parameters ADetailerParameters) {
	detailer.Args = append(detailer.Args, &parameters)
}

func (detailer *ADetailer) AppendSegModelByString(segmModelName string, genProperties *ImageGenerationRequest) {
	stringsToSlice := strings.Split(segmModelName, ",")
	for _, eachModel := range stringsToSlice {
		convertToParams := SegModelParameters(eachModel, genProperties) // SegModelParameters also sets the width and height
		detailer.AppendSegModel(convertToParams)
	}
}

var segModelDimensions = map[string][]int{
	"person_yolov8n-seg.pt": {768, 1152},
	"face_yolov8n.pt":       {768, 768},
}

// SetAdInpaintWidthAndHeight is a function that add width and height based on the segment model
func (parameters *ADetailerParameters) SetAdInpaintWidthAndHeight(segModel string, genProperties *ImageGenerationRequest) {
	calculatedWidth := int(genProperties.HrScale * float64(genProperties.Width))
	calculatedHeight := int(genProperties.HrScale * float64(genProperties.Height))

	if defaultDimensions, exist := segModelDimensions[segModel]; exist {
		parameters.AdInpaintWidth = max(defaultDimensions[0], int(genProperties.Width), genProperties.HrResizeX, calculatedWidth)
		parameters.AdInpaintHeight = max(defaultDimensions[1], genProperties.Height, genProperties.HrResizeY, calculatedHeight)
	}
}

// SegModelParameters creates an ADetailerParameters for a given segmentation model.
// It uses information from an ImageGenerationRequest instance to configure the parameters.
func SegModelParameters(segModel string, genProperties *ImageGenerationRequest) ADetailerParameters {
	parameters := ADetailerParameters{AdModel: segModel}

	parameters.SetAdInpaintWidthAndHeight(segModel, genProperties)

	if genProperties.SamplerName != "" {
		parameters.AdUseSampler = true
		parameters.AdSampler = genProperties.SamplerName
	}

	if genProperties.CFGScale != 0 {
		parameters.AdCfgScale = genProperties.CFGScale
	}

	return parameters
}

type ADetailer struct {
	Args []*ADetailerParameters `json:"args,omitempty"`
}

// Deprecated: use ImageGenerationRequest.NewADetailer() instead
func (g *ImageGeneration) NewADetailer() {
	if g.AlwaysOnScripts == nil {
		g.NewScripts()
	}
	g.AlwaysOnScripts.NewADetailerWithArgs()
}

func (g *ImageGenerationRequest) NewADetailer() {
	g.TextToImageRequest.Scripts.NewADetailerWithArgs()
}

func (s *Scripts) NewADetailerWithArgs() {
	s.ADetailer = &ADetailer{}
	s.ADetailer.CreateArgs()
}
