package websockets

import (
	"bytes"
	"testing"

	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
)

func TestSerializePixelIncludesServerRecvMsInJSON(t *testing.T) {
	p := &domain.Pixel{
		X: 0, Y: 0, Color: []uint{1, 2, 3},
		ServerRecvMs: 1700000000123,
	}
	data, err := utils.SerializePixel(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"server_recv_ms":1700000000123`)) {
		t.Fatalf("expected server_recv_ms in JSON, got %s", data)
	}
}

func TestBenchmarkUIDToCanonical(t *testing.T) {
	if _, ok := benchmarkUIDToCanonical(""); ok {
		t.Fatal("empty should fail")
	}
	if id, ok := benchmarkUIDToCanonical(" 42 "); !ok || id != "vk_42" {
		t.Fatalf("numeric uid: got %q ok=%v", id, ok)
	}
}
