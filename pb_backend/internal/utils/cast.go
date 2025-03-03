package utils

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"pb_backend/internal/core/domain"
)

// SerializePixel serializes a Pixel struct into a JSON byte slice.
func SerializePixel(p *domain.Pixel) ([]byte, error) {
	return json.Marshal(p)
}

// DeserializePixel deserializes a JSON byte slice into a Pixel struct.
func DeserializePixel(data []byte, p *domain.Pixel) error {
	return json.Unmarshal(data, p)
}

// SerializeRedisPixel serializes a RedisPixel struct into a JSON byte slice.
func SerializeRedisPixel(p *domain.RedisPixel) ([]byte, error) {
	return json.Marshal(p)
}

// DeserializeRedisPixel deserializes a JSON byte slice into a RedisPixel struct.
func DeserializeRedisPixel(data []byte, p *domain.RedisPixel) error {
	return json.Unmarshal(data, p)
}

// SerializeUser serializes a User struct into a JSON byte slice.
func SerializeUser(usr *domain.User) ([]byte, error) {
	return json.Marshal(usr)
}

// DeserializeUser deserializes a JSON byte slice into a User struct.
func DeserializeUser(data []byte, usr *domain.User) error {
	return json.Unmarshal(data, usr)
}

func toRGBA(img domain.Image) *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, int(img.Width), int(img.Height)))
	for _, pixel := range img.Data {
		rgba.Set(int(pixel.X), int(pixel.Y), color.RGBA{
			uint8(pixel.Color[0]),
			uint8(pixel.Color[1]),
			uint8(pixel.Color[2]),
			255,
		})
	}
	return rgba
}

func GetImageBytes(img *domain.Image) ([]byte, error) {

	rgba := toRGBA(*img)
	var buf bytes.Buffer
	err := png.Encode(&buf, rgba)
	return buf.Bytes(), err
}

