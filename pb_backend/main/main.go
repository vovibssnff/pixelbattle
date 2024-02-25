package main

import (
	"net/http"
	"pb_backend/service"
	"pb_backend/server"
	"pb_backend/websockets"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Starting service")
	
	ws := websockets.NewWebSocketServer("redis:6379", "redis", 1)
	go ws.Run()
	
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("websocket connection init")
		websockets.ServeWs(ws, "redis:6379", "redis", 2, w, r)
	})

	imgService := service.NewImageService()
	server := server.NewServer(imgService)

	http.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		server.HandleInitCanvas(w, r)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
