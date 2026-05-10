package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"pb_backend/internal/metrics"
)

// AvailabilityProbe periodically writes a sentinel pixel and reads it back,
// feeding the pixel_write_visible_seconds histogram. This gives the comparable
// KPI "storage write latency" (plan §2) — identical contract in both monolith
// and distributed architectures.
type AvailabilityProbe struct {
	rdb      redis.Cmdable
	interval time.Duration
	coord    string // Redis key for the sentinel pixel
	stop     chan struct{}
}

type sentinelPixel struct {
	Color     []uint `json:"color"`
	Timestamp int64  `json:"timestamp"`
	Probe     bool   `json:"probe"`
}

// NewAvailabilityProbe creates a probe that uses a fixed canvas coordinate for
// write-then-read latency measurements. The coordinate is the bottom-right
// corner (height-1, width-1) to minimise interference with real user pixels.
func NewAvailabilityProbe(rdb redis.Cmdable, canvasHeight, canvasWidth uint, interval time.Duration) *AvailabilityProbe {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	coord := fmt.Sprintf("pixel:%d:%d", canvasHeight-1, canvasWidth-1)
	return &AvailabilityProbe{
		rdb:      rdb,
		interval: interval,
		coord:    coord,
		stop:     make(chan struct{}),
	}
}

// Start begins the periodic probe goroutine.
func (p *AvailabilityProbe) Start() {
	go p.loop()
	logrus.Infof("availability_probe: started (coord=%s, interval=%s)", p.coord, p.interval)
}

// Stop halts the probe goroutine.
func (p *AvailabilityProbe) Stop() {
	close(p.stop)
}

func (p *AvailabilityProbe) loop() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stop:
			return
		case <-ticker.C:
			p.measure()
		}
	}
}

func (p *AvailabilityProbe) measure() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sentinel := sentinelPixel{
		Color:     []uint{254, 254, 254},
		Timestamp: time.Now().UnixMilli(),
		Probe:     true,
	}
	data, err := json.Marshal(sentinel)
	if err != nil {
		logrus.Errorf("availability_probe: marshal error: %v", err)
		return
	}

	start := time.Now()

	if err := p.rdb.RPush(ctx, p.coord, data).Err(); err != nil {
		logrus.Debugf("availability_probe: write failed: %v", err)
		return
	}

	result, err := p.rdb.LRange(ctx, p.coord, -1, -1).Result()
	if err != nil || len(result) == 0 {
		logrus.Debugf("availability_probe: read-back failed: %v", err)
		return
	}

	latency := time.Since(start)
	metrics.ObservePixelWriteVisible(latency)
}
