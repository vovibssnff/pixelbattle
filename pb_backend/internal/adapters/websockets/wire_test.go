package websockets

import (
	"testing"

	"pb_backend/internal/core/domain"
)

func TestEncodeDecodePixelV2(t *testing.T) {
	p := &domain.Pixel{
		X: 10, Y: 20,
		Color:        []uint{1, 2, 3},
		ServerRecvMs: 1700000000555,
	}
	b := encodePixelV2(p)
	if len(b) != wirePixelV2Len {
		t.Fatalf("len=%d", len(b))
	}
	got, err := decodeClientPixelV2(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.X != p.X || got.Y != p.Y || got.Color[0] != 1 || got.ClientSentMs != p.ServerRecvMs {
		t.Fatalf("decode mismatch: %+v", got)
	}
}

func TestEncodePixelV2RejectsLargeCoords(t *testing.T) {
	p := &domain.Pixel{X: 70000, Y: 1, Color: []uint{0, 0, 0}}
	if encodePixelV2(p) != nil {
		t.Fatal("expected nil for x > 65535")
	}
}
