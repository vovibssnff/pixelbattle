package server

import (
	"encoding/json"
	"net/http"
	"pb_backend/models"
	"pb_backend/service"
	"strconv"
	"github.com/gorilla/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Server struct {
	imgService     *service.ImageService
	historyService *redis.Client
	userService    *redis.Client
	serviceToken   string
	apiVer         string
	store          *sessions.CookieStore
	admIds         []int
}

func NewServer(imgService *service.ImageService, redisHistoryService *redis.Client, sessionStore *sessions.CookieStore, redisUserService *redis.Client, serviceToken string, apiVer string, ids []int) *Server {
	return &Server{
		imgService:     imgService,
		historyService: redisHistoryService,
		userService:    redisUserService,
		serviceToken:   serviceToken,
		apiVer:         apiVer,
		store:          sessionStore,
		admIds:         ids,
	}
}

func (s *Server) isAdmin(id int) bool {
	for _, admId := range s.admIds {
		if id == admId {
			return true
		}
	}
	return false
}

func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request, serviceToken string) {
	session, _ := s.store.Get(r, "user-session")
	var usr service.VKUserData
	err := json.NewDecoder(r.Body).Decode(&usr)
	if err != nil {
		logrus.Error(err)
		return
	}

	// vk_resp := service.ToVKResponse(r.URL.Query())
	// req := service.NewAccessReq(s.apiVer, vk_resp.Token, s.serviceToken, vk_resp.UUID)
	// accessToken := service.SilentToAccess(*req)
	// if accessToken == "" {
	// 	//TODO handle
	// }
	session.Values["ID"] = usr.UserID

	logrus.Info("Login request from ", usr.UserID)

	if usr.UserID == 0 ||
		usr.AccessToken == "" {
		// service.IsBanned(*service.NewCheckReq(usr.UserID, serviceToken, s.apiVer)) {
		http.Error(w, "lol", http.StatusBadRequest)
		return
	}

	status := ""

	if !service.UserExists(s.userService, usr.UserID) {
		session.Values["Authenticated"] = "in_process"
		usrInfo, err := service.GetUsrInfo(*service.NewNameReq(usr.UserID, usr.AccessToken, s.apiVer))
		if err != nil {
			logrus.Error(err)
			http.Error(w, "lol", http.StatusTeapot)
			return
		}
		redisUser := models.NewUser(usr.UserID, usrInfo.FirstName, usrInfo.LastName, usr.AccessToken, usr.RefreshToken, usr.IDToken, usr.DeviceID)
		if err := service.RegisterUser(s.userService, *redisUser); err != nil {
			logrus.Error(err)
		}
		session.Save(r, w)
		logrus.Info("not exists: ", session.Values)
		status = "redirect_to_faculty"

	} else if service.GetUsr(s.userService, usr.UserID).Faculty == "" || session.Values["Authenticated"] == "in_process" {
		session.Values["Authenticated"] = "in_process"
		session.Save(r, w)
		logrus.Info("exists/in_process: ", session.Values)
		status = "redirect_to_faculty"
	} else {
		session.Values["Authenticated"] = "true"
		usr := service.GetUsr(s.userService, usr.UserID)
		session.Values["Faculty"] = usr.Faculty
		session.Save(r, w)
		logrus.Info("default login: ", session.Values)
		status = "redirect_to_main"
	}
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"status": status}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleFaculty(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, "user-session")
	logrus.Info("api/faculty: ", session.Values)
	if session.Values["Authenticated"] != "in_process" {
		logrus.Warn("Unauthorized attempt to reach api/faculty")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	faculty := r.URL.Query().Get("faculty")
	usr := service.GetUsr(s.userService, session.Values["ID"].(int))
	usr.Faculty = faculty
	if err := service.RegisterUser(s.userService, usr); err != nil {
		logrus.Error(err)
	}
	session.Values["Authenticated"] = "true"
	session.Values["Faculty"] = faculty
	session.Save(r, w)
	service.IncrementOverallRegistrations()
	logrus.Info("New register")
	http.Redirect(w, r, "/main", http.StatusSeeOther)
}

func (server *Server) HandleInitCanvas(writer http.ResponseWriter, r *http.Request, h, w uint) {
	session, _ := server.store.Get(r, "user-session")
	if session.Values["Authenticated"] != "true" {
		logrus.Warn("Unauthorized attempt to reach /init_canvas")
		http.Redirect(writer, r, "/login", http.StatusSeeOther)
		return
	}
	img := service.NewImage(h, w)
	service.GetCanvas(server.historyService, img)
	b := server.imgService.GetImageBytes(img)
	writer.Header().Set("Content-Length", strconv.Itoa(len(b)))
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("Cache-Control", "no-cache, no-store")
	if server.isAdmin(session.Values["ID"].(int)) {
		writer.Header().Set("Is-God", "true")
	}
	writer.Write(b)
}

func (server *Server) WhiteCanvasInit(n, m uint) {
	if !service.CheckInitialized(server.historyService) {
		logrus.Info("Initialization is needed. Initializing...")
		service.InitializeCanvas(server.historyService, n, m)
		logrus.Info("Initialization successful")
	}
}
