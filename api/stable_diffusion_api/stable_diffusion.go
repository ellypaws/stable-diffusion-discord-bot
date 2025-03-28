package stable_diffusion_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"stable_diffusion_bot/discord_bot/handlers"
	"stable_diffusion_bot/entities"
)

type apiImplementation struct {
	host   string
	client *http.Client
}

type Config struct {
	Host string
}

func New(cfg Config) (StableDiffusionAPI, error) {
	if cfg.Host == "" {
		return nil, errors.New("missing host")
	}

	return &apiImplementation{
		host: cfg.Host,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}, nil
}

func (api *apiImplementation) Client() *http.Client { return api.client }
func (api *apiImplementation) Host(url ...string) string {
	if len(url) > 0 {
		url = slices.Insert(url, 0, api.host)
		return strings.Join(url, "")
	}
	return api.host
}

// Deprecated: Use the entities.ImageToImageResponse instead
type ImageToImageResponse struct {
	Images     []string          `json:"images,omitempty"`
	Info       string            `json:"info"`
	Parameters map[string]string `json:"parameters"`
}

type StableDiffusionModel struct {
	Title     string `json:"title"`
	ModelName string `json:"model_name"`
	Hash      string `json:"hash"`
	Sha256    string `json:"sha256"`
	Filename  string `json:"filename"`
	Config    string `json:"config"`
}

// Deprecated: Use the entities.TextToImageRequest in entities.ImageGeneration instead
type TextToImageRequest struct {
	Prompt            string            `json:"prompt"`
	NegativePrompt    string            `json:"negative_prompt"`
	Width             int               `json:"width"`
	Height            int               `json:"height"`
	RestoreFaces      bool              `json:"restore_faces"`
	EnableHR          bool              `json:"enable_hr"`
	HRUpscaleRate     float64           `json:"hr_scale"`
	HRUpscaler        string            `json:"hr_upscaler"`
	HRSteps           int64             `json:"hr_second_pass_steps"`
	HRResizeX         int               `json:"hr_resize_x"`
	HRResizeY         int               `json:"hr_resize_y"`
	DenoisingStrength float64           `json:"denoising_strength"`
	BatchSize         int               `json:"batch_size"`
	Seed              int64             `json:"seed"`
	Subseed           int64             `json:"subseed"`
	SubseedStrength   float64           `json:"subseed_strength"`
	SamplerName       string            `json:"sampler_name"`
	CfgScale          float64           `json:"cfg_scale"`
	Steps             int               `json:"steps"`
	NIter             int               `json:"n_iter"`
	AlwaysOnScripts   *entities.Scripts `json:"alwayson_scripts,omitempty"`
}

func (api *apiImplementation) CachePreview(c Cacheable) (Cacheable, error) {
	if c == nil {
		return nil, errors.New("cache is nil")
	}
	// _, err := c.GetCache(api)
	// if err != nil {
	//	return c, err
	// }
	if c.Len() > 2 {
		log.Printf("Successfully cached %v %T from api: %v...", c.Len(), c, c.String(0))
	}
	// return cache, nil

	return c, nil
}

func (api *apiImplementation) PopulateCache() (errors []error) {
	var caches = []Cacheable{
		CheckpointCache,
		LoraCache,
		VAECache,
		HypernetworkCache,
		EmbeddingCache,
	}
	if !handlers.CheckAPIAlive(api.host) {
		return []error{fmt.Errorf("could not populate caches: %s", handlers.DeadAPI)}
	}
	for _, cache := range caches {
		cache, err := cache.GetCache(api)
		if err != nil {
			errors = append(errors, fmt.Errorf("error caching %T: %w", cache, err))
		}
		_, err = api.CachePreview(cache)
		if err != nil {
			errors = append(errors, fmt.Errorf("error previewing %T: %w", cache, err))
		}
	}

	return
}

func (api *apiImplementation) RefreshCache(cache Cacheable) (Cacheable, error) {
	if cache == nil {
		return nil, errors.New("cache is nil")
	}
	return cache.Refresh(api)
}

func (api *apiImplementation) TextToImageRequest(req *entities.TextToImageRequest) (*entities.TextToImageResponse, error) {
	jsonData, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	return api.TextToImageRaw(jsonData)
}

func (api *apiImplementation) TextToImageRaw(req []byte) (*entities.TextToImageResponse, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	if req == nil {
		return nil, errors.New("missing request")
	}

	out := new(bytes.Buffer)
	err := Do(api.client, http.MethodPost, api.Host("/sdapi/v1/txt2img"), bytes.NewReader(req), out)
	if err != nil {
		return nil, err
	}

	return entities.JSONToTextToImageResponse(out.Bytes())
}

