package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"pb_backend/internal/core/service"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type adminJSONResponse struct {
	OK  bool   `json:"ok"`
	Msg string `json:"msg"`
}

func (h *RestHandlers) writeAdminJSON(w http.ResponseWriter, status int, ok bool, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(adminJSONResponse{OK: ok, Msg: msg})
}

// authorizeAdmin returns true when X-Admin-Token matches ADMIN_API_TOKEN or the session user is an effective admin.
func (h *RestHandlers) authorizeAdmin(w http.ResponseWriter, r *http.Request) bool {
	if h.adminAPIToken != "" && strings.TrimSpace(r.Header.Get("X-Admin-Token")) == h.adminAPIToken {
		return true
	}
	session, err := h.sessionService.GetSession(r)
	if err != nil {
		logrus.Error(err)
		service.IncrementSessionErrors()
		h.writeAdminJSON(w, http.StatusUnauthorized, false, "session error")
		return false
	}
	uid := h.sessionService.GetUserID(session)
	if uid == "" || !h.userService.IsEffectiveAdmin(r.Context(), uid) {
		h.writeAdminJSON(w, http.StatusForbidden, false, "forbidden")
		return false
	}
	return true
}

type adminUserIDBody struct {
	UserID string `json:"userid"`
}

func decodeAdminUserID(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var b adminUserIDBody
		data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(data, &b)
		return resolveBanTarget(strings.TrimSpace(b.UserID))
	}
	_ = r.ParseForm()
	if v := r.FormValue("userid"); v != "" {
		return resolveBanTarget(strings.TrimSpace(v))
	}
	return resolveBanTarget(strings.TrimSpace(r.FormValue("user_id")))
}

// HandleAdminBan POST /api/admin/users/ban
func (h *RestHandlers) HandleAdminBan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("ban", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("ban", "forbidden")
		return
	}
	target := decodeAdminUserID(r)
	if target == "" {
		service.IncrementAdminAction("ban", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "missing userid")
		return
	}
	if err := h.userService.BanUser(r.Context(), target); err != nil {
		logrus.Error(err)
		service.IncrementAdminAction("ban", "error")
		h.writeAdminJSON(w, http.StatusInternalServerError, false, err.Error())
		return
	}
	service.IncrementAdminAction("ban", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "banned")
}

// HandleAdminUnban POST /api/admin/users/unban
func (h *RestHandlers) HandleAdminUnban(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("unban", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("unban", "forbidden")
		return
	}
	target := decodeAdminUserID(r)
	if target == "" {
		service.IncrementAdminAction("unban", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "missing userid")
		return
	}
	if err := h.userService.UnbanUser(r.Context(), target); err != nil {
		logrus.Error(err)
		service.IncrementAdminAction("unban", "error")
		h.writeAdminJSON(w, http.StatusInternalServerError, false, err.Error())
		return
	}
	service.IncrementAdminAction("unban", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "unbanned")
}

// HandleAdminGrant POST /api/admin/users/grant_admin
func (h *RestHandlers) HandleAdminGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("grant_admin", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("grant_admin", "forbidden")
		return
	}
	target := decodeAdminUserID(r)
	if target == "" {
		service.IncrementAdminAction("grant_admin", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "missing userid")
		return
	}
	if err := h.userService.GrantAdminRole(r.Context(), target); err != nil {
		logrus.Error(err)
		service.IncrementAdminAction("grant_admin", "error")
		h.writeAdminJSON(w, http.StatusInternalServerError, false, err.Error())
		return
	}
	service.IncrementAdminAction("grant_admin", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "granted")
}

// HandleAdminRevoke POST /api/admin/users/revoke_admin
func (h *RestHandlers) HandleAdminRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("revoke_admin", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("revoke_admin", "forbidden")
		return
	}
	target := decodeAdminUserID(r)
	if target == "" {
		service.IncrementAdminAction("revoke_admin", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "missing userid")
		return
	}
	if err := h.userService.RevokeAdminRole(r.Context(), target); err != nil {
		logrus.Error(err)
		service.IncrementAdminAction("revoke_admin", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, err.Error())
		return
	}
	service.IncrementAdminAction("revoke_admin", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "revoked")
}

