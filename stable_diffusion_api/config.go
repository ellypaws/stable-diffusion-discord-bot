// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    aPIConfig, err := UnmarshalConfig(bytes)
//    bytes, err = aPIConfig.Marshal()

package stable_diffusion_api

import (
	"stable_diffusion_bot/entities"
)

func (api *apiImplementation) GetConfig() (*entities.Config, error) {
	getURL := "/sdapi/v1/options"

	body, err := api.GET(getURL)
	if err != nil {
		return nil, err
	}

	var apiConfig entities.Config
	apiConfig, err = entities.UnmarshalConfig(body)
	if err != nil {
		return nil, err
	}

	return &apiConfig, nil
}

func (api *apiImplementation) GetCheckpoint() (*string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return nil, err
	}

	return apiConfig.SDModelCheckpoint, nil
}

func (api *apiImplementation) GetVAE() (*string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return nil, err
	}

	return apiConfig.SDVae, nil
}

func (api *apiImplementation) GetHypernetwork() (*string, error) {
	apiConfig, err := api.GetConfig()
	if err != nil {
		return nil, err
	}

	return apiConfig.SDHypernetwork, nil
}
