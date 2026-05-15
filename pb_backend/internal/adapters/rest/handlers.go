package rest

import (
	"context"
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
	timerService   domain.TimerService
	snapshotter    *service.CanvasSnapshotter
	wsHub          WSResizeNotifier
	// adminAPIToken: when non-empty, requests with matching X-Admin-Token may call admin APIs without a session.
	adminAPIToken string
}

func NewRestHandlers(sessionService domain.SessionService, vkAuthProvider vk.VKAuthProvider, canvasService domain.CanvasService,
	userService domain.UserService, timerService domain.TimerService, adminAPIToken string, snapshotter *service.CanvasSnapshotter, wsHub WSResizeNotifier) *RestHandlers {
	return &RestHandlers{
		sessionService: sessionService,
		canvasService:  canvasService,
		userService:    userService,
		vkAuthProvider: vkAuthProvider,
		timerService:   timerService,
		snapshotter:    snapshotter,
		wsHub:          wsHub,
		adminAPIToken:  strings.TrimSpace(adminAPIToken),
	}
}

func (h *RestHandlers) effectiveCanvasDims(ctx context.Context, fbW, fbH uint) (uint, uint) {
	w, he, err := h.canvasService.CanvasDimensions(ctx)
	if err != nil {
		logrus.Debugf("canvas dimensions: %v", err)
		return fbW, fbH
	}
	if w == 0 || he == 0 {
		return fbW, fbH
	}
	return w, he
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

		if err := h.sessionService.SaveSession(session, w, r); err != nil {
			logrus.Error(err)
			service.IncrementSessionErrors()
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)

	} else if h.userService.GetUser(r.Context(), vkID).Faculty == "" || h.sessionService.IsInProcess(session) {
		h.sessionService.SetUserID(session, vkID)
		h.sessionService.SetAuthenticated(session, "in_process")
		if err := h.sessionService.SaveSession(session, w, r); err != nil {
			logrus.Error(err)
			service.IncrementSessionErrors()
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)

	} else {
		h.sessionService.SetUserID(session, vkID)
		h.sessionService.SetAuthenticated(session, "true")
		usr := h.userService.GetUser(r.Context(), h.sessionService.GetUserID(session))
		h.sessionService.SetFaculty(session, usr.Faculty)
		if err := h.sessionService.SaveSession(session, w, r); err != nil {
			logrus.Error(err)
			service.IncrementSessionErrors()
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
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
	if err := h.sessionService.SaveSession(session, w, r); err != nil {
		logrus.Error(err)
		service.IncrementSessionErrors()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
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

	cw, ch := h.effectiveCanvasDims(r.Context(), width, height)
	img := h.canvasService.CreateImage(ch, cw)
	if err := h.canvasService.GetCanvas(r.Context(), img); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	b, err := utils.GetImageBytes(img)
	if err != nil {
		logrus.Error(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	if h.userService.IsEffectiveAdmin(r.Context(), h.sessionService.GetUserID(session)) {
		w.Header().Set("Is-God", "true")
	}
	if _, err := w.Write(b); err != nil {
		logrus.Error(err)
		return
	}
	service.ObserveCanvasInitDuration(start)
}

// HandleCanvasPNG serves a cached PNG from the snapshotter when available (plan §11.3),
// with ETag / 304 and X-Snapshot-Ms for WS replay_after_ms; falls back to a live render.
func (h *RestHandlers) HandleCanvasPNG(w http.ResponseWriter, r *http.Request, height, width uint) {
	start := time.Now()
	session, _ := h.sessionService.GetSession(r)
	if !h.sessionService.IsAuthenticated(session) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var b []byte
	var etag string
	var unixMs int64

	if h.snapshotter != nil {
		if png, et, ms, ok := h.snapshotter.Get(); ok && len(png) > 0 {
			b, etag, unixMs = png, et, ms
		}
	}

	if len(b) == 0 {
		cw, ch := h.effectiveCanvasDims(r.Context(), width, height)
		img := h.canvasService.CreateImage(ch, cw)
		if err := h.canvasService.GetCanvas(r.Context(), img); err != nil {
			logrus.Error(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		var err error
		b, err = utils.GetImageBytes(img)
		if err != nil {
			logrus.Error(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		unixMs = time.Now().UnixMilli()
		etag = ""
	}

	if etag != "" {
		if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
			w.Header().Set("ETag", etag)
			w.Header().Set("X-Snapshot-Ms", strconv.FormatInt(unixMs, 10))
			w.Header().Set("Cache-Control", "public, max-age=2")
			w.WriteHeader(http.StatusNotModified)
			service.ObserveCanvasInitDuration(start)
			return
		}
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=2")
	w.Header().Set("X-Snapshot-Ms", strconv.FormatInt(unixMs, 10))
	if etag != "" {
		w.Header().Set("ETag", etag)
	}
	if h.userService.IsEffectiveAdmin(r.Context(), h.sessionService.GetUserID(session)) {
		w.Header().Set("Is-God", "true")
	}
	if _, err := w.Write(b); err != nil {
		logrus.Error(err)
		return
	}
	service.ObserveCanvasInitDuration(start)
}
