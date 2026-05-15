package main

import (
	"context"
	"net/http"
	"os"
	"pb_backend/internal/adapters/grpc"
	"pb_backend/internal/adapters/mongo"
	mongo_repo "pb_backend/internal/adapters/mongo/repository"
	redisadp "pb_backend/internal/adapters/redis"
	redis_repo "pb_backend/internal/adapters/redis/repository"
	"pb_backend/internal/adapters/rest"
	vk "pb_backend/internal/adapters/vk_auth"
	"pb_backend/internal/adapters/websockets"
	"pb_backend/internal/core/service"
	"pb_backend/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	goredis "github.com/redis/go-redis/v9"
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

	var canvasRDB goredis.Cmdable
	var timerRDB goredis.Cmdable
	hashTagKeys := config.RedisCanvasHashTagKeys
	if ca := strings.TrimSpace(config.RedisClusterAddrs); ca != "" {
		addrs := splitCommaNonEmpty(ca)
		if len(addrs) == 0 {
			logrus.Fatal("REDIS_CLUSTER_ADDRS is set but contains no addresses")
		}
		cluster := redisadp.NewRedisClusterConnection(addrs, config.RedisPsw)
		canvasRDB = cluster
		timerRDB = cluster
		hashTagKeys = true
		if config.RedisHistory != 0 || config.RedisTimer != 0 {
			logrus.Warn("Redis Cluster mode uses a single DB 0; REDIS_HISTORY / REDIS_TIMER indices are ignored")
		}
		logrus.Infof("Redis Cluster client enabled (%d seed addrs)", len(addrs))
	} else {
		canvasRDB = redisadp.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
		timerRDB = redisadp.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisTimer)
	}

	ch := uint(config.CanvasHeight)
	cw := uint(config.CanvasWidth)
	sentinelKey := redis_repo.PixelKey(hashTagKeys, cw-1, ch-1)
	probe := service.NewAvailabilityProbe(canvasRDB, sentinelKey, 0)
	probe.Start()
	defer probe.Stop()

	mongoUserDatabase, err := mongo.NewMongoConnection(config.MongoURI, "pixelbattle")
	if err != nil {
		logrus.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	canvasRepo := redis_repo.NewCanvasRepository(canvasRDB, hashTagKeys)
	if id := strings.TrimSpace(config.GatewayInstanceID); id != "" {
		canvasRepo.SetGatewayOrigin(id)
	}
	mongoUsrRepo := mongo_repo.NewUserRepository(mongoUserDatabase)

	timerRepo := redis_repo.NewTimerRepo(timerRDB)

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

	canvasService := service.NewCanvasService(canvasRepo)
	usrService := service.NewUserService(mongoUsrRepo, config.AdminIDs, config.AdminUsernames)
	timerService := service.NewTimerService(*timerRepo, config.PixelCooldownSec)
	sessionService := service.NewSessionService(sessionStore)
	vkAuthProvider := vk.NewVKAuthProvider(config.ServiceToken, config.APIVersion)

	if !canvasService.IsCanvasInitialized(context.Background()) {
		logrus.Info("Initializing canvas with white pixels")
		if err := canvasService.InitializeCanvas(context.Background(), uint(config.CanvasHeight), uint(config.CanvasWidth)); err != nil {
			logrus.Fatalf("Failed to initialize canvas: %v", err)
		}
	}
	service.SetCanvasDimensionsGauge(cw, ch)

	router := mux.NewRouter()
	router.Handle("/metrics", service.MetricsHandler())

	snapInterval := time.Duration(config.CanvasSnapshotIntervalSec) * time.Second
	snapshotter := service.NewCanvasSnapshotter(canvasService, ch, cw, snapInterval, strings.TrimSpace(config.CanvasSnapshotFile))
	snapshotter.Start()
	defer snapshotter.Stop()

	wsReplay := websockets.NewPixelReplayBuffer(200_000)

	if config.WSAllowAnonymous {
		logrus.Warn("WS_ALLOW_ANONYMOUS is enabled: /ws accepts connections without auth (for benchmarks only)")
	}
	wsLimit := websockets.NewLimiterHub(config.RateLimitPixelPerSec, config.RateLimitWSConnPerMinPerIP)
	ws := websockets.StartWebSocketServer(sessionService, canvasService, timerService, usrService, router,
		config.CanvasHeight, config.CanvasWidth, config.WSAllowAnonymous, wsLimit, wsReplay)

	rest.StartRestServer(sessionService, *vkAuthProvider, canvasService, usrService,
		timerService, strings.TrimSpace(config.AdminAPIToken), snapshotter, ws,
		config.CanvasHeight, config.CanvasWidth, router)

	// Phase 2 gateway-mode wiring (gRPC mesh + Redis Streams op-log consumer).
	// Skipped on Phase 1 monolith deployments where GATEWAY_INSTANCE_ID is unset.
	if instance := strings.TrimSpace(config.GatewayInstanceID); instance != "" {
		hostname, _ := os.Hostname()
		service.SetGatewayInstanceInfo(instance, hostname, "phase2")

		// gRPC server (received peer broadcasts → local WS hub).
		grpcServer := grpc.NewServer(instance, ws)
		if config.GatewayGRPCPort > 0 {
			addr := ":" + strconv.Itoa(config.GatewayGRPCPort)
			go func() {
				if err := grpcServer.Start(addr); err != nil {
					logrus.Errorf("gRPC server stopped: %v", err)
				}
			}()
			defer grpcServer.Stop()
		}

		// gRPC client pool (local WS hub → peer gateways).
		peers := splitCommaNonEmpty(config.GatewayPeers)
		if len(peers) > 0 {
			pool, perr := grpc.NewPeerPool(instance, peers)
			if perr != nil {
				logrus.Warnf("gRPC peer pool init failed: %v", perr)
			} else {
				ws.SetPeerFanout(pool)
				defer pool.Close()
				logrus.Infof("gRPC peer pool: %d peer(s) attached (%s)", len(peers), strings.Join(peers, ","))
			}
		}

		// Op-log XREADGROUP consumer (durable / cross-gateway path).
		group := strings.TrimSpace(config.GatewayOpStreamGroup)
		if group == "" {
			group = "pixelbattle-gateway"
		}
		consumer := strings.TrimSpace(config.GatewayOpStreamConsumer)
		if consumer == "" {
			consumer = instance
		}
		opConsumer := redis_repo.NewOpStreamConsumer(canvasRDB, ws, group, consumer, instance)
		if err := opConsumer.Start(context.Background()); err != nil {
			logrus.Errorf("op-log consumer: start failed: %v", err)
		}
		defer opConsumer.Stop()
	}

	logrus.Info("Starting server on port 8080")
	handler := service.InstrumentHandler(router)
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logrus.Fatal("Failed to start server: ", err)
	}
}

func splitCommaNonEmpty(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
