package main

import (
	"net/http"
	"pb_backend/models"
	"pb_backend/service"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Starting service")
	
	ws := models.NewWebSocketServer()
	go ws.Run()
	
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("websocket connection init")
		models.ServeWs(ws, w, r)
	})

	imgService := service.NewImageService()
	server := models.NewServer(imgService)

	http.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		server.HandleInitCanvas(w, r)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
