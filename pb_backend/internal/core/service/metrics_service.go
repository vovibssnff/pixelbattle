package service

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"pb_backend/internal/metrics"
	"time"
)

func InstrumentHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		handler.ServeHTTP(rr, r)
		metrics.RecordRequest(r.URL.Path, r.Method, time.Since(start), rr.statusCode)
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

func (rr *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rr.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

func (rr *responseRecorder) Flush() {
	if fl, ok := rr.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func MetricsHandler() http.Handler   { return metrics.Handler() }
func IncrementCurrentUsers()         { metrics.IncrementCurrentUsers() }
func DecrementCurrentUsers()         { metrics.DecrementCurrentUsers() }
func IncrementOverallRegistrations() { metrics.IncrementOverallRegistrations() }
func ObserveWebSocketMessageDuration(messageType string, start time.Time) {
	metrics.ObserveWebSocketMessageDuration(messageType, start)
}
func IncrementPixelsPlaced(faculty string)     { metrics.IncrementPixelsPlaced(faculty) }
func RecordLoginAttempt(method, result string) { metrics.RecordLoginAttempt(method, result) }
func IncrementWebSocketConnections()           { metrics.IncrementWebSocketConnections() }
func IncrementWebSocketMessagesReceived(msgType string) {
	metrics.IncrementWebSocketMessagesReceived(msgType)
}
func ObserveCanvasInitDuration(start time.Time) { metrics.ObserveCanvasInitDuration(start) }
func IncrementSessionErrors()                   { metrics.IncrementSessionErrors() }
func IncrementBannedRejected()                  { metrics.IncrementBannedRejected() }
func RecordHeatmapPixel(x, y uint)              { metrics.RecordHeatmapPixel(x, y) }
func SetPixelWriteQueueDepth(depth int)         { metrics.SetPixelWriteQueueDepth(depth) }
func ObserveDatabaseOperation(op, st string, d time.Duration, e error) {
	metrics.ObserveDatabaseOperation(op, st, d, e)
}
func SetDatabaseConnectionPoolSize(st string, o, i, u int) {
	metrics.SetDatabaseConnectionPoolSize(st, o, i, u)
}
func ObserveE2EPixelLatency(source string, d time.Duration) {
	metrics.ObserveE2EPixelLatency(source, d)
}
func IncrementWSError(kind string)             { metrics.IncrementWSError(kind) }
func ObservePixelWriteVisible(d time.Duration) { metrics.ObservePixelWriteVisible(d) }
func IncrementRUMBeacon(result string)         { metrics.IncrementRUMBeacon(result) }
func ObserveClientFPS(v float64)               { metrics.ObserveClientFPS(v) }
func ObserveClientFrameTimeP95Ms(v float64)    { metrics.ObserveClientFrameTimeP95Ms(v) }
func ObserveClientWebVitalMs(metric string, v float64) {
	metrics.ObserveClientWebVitalMs(metric, v)
}
func ObserveClientWSRenderLatencyMs(v float64) { metrics.ObserveClientWSRenderLatencyMs(v) }
func IncrementAdminAction(action, result string) {
	metrics.IncrementAdminAction(action, result)
}

func IncrementRejected(reason string) { metrics.IncrementRejected(reason) }

func IncrementOptimisticCorrection(reason string) {
	metrics.IncrementOptimisticCorrection(reason)
}

func SetCanvasDimensionsGauge(width, height uint) {
	metrics.SetCanvasDimensionsGauge(width, height)
}

// SetOpstreamLag re-exports metrics.SetOpstreamLag so adapter code does not import the
// metrics package directly (keeps ADR-002 adapter-boundary clean).
func SetOpstreamLag(group, consumer, shardID string, lag time.Duration) {
	metrics.SetOpstreamLag(group, consumer, shardID, lag)
}

// SetRedisReplicationLag re-exports metrics.SetRedisReplicationLag.
func SetRedisReplicationLag(shardID string, lag time.Duration) {
	metrics.SetRedisReplicationLag(shardID, lag)
}

// ObserveGatewayFanout re-exports metrics.ObserveGatewayFanout.
func ObserveGatewayFanout(path string, recipients int) {
	metrics.ObserveGatewayFanout(path, recipients)
}

// SetGatewayInstanceInfo re-exports metrics.SetGatewayInstanceInfo.
func SetGatewayInstanceInfo(instanceID, hostname, version string) {
	metrics.SetGatewayInstanceInfo(instanceID, hostname, version)
}
