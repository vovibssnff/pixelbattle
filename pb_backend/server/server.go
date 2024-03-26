package server

import (
	"github.com/gorilla/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"net/http"
	"pb_backend/models"
	"pb_backend/service"
	"strconv"
)

type Server struct {
	imgService     *service.ImageService
	historyService *redis.Client
	userService    *redis.Client
	serviceToken   string
	apiVer         string
	store          *sessions.CookieStore
}

func NewServer(imgService *service.ImageService, redisHistoryService *redis.Client, sessionStore *sessions.CookieStore, redisUserService *redis.Client, serviceToken string, apiVer string) *Server {
	return &Server{
		imgService:     imgService,
		historyService: redisHistoryService,
		userService:    redisUserService,
		serviceToken:   serviceToken,
		apiVer:         apiVer,
		store:          sessionStore,
	}
}

func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, "user-session")
	logrus.Info(session.Values)
	vk_resp := service.ToVKResponse(r.URL.Query())
	req := service.NewAccessReq(s.apiVer, vk_resp.Token, s.serviceToken, vk_resp.UUID)
	accessToken := service.SilentToAccess(*req)
	if accessToken == "" {
		//TODO handle
	}
	session.Values["ID"] = vk_resp.User.ID
	logrus.Info("Login request from ", vk_resp.User.FirstName, vk_resp.User.LastName)
	if !service.UserExists(s.userService, vk_resp.User.ID) {
		session.Values["Authenticated"] = "in_process"
		redisUser := models.NewUser(vk_resp.User.ID, vk_resp.User.FirstName, vk_resp.User.LastName, accessToken)
		if err := service.RegisterUser(s.userService, *redisUser); err != nil {
			logrus.Error(err)
		}
		logrus.Info(session.Values)
		session.Save(r, w)
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)
	} else if service.GetUsr(s.userService, vk_resp.User.ID).Faculty == "" || session.Values["Authenticated"] == "in_process" {
		session.Values["Authenticated"] = "in_process"
		logrus.Info(session.Values)
		session.Save(r, w)
		http.Redirect(w, r, "/faculty", http.StatusSeeOther)
	} else {
		logrus.Info("New login")
		session.Values["Authenticated"] = "true"
		usr := service.GetUsr(s.userService, vk_resp.User.ID)
		session.Values["Faculty"] = usr.Faculty
		logrus.Info(session.Values)
		session.Save(r, w)
		http.Redirect(w, r, "/main", http.StatusSeeOther)
	}
}

func (s *Server) HandleFaculty(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, "user-session")
	logrus.Info(session.Values)
	if session.Values["Authenticated"] != "in_process" {
		logrus.Warn("Unauthorized attempt to reach /faculty")
		logrus.Info(session.Values)
		http.Redirect(w, r, "login", http.StatusSeeOther)
		return
		// http.RedirectHandler("/login", http.StatusSeeOther)
	}
	facResp := service.ToFaculty(r)
	usr := service.GetUsr(s.userService, session.Values["ID"].(int))
	usr.Faculty = facResp.Faculty
	if err := service.RegisterUser(s.userService, usr); err != nil {
		logrus.Error(err)
	}
	session.Values["Authenticated"] = "true"
	session.Values["Faculty"] = facResp.Faculty
	logrus.Info(session.Values)
	session.Save(r, w)
	logrus.Info("New register")
}

func (server *Server) HandleInitCanvas(writer http.ResponseWriter, r *http.Request, h, w uint) {
	session, _ := server.store.Get(r, "user-session")
	if session.Values["Authenticated"] != "true" {
		logrus.Warn("Unauthorized attempt to reach /init_canvas")
		http.Redirect(writer, r, "login", http.StatusSeeOther)
		return
	}
	// logrus.Info("Received an init request")
	img := service.NewImage(h, w)
	service.GetCanvas(server.historyService, img)
	b := server.imgService.GetImageBytes(img)
	writer.Header().Set("Content-Length", strconv.Itoa(len(b)))
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Cache-Control", "no-cache, no-store")
	// logrus.Info("Canvas bytes sent")
	writer.Write(b)
}

func (server *Server) WhiteCanvasInit(n, m uint) {
	if !service.CheckInitialized(server.historyService) {
		logrus.Info("Initialization is needed. Initializing...")
		service.InitializeCanvas(server.historyService, n, m)
		logrus.Info("Initialization successful")

	}
}
