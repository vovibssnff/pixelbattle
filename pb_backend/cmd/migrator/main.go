package main

import (
	"context"
	"encoding/json"
	"log"
	mongo "pb_backend/internal/adapters/mongo"
	mongoRepo "pb_backend/internal/adapters/mongo/repository"
	"pb_backend/internal/adapters/redis"
	redisRepo "pb_backend/internal/adapters/redis/repository"
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

	// Check if MongoDB is empty before running migration
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

	logrus.Info(redisUserRepo, canvasRepo, mongoUserRepo)
	logrus.Info(redisUserRepo.GetUsr(ctx, 374040842))

	canvasHistory, err := canvasRepo.GetCanvasHistory(ctx)
	if err != nil {
		return err
	}

	currentCanvas, err := canvasRepo.GetCanvas(ctx)
	if err != nil {
		return err
	}
	logrus.Info(currentCanvas)

	totalPixels, activePixels := calculateUserStats(canvasHistory, currentCanvas)
	logrus.Info(totalPixels, activePixels)

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

		user := redisUserRepo.GetUsr(ctx, userID)
		if user.ID == 0 {
			log.Printf("Warning: Failed to get user data for ID %d", userID)
			continue
		}

		if err := mongoUserRepo.RegisterUser(ctx, user); err != nil {
			log.Printf("Warning: Failed to insert user %d: %v", user.ID, err)
			continue
		}

		totalPlaced := totalPixels[user.ID]
		active := activePixels[user.ID]
		if err := mongoUserRepo.UpdateUserStats(ctx, user, totalPlaced, active); err != nil {
			log.Printf("Warning: Failed to update stats for user %d: %v", user.ID, err)
			continue
		}

		migratedCount++
	}

	log.Printf("Migration completed in %v. Migrated %d users.", time.Since(startTime), migratedCount)
	return nil
}

func calculateUserStats(canvasHistory, currentCanvas map[string][]string) (map[int]int, map[int]int) {
	totalPixels := make(map[int]int)
	activePixels := make(map[int]int)

	for _, pixels := range canvasHistory {
		for _, pixelData := range pixels {
			var pixel struct {
				UserID int `json:"UserID"`
			}
			if err := json.Unmarshal([]byte(pixelData), &pixel); err != nil {
				log.Printf("Warning: Failed to unmarshal pixel data: %v", err)
				continue
			}

			totalPixels[pixel.UserID]++
		}
	}

	for _, pixels := range currentCanvas {
		if len(pixels) == 0 {
			continue
		}

		lastPixel := pixels[len(pixels)-1]

		var pixel struct {
			UserID int `json:"UserID"`
		}
		if err := json.Unmarshal([]byte(lastPixel), &pixel); err != nil {
			log.Printf("Warning: Failed to unmarshal pixel data: %v", err)
			continue
		}

		activePixels[pixel.UserID]++
	}

	return totalPixels, activePixels
}
