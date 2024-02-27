package service

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	// "os"
	"pb_backend/models"

	"github.com/sirupsen/logrus"
)

type Image struct {
	Height uint
	Width  uint
	Data   []models.Pixel
}

func NewImage(height uint, width uint) *Image {
	return &Image{
		Height: height,
		Width:  width,
		Data:   make([]models.Pixel, 0),
	}
}

type ImageService struct {
	img    draw.Image
	imgBuf []byte
}

func NewImageService() *ImageService {
	return &ImageService{
		img:    nil,
		imgBuf: nil,
	}
}

func (img *Image) toRGBA() *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, int(img.Width), int(img.Height)))
	for _, pixel := range img.Data {
		// logrus.Info(i)
		// logrus.Info(pixel.Color)
		rgba.Set(int(pixel.X), int(pixel.Y), color.RGBA{
			uint8(pixel.Color[0]),
			uint8(pixel.Color[1]),
			uint8(pixel.Color[2]),
			255,
		})
	}
	// for _, pixel := range img.Data {
	// 	logrus.Info(pixel)
	// }
	return rgba
}

func (service *ImageService) GetImageBytes(img *Image) []byte {
	logrus.Info("Entered GetImageBytes")

	

	// logrus.Info(img.Data)
	
	rgba := img.toRGBA()
	var buf bytes.Buffer
	err := png.Encode(&buf, rgba)
	if err != nil {
		logrus.Error("shit shit")
		logrus.Error(err)
	}
	return buf.Bytes()
}

// func loadImage(loadPath string) draw.Image {
// 	f, err := os.Open(loadPath)
// 	if err != nil {
// 		logrus.Error(err)
// 	}
// 	defer f.Close()
// 	pngImg, err := png.Decode(f)
// 	if err != nil {
// 		logrus.Error(err)
// 	}
// 	return pngImg.(draw.Image)
// }
