package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"sort"
	"sync"
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
	StorageType      string    `json:"storage_type"`
	Scenario         string    `json:"scenario"`
	Duration         float64   `json:"duration_seconds"`
	Operations       int64     `json:"operations"`
	Throughput       float64   `json:"throughput_ops_per_sec"`
	LatencyP50       float64   `json:"latency_p50_ms"`
	LatencyP95       float64   `json:"latency_p95_ms"`
	LatencyP99       float64   `json:"latency_p99_ms"`
	LatencyMin       float64   `json:"latency_min_ms"`
	LatencyMax       float64   `json:"latency_max_ms"`
	LatencyMean      float64   `json:"latency_mean_ms"`
	ErrorCount       int       `json:"error_count"`
	Timestamp        time.Time `json:"timestamp"`
}

type Benchmarker struct {
	repo         Repository
	storageType  string
	canvasHeight uint
	canvasWidth  uint
}

func NewBenchmarker(repo Repository, storageType string, height, width uint) *Benchmarker {
	return &Benchmarker{
		repo:         repo,
		storageType:  storageType,
		canvasHeight: height,
		canvasWidth:  width,
	}
}

func (b *Benchmarker) RunAllBenchmarks(concurrentWriters, historyMultiplier int) []BenchmarkResult {
	results := make([]BenchmarkResult, 0)

	logrus.Info("Running sequential write benchmark...")
	results = append(results, b.SequentialWrite())

	logrus.Info("Running concurrent write benchmark...")
	results = append(results, b.ConcurrentWrite(concurrentWriters))

	logrus.Info("Running full canvas read benchmark...")
	results = append(results, b.FullCanvasRead())

	logrus.Info("Running mixed workload benchmark...")
	results = append(results, b.MixedWorkload(concurrentWriters))

	logrus.Info("Running history growth benchmark...")
	results = append(results, b.HistoryGrowth(historyMultiplier))

	return results
}

func (b *Benchmarker) SequentialWrite() BenchmarkResult {
	totalPixels := int(b.canvasHeight * b.canvasWidth)
	latencies := make([]float64, 0, totalPixels)
	start := time.Now()
	errors := 0

	ctx := context.Background()
	for y := uint(0); y < b.canvasHeight; y++ {
		for x := uint(0); x < b.canvasWidth; x++ {
			pixelData := b.generatePixelData(x, y)
			opStart := time.Now()
			if err := b.repo.WritePixel(ctx, x, y, pixelData); err != nil {
				logrus.Error(err)
				errors++
			}
			latencies = append(latencies, time.Since(opStart).Seconds()*1000)
		}
	}

	duration := time.Since(start).Seconds()
	return b.calculateResult("sequential_write", duration, latencies, int64(totalPixels), errors)
}

func (b *Benchmarker) ConcurrentWrite(concurrency int) BenchmarkResult {
	totalPixels := int(b.canvasHeight * b.canvasWidth)
	pixelsPerWorker := totalPixels / concurrency
	latencies := make([]float64, 0, totalPixels)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := 0
	var errorMu sync.Mutex

	start := time.Now()
	ctx := context.Background()

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
				if err := b.repo.WritePixel(ctx, x, y, pixelData); err != nil {
					errorMu.Lock()
					errors++
					errorMu.Unlock()
					logrus.Error(err)
				}
				mu.Lock()
				latencies = append(latencies, time.Since(opStart).Seconds()*1000)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start).Seconds()
	return b.calculateResult("concurrent_write", duration, latencies, int64(totalPixels), errors)
}

func (b *Benchmarker) FullCanvasRead() BenchmarkResult {
	latencies := make([]float64, 10) // Run 10 reads
	errors := 0
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 10; i++ {
		readStart := time.Now()
		_, err := b.repo.GetCanvas(ctx)
		if err != nil {
			logrus.Error(err)
			errors++
		}
		latencies[i] = time.Since(readStart).Seconds() * 1000
	}
	duration := time.Since(start).Seconds()

	return b.calculateResult("full_canvas_read", duration, latencies, 10, errors)
}

func (b *Benchmarker) MixedWorkload(concurrency int) BenchmarkResult {
	totalOps := 10000
	writeOps := int(float64(totalOps) * 0.8)
	readOps := totalOps - writeOps
	latencies := make([]float64, 0, totalOps)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := 0
	var errorMu sync.Mutex

	start := time.Now()
	ctx := context.Background()

	// Writers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < writeOps/concurrency; j++ {
				x := uint(rand.Intn(int(b.canvasWidth)))
				y := uint(rand.Intn(int(b.canvasHeight)))
				pixelData := b.generatePixelData(x, y)
				opStart := time.Now()
				if err := b.repo.WritePixel(ctx, x, y, pixelData); err != nil {
					errorMu.Lock()
					errors++
					errorMu.Unlock()
				}
				mu.Lock()
				latencies = append(latencies, time.Since(opStart).Seconds()*1000)
				mu.Unlock()
			}
		}()
	}

	// Readers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < readOps/2; j++ {
				opStart := time.Now()
				_, err := b.repo.GetCanvas(ctx)
				if err != nil {
					errorMu.Lock()
					errors++
					errorMu.Unlock()
				}
				mu.Lock()
				latencies = append(latencies, time.Since(opStart).Seconds()*1000)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start).Seconds()
	return b.calculateResult("mixed_workload", duration, latencies, int64(totalOps), errors)
}

func (b *Benchmarker) HistoryGrowth(multiplier int) BenchmarkResult {
	// Write to same coordinates multiple times to simulate history growth
	totalPixels := int(b.canvasHeight * b.canvasWidth)
	totalOps := totalPixels * multiplier
	latencies := make([]float64, 0, totalOps)
	errors := 0

	ctx := context.Background()
	start := time.Now()

	for round := 0; round < multiplier; round++ {
		for y := uint(0); y < b.canvasHeight; y++ {
			for x := uint(0); x < b.canvasWidth; x++ {
				pixelData := b.generatePixelData(x, y)
				opStart := time.Now()
				if err := b.repo.WritePixel(ctx, x, y, pixelData); err != nil {
					logrus.Error(err)
					errors++
				}
				latencies = append(latencies, time.Since(opStart).Seconds()*1000)
			}
		}
	}

	duration := time.Since(start).Seconds()
	return b.calculateResult("history_growth", duration, latencies, int64(totalOps), errors)
}

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

func (b *Benchmarker) calculateResult(scenario string, duration float64, latencies []float64, operations int64, errors int) BenchmarkResult {
	if len(latencies) == 0 {
		return BenchmarkResult{
			StorageType: b.storageType,
			Scenario:    scenario,
			Duration:    duration,
			Operations:  operations,
			ErrorCount:  errors,
			Timestamp:   time.Now(),
		}
	}

	sort.Float64s(latencies)
	throughput := float64(operations) / duration

	mean := 0.0
	for _, lat := range latencies {
		mean += lat
	}
	mean /= float64(len(latencies))

	return BenchmarkResult{
		StorageType: b.storageType,
		Scenario:    scenario,
		Duration:    duration,
		Operations:  operations,
		Throughput:  throughput,
		LatencyP50:  latencies[len(latencies)/2],
		LatencyP95:  latencies[int(float64(len(latencies))*0.95)],
		LatencyP99:  latencies[int(float64(len(latencies))*0.99)],
		LatencyMin:  latencies[0],
		LatencyMax:  latencies[len(latencies)-1],
		LatencyMean: mean,
		ErrorCount:  errors,
		Timestamp:   time.Now(),
	}
}
