package server

import (
	"net/http"
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

func (server *Server) HandleInitCanvas(writer http.ResponseWriter, r *http.Request, h, w uint) {
	logrus.Info("Received an init request")
	img := service.NewImage(h, w)
	service.GetCanvas(server.rdb, img)
	b := server.imgService.GetImageBytes(img)
	writer.Header().Set("Content-Length", strconv.Itoa(len(b)))
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Cache-Control", "no-cache, no-store")
	logrus.Info("Bytes sent")
	writer.Write(b)
}

func (server *Server) WhiteCanvasInit(n, m uint) {
	service.InitializeCanvas(server.rdb, n, m)
}
