package server

import (
	"net/http"
	"pb_backend/service"
	"strconv"
	"github.com/sirupsen/logrus"
)

type Server struct {
	imgService 	*service.ImageService
}

func NewServer(imgService *service.ImageService) *Server {
	return &Server{
		imgService: imgService,
	}
}

func (server *Server) HandleInitCanvas(w http.ResponseWriter, r *http.Request) {
	logrus.Info("Received an init request")
	b := server.imgService.GetImageBytes("/usr/src/app/ava.png")
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Write(b)
}
