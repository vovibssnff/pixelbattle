package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"pb_backend/cmd/benchmark/benchmark"
	"pb_backend/cmd/benchmark/reporter"
	"pb_backend/internal/adapters/postgres"
	postgres_repo "pb_backend/internal/adapters/postgres/repository"
	"pb_backend/internal/adapters/redis"
	redis_repo "pb_backend/internal/adapters/redis/repository"
	sqlite_adapter "pb_backend/internal/adapters/sqlite"
	sqlite_repo "pb_backend/internal/adapters/sqlite/repository"
	"pb_backend/internal/utils"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		storageType    = flag.String("storage", "redis", "Storage type: redis, postgres, or sqlite")
		outputFile     = flag.String("output", "benchmark_results.json", "Output file for results")
		canvasHeight   = flag.Int("height", 250, "Canvas height")
		canvasWidth    = flag.Int("width", 500, "Canvas width")
		concurrentWriters = flag.Int("writers", 10, "Number of concurrent writers")
		historyMultiplier = flag.Int("history", 1, "History multiplier (how many times to write to same coordinates)")
	)
	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("Starting database benchmark")
	logrus.Infof("Storage type: %s", *storageType)
	logrus.Infof("Canvas size: %dx%d", *canvasWidth, *canvasHeight)
	logrus.Infof("Concurrent writers: %d", *concurrentWriters)
	logrus.Infof("History multiplier: %d", *historyMultiplier)

	config, err := utils.LoadConfig("app.env")
	if err != nil {
		logrus.Warnf("Failed to load config, using defaults: %v", err)
	}

	var repo benchmark.Repository
	var storageName string

	switch *storageType {
	case "redis":
		storageName = "redis"
		redisClient := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
		repo = redis_repo.NewCanvasRepository(redisClient)

	case "postgres":
		storageName = "postgres"
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
		repo = postgres_repo.NewCanvasRepository(postgresDB)

	case "sqlite":
		storageName = "sqlite"
		sqlitePath := config.SQLitePath
		if sqlitePath == "" {
			sqlitePath = "./sqlite/benchmark.db"
		}
		// Remove existing database for clean benchmark
		os.Remove(sqlitePath)
		os.MkdirAll("./sqlite", 0755)
		
		sqliteDB, err := sqlite_adapter.NewSQLiteConnection(sqlitePath)
		if err != nil {
			logrus.Fatalf("Failed to connect to SQLite: %v", err)
		}
		if err := sqlite_adapter.InitializeSchema(sqliteDB); err != nil {
			logrus.Fatalf("Failed to initialize SQLite schema: %v", err)
		}
		repo = sqlite_repo.NewCanvasRepository(sqliteDB)

	default:
		logrus.Fatalf("Unknown storage type: %s", *storageType)
	}

	benchmarker := benchmark.NewBenchmarker(repo, storageName, uint(*canvasHeight), uint(*canvasWidth))
	results := benchmarker.RunAllBenchmarks(*concurrentWriters, *historyMultiplier)

	// Generate report
	report := reporter.GenerateReport(results, storageName)
	fmt.Println("\n" + report)

	// Save results to JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		logrus.Fatalf("Failed to marshal results: %v", err)
	}

	if err := os.WriteFile(*outputFile, jsonData, 0644); err != nil {
		logrus.Fatalf("Failed to write results file: %v", err)
	}

	logrus.Infof("Benchmark results saved to %s", *outputFile)
}
