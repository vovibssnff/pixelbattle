package main

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"pb_backend/server"
	"pb_backend/service"
	"pb_backend/websockets"
	"strconv"
)

func main() {
	redis_db := os.Getenv("REDIS_ADDR")
	redis_psw := os.Getenv("REDIS_PSW")
	// var redis_history, redis_timer, canvas_height, canvas_width int

	redis_history, err := strconv.Atoi(os.Getenv("REDIS_HISTORY"))
	redis_timer, err := strconv.Atoi(os.Getenv("REDIS_TIMER"))
	canvas_height, err := strconv.Atoi(os.Getenv("CANVAS_HEIGHT"))
	canvas_width, err := strconv.Atoi(os.Getenv("CANVAS_WIDTH"))

	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Starting service")

	//server
	ws := websockets.NewWebSocketServer(redis_db, redis_psw, redis_history)
	go ws.Run()

	//client
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("websocket connection init")
		websockets.ServeWs(ws, redis_db, redis_psw, redis_timer, w, r)
	})

	imgService := service.NewImageService()
	server := server.NewServer(imgService, redis_db, redis_psw, redis_history)
	server.WhiteCanvasInit(uint(canvas_height), uint(canvas_width))

	http.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		server.HandleInitCanvas(w, r, uint(canvas_height), uint(canvas_width))
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
