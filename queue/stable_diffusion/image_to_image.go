package stable_diffusion

import (
	"bytes"
	"errors"
	"fmt"

	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/utils"
)

// TODO: Implement separate processing for Img2Img, possibly use github.com/SpenserCai/sd-webui-go/intersvc
// Deprecated: still using processCurrentImagine
func (q *SDQueue) processImg2ImgImagine() error {
	// defer q.done()
	return q.processCurrentImagine()
}

func (q *SDQueue) imageToImage() ([]string, error) {
	queue := q.currentImagine
	img2img := t2iToImg2Img(queue.TextToImageRequest)

	err := calculateImg2ImgDimensions(queue, &img2img)
	if err != nil {
		return nil, err
	}

	resp, err := q.stableDiffusionAPI.ImageToImageRequest(&img2img)
	if err != nil {
		return nil, err
	}

	return resp.Images, nil
}

func calculateImg2ImgDimensions(queue *SDQueueItem, img2img *entities.ImageToImageRequest) error {
	if queue.Img2ImgItem.Image == nil {
		return errors.New("no attached images found, skipping img2img generation")
	}

	bin := queue.Img2ImgItem.Image.Bytes()
	width, height, err := utils.GetImageSize(bytes.NewReader(bin))
	if err != nil {
		return fmt.Errorf("error getting image size: %w", err)
	}

	// calculate aspect ratio. e.g. 512x768 = 2:3 to the nearest whole number
	gcd := calculateGCD(width, height)
	aspectRatio := fmt.Sprintf("%d:%d", width/gcd, height/gcd)

	*img2img.Width, *img2img.Height = aspectRatioCalculation(aspectRatio, initializedWidth, initializedHeight)

	base64, err := queue.Img2ImgItem.Image.Base64()
	if err != nil {
		return fmt.Errorf("error converting image to base64: %w", err)
	}
	img2img.InitImages = append(img2img.InitImages, base64)
	return err
}

func calculateGCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func t2iToImg2Img(textToImage *entities.TextToImageRequest) entities.ImageToImageRequest {
	img2img := entities.ImageToImageRequest{
		Scripts:                           textToImage.Scripts,
		BatchSize:                         textToImage.BatchSize,
		CFGScale:                          &textToImage.CFGScale,
		DenoisingStrength:                 &textToImage.DenoisingStrength,
		Height:                            &textToImage.Height,
		ImageCFGScale:                     &textToImage.CFGScale,
		IncludeInitImages:                 nil,
		InitImages:                        nil,
		NIter:                             textToImage.NIter,
		NegativePrompt:                    &textToImage.NegativePrompt,
		OverrideSettings:                  textToImage.OverrideSettings,
		OverrideSettingsRestoreAfterwards: textToImage.OverrideSettingsRestoreAfterwards,
		Prompt:                            textToImage.Prompt,
		RefinerCheckpoint:                 textToImage.RefinerCheckpoint,
		RefinerSwitchAt:                   textToImage.RefinerSwitchAt,
		RestoreFaces:                      &textToImage.RestoreFaces,
		SChurn:                            textToImage.SChurn,
		SMinUncond:                        textToImage.SMinUncond,
		SNoise:                            textToImage.SNoise,
		STmax:                             textToImage.STmax,
		STmin:                             textToImage.STmin,
		SamplerIndex:                      textToImage.SamplerIndex,
		SamplerName:                       &textToImage.SamplerName,
		SaveImages:                        textToImage.SaveImages,
		ScriptArgs:                        textToImage.ScriptArgs,
		ScriptName:                        textToImage.ScriptName,
		Seed:                              &textToImage.Seed,
		SeedResizeFromH:                   textToImage.SeedResizeFromH,
		SeedResizeFromW:                   textToImage.SeedResizeFromW,
		SendImages:                        textToImage.SendImages,
		Steps:                             &textToImage.Steps,
		Styles:                            textToImage.Styles,
		Subseed:                           &textToImage.Subseed,
		SubseedStrength:                   &textToImage.SubseedStrength,
		Tiling:                            textToImage.Tiling,
		Width:                             &textToImage.Width,
	}
	return img2img
}
