package main

import (
	"context"
	"net/http"
	"pb_backend/internal/adapters/redis"
	"pb_backend/internal/adapters/redis/repository"
	"pb_backend/internal/adapters/rest"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/adapters/websockets"
	"pb_backend/internal/core/service"
	"pb_backend/internal/utils"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Loading config")

	config, err := utils.LoadConfig("app.env")
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	canvasDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
	usrDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisUsers)
	banListDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisBanned)
	timerDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisTimer)

	canvasRepo := repository.NewCanvasRepository(canvasDatabase)
	usrRepo := repository.NewUserRepository(usrDatabase, banListDatabase)

	timerRepo := repository.NewTimerRepo(timerDatabase)

	sessionStore := sessions.NewCookieStore([]byte(string(securecookie.GenerateRandomKey(32))))
	sessionStore.Options.MaxAge = 1800

	canvasService := service.NewCanvasService(*canvasRepo)
	usrService := service.NewUserService(*usrRepo, config.AdminIDs)
	timerService := service.NewTimerService(*timerRepo, 3)
	sessionService := service.NewSessionService(sessionStore)
	vkAuthProvider := vk.NewVKAuthProvider(config.ServiceToken, config.APIVersion)

	if !canvasService.IsCanvasInitialized(context.Background()) {
		logrus.Info("Initializing canvas with white pixels")
		if err := canvasService.InitializeCanvas(context.Background(), uint(config.CanvasHeight), uint(config.CanvasWidth)); err != nil {
			logrus.Fatalf("Failed to initialize canvas: %v", err)
		}
	}

	router := mux.NewRouter()
	rest.StartRestServer(sessionService, *vkAuthProvider, canvasService, usrService, timerService,
		config.CanvasHeight, config.CanvasWidth, router)

	websockets.StartWebSocketServer(sessionService, canvasService, timerService, usrService, router)

	logrus.Info("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		logrus.Fatal("Failed to start server: ", err)
	}

}
