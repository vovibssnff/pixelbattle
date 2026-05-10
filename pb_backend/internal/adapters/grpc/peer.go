// Package grpc implements the Phase 2 inter-gateway mesh (plan §p2-grpc-mesh, ADR-004).
//
// Two roles per gateway:
//   - GatewayServer: receives BroadcastPixel / BroadcastControl from peers and pushes them
//     into the local WebSocket hub via the WSHub interface.
//   - PeerClient: maintains one *grpc.ClientConn per peer in GATEWAY_PEERS env and exposes
//     FanOutPixel / FanOutControl, which the WS hub calls after a local Redis write.
//
// Dedupe: every request carries origin_instance. Receivers and senders skip echoes whose
// origin matches their own instance_id.
package grpc

import (
	"context"
	"errors"
	"sync"
	"time"

	"pb_backend/internal/core/domain"
	"pb_backend/internal/metrics"

	gatewayv1 "pb_backend/internal/adapters/grpc/gen/pixelbattle/gateway/v1"

	"github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// WSHub is the subset of the WebSocket hub that the gRPC server pushes incoming peer
// broadcasts into. WsServer in adapters/websockets satisfies this interface.
type WSHub interface {
	// BroadcastFromPeer fans out a pixel received via gRPC to local WS clients only.
	// Returns recipient count for gateway_fanout_recipients.
	BroadcastFromPeer(p *domain.Pixel) int
	// BroadcastControlFromPeer fans out a JSON control payload to local WS clients only.
	BroadcastControlFromPeer(payload []byte) int
}

// PeerFanout is the cross-gateway client interface installed into WsServer so it can
// forward locally-accepted pixels and control events to peer gateways.
type PeerFanout interface {
	// FanOutPixel fans out a pixel to every peer; returns total recipients across all peers.
	FanOutPixel(ctx context.Context, p *domain.Pixel, streamID string) int
	// FanOutControl fans out a JSON control event to every peer.
	FanOutControl(ctx context.Context, event string, payload []byte) int
	// Close closes all peer connections.
	Close()
}

// metricInterceptors builds shared Prometheus interceptors for gRPC server + client.
// Histograms are auto-registered to the default Prometheus registry on first call.
var (
	once       sync.Once
	srvMetrics *prometheus.ServerMetrics
	cliMetrics *prometheus.ClientMetrics
)

func ensureMetrics() {
	once.Do(func() {
		srvMetrics = prometheus.NewServerMetrics(
			prometheus.WithServerHandlingTimeHistogram(),
		)
		cliMetrics = prometheus.NewClientMetrics(
			prometheus.WithClientHandlingTimeHistogram(),
		)
		// Use the default registry (via promauto-style). The package-level helpers
		// only register lazily, so we register explicitly through the promhttp default.
		// metrics.Handler() exposes the default Prometheus registry, so registering here
		// makes grpc_server_handled_total + grpc_server_handling_seconds available on /metrics.
		// (errors are logged; double-registration is non-fatal for hot-reload binaries.)
		if err := metrics.RegisterCollector(srvMetrics); err != nil {
			logrus.Warnf("grpc: server metrics already registered: %v", err)
		}
		if err := metrics.RegisterCollector(cliMetrics); err != nil {
			logrus.Warnf("grpc: client metrics already registered: %v", err)
		}
	})
}

// Server bundles the gRPC server, its socket listener, and a graceful shutdown hook.
type Server struct {
	grpc       *grpc.Server
	addr       string
	hub        WSHub
	instance   string
	stopOnce   sync.Once
	listenAddr string
}

// NewServer wires the GatewayService implementation with Prometheus interceptors and a
// recovery handler. WSHub is the local WebSocket hub the server pushes into.
func NewServer(instance string, hub WSHub) *Server {
	ensureMetrics()
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			srvMetrics.UnaryServerInterceptor(),
			recovery.UnaryServerInterceptor(),
		),
	)
	gatewayv1.RegisterGatewayServiceServer(srv, &gatewaySvc{instance: instance, hub: hub})
	return &Server{grpc: srv, hub: hub, instance: instance}
}

// Start begins listening; blocking call (run in a goroutine).
func (s *Server) Start(addr string) error {
	if addr == "" {
		return errors.New("grpc: empty listen address")
	}
	s.listenAddr = addr
	lis, err := listen(addr)
	if err != nil {
		return err
	}
	logrus.Infof("grpc: gateway server listening on %s (instance=%s)", addr, s.instance)
	return s.grpc.Serve(lis)
}

// Stop shuts the server down gracefully.
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		if s.grpc != nil {
			s.grpc.GracefulStop()
		}
	})
}

// gatewaySvc implements gatewayv1.GatewayServiceServer.
type gatewaySvc struct {
	gatewayv1.UnimplementedGatewayServiceServer
	instance string
	hub      WSHub
}

