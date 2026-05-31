package rest

import (
	"net/http"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/core/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func StartRestServer(sessionService domain.SessionService, vkAuthProvider vk.VKAuthProvider,
	canvasService domain.CanvasService, userService domain.UserService,
	timerService domain.TimerService,
	adminAPIToken string,
	snapshotter *service.CanvasSnapshotter,
	wsHub WSRuntimeNotifier,
	height, width int, router *mux.Router) {

	logrus.Info("Initializing REST endpoints")

	handlers := NewRestHandlers(sessionService, vkAuthProvider, canvasService, userService, timerService, adminAPIToken, snapshotter, wsHub)

	router.HandleFunc("/api/vk-login", handlers.HandleVKLogin).Methods("GET")
	router.HandleFunc("/api/register", handlers.HandlePasswordRegister).Methods("POST")
	router.HandleFunc("/api/login", handlers.HandlePasswordLogin).Methods("POST")
	router.HandleFunc("/api/rum", handlers.HandleRUMBeacon).Methods("POST")
	router.HandleFunc("/api/config", handlers.HandleClientConfig).Methods("GET")
	router.HandleFunc("/api/canvas.png", func(w http.ResponseWriter, r *http.Request) {
		h := uint(height)
		wd := uint(width)
		if height <= 0 || width <= 0 {
			http.Error(w, "invalid canvas dimensions", http.StatusInternalServerError)
			return
		}
		handlers.HandleCanvasPNG(w, r, h, wd)
	}).Methods("GET")

	router.HandleFunc("/api/pixels/info", handlers.HandlePixelInfo).Methods("GET")

	router.HandleFunc("/api/admin/users/ban", handlers.HandleAdminBan).Methods("POST")
	router.HandleFunc("/api/admin/users/unban", handlers.HandleAdminUnban).Methods("POST")
	router.HandleFunc("/api/admin/users/grant_admin", handlers.HandleAdminGrant).Methods("POST")
	router.HandleFunc("/api/admin/users/revoke_admin", handlers.HandleAdminRevoke).Methods("POST")
	router.HandleFunc("/api/admin/users", handlers.HandleAdminUsers).Methods("GET")
	router.HandleFunc("/api/admin/pixels/info", handlers.HandleAdminPixelInfo).Methods("GET")
	router.HandleFunc("/api/admin/pixels/cache", handlers.HandleAdminPixelInfoCache).Methods("GET")
	router.HandleFunc("/api/admin/timer/set", handlers.HandleAdminTimerSet).Methods("POST")
	router.HandleFunc("/api/admin/canvas/freeze", handlers.HandleAdminFreeze).Methods("POST")
	router.HandleFunc("/api/admin/canvas/resize", handlers.HandleAdminResize).Methods("POST")

	router.HandleFunc("/api/faculty", handlers.HandleFaculty).Methods("GET")
	router.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleInitCanvas(w, r, uint(height), uint(width))
	}).Methods("GET")
}
