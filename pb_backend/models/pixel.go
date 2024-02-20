package models

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Color [3]uint

type Pixel struct {
	X     uint `json: "x"`
	Y     uint `json: "y"`
	Color []uint `json: "color"`
}

func NewPixel(x uint, y uint, color []uint) *Pixel {
	return &Pixel{
		X:     x,
		Y:     y,
		Color: color,
	}
}

type SerializationService interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

func (p *Pixel) Serialize() ([]byte, error) {
	logrus.Info("Serializer.Serialize received: ", p)
	resp, _ := json.Marshal(p)
	logrus.Info(string(resp))
	return json.Marshal(p)
}

func (p *Pixel) Deserialize(data []byte) error {
	logrus.Info("Serializer.Deserialize received: ", data)
	err := json.Unmarshal(data, p)
	logrus.Info("Serializer.Deserialize returned: ", p)
	return err
}
