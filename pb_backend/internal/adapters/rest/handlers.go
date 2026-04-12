package rest

import (
	"encoding/json"
	"io"
	"net/http"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/core/service"
	"pb_backend/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type RestHandlers struct {
	sessionService domain.SessionService
	vkAuthProvider vk.VKAuthProvider
	canvasService  domain.CanvasService
	userService    domain.UserService
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

// HandleVKLogin is the VK OAuth callback (GET with query payload).
func (h *RestHandlers) HandleVKLogin(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionService.GetSession(r)

	vkUsr, accessToken := h.vkAuthProvider.Register(r)

	logrus.Info("VK login request from ", vkUsr.FirstName, vkUsr.LastName, vkUsr.ID)

	if !h.vkAuthProvider.ValidVkUser(vkUsr, accessToken) {
		service.RecordLoginAttempt("vk", "fail")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	vkID := domain.VKUserID(vkUsr.ID)

	if !h.userService.UserExists(r.Context(), vkID) {
		h.sessionService.SetAuthenticated(session, "in_process")
		h.sessionService.SetUserID(session, vkID)

		u := h.userService.CreateUser(vkID, vkUsr.FirstName, vkUsr.LastName, accessToken)

		if err := h.userService.RegisterUser(r.Context(), *u); err != nil {
			logrus.Error(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		h.sessionService.SaveSession(session, w, r)
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)

	} else if h.userService.GetUser(r.Context(), vkID).Faculty == "" || h.sessionService.IsInProcess(session) {
		h.sessionService.SetUserID(session, vkID)
		h.sessionService.SetAuthenticated(session, "in_process")
		h.sessionService.SaveSession(session, w, r)
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)

	} else {
		h.sessionService.SetUserID(session, vkID)
		h.sessionService.SetAuthenticated(session, "true")
		usr := h.userService.GetUser(r.Context(), h.sessionService.GetUserID(session))
		h.sessionService.SetFaculty(session, usr.Faculty)
		h.sessionService.SaveSession(session, w, r)
		http.Redirect(w, r, "/main", http.StatusSeeOther)
	}
	service.RecordLoginAttempt("vk", "success")
}

type passwordRegisterBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Faculty  string `json:"faculty"`
}

type passwordLoginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminUserBody struct {
	UserID string `json:"user_id"`
}

// HandlePasswordRegister creates a local username/password user (POST).
func (h *RestHandlers) HandlePasswordRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username, password, faculty := parseRegisterPayload(r)
	if username == "" || password == "" || faculty == "" {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}
	usr, err := h.userService.RegisterWithPassword(r.Context(), username, password, faculty)
	if err != nil {
		logrus.Warn("register: ", err)
		service.RecordLoginAttempt("register", "fail")
		http.Error(w, "Could not register", http.StatusBadRequest)
		return
	}
	service.RecordLoginAttempt("register", "success")
	session, _ := h.sessionService.GetSession(r)
	h.sessionService.SetUserID(session, usr.ID)
	h.sessionService.SetAuthenticated(session, "true")
	h.sessionService.SetFaculty(session, usr.Faculty)
	if err := h.sessionService.SaveSession(session, w, r); err != nil {
		logrus.Error(err)
		service.IncrementSessionErrors()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	service.IncrementOverallRegistrations()
	http.Redirect(w, r, "/main", http.StatusSeeOther)
}

// HandlePasswordLogin logs in a local user (POST).
func (h *RestHandlers) HandlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username, password := parseLoginPayload(r)
	if username == "" || password == "" {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}
	usr, err := h.userService.LoginWithPassword(r.Context(), username, password)
	if err != nil {
		logrus.Warn("login: ", err)
		service.RecordLoginAttempt("password", "fail")
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	service.RecordLoginAttempt("password", "success")
	session, _ := h.sessionService.GetSession(r)
	h.sessionService.SetUserID(session, usr.ID)
	if usr.Faculty == "" {
		h.sessionService.SetAuthenticated(session, "in_process")
	} else {
		h.sessionService.SetAuthenticated(session, "true")
		h.sessionService.SetFaculty(session, usr.Faculty)
	}
	if err := h.sessionService.SaveSession(session, w, r); err != nil {
		logrus.Error(err)
		service.IncrementSessionErrors()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if usr.Faculty == "" {
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/main", http.StatusSeeOther)
	}
}

func parseRegisterPayload(r *http.Request) (username, password, faculty string) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var b passwordRegisterBody
		data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(data, &b)
		return strings.TrimSpace(b.Username), b.Password, strings.TrimSpace(b.Faculty)
	}
	_ = r.ParseForm()
	return r.FormValue("username"), r.FormValue("password"), r.FormValue("faculty")
}

func parseLoginPayload(r *http.Request) (username, password string) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var b passwordLoginBody
		data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(data, &b)
		return strings.TrimSpace(b.Username), b.Password
	}
	_ = r.ParseForm()
	return r.FormValue("username"), r.FormValue("password")
}

func parseAdminUserPayload(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var b adminUserBody
		data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(data, &b)
		return resolveBanTarget(strings.TrimSpace(b.UserID))
	}
	_ = r.ParseForm()
	return resolveBanTarget(strings.TrimSpace(r.FormValue("user_id")))
}

// resolveBanTarget accepts "vk_123", "username", or bare numeric VK id "123".
func resolveBanTarget(s string) string {
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "vk_") {
		return s
	}
	if _, err := strconv.Atoi(s); err == nil {
		return domain.VKUserID(mustAtoi(s))
	}
	return utils.NormalizeUsername(s)
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// HandleBan bans a user (admin only).
func (h *RestHandlers) HandleBan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, _ := h.sessionService.GetSession(r)
	if !h.userService.IsAdmin(h.sessionService.GetUserID(session)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	target := parseAdminUserPayload(r)
	if target == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}
	if err := h.userService.BanUser(r.Context(), target); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleUnban removes a ban (admin only).
func (h *RestHandlers) HandleUnban(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, _ := h.sessionService.GetSession(r)
	if !h.userService.IsAdmin(h.sessionService.GetUserID(session)) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	target := parseAdminUserPayload(r)
	if target == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}
	if err := h.userService.UnbanUser(r.Context(), target); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RestHandlers) HandleFaculty(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionService.GetSession(r)
	if !h.sessionService.IsInProcess(session) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	faculty := r.URL.Query().Get("faculty")
	if !utils.IsValidFaculty(faculty) {
		http.Error(w, "Invalid faculty", http.StatusBadRequest)
		return
	}

	usr := h.userService.GetUser(r.Context(), h.sessionService.GetUserID(session))
	usr.Faculty = faculty
	if err := h.userService.UpdateUser(r.Context(), usr); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h.sessionService.SetAuthenticated(session, "true")
	h.sessionService.SetFaculty(session, faculty)
	h.sessionService.SaveSession(session, w, r)
	service.IncrementOverallRegistrations()
	http.Redirect(w, r, "/main", http.StatusSeeOther)
}

func (h *RestHandlers) HandleInitCanvas(w http.ResponseWriter, r *http.Request, height, width uint) {
	start := time.Now()
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
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	if h.userService.IsAdmin(h.sessionService.GetUserID(session)) {
		w.Header().Set("Is-God", "true")
	}
	w.Write(b)
	service.ObserveCanvasInitDuration(start)
}
