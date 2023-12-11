package entities

type Scripts struct {
	ADetailer  *ADetailer  `json:"ADetailer,omitempty"`
	ControlNet *ControlNet `json:"ControlNet,omitempty"`
	CFGRescale *CFGRescale `json:"CFG Rescale Extension,omitempty"`
}

// Deprecated: use ImageGenerationRequest.NewScripts() instead
func (g *ImageGeneration) NewScripts() {
	g.AlwaysOnScripts = &Scripts{}
}

func (g *ImageGenerationRequest) NewScripts() {
	g.Scripts = Scripts{}
}
