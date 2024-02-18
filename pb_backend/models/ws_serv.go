package models

import (
	"github.com/sirupsen/logrus"
)

type WsServer struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewWebSocketServer() *WsServer {
	return &WsServer{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
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
			var deserialized Pixel
			if err := deserialized.Deserialize(pixel); err != nil {
				logrus.Error(err)
				return
			}
			server.setPixel(&deserialized)
		}
	}
}

func (server *WsServer) registerClient(client *Client) {
	server.clients[client] = true
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		delete(server.clients, client)
	}
}

func (server *WsServer) setPixel(pixel *Pixel) {
	for client := range server.clients {
		client.send <- pixel
	}
}
