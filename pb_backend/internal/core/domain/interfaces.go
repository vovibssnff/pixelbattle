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
	BanUser(ctx context.Context, userid string) error
	UnbanUser(ctx context.Context, userid string) error
}

type CanvasRepository interface {
	WritePixel(ctx context.Context, x, y uint, pixelData []byte) error
	CheckInitialized(ctx context.Context) bool
	GetCanvas(ctx context.Context) (map[string][]string, error)
	LoadHeatMap(ctx context.Context) (map[string]int64, error)
}

type CanvasService interface {
	WritePixel(ctx context.Context, p *Pixel) error
	InitializeCanvas(ctx context.Context, height uint, width uint) error
	IsCanvasInitialized(ctx context.Context) bool
	GetCanvas(ctx context.Context, img *Image) error
	GetHeatMap(ctx context.Context) ([]HeatMapUnit, error)
	CreateImage(h, w uint) *Image
}

type TimerRepository interface {
	SetTimer(ctx context.Context, userid string, delay int) error
	CheckTime(ctx context.Context, userid string) (int64, error)
}

type TimerService interface {
	SetTimer(ctx context.Context, userid string) error
	CheckTime(ctx context.Context, userid string) (int64, error)
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
