package server

import (
	"net/http"
	"os"
	"pb_backend/service"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Server struct {
	imgService 	*service.ImageService
	rdb *redis.Client
}

func NewServer(imgService *service.ImageService, addr string, psw string, db int) *Server {
	return &Server{
		imgService: imgService,
		rdb: service.NewRedisClient(addr, psw, db),
	}
}

func (server *Server) HandleInitCanvas(w http.ResponseWriter, r *http.Request, n, m uint) {
	logrus.Info("Received an init request")
	service.InitializeCanvas(server.rdb, n, m)
	img := service.NewImage(n, m)
	service.GetCanvas(server.rdb, img)

	logrus.New()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
	file, err := os.OpenFile("logrus.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		logrus.SetOutput(file)
		logrus.Info(img.Data)
	} else {
		logrus.Info(err)
	}

	b := server.imgService.GetImageBytes(img)
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Write(b)
}
