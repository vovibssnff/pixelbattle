package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"pb_backend/internal/core/domain"
	"pb_backend/internal/metrics"
)

// OpStreamConsumer drives the Phase 2 cross-gateway broadcast path (plan §11.4, p2-xreadgroup-fanout).
//
// Each gateway runs one consumer subscribed to the global fan-out stream `fanout:{global}`.
// The Lua write script XADDs to per-cell `opstream:{y:x}` (durable history); the canvas
// repository additionally XADDs to `fanout:{global}` so every gateway can replay events in
// arrival order.
//
// Dedupe: every XADD carries `origin_instance`. The consumer skips entries it produced
// itself (those were already fanned out locally by setPixel) and pushes the rest into the
// local WSHub via BroadcastFromPeer.
//
// Lag metric: opstream_lag_seconds = (now_ms - entry_ms) / 1000, sampled on every batch.
type OpStreamConsumer struct {
	rdb      redis.Cmdable
	hub      OpHub
	stream   string
	group    string
	consumer string
	self     string

	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// OpHub is the WebSocket-hub subset that the op-log consumer pushes into. WsServer
// (internal/adapters/websockets) satisfies this with BroadcastFromPeer.
type OpHub interface {
	BroadcastFromPeer(p *domain.Pixel) int
}

// FanoutGlobalKey is the canonical fan-out stream key. Hash-tagged so it lives in one slot
// in Redis Cluster.
func FanoutGlobalKey() string { return "fanout:{global}" }

// XAddFanout writes an entry to fanout:{global}. The canvas repository calls this from
// WritePixel after a successful Lua execution. Best-effort: a failed XADD does not roll
// back the canvas write; we just emit a warn so the candidate-column comparison sees the
// reliability characteristic.
func XAddFanout(ctx context.Context, rdb redis.Cmdable, origin string, p *domain.Pixel, streamID string) error {
	if rdb == nil {
		return errors.New("oplog: nil rdb")
	}
	payload, err := json.Marshal(struct {
		X            uint   `json:"x"`
		Y            uint   `json:"y"`
		Color        []uint `json:"c"`
		ClientSeq    uint32 `json:"seq,omitempty"`
		ServerRecvMs int64  `json:"r_ms,omitempty"`
	}{p.X, p.Y, p.Color, p.ClientSeq, p.ServerRecvMs})
	if err != nil {
		return err
	}
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: FanoutGlobalKey(),
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{
			"origin": origin,
			"data":   string(payload),
			"sid":    streamID,
			"ms":     strconv.FormatInt(time.Now().UnixMilli(), 10),
		},
	}).Err()
}

// NewOpStreamConsumer wires a consumer around the gateway's Redis client.
//   - group   : Redis Streams consumer group name (shared across the gateway fleet).
//   - consumer: this gateway's consumer name (must be unique inside the group).
//   - self    : the local gateway instance_id (used for dedupe of own-origin entries).
func NewOpStreamConsumer(rdb redis.Cmdable, hub OpHub, group, consumer, self string) *OpStreamConsumer {
	return &OpStreamConsumer{
		rdb:      rdb,
		hub:      hub,
		stream:   FanoutGlobalKey(),
		group:    group,
		consumer: consumer,
		self:     self,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start kicks off the consumer goroutine. Returns immediately after ensuring the group exists.
func (c *OpStreamConsumer) Start(ctx context.Context) error {
	if c.group == "" || c.consumer == "" {
		return errors.New("oplog: empty consumer group or consumer name")
	}
	// MKSTREAM creates the stream if missing; BUSYGROUP is non-fatal.
	if _, err := c.rdb.XGroupCreateMkStream(ctx, c.stream, c.group, "$").Result(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("oplog: XGROUP CREATE failed: %w", err)
		}
	}
	go c.runLoop(ctx)
	return nil
}

// Stop signals the consumer goroutine to exit and waits for it.
func (c *OpStreamConsumer) Stop() {
	c.stopOnce.Do(func() {
		close(c.stop)
		<-c.done
	})
}

func (c *OpStreamConsumer) runLoop(ctx context.Context) {
	defer close(c.done)
	logrus.Infof("oplog: consumer %s/%s starting on %s", c.group, c.consumer, c.stream)
	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second
	for {
		select {
		case <-c.stop:
			return
		case <-ctx.Done():
			return
		default:
		}
		batch, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{c.stream, ">"},
			Count:    256,
			Block:    200 * time.Millisecond,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				metrics.SetOpstreamLag(c.group, c.consumer, "global", 0)
				backoff = 100 * time.Millisecond
				continue
			}
			logrus.Warnf("oplog: XREADGROUP error (will back off %s): %v", backoff, err)
			select {
			case <-time.After(backoff):
			case <-c.stop:
				return
			}
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}
		backoff = 100 * time.Millisecond
		c.processBatch(ctx, batch)
	}
}

func (c *OpStreamConsumer) processBatch(ctx context.Context, batch []redis.XStream) {
	now := time.Now().UnixMilli()
	maxLag := time.Duration(0)
	for _, s := range batch {
		ackIDs := make([]string, 0, len(s.Messages))
		for _, msg := range s.Messages {
			ackIDs = append(ackIDs, msg.ID)
			origin, _ := msg.Values["origin"].(string)
			if origin == c.self {
				continue
			}
			data, _ := msg.Values["data"].(string)
			var p struct {
				X            uint   `json:"x"`
				Y            uint   `json:"y"`
				Color        []uint `json:"c"`
				ClientSeq    uint32 `json:"seq,omitempty"`
				ServerRecvMs int64  `json:"r_ms,omitempty"`
			}
			if err := json.Unmarshal([]byte(data), &p); err != nil {
				logrus.Debugf("oplog: bad entry %s: %v", msg.ID, err)
				continue
			}
			pixel := &domain.Pixel{
				X:            p.X,
				Y:            p.Y,
				Color:        p.Color,
				ClientSeq:    p.ClientSeq,
				ServerRecvMs: p.ServerRecvMs,
			}
			c.hub.BroadcastFromPeer(pixel)
			// Lag derived from the stream-id ms prefix (Redis Stream id is "ms-seq").
			if idMs, ok := parseStreamIDMs(msg.ID); ok {
				lag := time.Duration(now-idMs) * time.Millisecond
				if lag > maxLag {
					maxLag = lag
				}
			}
		}
		if len(ackIDs) > 0 {
			if err := c.rdb.XAck(ctx, c.stream, c.group, ackIDs...).Err(); err != nil {
				logrus.Debugf("oplog: XACK failed: %v", err)
			}
		}
	}
	metrics.SetOpstreamLag(c.group, c.consumer, "global", maxLag)
}

func parseStreamIDMs(id string) (int64, bool) {
	i := strings.IndexByte(id, '-')
	if i <= 0 {
		return 0, false
	}
	ms, err := strconv.ParseInt(id[:i], 10, 64)
	if err != nil {
		return 0, false
	}
	return ms, true
}
