package websockets

import (
	"context"
	"net/http"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/core/service"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// ResizeNotice carries a JSON control frame for all clients (processed on the WS Run goroutine).
type ResizeNotice struct {
	Width, Height uint
	Payload       []byte
}

// PeerFanout is the cross-gateway hook (gRPC PeerPool) installed by main when running in
// gateway mode. setPixel calls FanOutPixel after a successful Redis write so peers see
// new pixels immediately. nil in monolith mode (no peers).
type PeerFanout interface {
	FanOutPixel(ctx context.Context, p *domain.Pixel, streamID string) int
	FanOutControl(ctx context.Context, event string, payload []byte) int
}

// fromPeer wraps an inbound pixel from another gateway so the Run loop can fan it out
// to local WS clients only — bypassing the canvas write and the peer fan-out paths.
type fromPeer struct {
	pixel *domain.Pixel
	ack   chan int
}

// fromPeerControl wraps an inbound control payload from another gateway.
type fromPeerControl struct {
	payload []byte
	ack     chan int
}

type WsServer struct {
	clients          map[*Client]bool
	broadcast        chan *domain.Pixel
	register         chan *Client
	unregister       chan *Client
	resizeNotify     chan ResizeNotice
	peerPixel        chan fromPeer
	peerControl      chan fromPeerControl
	sessionService   domain.SessionService
	timerService     domain.TimerService
	userService      domain.UserService
	canvasService    domain.CanvasService
	canvasHeight     uint
	canvasWidth      uint
	allowAnonymousWS bool
	limiter          *LimiterHub
	replay           *PixelReplayBuffer

	peerMu     sync.RWMutex
	peerFanout PeerFanout
}

func NewWebSocketServer(
	sessionService domain.SessionService,
	timerService domain.TimerService,
	userService domain.UserService,
	canvasService domain.CanvasService,
	canvasHeight, canvasWidth uint,
	allowAnonymousWS bool,
	limiter *LimiterHub,
	replay *PixelReplayBuffer,
) *WsServer {
	return &WsServer{
		clients:          make(map[*Client]bool),
		broadcast:        make(chan *domain.Pixel),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		resizeNotify:     make(chan ResizeNotice, 4),
		peerPixel:        make(chan fromPeer, 256),
		peerControl:      make(chan fromPeerControl, 16),
		sessionService:   sessionService,
		timerService:     timerService,
		userService:      userService,
		canvasService:    canvasService,
		canvasHeight:     canvasHeight,
		canvasWidth:      canvasWidth,
		allowAnonymousWS: allowAnonymousWS,
		limiter:          limiter,
		replay:           replay,
	}
}

// SetPeerFanout installs the gRPC peer fan-out implementation (Phase 2 gateway mode).
// Safe to call before or after Run() begins.
func (s *WsServer) SetPeerFanout(p PeerFanout) {
	s.peerMu.Lock()
	defer s.peerMu.Unlock()
	s.peerFanout = p
}

func (s *WsServer) currentPeerFanout() PeerFanout {
	s.peerMu.RLock()
	defer s.peerMu.RUnlock()
	return s.peerFanout
}

// BroadcastFromPeer is called by the gRPC server when a peer gateway forwards a pixel.
// It blocks briefly to send into the Run goroutine (which then fans out to local WS
// clients), then returns the recipient count. Bypasses Redis write and peer fan-out
// to prevent loops.
func (s *WsServer) BroadcastFromPeer(p *domain.Pixel) int {
	if s == nil || p == nil {
		return 0
	}
	ack := make(chan int, 1)
	select {
	case s.peerPixel <- fromPeer{pixel: p, ack: ack}:
	case <-time.After(250 * time.Millisecond):
		service.IncrementWSError("peer_pixel_buffer_full")
		return 0
	}
	select {
	case n := <-ack:
		return n
	case <-time.After(500 * time.Millisecond):
		return 0
	}
}

// BroadcastControlFromPeer fans out a JSON control payload (e.g. a forwarded RESIZE) to
// local WS clients only.
func (s *WsServer) BroadcastControlFromPeer(payload []byte) int {
	if s == nil || len(payload) == 0 {
		return 0
	}
	ack := make(chan int, 1)
	select {
	case s.peerControl <- fromPeerControl{payload: append([]byte(nil), payload...), ack: ack}:
	case <-time.After(250 * time.Millisecond):
		service.IncrementWSError("peer_control_buffer_full")
		return 0
	}
	select {
	case n := <-ack:
		return n
	case <-time.After(500 * time.Millisecond):
		return 0
	}
}

func (server *WsServer) Run() {
	logrus.Info("WebSocket server running")
	for {
		start := time.Now()
		var tp string
		select {
		case client := <-server.register:
			server.registerClient(client)
			tp = "connect"
			logrus.Info("Current users: ", len(server.clients))
		case client := <-server.unregister:
			server.unregisterClient(client)
			tp = "disconnect"
			logrus.Info("Current users: ", len(server.clients))
		case pixel := <-server.broadcast:
			tp = "pixel"
			server.setPixel(pixel)
		case fp := <-server.peerPixel:
			tp = "peer_pixel"
			fp.ack <- server.broadcastLocalOnly(fp.pixel)
		case fpc := <-server.peerControl:
			tp = "peer_control"
			fpc.ack <- server.broadcastControlLocalOnly(fpc.payload)
		case rn := <-server.resizeNotify:
			tp = "resize"
			server.canvasWidth = rn.Width
			server.canvasHeight = rn.Height
			for client := range server.clients {
				p := append([]byte(nil), rn.Payload...)
				select {
				case client.control <- p:
				default:
					service.IncrementWSError("control_buffer_full")
				}
			}
			// Mirror to peers (gRPC fan-out path) so other gateways' clients also resize.
			if pf := server.currentPeerFanout(); pf != nil {
				go pf.FanOutControl(context.Background(), "RESIZE", append([]byte(nil), rn.Payload...))
			}
		}
		service.ObserveWebSocketMessageDuration(tp, start)
	}
}

// broadcastLocalOnly fans out a pixel to local WS clients without touching Redis or
// peers. Used by the peer-pixel and op-log paths. Returns recipient count.
func (server *WsServer) broadcastLocalOnly(pixel *domain.Pixel) int {
	if pixel == nil {
		return 0
	}
	pixel.Userid = ""
	pixel.Faculty = ""
	n := 0
	for client := range server.clients {
		select {
		case client.send <- pixel:
			n++
		default:
			service.IncrementWSError("send_buffer_full")
		}
	}
	service.ObserveGatewayFanout("local_ws", n)
	return n
}

// broadcastControlLocalOnly fans out an opaque control payload to local WS clients only.
// payload is the already-encoded JSON frame the client expects (e.g. RESIZE).
func (server *WsServer) broadcastControlLocalOnly(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	n := 0
	for client := range server.clients {
		p := append([]byte(nil), payload...)
		select {
		case client.control <- p:
			n++
		default:
			service.IncrementWSError("control_buffer_full")
		}
	}
	return n
}

// NotifyCanvasResize updates in-memory WS bounds and broadcasts a JSON control message (ADR-003).
func (s *WsServer) NotifyCanvasResize(width, height uint, payload []byte) {
	if s == nil {
		return
	}
	msg := ResizeNotice{Width: width, Height: height, Payload: append([]byte(nil), payload...)}
	select {
	case s.resizeNotify <- msg:
	default:
		logrus.Warn("ws: resize notify channel full")
	}
}

func (server *WsServer) registerClient(client *Client) {
	server.clients[client] = true
	service.IncrementCurrentUsers()
	service.IncrementWebSocketConnections()
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		service.DecrementCurrentUsers()
		delete(server.clients, client)
	}
}

