package websockets

import (
	"context"
	"net/http"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/core/service"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// ResizeNotice carries a JSON control frame for all clients (processed on the WS Run goroutine).
type ResizeNotice struct {
	Width, Height uint
	Payload       []byte
}

type WsServer struct {
	clients          map[*Client]bool
	broadcast        chan *domain.Pixel
	register         chan *Client
	unregister       chan *Client
	resizeNotify     chan ResizeNotice
	sessionService   domain.SessionService
	timerService     domain.TimerService
	userService      domain.UserService
	canvasService    domain.CanvasService
	canvasHeight     uint
	canvasWidth      uint
	allowAnonymousWS bool
	limiter          *LimiterHub
	replay           *PixelReplayBuffer
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
			logrus.Info("Server received pixel: ", pixel)
			server.setPixel(pixel)
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
		}
		service.ObserveWebSocketMessageDuration(tp, start)
	}
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
	for client := range server.clients {
		select {
		case client.send <- pixel:
		default:
			service.IncrementWSError("send_buffer_full")
		}
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
