package metrics

import (
	"net/http"
	"strconv"
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

	httpResponseStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_response_status_total",
			Help: "Total HTTP responses by path and status code",
		},
		[]string{"path", "status"},
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

	pixelsPlacedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pixels_placed_total",
			Help: "Total pixels placed on the canvas",
		},
		[]string{"faculty"},
	)

	loginAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "login_attempts_total",
			Help: "Total login attempts by method and result",
		},
		[]string{"method", "result"},
	)

	websocketConnectionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "websocket_connections_total",
			Help: "Cumulative WebSocket connections opened",
		},
	)

	websocketMessagesReceivedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_received_total",
			Help: "Total inbound WebSocket messages by type",
		},
		[]string{"type"},
	)

	canvasInitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "canvas_init_duration_seconds",
			Help:    "Time to serve /init_canvas responses",
			Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
	)

	sessionErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "session_errors_total",
			Help: "Total session decode or save failures",
		},
	)

	bannedActionRejectedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "banned_action_rejected_total",
			Help: "Total actions rejected because user is banned",
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
		[]string{"storage_type", "state"},
	)

	pixelWriteQueueDepth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pixel_write_queue_depth",
			Help: "Number of pending pixel writes in queue",
		},
	)
)

func init() {
	prometheus.MustRegister(
		requestsPerSecond,
		requestDuration,
		httpResponseStatus,
		currentUsers,
		overallRegistrations,
		pixelsPlacedTotal,
		loginAttemptsTotal,
		websocketConnectionsTotal,
		websocketMessagesReceivedTotal,
		canvasInitDuration,
		sessionErrorsTotal,
		bannedActionRejectedTotal,
		webSocketMessageDuration,
		heatmapMetrics,
		databaseOperationDuration,
		databaseOperationTotal,
		databaseConnectionPoolSize,
		pixelWriteQueueDepth,
	)
}

func RecordRequest(path, method string, duration time.Duration, statusCode int) {
	requestDuration.WithLabelValues(path, method).Observe(duration.Seconds())
	requestsPerSecond.WithLabelValues(path, method).Inc()
	httpResponseStatus.WithLabelValues(path, strconv.Itoa(statusCode)).Inc()
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func IncrementCurrentUsers()         { currentUsers.Inc() }
func DecrementCurrentUsers()         { currentUsers.Dec() }
func IncrementOverallRegistrations() { overallRegistrations.Inc() }
func IncrementWebSocketConnections() { websocketConnectionsTotal.Inc() }
func IncrementSessionErrors()        { sessionErrorsTotal.Inc() }
func IncrementBannedRejected()       { bannedActionRejectedTotal.Inc() }

func IncrementPixelsPlaced(faculty string) {
	pixelsPlacedTotal.WithLabelValues(faculty).Inc()
}

func RecordLoginAttempt(method, result string) {
	loginAttemptsTotal.WithLabelValues(method, result).Inc()
}

func IncrementWebSocketMessagesReceived(msgType string) {
	websocketMessagesReceivedTotal.WithLabelValues(msgType).Inc()
}

func ObserveWebSocketMessageDuration(messageType string, start time.Time) {
	webSocketMessageDuration.WithLabelValues(messageType).Observe(time.Since(start).Seconds())
}

func ObserveCanvasInitDuration(start time.Time) {
	canvasInitDuration.Observe(time.Since(start).Seconds())
}

func ObserveDatabaseOperation(operation, storageType string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	databaseOperationDuration.WithLabelValues(operation, storageType).Observe(duration.Seconds())
	databaseOperationTotal.WithLabelValues(operation, storageType, status).Inc()
}

func SetDatabaseConnectionPoolSize(storageType string, open, idle, inUse int) {
	databaseConnectionPoolSize.WithLabelValues(storageType, "open").Set(float64(open))
	databaseConnectionPoolSize.WithLabelValues(storageType, "idle").Set(float64(idle))
	databaseConnectionPoolSize.WithLabelValues(storageType, "in_use").Set(float64(inUse))
}

func SetPixelWriteQueueDepth(depth int) {
	pixelWriteQueueDepth.Set(float64(depth))
}

func RecordHeatmapPixel(x, y uint) {
	heatmapMetrics.WithLabelValues(strconv.FormatUint(uint64(x), 10), strconv.FormatUint(uint64(y), 10)).Inc()
}
