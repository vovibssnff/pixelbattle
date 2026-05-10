package grpc

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	gatewayv1 "pb_backend/internal/adapters/grpc/gen/pixelbattle/gateway/v1"
	"pb_backend/internal/core/domain"
)

// Phase 2 backend test (plan §p2-backend-test-grow):
// Verify the inter-gateway gRPC mesh:
//   1. BroadcastPixel from a peer ends up in the local WS hub.
//   2. BroadcastPixel from our own origin_instance is treated as an echo and dropped.
//   3. BroadcastControl from a peer ends up in the local WS hub.
//   4. Ping returns our own instance id.
//   5. PeerPool.FanOutPixel reaches every dialed peer.

const testInstanceID = "gw-srv-1"

type captureHub struct {
	pixels   int64
	controls int64
	last     atomic.Pointer[domain.Pixel]
	payload  atomic.Pointer[[]byte]
}

func (h *captureHub) BroadcastFromPeer(p *domain.Pixel) int {
	atomic.AddInt64(&h.pixels, 1)
	h.last.Store(p)
	return 1
}

func (h *captureHub) BroadcastControlFromPeer(payload []byte) int {
	atomic.AddInt64(&h.controls, 1)
	cp := append([]byte(nil), payload...)
	h.payload.Store(&cp)
	return 1
}

func (h *captureHub) pixelCount() int64   { return atomic.LoadInt64(&h.pixels) }
func (h *captureHub) controlCount() int64 { return atomic.LoadInt64(&h.controls) }

// newBufServer spins up the GatewayService over a bufconn listener and returns the live
// client + a stop func. Reuses the production grpc.NewServer wiring so interceptors and
// metrics paths are exercised.
func newBufServer(t *testing.T, instance string, hub WSHub) (gatewayv1.GatewayServiceClient, func()) {
	t.Helper()
	ensureMetrics()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	gatewayv1.RegisterGatewayServiceServer(srv, &gatewaySvc{instance: instance, hub: hub})
	go func() {
		_ = srv.Serve(lis)
	}()
	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	return gatewayv1.NewGatewayServiceClient(conn), func() {
		_ = conn.Close()
		srv.Stop()
		_ = lis.Close()
	}
}

func TestBroadcastPixelFromPeerHitsLocalHub(t *testing.T) {
	hub := &captureHub{}
	cli, stop := newBufServer(t, testInstanceID, hub)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ack, err := cli.BroadcastPixel(ctx, &gatewayv1.BroadcastPixelRequest{
		OriginInstance: "gw-peer",
		X:              10, Y: 20,
		R: 255, G: 128, B: 0,
		ClientSeq:    42,
		ServerRecvMs: time.Now().UnixMilli(),
		Writer:       "tester",
	})
	if err != nil {
		t.Fatalf("BroadcastPixel: %v", err)
	}
	if got := ack.GetRecipients(); got != 1 {
		t.Fatalf("recipients=%d, want 1", got)
	}
	if got := hub.pixelCount(); got != 1 {
		t.Fatalf("hub pixels=%d, want 1", got)
	}
	if p := hub.last.Load(); p == nil || p.X != 10 || p.Y != 20 || p.ClientSeq != 42 {
		t.Fatalf("hub last pixel mismatch: %+v", p)
	}
}

func TestBroadcastPixelOwnOriginIsDropped(t *testing.T) {
	hub := &captureHub{}
	cli, stop := newBufServer(t, testInstanceID, hub)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ack, err := cli.BroadcastPixel(ctx, &gatewayv1.BroadcastPixelRequest{
		OriginInstance: testInstanceID, // self echo
		X:              1, Y: 2, R: 1, G: 2, B: 3,
	})
	if err != nil {
		t.Fatalf("BroadcastPixel: %v", err)
	}
	if got := ack.GetRecipients(); got != 0 {
		t.Fatalf("recipients=%d, want 0 (self echo must be dropped)", got)
	}
	if got := hub.pixelCount(); got != 0 {
		t.Fatalf("hub pixels=%d, want 0", got)
	}
}

func TestBroadcastControlFromPeerHitsLocalHub(t *testing.T) {
	hub := &captureHub{}
	cli, stop := newBufServer(t, testInstanceID, hub)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	payload := []byte(`{"event":"canvas_resized","width":600,"height":300}`)
	ack, err := cli.BroadcastControl(ctx, &gatewayv1.BroadcastControlRequest{
		OriginInstance: "gw-peer",
		Event:          "canvas_resized",
		Payload:        payload,
	})
	if err != nil {
		t.Fatalf("BroadcastControl: %v", err)
	}
	if got := ack.GetRecipients(); got != 1 {
		t.Fatalf("recipients=%d, want 1", got)
	}
	if got := hub.controlCount(); got != 1 {
		t.Fatalf("hub controls=%d, want 1", got)
	}
}

func TestPingReturnsInstanceID(t *testing.T) {
	hub := &captureHub{}
	cli, stop := newBufServer(t, testInstanceID, hub)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := cli.Ping(ctx, &gatewayv1.PingRequest{})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if got := resp.GetInstanceId(); got != testInstanceID {
		t.Fatalf("instance_id=%q, want %q", got, testInstanceID)
	}
	if resp.GetServerMs() == 0 {
		t.Fatal("server_ms is zero")
	}
}

func TestPeerPoolFanOutReachesAllPeers(t *testing.T) {
	hubA := &captureHub{}
	hubB := &captureHub{}

	// Two independent bufconn servers, one TCP fall-back so we can use real PeerPool.
	// PeerPool.NewPeerPool dials via real net so we wire actual TCP loopback listeners.
	srvA, addrA := newTCPServer(t, "gw-a", hubA)
	defer srvA.Stop()
	srvB, addrB := newTCPServer(t, "gw-b", hubB)
	defer srvB.Stop()

	pool, err := NewPeerPool("gw-self", []string{addrA, addrB})
	if err != nil {
		t.Fatalf("NewPeerPool: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	total := pool.FanOutPixel(ctx, &domain.Pixel{X: 5, Y: 7, Color: []uint{0, 200, 50}}, "1700000000000-0")
	if total < 1 {
		t.Fatalf("fan-out reached %d peers; want >=1", total)
	}
	if hubA.pixelCount() == 0 || hubB.pixelCount() == 0 {
		t.Fatalf("not all hubs received the pixel: A=%d B=%d", hubA.pixelCount(), hubB.pixelCount())
	}
}

func newTCPServer(t *testing.T, instance string, hub WSHub) (*Server, string) {
	t.Helper()
	ensureMetrics()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	srv := grpc.NewServer()
	gatewayv1.RegisterGatewayServiceServer(srv, &gatewaySvc{instance: instance, hub: hub})
	go func() {
		_ = srv.Serve(lis)
	}()
	return &Server{grpc: srv, hub: hub, instance: instance, listenAddr: lis.Addr().String()}, lis.Addr().String()
}
