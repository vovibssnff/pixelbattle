package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type Repository interface {
	WritePixel(ctx context.Context, x, y uint, pixelData []byte) error
	GetCanvas(ctx context.Context) (map[string][]string, error)
	CheckInitialized(ctx context.Context) bool
	LoadHeatMap(ctx context.Context) (map[string]int64, error)
}

type BenchmarkResult struct {
	StorageType         string       `json:"storage_type"`
	Scenario            string       `json:"scenario"`
	Duration            float64      `json:"duration_seconds"`
	WallClockDuration   float64      `json:"wall_clock_duration_seconds"`
	Operations          int64        `json:"operations"`
	Throughput          float64      `json:"throughput_ops_per_sec"`
	LatencyP50    float64      `json:"latency_p50_ms"`
	LatencyP95    float64      `json:"latency_p95_ms"`
	LatencyP99    float64      `json:"latency_p99_ms"`
	LatencyMin    float64      `json:"latency_min_ms"`
	LatencyMax    float64      `json:"latency_max_ms"`
	LatencyMean   float64      `json:"latency_mean_ms"`
	ErrorCount    int          `json:"error_count"`
	Timestamp     time.Time    `json:"timestamp"`
	TargetRate    float64      `json:"target_rate_ops_per_sec,omitempty"`
	HistoryDepth  int          `json:"history_depth,omitempty"`
	TimeBuckets   []TimeBucket `json:"time_buckets,omitempty"`
	WarmupSkipped int64        `json:"warmup_skipped_ops,omitempty"`
}

type BenchmarkConfig struct {
	ConcurrentWriters int
	HistoryRounds     int
	WarmupDuration    time.Duration
	BucketWidth       time.Duration
	Distribution      string // "uniform" or "zipfian"

	// SustainedThroughput
	TargetRates       []int
	SustainedDuration time.Duration

	// ReadUnderWrite
	RUWDuration time.Duration
	RUWReaders  int
}

// warmupMinOps disables time-based warmup when expected op count is below this threshold.
const warmupMinOps int64 = 100

type Benchmarker struct {
	repo         Repository
	storageType  string
	canvasHeight uint
	canvasWidth  uint
	config       BenchmarkConfig
}

func NewBenchmarker(repo Repository, storageType string, height, width uint, cfg BenchmarkConfig) *Benchmarker {
	return &Benchmarker{
		repo:         repo,
		storageType:  storageType,
		canvasHeight: height,
		canvasWidth:  width,
		config:       cfg,
	}
}

// newCollector is for duration-based scenarios with no op-count warmup cap (expectedOps < 0).
func (b *Benchmarker) newCollector(scenario string) *LatencyCollector {
	return b.newCollectorWithOps(scenario, -1)
}

// newCollectorWithOps configures warmup: short scenarios (expectedOps < warmupMinOps) skip
// time warmup; otherwise warmup is capped by max 20% of ops (up to 25000). Pass expectedOps < 0
// for open-ended runs (read under write, sustained throughput) — full warmup window, no cap.
func (b *Benchmarker) newCollectorWithOps(scenario string, expectedOps int64) *LatencyCollector {
	warmup := b.config.WarmupDuration
	if expectedOps >= 0 && expectedOps < warmupMinOps {
		warmup = 0
	}
	lc := NewLatencyCollector(warmup, b.config.BucketWidth)
	var maxWU int64
	if warmup > 0 && expectedOps > 0 {
		maxWU = expectedOps / 5
		if maxWU > 25000 {
			maxWU = 25000
		}
		if maxWU < 1 {
			maxWU = 1
		}
	}
	if expectedOps < 0 {
		maxWU = 0
	}
	lc.SetMaxWarmupOps(maxWU)
	lc.SetOnRecord(makePromCallback(b.storageType, scenario))
	return lc
}

func (b *Benchmarker) markActive(scenario string) {
	BenchScenarioActive.WithLabelValues(b.storageType, scenario).Set(1)
}

func (b *Benchmarker) markDone(scenario string) {
	BenchScenarioActive.WithLabelValues(b.storageType, scenario).Set(0)
}

func (b *Benchmarker) newCoordGen(seed int64) CoordinateGenerator {
	if b.config.Distribution == "uniform" {
		return NewUniformGenerator(b.canvasWidth, b.canvasHeight, seed)
	}
	return NewZipfianGenerator(b.canvasWidth, b.canvasHeight, seed)
}

