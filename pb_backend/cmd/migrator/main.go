package main

import (
	"context"
	"encoding/json"
	"log"
	mongo "pb_backend/internal/adapters/mongo"
	mongoRepo "pb_backend/internal/adapters/mongo/repository"
	"pb_backend/internal/adapters/redis"
	redisRepo "pb_backend/internal/adapters/redis/repository"
	"pb_backend/internal/core/domain"
	"pb_backend/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Info("Loading config")

	config, err := utils.LoadConfig("app.env")
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	canvasDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
	usrDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisUsers)
	banListDatabase := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisBanned)

	canvasRepo := redisRepo.NewCanvasRepository(canvasDatabase)
	redisUserRepo := redisRepo.NewUserRepository(usrDatabase, banListDatabase)

	mongoDb, err := mongo.NewMongoConnection(config.MongoURI, "pixelbattle")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	mongoUserRepo := mongoRepo.NewUserRepository(mongoDb)

	isEmpty, err := mongoUserRepo.IsEmpty(ctx)
	if err != nil {
		log.Fatalf("Failed to check if MongoDB is empty: %v", err)
	}

	if isEmpty {
		logrus.Info("MongoDB database is empty. Starting migration...")
		if err := migrateData(ctx, *redisUserRepo, *canvasRepo, *mongoUserRepo); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migration completed successfully")
	} else {
		logrus.Info("MongoDB database already contains users. Skipping migration.")
	}
}

func migrateData(
	ctx context.Context,
	redisUserRepo redisRepo.UserRepository,
	canvasRepo redisRepo.CanvasRepository,
	mongoUserRepo mongoRepo.UserRepository,
) error {
	startTime := time.Now()
	log.Println("Starting data migration...")

	canvasHistory, err := canvasRepo.GetCanvasHistory(ctx)
	if err != nil {
		return err
	}

	currentCanvas, err := canvasRepo.GetCanvas(ctx)
	if err != nil {
		return err
	}

	totalPixels, activePixels := calculateUserStats(canvasHistory, currentCanvas)

	keys, err := redisUserRepo.GetAllUserKeys(ctx)
	if err != nil {
		return err
	}

	migratedCount := 0
	for _, key := range keys {
		userIDStr := strings.TrimPrefix(key, "usr:")
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			log.Printf("Warning: Invalid user key format %s: %v", key, err)
			continue
		}

		vkKey := domain.VKUserID(userID)
		user := redisUserRepo.GetUsr(ctx, userIDStr)
		user.ID = vkKey
		if user.ID == "" {
			log.Printf("Warning: Failed to get user data for key %s", key)
			continue
		}

		if err := mongoUserRepo.RegisterUser(ctx, user); err != nil {
			log.Printf("Warning: Failed to insert user %s: %v", user.ID, err)
			continue
		}

		totalPlaced := totalPixels[user.ID]
		active := activePixels[user.ID]
		if err := mongoUserRepo.UpdateUserStats(ctx, user, totalPlaced, active); err != nil {
			log.Printf("Warning: Failed to update stats for user %s: %v", user.ID, err)
			continue
		}

		migratedCount++
	}

	log.Printf("Migration completed in %v. Migrated %d users.", time.Since(startTime), migratedCount)
	return nil
}

func pixelUserKeyFromJSON(pixelJSON string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(pixelJSON), &m); err != nil {
		return ""
	}
	raw, ok := m["userid"]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case float64:
		return domain.VKUserID(int(v))
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return ""
		}
		if n, err := strconv.Atoi(s); err == nil {
			return domain.VKUserID(n)
		}
		return utils.NormalizeUsername(s)
	default:
		return ""
	}
}

func calculateUserStats(canvasHistory, currentCanvas map[string][]string) (map[string]int, map[string]int) {
	totalPixels := make(map[string]int)
	activePixels := make(map[string]int)

	for _, pixels := range canvasHistory {
		for _, pixelData := range pixels {
			uid := pixelUserKeyFromJSON(pixelData)
			if uid == "" {
				continue
			}
			totalPixels[uid]++
		}
	}

	for _, pixels := range currentCanvas {
		if len(pixels) == 0 {
			continue
		}
		lastPixel := pixels[len(pixels)-1]
		uid := pixelUserKeyFromJSON(lastPixel)
		if uid == "" {
			continue
		}
		activePixels[uid]++
	}

	return totalPixels, activePixels
}