type adminTimerBody struct {
	Seconds int `json:"seconds"`
}

// HandleAdminTimerSet POST /api/admin/timer/set
func (h *RestHandlers) HandleAdminTimerSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("set_timer", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("set_timer", "forbidden")
		return
	}
	sec := 0
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var b adminTimerBody
		data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err == nil {
			_ = json.Unmarshal(data, &b)
			sec = b.Seconds
		}
	} else {
		_ = r.ParseForm()
		sec, _ = strconv.Atoi(strings.TrimSpace(r.FormValue("seconds")))
	}
	if sec <= 0 {
		service.IncrementAdminAction("set_timer", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "invalid seconds")
		return
	}
	if err := h.timerService.SetCooldownSeconds(sec); err != nil {
		service.IncrementAdminAction("set_timer", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, err.Error())
		return
	}
	service.IncrementAdminAction("set_timer", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "timer updated")
}

type adminFreezeBody struct {
	Frozen bool `json:"frozen"`
}

// HandleAdminFreeze POST /api/admin/canvas/freeze
func (h *RestHandlers) HandleAdminFreeze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("freeze", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("freeze", "forbidden")
		return
	}
	frozen := false
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		var b adminFreezeBody
		data, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(data, &b)
		frozen = b.Frozen
	} else {
		_ = r.ParseForm()
		frozen = strings.EqualFold(r.FormValue("frozen"), "true") || r.FormValue("frozen") == "1"
	}
	service.SetCanvasFrozen(frozen)
	service.IncrementAdminAction("freeze", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "freeze toggled")
}

type adminResizeBody struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`
}

// HandleAdminResize POST /api/admin/canvas/resize — expand-only; see ADR-003.
func (h *RestHandlers) HandleAdminResize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementAdminAction("resize", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("resize", "forbidden")
		return
	}
	var body adminResizeBody
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		service.IncrementAdminAction("resize", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "read body")
		return
	}
	if err := json.Unmarshal(data, &body); err != nil || body.Width == 0 || body.Height == 0 {
		service.IncrementAdminAction("resize", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, "invalid width/height")
		return
	}
	if err := h.canvasService.ExpandCanvas(r.Context(), body.Width, body.Height); err != nil {
		logrus.Error(err)
		service.IncrementAdminAction("resize", "error")
		h.writeAdminJSON(w, http.StatusBadRequest, false, err.Error())
		return
	}
	service.SetCanvasDimensionsGauge(body.Width, body.Height)
	payload, err := json.Marshal(map[string]any{
		"event":  "RESIZE",
		"width":  body.Width,
		"height": body.Height,
	})
	if err != nil {
		service.IncrementAdminAction("resize", "error")
		h.writeAdminJSON(w, http.StatusInternalServerError, false, "marshal")
		return
	}
	if h.wsHub != nil {
		h.wsHub.NotifyCanvasResize(body.Width, body.Height, payload)
	}
	service.IncrementAdminAction("resize", "ok")
	h.writeAdminJSON(w, http.StatusOK, true, "resized")
}

// HandleAdminUsers GET /api/admin/users
func (h *RestHandlers) HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		service.IncrementAdminAction("list_users", "error")
		h.writeAdminJSON(w, http.StatusMethodNotAllowed, false, "method not allowed")
		return
	}
	if !h.authorizeAdmin(w, r) {
		service.IncrementAdminAction("list_users", "forbidden")
		return
	}
	ids, err := h.userService.ListUserIDs(r.Context(), 500)
	if err != nil {
		logrus.Error(err)
		service.IncrementAdminAction("list_users", "error")
		h.writeAdminJSON(w, http.StatusInternalServerError, false, err.Error())
		return
	}
	service.IncrementAdminAction("list_users", "ok")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_ids": ids})
}
