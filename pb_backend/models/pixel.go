package models

import (
	"encoding/json"
)

type Color [3]uint

type Pixel struct {
	X       uint   `json: "x"`
	Y       uint   `json: "y"`
	Color   []uint `json: "color"`
	Userid  int    `json: "userid"`
	Faculty string `json: "faculty"`
}

func NewPixel(x uint, y uint, color []uint) *Pixel {
	return &Pixel{
		X:     x,
		Y:     y,
		Color: color,
	}
}

type RedisPixel struct {
	UserID    int    `json: "userid"`
	Faculty   string `json: "faculty"`
	Color     []uint `json: "color"`
	Timestamp int64  `json: "timestamp"`
}

func NewRedisPixel(userid int, faculty string, color []uint, timestamp int64) *RedisPixel {
	return &RedisPixel{
		UserID:    userid,
		Faculty:   faculty,
		Color:     color,
		Timestamp: timestamp,
	}
}

type HeatMapUnit struct {
	X   uint
	Y   uint
	Len uint
}

type SerializationService interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	ToRedisFormat() ([]byte, error)
	FromRedisFormat(data []byte) error
}

func (p *Pixel) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Pixel) Deserialize(data []byte) error {
	err := json.Unmarshal(data, p)
	return err
}

func (p *RedisPixel) ToRedisFormat() ([]byte, error) {
	return json.Marshal(p)
}

func (p *RedisPixel) FromRedisFormat(data []byte) error {
	err := json.Unmarshal(data, p)
	return err
}
