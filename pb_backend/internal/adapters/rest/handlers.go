package rest

import (
	"net/http"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/core/service"
	"pb_backend/internal/utils"
	"strconv"

	"github.com/sirupsen/logrus"
)

type RestHandlers struct {
	sessionService domain.SessionService
	vkAuthProvider vk.VKAuthProvider
	canvasService  domain.CanvasService
	userService    domain.UserService
	// imgService     *domain.ImageService
}

func NewRestHandlers(sessionService domain.SessionService, vkAuthProvider vk.VKAuthProvider, canvasService domain.CanvasService,
	userService domain.UserService) *RestHandlers {
	return &RestHandlers{
		sessionService: sessionService,
		canvasService:  canvasService,
		userService:    userService,
		vkAuthProvider: vkAuthProvider,
	}
}

func (h *RestHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionService.GetSession(r)

	vkUsr, accessToken := h.vkAuthProvider.Register(r)

	h.sessionService.SetUserID(session, vkUsr.ID)

	logrus.Info("Login request from ", vkUsr.FirstName, vkUsr.LastName, vkUsr.ID)

	if !h.vkAuthProvider.ValidVkUser(vkUsr, accessToken) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !h.userService.UserExists(r.Context(), vkUsr.ID) {
		h.sessionService.SetAuthenticated(session, "in_process")

		redisUser := h.userService.CreateUser(vkUsr.ID, vkUsr.FirstName, vkUsr.LastName, accessToken)

		if err := h.userService.RegisterUser(r.Context(), *redisUser); err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h.sessionService.SaveSession(session, w, r)
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)

	} else if h.userService.GetUser(r.Context(), vkUsr.ID).Faculty == "" || h.sessionService.IsInProcess(session) {
		h.sessionService.SetAuthenticated(session, "in_process")
		h.sessionService.SaveSession(session, w, r)
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)

	} else {
		h.sessionService.SetAuthenticated(session, "true")
		usr := h.userService.GetUser(r.Context(), h.sessionService.GetUserID(session))
		h.sessionService.SetFaculty(session, usr.Faculty)
		h.sessionService.SaveSession(session, w, r)
		http.Redirect(w, r, "/main", http.StatusSeeOther)
	}
}

func (h *RestHandlers) HandleFaculty(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionService.GetSession(r)
	if !h.sessionService.IsInProcess(session) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	faculty := r.URL.Query().Get("faculty")

	usr := h.userService.GetUser(r.Context(), h.sessionService.GetUserID(session))
	usr.Faculty = faculty
	if err := h.userService.RegisterUser(r.Context(), usr); err != nil {
		logrus.Error(err)
	}
	h.sessionService.SetAuthenticated(session, "true")
	h.sessionService.SetFaculty(session, faculty)
	h.sessionService.SaveSession(session, w, r)
	service.IncrementOverallRegistrations()
	http.Redirect(w, r, "/main", http.StatusSeeOther)
}

func (h *RestHandlers) HandleInitCanvas(w http.ResponseWriter, r *http.Request, height, width uint) {
	session, _ := h.sessionService.GetSession(r)
	if !h.sessionService.IsAuthenticated(session) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	img := h.canvasService.CreateImage(height, width)
	h.canvasService.GetCanvas(r.Context(), img)
	b, err := utils.GetImageBytes(img)
	if err != nil {
		logrus.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	if h.userService.IsAdmin(h.sessionService.GetUserID(session)) {
		w.Header().Set("Is-God", "true")
	}
	w.Write(b)
}