func (api *apiImplementation) ImageToImageRequest(req *entities.ImageToImageRequest) (*entities.ImageToImageResponse, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	if req == nil {
		return nil, errors.New("missing request")
	}

	response := new(entities.ImageToImageResponse)
	err := POST(api.client, api.Host("/sdapi/v1/img2img"), req, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

type UpscaleRequest struct {
	ResizeMode         int                          `json:"resize_mode"`
	UpscalingResize    int                          `json:"upscaling_resize"`
	Upscaler1          string                       `json:"upscaler_1"`
	TextToImageRequest *entities.TextToImageRequest `json:"text_to_image_request"`
}

type upscaleJSONRequest struct {
	ResizeMode      int    `json:"resize_mode"`
	UpscalingResize int    `json:"upscaling_resize"`
	Upscaler1       string `json:"upscaler_1"`
	Image           string `json:"image"`
}

type UpscaleResponse struct {
	Image string `json:"image"`
}

func (api *apiImplementation) UpscaleImage(upscaleReq *UpscaleRequest) (*UpscaleResponse, error) {
	if !handlers.CheckAPIAlive(api.host) {
		return nil, errors.New(handlers.DeadAPI)
	}
	if upscaleReq == nil {
		return nil, errors.New("missing request")
	}

	regenerateRequest := upscaleReq.TextToImageRequest
	if regenerateRequest == nil {
		return nil, errors.New("missing text to image request")
	}
	regenerateRequest.NIter = 1

	regeneratedImage, err := api.TextToImageRequest(regenerateRequest)
	if err != nil {
		return nil, err
	}

	if len(regeneratedImage.Images) < 1 {
		return nil, errors.New("no images returned from text to image request to upscale")
	}

	jsonReq := &upscaleJSONRequest{
		ResizeMode:      upscaleReq.ResizeMode,
		UpscalingResize: upscaleReq.UpscalingResize,
		Upscaler1:       upscaleReq.Upscaler1,
		Image:           regeneratedImage.Images[0],
	}

	upscaleResponse := new(UpscaleResponse)
	err = POST(api.client, api.Host("/sdapi/v1/extra-single-image"), jsonReq, upscaleResponse)
	if err != nil {
		return nil, err
	}

	return upscaleResponse, nil
}

type ProgressResponse struct {
	Progress    float64 `json:"progress"`
	EtaRelative float64 `json:"eta_relative"`
}

func (api *apiImplementation) GetCurrentProgress() (*ProgressResponse, error) {
	getURL := api.Host("/sdapi/v1/progress")

	progress, err := GET[ProgressResponse](api.client, getURL)
	if err != nil {
		return nil, err
	}

	return progress, nil
}

// Deprecated: Use entities.Config instead
type POSTConfig struct {
	SdModelCheckpoint string `json:"sd_model_checkpoint,omitempty"`
}

// GET is a generic function to make a GET request to the API
// It returns the response body as the specified type
func GET[T any](client *http.Client, url string) (*T, error) {
	v := new(T)
	err := Do(client, http.MethodGet, url, nil, v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// POST is a generic function to make a POST request to the API
// It writes to v the response body as the specified type
func POST[T any](client *http.Client, url string, body any, v *T) error {
	if body == nil {
		return Do(client, http.MethodPost, url, nil, v)
	}
	var reader io.Reader
	switch body := body.(type) {
	case *bytes.Buffer:
		reader = body
	case []byte:
		reader = bytes.NewReader(body)
	default:
		writer := new(bytes.Buffer)
		if err := json.NewEncoder(writer).Encode(body); err != nil {
			return err
		}
		reader = writer
	}
	return Do(client, http.MethodPost, url, reader, v)
}

func Do(client *http.Client, method string, url string, body io.Reader, v any) error {
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	request, err := http.NewRequestWithContext(timeout, method, url, body)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer closeResponseBody(response.Body)

	if response.StatusCode != http.StatusOK {
		responseString := " (unknown error)"
		body, _ := io.ReadAll(response.Body)
		if len(body) > 0 {
			responseString = fmt.Sprintf("\n```json\n%s\n```", body)
		}
		return fmt.Errorf("unexpected status code: `%s`%s", response.Status, responseString)
	}

	if v == nil {
		return nil
	}

	switch v := v.(type) {
	case *bytes.Buffer:
		_, err = io.Copy(v, response.Body)
		if err != nil {
			return err
		}
	default:
		err = json.NewDecoder(response.Body).Decode(&v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (api *apiImplementation) UpdateConfiguration(config entities.Config) error {
	if !handlers.CheckAPIAlive(api.host) {
		return errors.New(handlers.DeadAPI)
	}

	err := POST(api.client, api.Host("/sdapi/v1/options"), config, (*map[string]any)(nil))
	if err != nil {
		return err
	}

	return nil
}

func closeResponseBody(closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Printf("Error closing response body: %v", err)
	}
}

// interrupt by posting to /sdapi/v1/interrupt using the POST() function
func (api *apiImplementation) Interrupt() error {
	if !handlers.CheckAPIAlive(api.host) {
		return errors.New(handlers.DeadAPI)
	}

	err := POST[error](api.client, api.Host("/sdapi/v1/interrupt"), nil, nil)
	if err != nil {
		return err
	}

	return nil
}