func (g *gatewaySvc) BroadcastPixel(ctx context.Context, in *gatewayv1.BroadcastPixelRequest) (*gatewayv1.BroadcastPixelResponse, error) {
	// Skip echoes from our own instance — defensive, peers should never send these.
	if in.GetOriginInstance() == g.instance {
		return &gatewayv1.BroadcastPixelResponse{Recipients: 0}, nil
	}
	p := &domain.Pixel{
		X:            uint(in.GetX()),
		Y:            uint(in.GetY()),
		Color:        []uint{uint(in.GetR()), uint(in.GetG()), uint(in.GetB())},
		ClientSeq:    in.GetClientSeq(),
		ServerRecvMs: in.GetServerRecvMs(),
		Userid:       "",
		Faculty:      "",
	}
	recipients := g.hub.BroadcastFromPeer(p)
	metrics.ObserveGatewayFanout("local_ws_from_peer", recipients)
	return &gatewayv1.BroadcastPixelResponse{Recipients: uint32(recipients)}, nil
}

func (g *gatewaySvc) BroadcastControl(ctx context.Context, in *gatewayv1.BroadcastControlRequest) (*gatewayv1.BroadcastControlResponse, error) {
	if in.GetOriginInstance() == g.instance {
		return &gatewayv1.BroadcastControlResponse{Recipients: 0}, nil
	}
	recipients := g.hub.BroadcastControlFromPeer(in.GetPayload())
	metrics.ObserveGatewayFanout("local_ws_control_from_peer", recipients)
	return &gatewayv1.BroadcastControlResponse{Recipients: uint32(recipients)}, nil
}

func (g *gatewaySvc) Ping(_ context.Context, _ *gatewayv1.PingRequest) (*gatewayv1.PingResponse, error) {
	return &gatewayv1.PingResponse{
		InstanceId: g.instance,
		ServerMs:   time.Now().UnixMilli(),
	}, nil
}

// peerClient wraps one ClientConn + GatewayServiceClient.
type peerClient struct {
	addr string
	conn *grpc.ClientConn
	api  gatewayv1.GatewayServiceClient
}

// PeerPool implements PeerFanout: round-robins fan-out across all configured peers.
type PeerPool struct {
	instance string
	peers    []*peerClient
	timeout  time.Duration
}

// NewPeerPool dials every peer in addrs (host:port). Failed dials are retried lazily.
func NewPeerPool(instance string, addrs []string) (*PeerPool, error) {
	ensureMetrics()
	if len(addrs) == 0 {
		return &PeerPool{instance: instance, timeout: 500 * time.Millisecond}, nil
	}
	peers := make([]*peerClient, 0, len(addrs))
	for _, a := range addrs {
		conn, err := grpc.NewClient(
			a,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithChainUnaryInterceptor(cliMetrics.UnaryClientInterceptor()),
		)
		if err != nil {
			logrus.Warnf("grpc: failed to dial peer %s: %v", a, err)
			continue
		}
		peers = append(peers, &peerClient{
			addr: a,
			conn: conn,
			api:  gatewayv1.NewGatewayServiceClient(conn),
		})
	}
	return &PeerPool{instance: instance, peers: peers, timeout: 500 * time.Millisecond}, nil
}

// Close all peer connections.
func (p *PeerPool) Close() {
	for _, peer := range p.peers {
		_ = peer.conn.Close()
	}
}

// FanOutPixel forwards pixel to every peer. Returns sum of recipient counts (each peer's
// local WS fan-out). Logs but does not propagate per-peer errors so one slow peer doesn't
// stall the WS hub.
func (p *PeerPool) FanOutPixel(ctx context.Context, px *domain.Pixel, streamID string) int {
	if len(p.peers) == 0 {
		return 0
	}
	color := px.Color
	for len(color) < 3 {
		color = append(color, 0)
	}
	req := &gatewayv1.BroadcastPixelRequest{
		OriginInstance: p.instance,
		X:              uint32(px.X),
		Y:              uint32(px.Y),
		R:              uint32(color[0]),
		G:              uint32(color[1]),
		B:              uint32(color[2]),
		ClientSeq:      px.ClientSeq,
		ServerRecvMs:   px.ServerRecvMs,
		Writer:         px.Userid,
		StreamId:       streamID,
	}
	total := 0
	for _, peer := range p.peers {
		callCtx, cancel := context.WithTimeout(ctx, p.timeout)
		ack, err := peer.api.BroadcastPixel(callCtx, req)
		cancel()
		if err != nil {
			logrus.Debugf("grpc: BroadcastPixel to %s failed: %v", peer.addr, err)
			continue
		}
		total += int(ack.GetRecipients())
	}
	metrics.ObserveGatewayFanout("grpc_peer", total)
	return total
}

// FanOutControl forwards an admin / control event to every peer.
func (p *PeerPool) FanOutControl(ctx context.Context, event string, payload []byte) int {
	if len(p.peers) == 0 {
		return 0
	}
	req := &gatewayv1.BroadcastControlRequest{
		OriginInstance: p.instance,
		Event:          event,
		Payload:        payload,
	}
	total := 0
	for _, peer := range p.peers {
		callCtx, cancel := context.WithTimeout(ctx, p.timeout)
		ack, err := peer.api.BroadcastControl(callCtx, req)
		cancel()
		if err != nil {
			logrus.Debugf("grpc: BroadcastControl to %s failed: %v", peer.addr, err)
			continue
		}
		total += int(ack.GetRecipients())
	}
	return total
}
