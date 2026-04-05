package main

import (
	"context"
	"net/http"
	"pb_backend/internal/adapters/mongo"
	mongo_repo "pb_backend/internal/adapters/mongo/repository"
	"pb_backend/internal/adapters/postgres"
	postgres_repo "pb_backend/internal/adapters/postgres/repository"
	"pb_backend/internal/adapters/redis"
	redis_repo "pb_backend/internal/adapters/redis/repository"
	"pb_backend/internal/adapters/rest"
	sqlite_adapter "pb_backend/internal/adapters/sqlite"
	sqlite_repo "pb_backend/internal/adapters/sqlite/repository"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/adapters/websockets"
	"pb_backend/internal/core/domain"
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

	// Initialize canvas repository based on storage type
	storageType := config.StorageType
	if storageType == "" {
		storageType = "redis" // Default to redis for backward compatibility
	}

	var canvasRepo domain.CanvasRepository

	switch storageType {
	case "postgres":
		logrus.Info("Initializing PostgreSQL storage")
		postgresDB, err := postgres.NewPostgresConnection(
			config.PostgresHost,
			config.PostgresPort,
			config.PostgresUser,
			config.PostgresPassword,
			config.PostgresDB,
		)
		if err != nil {
			logrus.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		if err := postgres.InitializeSchema(postgresDB); err != nil {
			logrus.Fatalf("Failed to initialize PostgreSQL schema: %v", err)
		}
		canvasRepo = postgres_repo.NewCanvasRepository(postgresDB)

	case "sqlite":
		logrus.Info("Initializing SQLite storage")
		sqlitePath := config.SQLitePath
		if sqlitePath == "" {
			sqlitePath = "./sqlite/pixelbattle.db"
		}
		sqliteDB, err := sqlite_adapter.NewSQLiteConnection(sqlitePath)
		if err != nil {
			logrus.Fatalf("Failed to connect to SQLite: %v", err)
		}
		if err := sqlite_adapter.InitializeSchema(sqliteDB); err != nil {
			logrus.Fatalf("Failed to initialize SQLite schema: %v", err)
		}
		canvasRepo = sqlite_repo.NewCanvasRepository(sqliteDB)

	case "redis", "":
		logrus.Info("Initializing Redis storage")
		canvasDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
		canvasRepo = redis_repo.NewCanvasRepository(canvasDatabase)

	default:
		logrus.Fatalf("Unknown storage type: %s. Supported types: redis, postgres, sqlite", storageType)
	}

	// Timer and user repositories still use Redis/MongoDB
	timerDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisTimer)
	timerRepo := redis_repo.NewTimerRepo(timerDatabase)

	mongoUserDatabase, err := mongo.NewMongoConnection(config.MongoURI, "pixelbattle")
	if err != nil {
		logrus.Error(err)
	}
	mongoUsrRepo := mongo_repo.NewUserRepository(mongoUserDatabase)

	sessionStore := sessions.NewCookieStore([]byte(string(securecookie.GenerateRandomKey(32))))
	sessionStore.Options.MaxAge = 1800

	canvasService := service.NewCanvasService(canvasRepo)
	usrService := service.NewUserService(mongoUsrRepo, config.AdminIDs)
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
	
	// Register metrics endpoint
	router.Handle("/metrics", service.MetricsHandler())
	
	rest.StartRestServer(sessionService, *vkAuthProvider, canvasService, usrService, timerService,
		config.CanvasHeight, config.CanvasWidth, router)

	if config.WSAllowAnonymous {
		logrus.Warn("WS_ALLOW_ANONYMOUS is enabled: /ws accepts connections without auth (for benchmarks only)")
	}
	websockets.StartWebSocketServer(sessionService, canvasService, timerService, usrService, router, config.WSAllowAnonymous)

	logrus.Info("Starting server on port 8080")
	handler := service.InstrumentHandler(router)
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logrus.Fatal("Failed to start server: ", err)
	}

}
