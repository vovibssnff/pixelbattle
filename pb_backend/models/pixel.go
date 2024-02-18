package models

import (
	"bytes"
	"encoding/gob"
)

type Pixel struct {
	x     uint32
	y     uint32
	color []byte
}

func NewPixel(x uint32, y uint32, color []byte) *Pixel {
	return &Pixel{
		x:     x,
		y:     y,
		color: color,
	}
}

type SerializationService interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

func (p *Pixel) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize deserializes bytes into a Pixel using gob.
func (p *Pixel) Deserialize(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(p)
}
