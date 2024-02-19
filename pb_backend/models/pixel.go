package models

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Pixel struct {
	X     uint32 `json: "x"`
	Y     uint32 `json: "y"`
	Color []byte `json: "color"`
}

func NewPixel(x uint32, y uint32, color []byte) *Pixel {
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
	return json.Marshal(p)
}

func (p *Pixel) Deserialize(data []byte) error {
	logrus.Info("Serializer.Deserialize received: ", data)
	err := json.Unmarshal(data, p)
	logrus.Info("Serializer.Deserialize returned: ", p)
	return err
}