func (b *Benchmarker) RunAllBenchmarks() []BenchmarkResult {
	results := make([]BenchmarkResult, 0, 32)

	logrus.Info("=== Phase 1: Sequential write (fills canvas) ===")
	results = append(results, b.SequentialWrite())

	logrus.Info("=== Phase 2: HeatMap load (post-fill) ===")
	results = append(results, b.HeatMapLoad(20))

	logrus.Info("=== Phase 3: Concurrent write ===")
	results = append(results, b.ConcurrentWrite(b.config.ConcurrentWriters))

	logrus.Info("=== Phase 4: Full canvas read ===")
	results = append(results, b.FullCanvasRead())

	logrus.Info("=== Phase 5: Mixed workload ===")
	results = append(results, b.MixedWorkload(b.config.ConcurrentWriters))

	logrus.Info("=== Phase 6: Read under write ===")
	results = append(results, b.ReadUnderWrite(b.config.ConcurrentWriters, b.config.RUWReaders, b.config.RUWDuration)...)

	logrus.Info("=== Phase 7: Sustained throughput ===")
	results = append(results, b.SustainedThroughput(b.config.TargetRates, b.config.SustainedDuration, b.config.ConcurrentWriters)...)

	logrus.Info("=== Phase 8: History growth degradation ===")
	results = append(results, b.HistoryGrowth(b.config.HistoryRounds, b.config.ConcurrentWriters)...)

	return results
}

// ---------------------------------------------------------------------------
// Existing scenarios (refactored)
// ---------------------------------------------------------------------------

func (b *Benchmarker) SequentialWrite() BenchmarkResult {
	const scenario = "sequential_write"
	totalPixels := int(b.canvasHeight * b.canvasWidth)
	lc := b.newCollectorWithOps(scenario, int64(totalPixels))
	b.markActive(scenario)
	defer b.markDone(scenario)
	lc.Start()

	ctx := context.Background()
	for y := uint(0); y < b.canvasHeight; y++ {
		for x := uint(0); x < b.canvasWidth; x++ {
			pixelData := b.generatePixelData(x, y)
			opStart := time.Now()
			err := b.repo.WritePixel(ctx, x, y, pixelData)
			latMs := time.Since(opStart).Seconds() * 1000
			lc.Record(latMs, err != nil)
			if err != nil {
				logrus.Error(err)
			}
		}
	}

	return b.buildResult("sequential_write", lc, int64(totalPixels))
}

func (b *Benchmarker) ConcurrentWrite(concurrency int) BenchmarkResult {
	const scenario = "concurrent_write"
	totalPixels := int(b.canvasHeight * b.canvasWidth)
	pixelsPerWorker := totalPixels / concurrency
	lc := b.newCollectorWithOps(scenario, int64(totalPixels))
	b.markActive(scenario)
	defer b.markDone(scenario)
	lc.Start()

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			gen := b.newCoordGen(int64(workerID) + 100)
			count := pixelsPerWorker
			if workerID == concurrency-1 {
				count = totalPixels - workerID*pixelsPerWorker
			}
			for j := 0; j < count; j++ {
				x, y := gen.Next()
				pixelData := b.generatePixelData(x, y)
				opStart := time.Now()
				err := b.repo.WritePixel(ctx, x, y, pixelData)
				lc.Record(time.Since(opStart).Seconds()*1000, err != nil)
				if err != nil {
					logrus.Error(err)
				}
			}
		}(i)
	}

	wg.Wait()
	return b.buildResult("concurrent_write", lc, int64(totalPixels))
}

func (b *Benchmarker) FullCanvasRead() BenchmarkResult {
	const scenario = "full_canvas_read"
	const iterations = 20
	lc := b.newCollectorWithOps(scenario, iterations)
	b.markActive(scenario)
	defer b.markDone(scenario)
	lc.Start()
	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		opStart := time.Now()
		_, err := b.repo.GetCanvas(ctx)
		lc.Record(time.Since(opStart).Seconds()*1000, err != nil)
		if err != nil {
			logrus.Error(err)
		}
	}

	return b.buildResult("full_canvas_read", lc, iterations)
}

func (b *Benchmarker) MixedWorkload(concurrency int) BenchmarkResult {
	const scenario = "mixed_workload"
	totalOps := 10000
	writeOps := int(float64(totalOps) * 0.8)
	readOps := totalOps - writeOps
	lc := b.newCollectorWithOps(scenario, int64(totalOps))
	b.markActive(scenario)
	defer b.markDone(scenario)
	lc.Start()

	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			gen := b.newCoordGen(int64(id) + 200)
			for j := 0; j < writeOps/concurrency; j++ {
				x, y := gen.Next()
				pixelData := b.generatePixelData(x, y)
				opStart := time.Now()
				err := b.repo.WritePixel(ctx, x, y, pixelData)
				lc.Record(time.Since(opStart).Seconds()*1000, err != nil)
			}
		}(i)
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < readOps/2; j++ {
				opStart := time.Now()
				_, err := b.repo.GetCanvas(ctx)
				lc.Record(time.Since(opStart).Seconds()*1000, err != nil)
			}
		}()
	}

	wg.Wait()
	return b.buildResult("mixed_workload", lc, int64(totalOps))
}

