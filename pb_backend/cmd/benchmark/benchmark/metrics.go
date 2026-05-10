package benchmark

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	BenchOpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "benchmark_operation_duration_seconds",
			Help:    "Histogram of individual benchmark operation latencies",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"storage_type", "scenario"},
	)

	BenchOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "benchmark_operations_total",
			Help: "Total number of benchmark operations",
		},
		[]string{"storage_type", "scenario", "status"},
	)

	BenchScenarioActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "benchmark_scenario_active",
			Help: "1 when a scenario is currently running, 0 otherwise",
		},
		[]string{"storage_type", "scenario"},
	)

	BenchShardOpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "benchmark_shard_operation_duration_seconds",
			Help:    "Histogram of benchmark latencies by synthetic shard",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"storage_type", "scenario", "shard_id"},
	)

	BenchShardOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "benchmark_shard_operations_total",
			Help: "Total benchmark operations by synthetic shard",
		},
		[]string{"storage_type", "scenario", "shard_id", "status"},
	)
)

func init() {
	prometheus.MustRegister(BenchOpDuration)
	prometheus.MustRegister(BenchOpsTotal)
	prometheus.MustRegister(BenchScenarioActive)
	prometheus.MustRegister(BenchShardOpDuration)
	prometheus.MustRegister(BenchShardOpsTotal)
}

// StartMetricsServer starts an HTTP server exposing /metrics for Prometheus.
// Returns immediately; the server runs in the background until the process exits.
// Pass port=0 to disable.
func StartMetricsServer(port int) {
	if port <= 0 {
		logrus.Info("Prometheus metrics server disabled")
		return
	}
	addr := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Metrics server error: %v", err)
		}
	}()
}

// makePromCallback builds a function that observes into Prometheus histogram + counter.
func makePromCallback(storageType, scenario string) func(latencyMs float64, isError bool) {
	observer := BenchOpDuration.WithLabelValues(storageType, scenario)
	successCounter := BenchOpsTotal.WithLabelValues(storageType, scenario, "success")
	errorCounter := BenchOpsTotal.WithLabelValues(storageType, scenario, "error")
	return func(latencyMs float64, isError bool) {
		observer.Observe(latencyMs / 1000.0) // convert ms → seconds for Prometheus
		if isError {
			errorCounter.Inc()
		} else {
			successCounter.Inc()
		}
	}
}

func observePromShard(storageType, scenario, shardID string, latencyMs float64, isError bool) {
	BenchShardOpDuration.WithLabelValues(storageType, scenario, shardID).Observe(latencyMs / 1000.0)
	status := "success"
	if isError {
		status = "error"
	}
	BenchShardOpsTotal.WithLabelValues(storageType, scenario, shardID, status).Inc()
}
