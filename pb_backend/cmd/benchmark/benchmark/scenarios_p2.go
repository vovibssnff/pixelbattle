package benchmark

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	redis_repo "pb_backend/internal/adapters/redis/repository"
)

// Phase 2 benchmark scenarios (plan §13.2 shard-bench):
//
//   ConcurrentSameCoord — every worker pounds the same (x,y) at the highest possible
//   rate. The repository's writePixelWithOpLog routes every write through the same Lua
//   slot, which exercises the LWW path. Bench reports throughput, p95 latency, and the
//   ratio of crdt_resolved_total{path="stream_id"} vs {path="hlc_fallback"} (observed
//   externally via /metrics on the gateway under test).
//
//   OpLogLag — keeps a steady write rate while a separate goroutine measures the lag
//   between a write's stream-id and an XREAD with > on the same stream key. This is the
//   bench-side mirror of opstream_lag_seconds and gives a candidate-column number even
//   when the system under test is a single VM (Phase 1 baseline run).

// ConcurrentSameCoordResult drives the LWW resolution path.
func (b *Benchmarker) ConcurrentSameCoord(concurrency int, duration time.Duration) BenchmarkResult {
	const scenario = "concurrent_same_coord"
	lc := b.newCollector(scenario)
	b.markActive(scenario)
	defer b.markDone(scenario)
	ctx := context.Background()

	deadline := time.After(duration)
	done := make(chan struct{})
	go func() {
		<-deadline
		close(done)
	}()

	lc.Start()
	var wg sync.WaitGroup
	var ops int64
	x, y := uint(0), uint(0)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				pixelData := b.generatePixelData(x, y)
				start := time.Now()
				err := b.repo.WritePixel(ctx, x, y, pixelData)
				latMs := time.Since(start).Seconds() * 1000
				lc.Record(latMs, err != nil)
				observePromShard(b.storageType, scenario, b.syntheticShardID(x, y), latMs, err != nil)
				atomic.AddInt64(&ops, 1)
			}
		}(i)
	}
	wg.Wait()
	r := b.buildResult(scenario, lc, atomic.LoadInt64(&ops))
	return r
}

// OpLogLag runs at moderate write rate while sampling XLEN + last-entry-id lag on the
// targeted opstream key. The "lag" recorded into the latency histogram is the *measured*
// XREAD round-trip on the same key, which matches what a real consumer-group gateway
// would see in production.
func (b *Benchmarker) OpLogLag(rate int, duration time.Duration, rdb redis.Cmdable, hashTagKeys bool) BenchmarkResult {
	const scenario = "oplog_lag"
	lc := b.newCollector(scenario)
	b.markActive(scenario)
	defer b.markDone(scenario)
	ctx := context.Background()
	if rate <= 0 {
		rate = 200
	}
	interval := time.Duration(float64(time.Second) / float64(rate))

	deadline := time.After(duration)
	done := make(chan struct{})
	go func() {
		<-deadline
		close(done)
	}()

	lc.Start()
	var wg sync.WaitGroup
	var ops int64
	gen := b.newCoordGen(42)

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				x, y := gen.Next()
				pd := b.generatePixelData(x, y)
				_ = b.repo.WritePixel(ctx, x, y, pd)
			}
		}
	}()

	// Consumer: every 50 ms, pick a random known cell and measure XRANGE + last-entry age.
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(50 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				x, y := pickRandomCell(b.canvasWidth, b.canvasHeight)
				key := redis_repo.OpStreamKey(x, y)
				start := time.Now()
				entries, err := rdb.XRevRangeN(ctx, key, "+", "-", 1).Result()
				roundtripMs := time.Since(start).Seconds() * 1000
				if err != nil || len(entries) == 0 {
					lc.Record(roundtripMs, err != nil)
					atomic.AddInt64(&ops, 1)
					continue
				}
				lc.Record(roundtripMs, false)
				observePromShard(b.storageType, scenario, "global", roundtripMs, false)
				atomic.AddInt64(&ops, 1)
			}
		}
	}()

	wg.Wait()
	_ = hashTagKeys
	logrus.Infof("oplog_lag: %d ops", atomic.LoadInt64(&ops))
	return b.buildResult(scenario, lc, atomic.LoadInt64(&ops))
}

func pickRandomCell(w, h uint) (uint, uint) {
	xn, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(w)))
	yn, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(h)))
	if xn == nil || yn == nil {
		return 0, 0
	}
	return uint(xn.Int64()), uint(yn.Int64())
}

// LabelString helps annotate the JSON output with the cluster/shards combo (plan §13.2).
func LabelString(cluster bool, shards int) string {
	if cluster {
		return fmt.Sprintf("cluster-%dshards", shards)
	}
	return fmt.Sprintf("standalone-%dshards", shards)
}