// ---------------------------------------------------------------------------
// New scenarios
// ---------------------------------------------------------------------------

// HeatMapLoad benchmarks LoadHeatMap after the canvas is populated.
func (b *Benchmarker) HeatMapLoad(iterations int) BenchmarkResult {
	const scenario = "heatmap_load"
	lc := b.newCollectorWithOps(scenario, int64(iterations))
	b.markActive(scenario)
	defer b.markDone(scenario)
	lc.Start()
	ctx := context.Background()

	for i := 0; i < iterations; i++ {
		opStart := time.Now()
		_, err := b.repo.LoadHeatMap(ctx)
		lc.Record(time.Since(opStart).Seconds()*1000, err != nil)
		if err != nil {
			logrus.Error(err)
		}
	}

	return b.buildResult("heatmap_load", lc, int64(iterations))
}

// ReadUnderWrite runs concurrent writers and readers for a fixed duration,
// returning separate results for writes and reads.
func (b *Benchmarker) ReadUnderWrite(writers, readers int, duration time.Duration) []BenchmarkResult {
	const writeScenario = "read_under_write_writes"
	const readScenario = "read_under_write_reads"
	writeLc := b.newCollector(writeScenario)
	readLc := b.newCollector(readScenario)
	b.markActive(writeScenario)
	b.markActive(readScenario)
	defer b.markDone(writeScenario)
	defer b.markDone(readScenario)
	ctx := context.Background()

	var wg sync.WaitGroup
	var writeOps, readOps int64
	deadline := time.After(duration)
	done := make(chan struct{})

	go func() {
		<-deadline
		close(done)
	}()

	writeLc.Start()
	readLc.Start()

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			gen := b.newCoordGen(int64(id) + 300)
			for {
				select {
				case <-done:
					return
				default:
				}
				x, y := gen.Next()
				pixelData := b.generatePixelData(x, y)
				opStart := time.Now()
				err := b.repo.WritePixel(ctx, x, y, pixelData)
				writeLc.Record(time.Since(opStart).Seconds()*1000, err != nil)
				atomic.AddInt64(&writeOps, 1)
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			// do one read immediately
			opStart := time.Now()
			_, err := b.repo.GetCanvas(ctx)
			readLc.Record(time.Since(opStart).Seconds()*1000, err != nil)
			atomic.AddInt64(&readOps, 1)
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					opStart := time.Now()
					_, err := b.repo.GetCanvas(ctx)
					readLc.Record(time.Since(opStart).Seconds()*1000, err != nil)
					atomic.AddInt64(&readOps, 1)
				}
			}
		}()
	}

	wg.Wait()

	wResult := b.buildResult("read_under_write_writes", writeLc, atomic.LoadInt64(&writeOps))
	rResult := b.buildResult("read_under_write_reads", readLc, atomic.LoadInt64(&readOps))
	return []BenchmarkResult{wResult, rResult}
}

