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
			Help:    "Time to serve canvas init responses (/init_canvas and /api/canvas.png)",
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
		[]string{"operation", "storage_type", "shard_id", "instance_id"},
	)

	databaseOperationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operation_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "storage_type", "status", "shard_id", "instance_id"},
	)

	crdtResolvedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "crdt_resolved_total",
			Help: "CRDT / LWW conflict resolutions by ordering path (stream_id primary, hlc_fallback)",
		},
		[]string{"path"},
	)

	crdtMergeDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "crdt_merge_duration_seconds",
			Help:    "Wall time spent in CRDT merge on the HLC fallback path",
			Buckets: []float64{.000001, .000005, .00001, .000025, .00005, .0001, .00025, .0005, .001, .0025, .005},
		},
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

	e2ePixelLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "e2e_pixel_latency_seconds",
			Help:    "End-to-end pixel latency from client stamp to server observation",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"source"},
	)

	wsErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_errors_total",
			Help: "Total websocket errors by kind",
		},
		[]string{"kind"},
	)

	availabilitySLITotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "availability_sli_total",
			Help: "Availability SLI total requests by result",
		},
		[]string{"result"},
	)

	pixelWriteVisibleSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "pixel_write_visible_seconds",
			Help:    "Time from WS pixel ingest to write completion for visibility",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
	)

	rumBeaconTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rum_beacon_total",
			Help: "Total RUM beacons received by result",
		},
		[]string{"result"},
	)

	clientFPS = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "client_fps",
			Help:    "Client-reported FPS average",
			Buckets: []float64{5, 10, 15, 20, 24, 30, 45, 60, 90, 120},
		},
	)

	clientFrameTimeP95Ms = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "client_frame_time_p95_ms",
			Help:    "Client-reported p95 frame time in milliseconds",
			Buckets: []float64{4, 8, 12, 16, 20, 24, 33, 50, 66, 100, 200},
		},
	)

	clientWebVitalMs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "client_web_vital_ms",
			Help:    "Client web-vitals in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"metric"},
	)

	clientWSRenderLatencyMs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "client_ws_render_latency_ms",
			Help:    "Client-reported websocket-to-render latency p95 in milliseconds",
			Buckets: []float64{1, 2, 4, 8, 16, 25, 50, 100, 250, 500, 1000, 2500},
		},
	)

	adminActionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_action_total",
			Help: "Admin actions (ban, unban, grant_admin, revoke_admin, set_timer, freeze, resize, list_users) by outcome",
		},
		[]string{"action", "result"},
	)

	rejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rejected_total",
			Help: "Rejected requests or actions (rate limits, policy); see label reason",
		},
		[]string{"reason"},
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
		crdtResolvedTotal,
		crdtMergeDurationSeconds,
		databaseConnectionPoolSize,
		pixelWriteQueueDepth,
		e2ePixelLatencySeconds,
		wsErrorsTotal,
		availabilitySLITotal,
		pixelWriteVisibleSeconds,
		rumBeaconTotal,
		clientFPS,
		clientFrameTimeP95Ms,
		clientWebVitalMs,
		clientWSRenderLatencyMs,
		adminActionTotal,
		rejectedTotal,
	)
}

func RecordRequest(path, method string, duration time.Duration, statusCode int) {
	requestDuration.WithLabelValues(path, method).Observe(duration.Seconds())
	requestsPerSecond.WithLabelValues(path, method).Inc()
	httpResponseStatus.WithLabelValues(path, strconv.Itoa(statusCode)).Inc()
	if statusCode < http.StatusInternalServerError {
		availabilitySLITotal.WithLabelValues("ok").Inc()
		return
	}
	availabilitySLITotal.WithLabelValues("error").Inc()
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

// MonolithShardID and MonolithInstanceID are the default shard labels for the single-node deployment (Phase 1 / baseline).
const MonolithShardID = "single"
const MonolithInstanceID = "single"

func ObserveDatabaseOperation(operation, storageType string, duration time.Duration, err error) {
	ObserveDatabaseOperationScoped(operation, storageType, MonolithShardID, MonolithInstanceID, duration, err)
}

// ObserveDatabaseOperationScoped records DB latency with architecture-internal shard labels (gateways set shard_id / instance_id).
func ObserveDatabaseOperationScoped(operation, storageType, shardID, instanceID string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	if shardID == "" {
		shardID = MonolithShardID
	}
	if instanceID == "" {
		instanceID = MonolithInstanceID
	}
	databaseOperationDuration.WithLabelValues(operation, storageType, shardID, instanceID).Observe(duration.Seconds())
	databaseOperationTotal.WithLabelValues(operation, storageType, status, shardID, instanceID).Inc()
}

// IncrementCRDTResolved records which ordering path won (path: stream_id | hlc_fallback).
// Call from adapters after domain/crdt.DecideLWW — see adr/002-phase2-adapter-boundaries.md (ADR-002).
func IncrementCRDTResolved(path string) {
	crdtResolvedTotal.WithLabelValues(path).Inc()
}

// ObserveCRDTMergeDuration records merge wall time on the HLC fallback path.
func ObserveCRDTMergeDuration(d time.Duration) {
	crdtMergeDurationSeconds.Observe(d.Seconds())
}

func SetDatabaseConnectionPoolSize(storageType string, open, idle, inUse int) {
	databaseConnectionPoolSize.WithLabelValues(storageType, "open").Set(float64(open))
	databaseConnectionPoolSize.WithLabelValues(storageType, "idle").Set(float64(idle))
	databaseConnectionPoolSize.WithLabelValues(storageType, "in_use").Set(float64(inUse))
}

func SetPixelWriteQueueDepth(depth int) {
	pixelWriteQueueDepth.Set(float64(depth))
}

func ObserveE2EPixelLatency(source string, d time.Duration) {
	e2ePixelLatencySeconds.WithLabelValues(source).Observe(d.Seconds())
}

func IncrementWSError(kind string) {
	wsErrorsTotal.WithLabelValues(kind).Inc()
}

func ObservePixelWriteVisible(d time.Duration) {
	pixelWriteVisibleSeconds.Observe(d.Seconds())
}

func IncrementRUMBeacon(result string) {
	rumBeaconTotal.WithLabelValues(result).Inc()
}

func ObserveClientFPS(v float64) {
	clientFPS.Observe(v)
}

func ObserveClientFrameTimeP95Ms(v float64) {
	clientFrameTimeP95Ms.Observe(v)
}

func ObserveClientWebVitalMs(metric string, v float64) {
	clientWebVitalMs.WithLabelValues(metric).Observe(v)
}

func ObserveClientWSRenderLatencyMs(v float64) {
	clientWSRenderLatencyMs.Observe(v)
}

// IncrementAdminAction records a runtime-admin call. action is the canonical action
// name (e.g. "ban", "grant_admin", "set_timer", "freeze", "list_users"); result is
// "ok", "forbidden", "not_found", or "error". Used by the Grafana Runtime Admin
// dashboard's audit table and by the Phase 1 §13.1 acceptance check.
func IncrementAdminAction(action, result string) {
	adminActionTotal.WithLabelValues(action, result).Inc()
}

// IncrementRejected records gateway-style rejections (plan §11.5). reason examples:
// rate_limit_pixel, rate_limit_ws_ip, malformed, anonymous_disabled.
func IncrementRejected(reason string) {
	rejectedTotal.WithLabelValues(reason).Inc()
}

func RecordHeatmapPixel(x, y uint) {
	heatmapMetrics.WithLabelValues(strconv.FormatUint(uint64(x), 10), strconv.FormatUint(uint64(y), 10)).Inc()
}
