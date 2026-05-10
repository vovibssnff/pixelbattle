package websockets

import (
	"sync"

	"pb_backend/internal/core/domain"
)

type replayEntry struct {
	pixel  *domain.Pixel
	wallMs int64
}

// PixelReplayBuffer keeps recent broadcast pixels with server wall time for
// WS replay_after_ms (plan §11.3 tail until full op-log XREAD exists).
type PixelReplayBuffer struct {
	mu      sync.Mutex
	max     int
	entries []replayEntry
}

func NewPixelReplayBuffer(maxEntries int) *PixelReplayBuffer {
	if maxEntries <= 0 {
		maxEntries = 200_000
	}
	return &PixelReplayBuffer{max: maxEntries}
}

func clonePixelForReplay(p *domain.Pixel) *domain.Pixel {
	if p == nil {
		return nil
	}
	c := *p
	if len(p.Color) > 0 {
		c.Color = append([]uint(nil), p.Color...)
	}
	return &c
}

// Add records a pixel after it was persisted and cleared of userid/faculty (broadcast shape).
func (b *PixelReplayBuffer) Add(p *domain.Pixel, wallMs int64) {
	if b == nil || p == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, replayEntry{
		pixel:  clonePixelForReplay(p),
		wallMs: wallMs,
	})
	if len(b.entries) > b.max {
		b.entries = b.entries[len(b.entries)-b.max:]
	}
}

// Since returns copies of pixels with wallMs strictly greater than ms, in order.
func (b *PixelReplayBuffer) Since(ms int64) []*domain.Pixel {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*domain.Pixel, 0, 32)
	for _, e := range b.entries {
		if e.wallMs > ms {
			out = append(out, clonePixelForReplay(e.pixel))
		}
	}
	return out
}