// SustainedThroughput issues writes at controlled rates and measures latency
// at each level, producing a throughput-latency curve.
func (b *Benchmarker) SustainedThroughput(rates []int, durationPerRate time.Duration, concurrency int) []BenchmarkResult {
	results := make([]BenchmarkResult, 0, len(rates))
	ctx := context.Background()

	for _, rate := range rates {
		scenario := fmt.Sprintf("sustained_throughput_%d", rate)
		logrus.Infof("  Sustained throughput @ %d ops/s for %v ...", rate, durationPerRate)
		lc := b.newCollector(scenario)
		b.markActive(scenario)
		var wg sync.WaitGroup
		var ops int64

		ratePerWorker := rate / concurrency
		if ratePerWorker < 1 {
			ratePerWorker = 1
		}
		interval := time.Duration(float64(time.Second) / float64(ratePerWorker))

		deadline := time.After(durationPerRate)
		done := make(chan struct{})
		go func() {
			<-deadline
			close(done)
		}()

		lc.Start()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				gen := b.newCoordGen(int64(id) + int64(rate)*100)
				ticker := time.NewTicker(interval)
				defer ticker.Stop()
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						x, y := gen.Next()
						pixelData := b.generatePixelData(x, y)
						opStart := time.Now()
						err := b.repo.WritePixel(ctx, x, y, pixelData)
						lc.Record(time.Since(opStart).Seconds()*1000, err != nil)
						atomic.AddInt64(&ops, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		b.markDone(scenario)

		r := b.buildResult(scenario, lc, atomic.LoadInt64(&ops))
		r.TargetRate = float64(rate)
		results = append(results, r)
	}

	return results
}

// HistoryGrowth writes multiple rounds to the canvas and measures
// GetCanvas + LoadHeatMap latency after each round to show degradation.
func (b *Benchmarker) HistoryGrowth(rounds, concurrency int) []BenchmarkResult {
	totalPixels := int(b.canvasHeight * b.canvasWidth)
	results := make([]BenchmarkResult, 0, rounds*3)
	ctx := context.Background()

	for round := 1; round <= rounds; round++ {
		logrus.Infof("  History round %d/%d (writing %d pixels) ...", round, rounds, totalPixels)

		writeScenario := fmt.Sprintf("history_growth_write_round_%d", round)
		readScenario := fmt.Sprintf("history_growth_read_round_%d", round)
		hmScenario := fmt.Sprintf("history_growth_heatmap_round_%d", round)

		// Write phase
		writeLc := b.newCollectorWithOps(writeScenario, int64(totalPixels))
		b.markActive(writeScenario)
		writeLc.Start()
		pixelsPerWorker := totalPixels / concurrency
		var wg sync.WaitGroup

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				startIdx := workerID * pixelsPerWorker
				endIdx := startIdx + pixelsPerWorker
				if workerID == concurrency-1 {
					endIdx = totalPixels
				}
				for idx := startIdx; idx < endIdx; idx++ {
					x := uint(idx % int(b.canvasWidth))
					y := uint(idx / int(b.canvasWidth))
					pixelData := b.generatePixelData(x, y)
					opStart := time.Now()
					err := b.repo.WritePixel(ctx, x, y, pixelData)
					writeLc.Record(time.Since(opStart).Seconds()*1000, err != nil)
				}
			}(i)
		}
		wg.Wait()
		b.markDone(writeScenario)

		depth := round * totalPixels
		wr := b.buildResult(writeScenario, writeLc, int64(totalPixels))
		wr.HistoryDepth = depth
		results = append(results, wr)

		// GetCanvas measurement
		readLc := b.newCollectorWithOps(readScenario, 5)
		b.markActive(readScenario)
		readLc.Start()
		for i := 0; i < 5; i++ {
			opStart := time.Now()
			_, err := b.repo.GetCanvas(ctx)
			readLc.Record(time.Since(opStart).Seconds()*1000, err != nil)
		}
		b.markDone(readScenario)
		rr := b.buildResult(readScenario, readLc, 5)
		rr.HistoryDepth = depth
		results = append(results, rr)

		// LoadHeatMap measurement
		hmLc := b.newCollectorWithOps(hmScenario, 5)
		b.markActive(hmScenario)
		hmLc.Start()
		for i := 0; i < 5; i++ {
			opStart := time.Now()
			_, err := b.repo.LoadHeatMap(ctx)
			hmLc.Record(time.Since(opStart).Seconds()*1000, err != nil)
		}
		b.markDone(hmScenario)
		hr := b.buildResult(hmScenario, hmLc, 5)
		hr.HistoryDepth = depth
		results = append(results, hr)
	}

	return results
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (b *Benchmarker) generatePixelData(x, y uint) []byte {
	pixel := &domain.RedisPixel{
		UserId:    rand.Intn(1000),
		Faculty:   fmt.Sprintf("faculty_%d", rand.Intn(5)),
		Color:     []uint{uint(rand.Intn(256)), uint(rand.Intn(256)), uint(rand.Intn(256))},
		Timestamp: time.Now().Unix(),
	}
	data, err := utils.SerializeRedisPixel(pixel)
	if err != nil {
		logrus.Error("Failed to serialize pixel: ", err)
		return []byte{}
	}
	return data
}

func (b *Benchmarker) buildResult(scenario string, lc *LatencyCollector, totalOps int64) BenchmarkResult {
	cr := lc.Finalize()
	wallClock := lc.TotalElapsed().Seconds()

	result := BenchmarkResult{
		StorageType:       b.storageType,
		Scenario:          scenario,
		Timestamp:         time.Now(),
		TimeBuckets:       cr.Buckets,
		WarmupSkipped:     cr.WarmupSkipped,
		WallClockDuration: wallClock,
	}

	latencies := cr.Latencies
	ops := int64(len(latencies))
	if ops == 0 {
		result.Operations = 0
		result.ErrorCount = cr.ErrorCount
		return result
	}

	sort.Float64s(latencies)

	mean := 0.0
	for _, v := range latencies {
		mean += v
	}
	mean /= float64(ops)

	duration := cr.MeasuredDuration.Seconds()
	if duration <= 0 && ops > 0 {
		duration = mean * float64(ops) / 1000.0
	}

	result.Operations = ops
	result.Duration = duration
	result.ErrorCount = cr.ErrorCount
	result.Throughput = float64(ops) / duration
	result.LatencyP50 = percentile(latencies, 0.50)
	result.LatencyP95 = percentile(latencies, 0.95)
	result.LatencyP99 = percentile(latencies, 0.99)
	result.LatencyMin = latencies[0]
	result.LatencyMax = latencies[len(latencies)-1]
	result.LatencyMean = mean

	return result
}
