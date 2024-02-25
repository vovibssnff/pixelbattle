package models

import (
	"encoding/json"
	// "github.com/sirupsen/logrus"

)

type Color [3]uint

type Pixel struct {
	X     uint `json: "x"`
	Y     uint `json: "y"`
	Color []uint `json: "color"`
	Userid string `json: "userid"`
}

func NewPixel(x uint, y uint, color []uint) *Pixel {
	return &Pixel{
		X:     x,
		Y:     y,
		Color: color,
	}
}

type RedisPixel struct {
	UserID string `json: "userid"`
	Color []uint `json: "color"`
	Timestamp int64 `json: "timestamp"`
}

func NewRedisPixel(userid string, color []uint, timestamp int64) *RedisPixel{
	return &RedisPixel{
		UserID: userid,
		Color: color,
		Timestamp: timestamp,
	}
}

type SerializationService interface {
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	ToRedisFormat() ([]byte, error)
	FromRedisFormat(data []byte) error
}

func (p *Pixel) Serialize() ([]byte, error) {
	// logrus.Info("Serializer.Serialize received: ", p)
	// resp, _ := json.Marshal(p)
	// logrus.Info(string(resp))
	return json.Marshal(p)
}

func (p *Pixel) Deserialize(data []byte) error {
	// logrus.Info("Serializer.Deserialize received: ", data)
	err := json.Unmarshal(data, p)
	// logrus.Info("Serializer.Deserialize returned: ", p)
	return err
}

func (p *RedisPixel) ToRedisFormat() ([]byte, error) {
	return json.Marshal(p)
}

func (p *RedisPixel) FromRedisFormat(data []byte) error {
	err := json.Unmarshal(data, p)
	return err
}
