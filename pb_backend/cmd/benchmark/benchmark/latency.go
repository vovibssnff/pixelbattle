package benchmark

import (
	"sort"
	"sync"
	"time"
)

type TimeBucket struct {
	BucketStartSec float64 `json:"bucket_start_sec"`
	Operations     int64   `json:"operations"`
	LatencyP50     float64 `json:"latency_p50_ms"`
	LatencyP95     float64 `json:"latency_p95_ms"`
	LatencyP99     float64 `json:"latency_p99_ms"`
	LatencyMean    float64 `json:"latency_mean_ms"`
	ErrorCount     int     `json:"error_count"`
}

// LatencyCollector records per-operation latencies with warmup filtering
// and time-bucketed aggregation. Safe for concurrent use.
type LatencyCollector struct {
	mu             sync.Mutex
	warmupDuration time.Duration
	bucketWidth    time.Duration
	maxWarmupOps   int64
	startTime      time.Time
	lastRecordTime time.Time

	// measuredPhaseStart is the wall-clock time when the measured (post-warmup)
	// window begins. It may differ from startTime+warmupDuration when warmup
	// ends early due to maxWarmupOps.
	measuredPhaseStart time.Time

	warmupLatencies []float64
	latencies       []float64
	buckets         map[int]*bucketAccum
	errorCount      int
	warmupErrors    int

	// onRecord is an optional callback invoked on every Record() call
	// (including warmup ops). Called outside the mutex — must be thread-safe.
	// Typically wired to Prometheus histogram + counter observations.
	onRecord func(latencyMs float64, isError bool)
}

type bucketAccum struct {
	latencies []float64
	errors    int
}

func NewLatencyCollector(warmup, bucketWidth time.Duration) *LatencyCollector {
	return &LatencyCollector{
		warmupDuration:  warmup,
		bucketWidth:     bucketWidth,
		latencies:       make([]float64, 0, 4096),
		warmupLatencies: make([]float64, 0, 256),
		buckets:         make(map[int]*bucketAccum),
	}
}

// SetMaxWarmupOps caps how many samples count as warmup; when reached, measured
// ops begin even if warmupDuration has not elapsed. Zero disables the cap.
// Must be called before Start().
func (lc *LatencyCollector) SetMaxWarmupOps(n int64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.maxWarmupOps = n
}

func (lc *LatencyCollector) Start() {
	lc.mu.Lock()
	lc.startTime = time.Now()
	lc.mu.Unlock()
}

// TotalElapsed returns wall-clock time since Start() (including warmup).
func (lc *LatencyCollector) TotalElapsed() time.Duration {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.startTime.IsZero() {
		return 0
	}
	return time.Since(lc.startTime)
}

// SetOnRecord sets the Prometheus observation callback.
// Must be called before Start().
func (lc *LatencyCollector) SetOnRecord(fn func(latencyMs float64, isError bool)) {
	lc.onRecord = fn
}

func (lc *LatencyCollector) Record(latencyMs float64, isError bool) {
	if lc.onRecord != nil {
		lc.onRecord(latencyMs, isError)
	}

	now := time.Now()
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.lastRecordTime = now
	elapsed := now.Sub(lc.startTime)

	inWarmup := elapsed < lc.warmupDuration &&
		(lc.maxWarmupOps == 0 || int64(len(lc.warmupLatencies)) < lc.maxWarmupOps)

	if inWarmup {
		lc.warmupLatencies = append(lc.warmupLatencies, latencyMs)
		if isError {
			lc.warmupErrors++
		}
		return
	}

	if lc.measuredPhaseStart.IsZero() {
		if elapsed >= lc.warmupDuration {
			lc.measuredPhaseStart = lc.startTime.Add(lc.warmupDuration)
		} else {
			lc.measuredPhaseStart = now
		}
	}

	lc.latencies = append(lc.latencies, latencyMs)
	if isError {
		lc.errorCount++
	}

	if lc.bucketWidth > 0 {
		measuredElapsed := now.Sub(lc.measuredPhaseStart)
		bucketIdx := int(measuredElapsed / lc.bucketWidth)
		b, ok := lc.buckets[bucketIdx]
		if !ok {
			b = &bucketAccum{latencies: make([]float64, 0, 256)}
			lc.buckets[bucketIdx] = b
		}
		b.latencies = append(b.latencies, latencyMs)
		if isError {
			b.errors++
		}
	}
}

type CollectorResult struct {
	Latencies        []float64
	Buckets          []TimeBucket
	WarmupSkipped    int64
	ErrorCount       int
	MeasuredDuration time.Duration // wall-clock from measured phase start to last record
}

func (lc *LatencyCollector) Finalize() CollectorResult {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var buckets []TimeBucket
	if lc.bucketWidth > 0 && len(lc.buckets) > 0 {
		maxIdx := 0
		for idx := range lc.buckets {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
		buckets = make([]TimeBucket, 0, maxIdx+1)
		for i := 0; i <= maxIdx; i++ {
			b, ok := lc.buckets[i]
			if !ok || len(b.latencies) == 0 {
				buckets = append(buckets, TimeBucket{
					BucketStartSec: float64(i) * lc.bucketWidth.Seconds(),
				})
				continue
			}
			buckets = append(buckets, computeBucket(
				float64(i)*lc.bucketWidth.Seconds(),
				b.latencies,
				b.errors,
			))
		}
	}

	var measuredDuration time.Duration
	if !lc.measuredPhaseStart.IsZero() && !lc.lastRecordTime.IsZero() {
		measuredDuration = lc.lastRecordTime.Sub(lc.measuredPhaseStart)
		if measuredDuration < 0 {
			measuredDuration = 0
		}
	}

	return CollectorResult{
		Latencies:        lc.latencies,
		Buckets:          buckets,
		WarmupSkipped:    int64(len(lc.warmupLatencies)),
		ErrorCount:       lc.errorCount,
		MeasuredDuration: measuredDuration,
	}
}

func computeBucket(startSec float64, latencies []float64, errors int) TimeBucket {
	sorted := make([]float64, len(latencies))
	copy(sorted, latencies)
	sort.Float64s(sorted)

	mean := 0.0
	for _, v := range sorted {
		mean += v
	}
	mean /= float64(len(sorted))

	return TimeBucket{
		BucketStartSec: startSec,
		Operations:     int64(len(sorted)),
		LatencyP50:     percentile(sorted, 0.50),
		LatencyP95:     percentile(sorted, 0.95),
		LatencyP99:     percentile(sorted, 0.99),
		LatencyMean:    mean,
		ErrorCount:     errors,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
