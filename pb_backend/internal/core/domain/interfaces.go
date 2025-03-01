package domain

// type User struct {
// 	ID           int    `json: "id"`
// 	FirstName    string `json: "name"`
// 	LastName     string `json: "surname"`
// 	AccessToken  string `json: "access_token"`
// 	RefreshToken string `json: "refresh_token"`
// 	IDToken      string `json: "id_token"`
// 	DeviceID     string `json: "device_id"`
// 	Faculty      string `json: "faculty"`
// }

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
)

type UserRepository interface {
	RegisterUser(ctx context.Context, usr User) error
	UserExists(ctx context.Context, usrID int) bool
	GetUsr(ctx context.Context, usrID int) User
	DelUsr(ctx context.Context, usrID int)
	CheckBanned(ctx context.Context, userid int) bool
}

type UserService interface {
	CreateUser(id int, firstName, lastName, accessToken string) *User
	RegisterUser(ctx context.Context, usr User) error
	UserExists(ctx context.Context, usrID int) bool
	GetUser(ctx context.Context, usrID int) User
	DeleteUser(ctx context.Context, usrID int)
	IsUserBanned(ctx context.Context, userid int) bool
	IsAdmin(id int) bool
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
	SetTimer(ctx context.Context, userid int, delay int) error
	CheckTime(ctx context.Context, userid int) (int64, error)
}

type TimerService interface {
	SetTimer(ctx context.Context, userid int) error
	CheckTime(ctx context.Context, userid int) (int64, error)
}

type SessionService interface {
	GetSession(r *http.Request) (*sessions.Session, error)
	SaveSession(session *sessions.Session, w http.ResponseWriter, r *http.Request) error
	SetAuthenticated(session *sessions.Session, value string)
	SetFaculty(session *sessions.Session, value string)
	SetUserID(session *sessions.Session, id int)
	IsAuthenticated(session *sessions.Session) bool
	IsInProcess(session *sessions.Session) bool
	GetUserID(session *sessions.Session) int
	GetFaculty(session *sessions.Session) string
}
