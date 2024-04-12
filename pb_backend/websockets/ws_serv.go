package websockets

import (
	"pb_backend/models"
	"pb_backend/service"
	"github.com/gorilla/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type WsServer struct {
	clients    map[*Client]bool
	broadcast  chan *models.Pixel
	register   chan *Client
	unregister chan *Client
	redis_service *redis.Client
	redis_banned *redis.Client
	store 		*sessions.CookieStore
}

func NewWebSocketServer(redisService *redis.Client, sessionStore *sessions.CookieStore, redisBanned *redis.Client) *WsServer {
	return &WsServer{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *models.Pixel),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis_service: redisService,
		redis_banned: redisBanned,
		store:		sessionStore,
	}
}

func (server *WsServer) Run() {
	logrus.Info("WebSocket server running")
	for {
		select {
		case client := <-server.register:
			server.registerClient(client)
			logrus.Info("Current users: ", len(server.clients))
		case client := <-server.unregister:
			server.unregisterClient(client)
			logrus.Info("Current users: ", len(server.clients))
		case pixel := <-server.broadcast:
			logrus.Info("Server received pixel: ", pixel)
			server.setPixel(pixel)
		}
	}
}

func (server *WsServer) registerClient(client *Client) {
	// session, _ := server.store.Get(r, "user-session")
	// client.userid = session.Values["ID"]

	server.clients[client] = true
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		delete(server.clients, client)
	}
}

func (server *WsServer) setPixel(pixel *models.Pixel) {
	if err := service.WritePixel(server.redis_service, pixel); err != nil {
		logrus.Error(err)
		return
	}
	// logrus.Info("Pixel written to redis db")
	pixel.Userid = 0
	pixel.Faculty = ""
	for client := range server.clients {
		client.send <- pixel
	}
}
