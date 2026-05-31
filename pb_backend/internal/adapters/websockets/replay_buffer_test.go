package websockets

import (
	"testing"

	"pb_backend/internal/core/domain"
)

func TestPixelReplayBufferSince(t *testing.T) {
	b := NewPixelReplayBuffer(100)
	p1 := &domain.Pixel{X: 1, Y: 2, Color: []uint{1, 2, 3}}
	p2 := &domain.Pixel{X: 3, Y: 4, Color: []uint{4, 5, 6}}
	b.Add(p1, 100)
	b.Add(p2, 200)

	got := b.Since(100)
	if len(got) != 1 || got[0].X != 3 {
		t.Fatalf("Since(100): %+v", got)
	}
	got = b.Since(99)
	if len(got) != 2 {
		t.Fatalf("Since(99): len=%d", len(got))
	}
	if got[0].Color[0] != 1 || got[1].Color[0] != 4 {
		t.Fatalf("order/colors wrong: %+v", got)
	}
}

func TestPixelReplayBufferCap(t *testing.T) {
	b := NewPixelReplayBuffer(3)
	for i := 0; i < 5; i++ {
		b.Add(&domain.Pixel{X: uint(i), Y: 0, Color: []uint{uint(i), 0, 0}}, int64(10+i))
	}
	if len(b.entries) != 3 {
		t.Fatalf("cap: len=%d", len(b.entries))
	}
	last := b.Since(0)
	if len(last) != 3 || last[0].X != 2 || last[2].X != 4 {
		t.Fatalf("tail wrong: %+v", last)
	}
}
