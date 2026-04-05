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
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		storageType       = flag.String("storage", "redis", "Storage type: redis, postgres, or sqlite")
		outputFile        = flag.String("output", "benchmark_results.json", "Output file for results")
		canvasHeight      = flag.Int("height", 250, "Canvas height")
		canvasWidth       = flag.Int("width", 500, "Canvas width")
		concurrentWriters = flag.Int("writers", 10, "Number of concurrent writers")
		historyRounds     = flag.Int("history", 3, "Number of history growth rounds")
		distribution      = flag.String("distribution", "zipfian", "Coordinate distribution: uniform or zipfian")
		warmupSec         = flag.Int("warmup", 5, "Warmup duration in seconds (0 to disable)")
		targetRatesStr    = flag.String("target-rates", "100,500,1000,2000", "Comma-separated target ops/s for sustained throughput test")
		sustainedSec      = flag.Int("sustained-duration", 30, "Seconds per rate level for sustained throughput")
		ruwSec            = flag.Int("ruw-duration", 60, "Read-under-write duration in seconds")
		ruwReaders        = flag.Int("ruw-readers", 2, "Number of concurrent readers for read-under-write")
		metricsPort       = flag.Int("metrics-port", 9091, "Prometheus /metrics HTTP port (0 to disable)")
	)
	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	benchmark.StartMetricsServer(*metricsPort)
	logrus.Info("Starting database benchmark")
	logrus.Infof("Storage type: %s", *storageType)
	logrus.Infof("Canvas size: %dx%d", *canvasWidth, *canvasHeight)
	logrus.Infof("Concurrent writers: %d", *concurrentWriters)
	logrus.Infof("History rounds: %d", *historyRounds)
	logrus.Infof("Distribution: %s", *distribution)
	logrus.Infof("Warmup: %ds", *warmupSec)

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

	rates := parseRates(*targetRatesStr)

	cfg := benchmark.BenchmarkConfig{
		ConcurrentWriters: *concurrentWriters,
		HistoryRounds:     *historyRounds,
		WarmupDuration:    time.Duration(*warmupSec) * time.Second,
		BucketWidth:       5 * time.Second,
		Distribution:      *distribution,
		TargetRates:       rates,
		SustainedDuration: time.Duration(*sustainedSec) * time.Second,
		RUWDuration:       time.Duration(*ruwSec) * time.Second,
		RUWReaders:        *ruwReaders,
	}

	benchmarker := benchmark.NewBenchmarker(repo, storageName, uint(*canvasHeight), uint(*canvasWidth), cfg)
	results := benchmarker.RunAllBenchmarks()

	report := reporter.GenerateReport(results, storageName)
	fmt.Println("\n" + report)

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		logrus.Fatalf("Failed to marshal results: %v", err)
	}

	if err := os.WriteFile(*outputFile, jsonData, 0644); err != nil {
		logrus.Fatalf("Failed to write results file: %v", err)
	}

	logrus.Infof("Benchmark results saved to %s", *outputFile)
}

func parseRates(s string) []int {
	parts := strings.Split(s, ",")
	rates := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			rates = append(rates, v)
		}
	}
	return rates
}
