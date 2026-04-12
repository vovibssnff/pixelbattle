package service

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsPerSecond = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_per_second",
			Help: "Number of requests per second",
		},
		[]string{"path", "method"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Histogram of request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	currentUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "current_users",
			Help: "Current number of users",
		},
	)

	overallRegistrations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "overall_registrations",
			Help: "Total number of user registrations",
		},
	)

	webSocketMessageDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "websocket_message_duration_seconds",
			Help:    "Histogram of WebSocket message processing durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	heatmapMetrics = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pixel_heatmap",
			Help: "Number of pixels set at each coordinate on the canvas",
		},
		[]string{"x", "y"},
	)

	databaseOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Histogram of database operation durations",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"operation", "storage_type"},
	)

	databaseOperationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operation_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "storage_type", "status"},
	)

	databaseConnectionPoolSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connection_pool_size",
			Help: "Database connection pool size",
		},
		[]string{"storage_type", "state"}, // state: open, idle, in_use
	)

	pixelWriteQueueDepth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pixel_write_queue_depth",
			Help: "Number of pending pixel writes in queue",
		},
	)
)

func init() {
	prometheus.MustRegister(requestsPerSecond)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(currentUsers)
	prometheus.MustRegister(overallRegistrations)
	prometheus.MustRegister(webSocketMessageDuration)
	prometheus.MustRegister(heatmapMetrics)
	prometheus.MustRegister(databaseOperationDuration)
	prometheus.MustRegister(databaseOperationTotal)
	prometheus.MustRegister(databaseConnectionPoolSize)
	prometheus.MustRegister(pixelWriteQueueDepth)
}

func InstrumentHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		handler.ServeHTTP(rr, r)

		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues(r.URL.Path, r.Method).Observe(duration)
		requestsPerSecond.WithLabelValues(r.URL.Path, r.Method).Inc()
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// Hijack forwards to the underlying ResponseWriter so WebSocket upgrades work.
func (rr *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rr.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// Flush forwards to the underlying ResponseWriter for SSE / chunked responses.
func (rr *responseRecorder) Flush() {
	if fl, ok := rr.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func IncrementCurrentUsers() {
	currentUsers.Inc()
}

func DecrementCurrentUsers() {
	currentUsers.Dec()
}

func IncrementOverallRegistrations() {
	overallRegistrations.Inc()
}

func ObserveWebSocketMessageDuration(messageType string, start time.Time) {
	duration := time.Since(start).Seconds()
	webSocketMessageDuration.WithLabelValues(messageType).Observe(duration)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// ObserveDatabaseOperation records database operation duration and count.
func ObserveDatabaseOperation(operation, storageType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	databaseOperationDuration.WithLabelValues(operation, storageType).Observe(duration.Seconds())
	databaseOperationTotal.WithLabelValues(operation, storageType, status).Inc()
}

// SetDatabaseConnectionPoolSize sets the connection pool size metrics.
func SetDatabaseConnectionPoolSize(storageType string, open, idle, inUse int) {
	databaseConnectionPoolSize.WithLabelValues(storageType, "open").Set(float64(open))
	databaseConnectionPoolSize.WithLabelValues(storageType, "idle").Set(float64(idle))
	databaseConnectionPoolSize.WithLabelValues(storageType, "in_use").Set(float64(inUse))
}

// SetPixelWriteQueueDepth sets the pixel write queue depth.
func SetPixelWriteQueueDepth(depth int) {
	pixelWriteQueueDepth.Set(float64(depth))
}
