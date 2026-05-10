package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// unmarshalUserIDJSON accepts JSON string or number (legacy Redis / clients).
func unmarshalUserIDJSON(raw json.RawMessage) (string, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return "", nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return "", err
		}
		return s, nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		return n.String(), nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return "", fmt.Errorf("userid: %w", err)
	}
	return strconv.FormatInt(int64(f), 10), nil
}

// VKUserID returns the canonical MongoDB _id for a VK numeric user id (e.g. "vk_12345").
func VKUserID(vkNumericID int) string {
	return fmt.Sprintf("vk_%d", vkNumericID)
}

type Color [3]uint

type Pixel struct {
	X            uint   `json:"x"`
	Y            uint   `json:"y"`
	Color        []uint `json:"color"`
	Userid       string `json:"userid"`
	Faculty      string `json:"faculty"`
	ClientSentMs int64  `json:"client_sent_ms,omitempty"`
	ServerRecvMs int64  `json:"server_recv_ms,omitempty"`
}

// UnmarshalJSON accepts userid as string or number (WebSocket clients vary).
func (p *Pixel) UnmarshalJSON(data []byte) error {
	var aux struct {
		X            uint            `json:"x"`
		Y            uint            `json:"y"`
		Color        []uint          `json:"color"`
		Userid       json.RawMessage `json:"userid"`
		Faculty      string          `json:"faculty"`
		ClientSentMs int64           `json:"client_sent_ms"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := unmarshalUserIDJSON(aux.Userid)
	if err != nil {
		return err
	}
	p.X, p.Y, p.Color, p.Faculty = aux.X, aux.Y, aux.Color, aux.Faculty
	p.Userid = uid
	p.ClientSentMs = aux.ClientSentMs
	return nil
}

type RedisPixel struct {
	UserId    string `json:"userid"`
	Faculty   string `json:"faculty"`
	Color     []uint `json:"color"`
	Timestamp int64  `json:"timestamp"`
}

// UnmarshalJSON accepts userid as string or number (legacy keys in Redis).
func (p *RedisPixel) UnmarshalJSON(data []byte) error {
	var aux struct {
		UserId    json.RawMessage `json:"userid"`
		Faculty   string          `json:"faculty"`
		Color     []uint          `json:"color"`
		Timestamp int64           `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	uid, err := unmarshalUserIDJSON(aux.UserId)
	if err != nil {
		return err
	}
	p.UserId, p.Faculty, p.Color, p.Timestamp = uid, aux.Faculty, aux.Color, aux.Timestamp
	return nil
}

type HeatMapUnit struct {
	X   uint
	Y   uint
	Len uint
}

type UserStats struct {
	TotalPixelsPlaced int `bson:"total_pixels_placed"`
	ActivePixels      int `bson:"active_pixels"`
}

type User struct {
	ID           string    `json:"id" bson:"_id"`
	FirstName    string    `json:"name" bson:"first_name"`
	LastName     string    `json:"surname" bson:"last_name"`
	PasswordHash string    `json:"-" bson:"password_hash,omitempty"`
	AccessToken  string    `json:"-" bson:"access_token,omitempty"`
	Faculty      string    `json:"faculty" bson:"faculty"`
	Stats        UserStats `json:"-" bson:"stats"`
}

type Image struct {
	Height uint
	Width  uint
	Data   []Pixel
}

type BroadcastStats struct {
	ID                string `json:"id" bson:"_id"`
	FirstName         string `json:"name" bson:"first_name"`
	LastName          string `json:"surname" bson:"last_name"`
	TotalPixelsPlaced int    `json:"total_pixels_placed" bson:"total_pixels_placed"`
	ActivePixels      int    `json:"active_pixels" bson:"active_pixels"`
}
