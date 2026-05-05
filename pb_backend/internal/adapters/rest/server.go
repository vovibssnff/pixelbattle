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
	height, width int, router *mux.Router) {

	logrus.Info("Initializing REST endpoints")

	handlers := NewRestHandlers(sessionService, vkAuthProvider, canvasService, userService)

	router.HandleFunc("/api/vk-login", handlers.HandleVKLogin).Methods("GET")
	router.HandleFunc("/api/register", handlers.HandlePasswordRegister).Methods("POST")
	router.HandleFunc("/api/login", handlers.HandlePasswordLogin).Methods("POST")
	router.HandleFunc("/api/rum", handlers.HandleRUMBeacon).Methods("POST")
	router.HandleFunc("/api/config", handlers.HandleClientConfig).Methods("GET")
	router.HandleFunc("/api/admin/ban", handlers.HandleBan).Methods("POST")
	router.HandleFunc("/api/admin/unban", handlers.HandleUnban).Methods("POST")

	router.HandleFunc("/api/faculty", handlers.HandleFaculty).Methods("GET")
	router.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleInitCanvas(w, r, uint(height), uint(width))
	}).Methods("GET")
}
