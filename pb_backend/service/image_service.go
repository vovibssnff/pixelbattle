package service

import (
	"bytes"
	"image/draw"
	"image/png"
	"os"
	"github.com/sirupsen/logrus"
)

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

func (service *ImageService) GetImageBytes(loadPath string) []byte {
	service.img = loadImage(loadPath)
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, service.img); err != nil {
		logrus.Error(err)
	}
	service.imgBuf = buf.Bytes()
	return service.imgBuf
}

func loadImage(loadPath string) draw.Image {
	f, err := os.Open(loadPath)
	if err != nil {
		logrus.Error(err)
	}
	defer f.Close()
	pngImg, err := png.Decode(f)
	if err != nil {
		logrus.Error(err)
	}
	return pngImg.(draw.Image)
}
