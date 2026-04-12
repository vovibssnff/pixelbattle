package service

import (
	"net/http"
	"pb_backend/internal/core/domain"

	"github.com/gorilla/sessions"
)

type SessionService struct {
	store *sessions.CookieStore
}

func NewSessionService(store *sessions.CookieStore) *SessionService {
	return &SessionService{
		store: store,
	}
}

// GetSession retrieves the session for the given request.
func (s *SessionService) GetSession(r *http.Request) (*sessions.Session, error) {
	return s.store.Get(r, "user-session")
}

// SaveSession saves the session and writes the response.
func (s *SessionService) SaveSession(session *sessions.Session, w http.ResponseWriter, r *http.Request) error {
	return session.Save(r, w)
}

// SetAuthenticated sets the session's "Authenticated" value.
func (s *SessionService) SetAuthenticated(session *sessions.Session, value string) {
	session.Values["Authenticated"] = value
}

// SetFaculty sets the session's "Faculty" value.
func (s *SessionService) SetFaculty(session *sessions.Session, value string) {
	session.Values["Faculty"] = value
}

// SetUserID sets the session's user id (VK "vk_*" or local username).
func (s *SessionService) SetUserID(session *sessions.Session, id string) {
	session.Values["ID"] = id
}

// IsAuthenticated checks if the session is authenticated.
func (s *SessionService) IsAuthenticated(session *sessions.Session) bool {
	return session.Values["Authenticated"] == "true"
}

// IsInProcess checks if the session is in the authentication process.
func (s *SessionService) IsInProcess(session *sessions.Session) bool {
	return session.Values["Authenticated"] == "in_process"
}

// GetUserID retrieves the user id from the session.
// Accepts string ids, or legacy int session values (older VK-only builds).
func (s *SessionService) GetUserID(session *sessions.Session) string {
	if id, ok := session.Values["ID"].(string); ok {
		return id
	}
	if n, ok := session.Values["ID"].(int); ok {
		return domain.VKUserID(n)
	}
	if f, ok := session.Values["ID"].(float64); ok {
		return domain.VKUserID(int(f))
	}
	return ""
}

// GetFaculty retrieves the faculty from the session.
func (s *SessionService) GetFaculty(session *sessions.Session) string {
	if faculty, ok := session.Values["Faculty"].(string); ok {
		return faculty
	}
	return ""
}
