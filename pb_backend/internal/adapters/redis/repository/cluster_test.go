package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"pb_backend/internal/core/domain"
)

// Phase 2 backend test (plan §p2-backend-test-grow): hash-tag colocation + LWW path coverage.
// Uses miniredis (single-node, Lua-compatible) because go-redis ClusterClient does not work
// against a fake — the LWW invariants we care about are slot-routing-independent and the
// real cluster path is exercised by the Ansible suite (plan §13.2).

// TestHashTagColocation: pixel:{y:x}, opstream:{y:x}, pixmeta:{y:x} must all hash to the
// same Redis Cluster slot — otherwise the Lua EVAL in oplog.go would CROSSSLOT-error.
func TestHashTagColocation(t *testing.T) {
	cases := []struct {
		x, y uint
	}{
		{0, 0}, {1, 1}, {499, 249}, {100, 50}, {250, 125},
	}
	for _, tc := range cases {
		px := PixelKey(true, tc.x, tc.y)
		os := OpStreamKey(tc.x, tc.y)
		pm := PixelMetaKey(tc.x, tc.y)
		if slot(px) != slot(os) || slot(px) != slot(pm) {
			t.Fatalf("hash-tag colocation broken for (%d,%d): pixel=%d opstream=%d pixmeta=%d",
				tc.x, tc.y, slot(px), slot(os), slot(pm))
		}
	}
}

// slot mimics the redis-cluster CRC16-based slot for our hash-tagged keys: pick the
// substring between the first '{' and the next '}' and hash that. Two keys with the
// same tag hash to the same slot.
func slot(key string) int {
	startBrace := -1
	for i, c := range key {
		if c == '{' {
			startBrace = i
			break
		}
	}
	if startBrace == -1 {
		return crc16(key)
	}
	endBrace := -1
	for i := startBrace + 1; i < len(key); i++ {
		if key[i] == '}' {
			endBrace = i
			break
		}
	}
	if endBrace == -1 {
		return crc16(key)
	}
	tag := key[startBrace+1 : endBrace]
	if tag == "" {
		return crc16(key)
	}
	return crc16(tag)
}

func crc16(s string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return int(h.Sum32() & 0xffff)
}

// TestWritePixelOpLogWithGatewayOriginEmitsFanout: after a successful Lua write the
// repository must XADD to fanout:{global} with origin=<self> when GATEWAY_INSTANCE_ID is set.
func TestWritePixelOpLogWithGatewayOriginEmitsFanout(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	repo := NewCanvasRepository(rdb, true)
	repo.SetGatewayOrigin("gw-test")

	ctx := context.Background()
	pixel := domain.RedisPixel{UserId: "u1", Faculty: "KTU", Color: []uint{255, 0, 0}, Timestamp: time.Now().Unix()}
	pixelBytes, err := json.Marshal(pixel)
	if err != nil {
		t.Fatalf("marshal pixel: %v", err)
	}

	if err := repo.WritePixel(ctx, 10, 20, pixelBytes); err != nil {
		t.Fatalf("WritePixel: %v", err)
	}

	// fanout:{global} must have one entry now.
	entries, err := rdb.XLen(ctx, FanoutGlobalKey()).Result()
	if err != nil {
		t.Fatalf("XLEN: %v", err)
	}
	if entries != 1 {
		t.Fatalf("fanout:{global} entries = %d, want 1", entries)
	}

	// Verify the entry has origin=gw-test.
	rng, err := rdb.XRange(ctx, FanoutGlobalKey(), "-", "+").Result()
	if err != nil {
		t.Fatalf("XRANGE: %v", err)
	}
	if len(rng) != 1 {
		t.Fatalf("XRANGE returned %d entries", len(rng))
	}
	if got, _ := rng[0].Values["origin"].(string); got != "gw-test" {
		t.Fatalf("origin = %q, want gw-test", got)
	}
}

// TestWritePixelOpLogMonolithSkipsFanout: a monolith deployment (no GATEWAY_INSTANCE_ID)
// must NOT add to fanout:{global} — that would emit an extra Redis op on Phase 1 baseline.
func TestWritePixelOpLogMonolithSkipsFanout(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	repo := NewCanvasRepository(rdb, true) // no SetGatewayOrigin call

	ctx := context.Background()
	pixel := domain.RedisPixel{UserId: "u1", Faculty: "KTU", Color: []uint{1, 2, 3}, Timestamp: time.Now().Unix()}
	pixelBytes, _ := json.Marshal(pixel)

	if err := repo.WritePixel(ctx, 7, 7, pixelBytes); err != nil {
		t.Fatalf("WritePixel: %v", err)
	}

	entries, err := rdb.XLen(ctx, FanoutGlobalKey()).Result()
	if err != nil {
		t.Fatalf("XLEN: %v", err)
	}
	if entries != 0 {
		t.Fatalf("fanout:{global} entries = %d, want 0 (monolith mode)", entries)
	}
}

// TestOpStreamConsumerDedupesOwnOrigin: a gateway must not re-broadcast pixels that came
// from its own setPixel path (already fanned out locally).
func TestOpStreamConsumerDedupesOwnOrigin(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	hub := &fakeHub{}
	c := NewOpStreamConsumer(rdb, hub, "test-group", "gw-self", "gw-self")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	// Push two entries: one from self, one from a peer.
	for _, origin := range []string{"gw-self", "gw-peer"} {
		_, err := rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: FanoutGlobalKey(),
			Values: map[string]interface{}{
				"origin": origin,
				"data":   `{"x":1,"y":2,"c":[1,2,3]}`,
				"sid":    fmt.Sprintf("%d-0", time.Now().UnixMilli()),
				"ms":     fmt.Sprintf("%d", time.Now().UnixMilli()),
			},
		}).Result()
		if err != nil {
			t.Fatalf("XAdd: %v", err)
		}
	}

	// Give the consumer time to drain the batch.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.calls() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := hub.calls(); got != 1 {
		t.Fatalf("hub.BroadcastFromPeer called %d times, want 1 (gw-peer only, gw-self skipped)", got)
	}
}

type fakeHub struct {
	c int64
}

func (f *fakeHub) BroadcastFromPeer(p *domain.Pixel) int {
	atomic.AddInt64(&f.c, 1)
	return 1
}
func (f *fakeHub) calls() int { return int(atomic.LoadInt64(&f.c)) }
