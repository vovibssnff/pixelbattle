package service

import (
	// "fmt"
	// "github.com/redis/go-redis/v9"
	// "github.com/sirupsen/logrus"
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
)

func init() {
	prometheus.MustRegister(requestsPerSecond)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(currentUsers)
	prometheus.MustRegister(overallRegistrations)
	prometheus.MustRegister(webSocketMessageDuration)
	prometheus.MustRegister(heatmapMetrics)
}

// func updateHeatMap(rdb *redis.Client) {
// 	heatmap, err := loadHeatMap(rdb)
// 	if err != nil {
// 		logrus.Error(err)
// 	}
// 	for _, val := range heatmap {
// 		heatmapMetrics.WithLabelValues(fmt.Sprintf("%d", val.X), fmt.Sprintf("%d", val.Y)).Set(float64(val.Len))
// 	}
// }

// func StartHeatmapUpdater(rdb *redis.Client) {
// 	ticker := time.NewTicker(30 * time.Second)
// 	defer ticker.Stop()
// 	for {
// 		select {
// 		case <-ticker.C:
// 			updateHeatMap(rdb)
// 			logrus.Info("HeatMap updated")
// 		}
// 	}
// }

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
