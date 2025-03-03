package rest

import (
	"net/http"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/core/domain"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func StartRestServer(sessionService domain.SessionService, vkAuthProvider vk.VKAuthProvider,
	canvasService domain.CanvasService, userService domain.UserService,
	timerService domain.TimerService, height, width int, router *mux.Router) {

	logrus.Info("Initializing REST endpoints")

	handlers := NewRestHandlers(sessionService, vkAuthProvider, canvasService, userService)

	router.HandleFunc("/api/login", handlers.HandleRegister).Methods("GET")
	router.HandleFunc("/api/faculty", handlers.HandleFaculty).Methods("GET")
	router.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleInitCanvas(w, r, uint(height), uint(width))
	}).Methods("GET")
}
