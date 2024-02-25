package websockets

import (
	"pb_backend/models"
	"pb_backend/service"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type WsServer struct {
	clients    map[*Client]bool
	broadcast  chan *models.Pixel
	register   chan *Client
	unregister chan *Client
	redis_service *redis.Client
}

func NewWebSocketServer(dbAddr string, dbPsw string, db int) *WsServer {
	return &WsServer{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *models.Pixel),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis_service: service.NewRedisClient(dbAddr, dbPsw, db),
	}
}

func (server *WsServer) Run() {
	logrus.Info("WebSocket server running")
	for {
		select {
		case client := <-server.register:
			logrus.Info("Server received register request")
			server.registerClient(client)
		case client := <-server.unregister:
			logrus.Info("Server received unregister request")
			server.unregisterClient(client)
		case pixel := <-server.broadcast:
			logrus.Info("Server received pixel")
			server.setPixel(pixel)
		}
	}
}

func (server *WsServer) registerClient(client *Client) {
	client.userid = uuid.New().String()
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
		// for client := range server.clients {
		// 	if client.userid == pixel.Userid {
		// 		client.send <- err
		// 	}
		// }
	}
	logrus.Info("Pixel written to redis db")
	for client := range server.clients {
		client.send <- pixel
	}
}

// func (server *WsServer) sendError(err error, userid string) {
// 	for client := range server.clients {
// 		if client.userid == userid {
// 			client.send 
// 		}
// 	}
// }