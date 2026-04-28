package main

import (
	"context"
	"net/http"
	"pb_backend/internal/adapters/mongo"
	mongo_repo "pb_backend/internal/adapters/mongo/repository"
	"pb_backend/internal/adapters/redis"
	redis_repo "pb_backend/internal/adapters/redis/repository"
	"pb_backend/internal/adapters/rest"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/adapters/websockets"
	"pb_backend/internal/core/service"
	"pb_backend/internal/utils"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Info("Loading config")

	config, err := utils.LoadConfig("app.env")
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	level, err := logrus.ParseLevel(strings.ToLower(config.LogLevel))
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	canvasDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
	timerDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisTimer)

	mongoUserDatabase, err := mongo.NewMongoConnection(config.MongoURI, "pixelbattle")
	if err != nil {
		logrus.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	canvasRepo := redis_repo.NewCanvasRepository(canvasDatabase)
	mongoUsrRepo := mongo_repo.NewUserRepository(mongoUserDatabase)

	timerRepo := redis_repo.NewTimerRepo(timerDatabase)

	var authKey []byte
	sk := strings.TrimSpace(config.SessionKey)
	if len(sk) >= 32 {
		authKey = []byte(sk)
	} else {
		if len(sk) > 0 {
			logrus.Warn("SESSION_KEY is shorter than 32 bytes; generating ephemeral session key (sessions invalid after restart)")
		} else {
			logrus.Warn("SESSION_KEY empty; generating ephemeral session key (sessions invalid after restart)")
		}
		k := securecookie.GenerateRandomKey(32)
		if k == nil {
			logrus.Fatal("failed to generate session key")
		}
		authKey = k
	}

	sessionStore := sessions.NewCookieStore(authKey)
	sessionStore.Options.MaxAge = 1800
	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = config.SessionSecure
	sessionStore.Options.SameSite = http.SameSiteLaxMode

	canvasService := service.NewCanvasService(*canvasRepo)
	usrService := service.NewUserService(mongoUsrRepo, config.AdminIDs, config.AdminUsernames)
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
	router.Handle("/metrics", service.MetricsHandler())

	rest.StartRestServer(sessionService, *vkAuthProvider, canvasService, usrService,
		config.CanvasHeight, config.CanvasWidth, router)

	if config.WSAllowAnonymous {
		logrus.Warn("WS_ALLOW_ANONYMOUS is enabled: /ws accepts connections without auth (for benchmarks only)")
	}
	websockets.StartWebSocketServer(sessionService, canvasService, timerService, usrService, router,
		config.CanvasHeight, config.CanvasWidth, config.WSAllowAnonymous)

	logrus.Info("Starting server on port 8080")
	handler := service.InstrumentHandler(router)
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logrus.Fatal("Failed to start server: ", err)
	}
}