func (server *WsServer) setPixel(pixel *domain.Pixel) {
	visibleStart := time.Now()
	if pixel.ServerRecvMs > 0 {
		visibleStart = time.UnixMilli(pixel.ServerRecvMs)
	}
	if err := server.canvasService.WritePixel(context.Background(), pixel); err != nil {
		service.IncrementWSError("write_pixel")
		return
	}
	service.IncrementPixelsPlaced(pixel.Faculty)
	service.RecordHeatmapPixel(pixel.X, pixel.Y)
	service.SetPixelWriteQueueDepth(len(server.broadcast))
	pixel.Userid = ""
	pixel.Faculty = ""
	replayMs := time.Now().UnixMilli()
	if server.replay != nil {
		server.replay.Add(pixel, replayMs)
	}
	local := 0
	for client := range server.clients {
		select {
		case client.send <- pixel:
			local++
		default:
			service.IncrementWSError("send_buffer_full")
		}
	}
	service.ObserveGatewayFanout("local_ws", local)
	// Phase 2: forward to peer gateways in a goroutine so the Run loop is not blocked.
	if pf := server.currentPeerFanout(); pf != nil {
		px := *pixel
		go pf.FanOutPixel(context.Background(), &px, "")
	}
	service.ObservePixelWriteVisible(time.Since(visibleStart))
}

func StartWebSocketServer(
	sessionService domain.SessionService,
	canvasService domain.CanvasService,
	timerService domain.TimerService,
	userService domain.UserService,
	router *mux.Router,
	canvasHeight, canvasWidth int,
	allowAnonymousWS bool,
	limiter *LimiterHub,
	replay *PixelReplayBuffer,
) *WsServer {
	var ch, cw uint
	if canvasHeight > 0 {
		ch = uint(canvasHeight)
	}
	if canvasWidth > 0 {
		cw = uint(canvasWidth)
	}
	ws := NewWebSocketServer(sessionService, timerService, userService, canvasService, ch, cw, allowAnonymousWS, limiter, replay)
	go ws.Run()

	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(ws, w, r)
	})
	return ws
}
