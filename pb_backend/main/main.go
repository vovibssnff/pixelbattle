package main

import (
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"pb_backend/server"
	"pb_backend/service"
	"pb_backend/websockets"
	"strconv"
	"strings"
)

func main() {
	redis_db := os.Getenv("REDIS_ADDR")
	redis_psw := os.Getenv("REDIS_PSW")
	redis_history, err := strconv.Atoi(os.Getenv("REDIS_HISTORY"))
	redis_timer, err := strconv.Atoi(os.Getenv("REDIS_TIMER"))
	redis_users, err := strconv.Atoi(os.Getenv("REDIS_USERS"))
	redis_banned, err := strconv.Atoi(os.Getenv("REDIS_BANNED"))

	canvas_height, err := strconv.Atoi(os.Getenv("CANVAS_HEIGHT"))
	canvas_width, err := strconv.Atoi(os.Getenv("CANVAS_WIDTH"))

	serviceToken := os.Getenv("SERVICE_TOKEN")
	apiVer := os.Getenv("API_VERSION")

	adminIDsString := strings.Trim(os.Getenv("ADMIN_IDS"), "[]")
	idStrs := strings.Split(adminIDsString, ",")
	var ids []int

	for _, idStr := range idStrs {
		id, _ := strconv.Atoi(strings.TrimSpace(idStr))
		ids = append(ids, id)
	}

	if os.Getenv("SESSION_KEY") == "" {
		os.Setenv("SESSION_KEY", string(securecookie.GenerateRandomKey(32)))
	}
	sessionKey := os.Getenv("SESSION_KEY")

	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Starting service")

	redisHistoryService := service.NewRedisClient(redis_db, redis_psw, redis_history)
	redisUserService := service.NewRedisClient(redis_db, redis_psw, redis_users)
	redisBannedService := service.NewRedisClient(redis_db, redis_psw, redis_banned)

	sessionStore := sessions.NewCookieStore([]byte(sessionKey))
	sessionStore.Options.MaxAge = 1800
	//server
	ws := websockets.NewWebSocketServer(redisHistoryService, sessionStore, redisBannedService)
	go ws.Run()

	//client
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		redisTimerService := service.NewRedisClient(redis_db, redis_psw, redis_timer)
		websockets.ServeWs(ws, redisTimerService, w, r, ids, redisBannedService, redisUserService)
	})

	imgService := service.NewImageService()
	server := server.NewServer(imgService, redisHistoryService, sessionStore, redisUserService, serviceToken, apiVer, ids)

	server.WhiteCanvasInit(uint(canvas_height), uint(canvas_width))

	http.HandleFunc("/init_canvas", func(w http.ResponseWriter, r *http.Request) {
		server.HandleInitCanvas(w, r, uint(canvas_height), uint(canvas_width))
	})

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		server.HandleRegister(w, r, serviceToken)
	})

	http.HandleFunc("/api/faculty", func(w http.ResponseWriter, r *http.Request) {
		server.HandleFaculty(w, r)
	})

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
