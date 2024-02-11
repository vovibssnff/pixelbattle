package main

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Server struct {
	img    draw.Image
	imgBuf []byte
}

func loadImage(loadPath string) draw.Image {
	f, err := os.Open(loadPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	pngimg, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	return pngimg.(draw.Image)
}

func (sv *Server) GetImageBytes() []byte {
	if sv.imgBuf == nil {
		buf := bytes.NewBuffer(nil)
		if err := png.Encode(buf, sv.img); err != nil {
			log.Println(err)
		}
		sv.imgBuf = buf.Bytes()
	}
	return sv.imgBuf
}

func (sv *Server) HandleGetImage(w http.ResponseWriter, req *http.Request) {
	logrus.Info("Received a request")
	b := sv.GetImageBytes() // Placeholder function call, replace with actual implementation
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Write(b)
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Starting service")
	server := &Server{}

	loadPath := "/usr/src/app/ava.png"

	server.img = loadImage(loadPath)

	http.HandleFunc("/init_canvas", server.HandleGetImage)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
