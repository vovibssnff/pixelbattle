package websockets

import (
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// tokenBucket is a simple refill bucket (tokens per second, capped burst).
type tokenBucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	mu     sync.Mutex
}

func newTokenBucket(ratePerSec float64, burst int) *tokenBucket {
	b := float64(burst)
	if b < 1 {
		b = 1
	}
	now := time.Now()
	return &tokenBucket{
		rate:   ratePerSec,
		burst:  b,
		tokens: b,
		last:   now,
	}
}

func (t *tokenBucket) allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(t.last).Seconds()
	t.last = now
	t.tokens = math.Min(t.burst, t.tokens+elapsed*t.rate)
	if t.tokens >= 1 {
		t.tokens -= 1
		return true
	}
	return false
}

// LimiterHub holds per-user pixel buckets and per-IP WS handshake buckets.
// Nil or constructed with both dimensions disabled means no limiting.
type LimiterHub struct {
	pixelRate   float64
	pixelBurst  int
	connRate    float64
	connBurst   int
	pixelBucket sync.Map // userid -> *tokenBucket
	connBucket  sync.Map // client IP -> *tokenBucket
}

// NewLimiterHub returns nil if both limits are disabled (<= 0).
func NewLimiterHub(pixelPerSec, wsConnPerMinPerIP int) *LimiterHub {
	if pixelPerSec <= 0 && wsConnPerMinPerIP <= 0 {
		return nil
	}
	h := &LimiterHub{}
	if pixelPerSec > 0 {
		h.pixelRate = float64(pixelPerSec)
		h.pixelBurst = max(1, pixelPerSec*2)
	}
	if wsConnPerMinPerIP > 0 {
		h.connRate = float64(wsConnPerMinPerIP) / 60.0
		h.connBurst = max(1, wsConnPerMinPerIP)
	}
	return h
}

// AllowPlacement enforces per-userid pixel messages (non-admins only at call site).
func (h *LimiterHub) AllowPlacement(userID string) bool {
	if h == nil || h.pixelRate <= 0 {
		return true
	}
	v, _ := h.pixelBucket.LoadOrStore(userID, newTokenBucket(h.pixelRate, h.pixelBurst))
	return v.(*tokenBucket).allow()
}

// AllowWSConn enforces new WebSocket handshakes per client IP.
func (h *LimiterHub) AllowWSConn(ip string) bool {
	if h == nil || h.connRate <= 0 || ip == "" {
		return true
	}
	v, _ := h.connBucket.LoadOrStore(ip, newTokenBucket(h.connRate, h.connBurst))
	return v.(*tokenBucket).allow()
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
