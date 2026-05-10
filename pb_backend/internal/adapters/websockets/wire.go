package websockets

import (
	"encoding/binary"
	"errors"

	"pb_backend/internal/core/domain"
)

// WebSocket subprotocols (plan §11.2).
const (
	SubprotocolV2 = "pixelbattle.v2"
	SubprotocolV1 = "pixelbattle.v1"
)

const (
	wirePixelV2Type     = byte(1)
	wirePixelV2Len      = 20 // type + xy + rgb + client_seq + ts
	wirePixelV2LenLegacy = 16 // pre–client_seq v2 frames (client_seq=0)
	wirePixelV2MaxCoord  = 65535
)

var errWirePixelV2 = errors.New("invalid pixelbattle.v2 binary frame")

// encodePixelV2 encodes a broadcast pixel: type, x, y, rgb, client_seq, server_recv_ms (LE).
func encodePixelV2(p *domain.Pixel) []byte {
	if p == nil || len(p.Color) != 3 || !coordsFitWireV2(p.X, p.Y) {
		return nil
	}
	b := make([]byte, wirePixelV2Len)
	b[0] = wirePixelV2Type
	binary.LittleEndian.PutUint16(b[1:3], uint16(p.X))
	binary.LittleEndian.PutUint16(b[3:5], uint16(p.Y))
	b[5] = uint8(p.Color[0])
	b[6] = uint8(p.Color[1])
	b[7] = uint8(p.Color[2])
	binary.LittleEndian.PutUint32(b[8:12], p.ClientSeq)
	binary.LittleEndian.PutUint64(b[12:20], uint64(p.ServerRecvMs))
	return b
}

// decodeClientPixelV2 parses an inbound v2 frame. Timestamp tail is ClientSentMs.
func decodeClientPixelV2(data []byte) (domain.Pixel, error) {
	if len(data) < 1 || data[0] != wirePixelV2Type {
		return domain.Pixel{}, errWirePixelV2
	}
	switch len(data) {
	case wirePixelV2LenLegacy:
		x := uint(binary.LittleEndian.Uint16(data[1:3]))
		y := uint(binary.LittleEndian.Uint16(data[3:5]))
		return domain.Pixel{
			X: x,
			Y: y,
			Color: []uint{
				uint(data[5]),
				uint(data[6]),
				uint(data[7]),
			},
			ClientSentMs: int64(binary.LittleEndian.Uint64(data[8:16])),
			ClientSeq:    0,
		}, nil
	case wirePixelV2Len:
		x := uint(binary.LittleEndian.Uint16(data[1:3]))
		y := uint(binary.LittleEndian.Uint16(data[3:5]))
		return domain.Pixel{
			X: x,
			Y: y,
			Color: []uint{
				uint(data[5]),
				uint(data[6]),
				uint(data[7]),
			},
			ClientSeq:    binary.LittleEndian.Uint32(data[8:12]),
			ClientSentMs: int64(binary.LittleEndian.Uint64(data[12:20])),
		}, nil
	default:
		return domain.Pixel{}, errWirePixelV2
	}
}

func coordsFitWireV2(x, y uint) bool {
	return x <= wirePixelV2MaxCoord && y <= wirePixelV2MaxCoord
}
