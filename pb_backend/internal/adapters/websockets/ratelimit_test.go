package websockets

import (
	"net/http"
	"testing"
	"time"
)

func TestLimiterHub_AllowPlacement(t *testing.T) {
	h := NewLimiterHub(5, 0)
	if h == nil {
		t.Fatal("expected hub")
	}
	if !h.AllowPlacement("u1") {
		t.Fatal("first allow")
	}
}

func TestLimiterHub_DisabledReturnsNil(t *testing.T) {
	if NewLimiterHub(0, 0) != nil {
		t.Fatal("expected nil")
	}
}

func TestTokenBucketRefills(t *testing.T) {
	b := newTokenBucket(10, 10)
	n := 0
	for i := 0; i < 20; i++ {
		if b.allow() {
			n++
		}
	}
	if n != 10 {
		t.Fatalf("burst cap: allowed %d", n)
	}
	time.Sleep(150 * time.Millisecond)
	if !b.allow() {
		t.Fatal("expected refill after sleep")
	}
}

func TestClientIPForwardedFor(t *testing.T) {
	r := &http.Request{Header: make(http.Header)}
	r.Header.Set("X-Forwarded-For", " 203.0.113.1, 10.0.0.1 ")
	if got := clientIP(r); got != "203.0.113.1" {
		t.Fatalf("got %q", got)
	}
}
