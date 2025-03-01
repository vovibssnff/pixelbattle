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

type WsServer struct {
	clients        map[*Client]bool
	broadcast      chan *domain.Pixel
	register       chan *Client
	unregister     chan *Client
	sessionService domain.SessionService
	timerService   domain.TimerService
	userService    domain.UserService
	canvasService  domain.CanvasService
}

func NewWebSocketServer(
	sessionService domain.SessionService,
	timerService domain.TimerService,
	userService domain.UserService,
	canvasService domain.CanvasService,
) *WsServer {
	return &WsServer{
		clients:        make(map[*Client]bool),
		broadcast:      make(chan *domain.Pixel),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		sessionService: sessionService,
		timerService:   timerService,
		userService:    userService,
		canvasService:  canvasService,
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
		}
		service.ObserveWebSocketMessageDuration(tp, start)
	}
}

func (server *WsServer) registerClient(client *Client) {
	server.clients[client] = true
	service.IncrementCurrentUsers()
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		service.DecrementCurrentUsers()
		delete(server.clients, client)
	}
}

func (server *WsServer) setPixel(pixel *domain.Pixel) {
	if err := server.canvasService.WritePixel(context.Background(), pixel); err != nil {
		// logrus.Error(err)
		return
	}
	pixel.Userid = 0
	pixel.Faculty = ""
	for client := range server.clients {
		client.send <- pixel
	}
}

func StartWebSocketServer(
	sessionService domain.SessionService,
	canvasService domain.CanvasService,
	timerService domain.TimerService,
	userService domain.UserService,
	router *mux.Router,
) {
	ws := NewWebSocketServer(sessionService, timerService, userService, canvasService)
	go ws.Run()

	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(ws, w, r)
	})
}
