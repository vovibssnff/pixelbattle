package domain

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
)

type UserRepository interface {
	RegisterUser(ctx context.Context, usr User) error
	UpdateUser(ctx context.Context, usr User) error
	UserExists(ctx context.Context, usrID string) bool
	GetUsr(ctx context.Context, usrID string) User
	GetUserWithHash(ctx context.Context, usrID string) (User, error)
	DelUsr(ctx context.Context, usrID string)
	CheckBanned(ctx context.Context, userid string) bool
	BanUser(ctx context.Context, userid string) error
	UnbanUser(ctx context.Context, userid string) error
	// Runtime admin grants (Mongo collection admin_grants, _id = canonical user id).
	GrantAdminRole(ctx context.Context, userid string) error
	RevokeAdminRole(ctx context.Context, userid string) error
	IsDynamicAdmin(ctx context.Context, userid string) bool
	ListUserIDs(ctx context.Context, limit int) ([]string, error)
	ListAdminUsers(ctx context.Context, limit int) ([]AdminUserInfo, error)
}

type UserService interface {
	CreateUser(id string, firstName, lastName, accessToken string) *User
	RegisterUser(ctx context.Context, usr User) error
	RegisterWithPassword(ctx context.Context, username, password, faculty string) (*User, error)
	LoginWithPassword(ctx context.Context, username, password string) (*User, error)
	UpdateUser(ctx context.Context, usr User) error
	UserExists(ctx context.Context, usrID string) bool
	GetUser(ctx context.Context, usrID string) User
	DeleteUser(ctx context.Context, usrID string)
	IsUserBanned(ctx context.Context, userid string) bool
	IsAdmin(id string) bool
	// IsEffectiveAdmin is true for static config admins, Mongo-granted admins, or both.
	IsEffectiveAdmin(ctx context.Context, id string) bool
	GrantAdminRole(ctx context.Context, userid string) error
	RevokeAdminRole(ctx context.Context, userid string) error
	ListUserIDs(ctx context.Context, limit int) ([]string, error)
	ListAdminUsers(ctx context.Context, limit int) ([]AdminUserInfo, error)
	BanUser(ctx context.Context, userid string) error
	UnbanUser(ctx context.Context, userid string) error
}

type CanvasRepository interface {
	WritePixel(ctx context.Context, x, y uint, pixelData []byte) error
	CheckInitialized(ctx context.Context) bool
	GetCanvas(ctx context.Context) (map[string][]string, error)
	GetLatestPixel(ctx context.Context, x, y uint) (RedisPixel, error)
	LoadHeatMap(ctx context.Context) (map[string]int64, error)
	// GetCanvasDimensions returns persisted logical size (0,0 if unknown). See ADR-003.
	GetCanvasDimensions(ctx context.Context) (width, height uint, err error)
	SetCanvasDimensions(ctx context.Context, width, height uint) error
}

type CanvasService interface {
	WritePixel(ctx context.Context, p *Pixel) error
	InitializeCanvas(ctx context.Context, height uint, width uint) error
	EnsureCanvasInitialized(ctx context.Context, height uint, width uint) error
	IsCanvasInitialized(ctx context.Context) bool
	GetCanvas(ctx context.Context, img *Image) error
	GetPixelInfo(ctx context.Context, x, y uint) (PixelInfo, error)
	GetPixelInfoCache(ctx context.Context) ([]PixelInfo, error)
	GetHeatMap(ctx context.Context) ([]HeatMapUnit, error)
	CreateImage(h, w uint) *Image
	// CanvasDimensions returns stored size or infers from canvas keys (ADR-003).
	CanvasDimensions(ctx context.Context) (width, height uint, err error)
	// ExpandCanvas grows the canvas (new cells white); shrink rejected.
	ExpandCanvas(ctx context.Context, width, height uint) error
}

type TimerRepository interface {
	SetTimer(ctx context.Context, userid string, delay int) error
	CheckTime(ctx context.Context, userid string) (int64, error)
}

type TimerService interface {
	SetTimer(ctx context.Context, userid string) error
	CheckTime(ctx context.Context, userid string) (int64, error)
	// SetCooldownSeconds updates the Redis TTL used for the per-user placement timer (runtime admin).
	SetCooldownSeconds(ctx context.Context, sec int) error
	CooldownSeconds(ctx context.Context) (int, error)
}

type SessionService interface {
	GetSession(r *http.Request) (*sessions.Session, error)
	SaveSession(session *sessions.Session, w http.ResponseWriter, r *http.Request) error
	SetAuthenticated(session *sessions.Session, value string)
	SetFaculty(session *sessions.Session, value string)
	SetUserID(session *sessions.Session, id string)
	IsAuthenticated(session *sessions.Session) bool
	IsInProcess(session *sessions.Session) bool
	GetUserID(session *sessions.Session) string
	GetFaculty(session *sessions.Session) string
}
