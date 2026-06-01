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
	redis_adapter "pb_backend/internal/adapters/redis"
	redis_repo "pb_backend/internal/adapters/redis/repository"
	sqlite_adapter "pb_backend/internal/adapters/sqlite"
	sqlite_repo "pb_backend/internal/adapters/sqlite/repository"
	"pb_backend/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
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
		redisCluster      = flag.Bool("redis-cluster", false, "Use Redis cluster client")
		redisAddrsStr     = flag.String("redis-addrs", "", "Comma-separated Redis cluster addrs (default: REDIS_ADDR)")
		shards            = flag.Int("shards", 1, "Synthetic shard count for per-shard labels")
		zipfianSkew       = flag.Float64("zipfian-skew", 1.1, "Zipfian skew parameter (>=1.01)")
		// Phase 2 only — concurrent-same-coord (CRDT LWW path) and oplog_lag scenarios.
		concurrentSameCoordSec = flag.Int("concurrent-same-coord-duration", 0, "Run concurrent_same_coord scenario for N seconds (0 = skip)")
		opLogLagSec            = flag.Int("oplog-lag-duration", 0, "Run oplog_lag scenario for N seconds (0 = skip)")
		opLogLagRate           = flag.Int("oplog-lag-rate", 500, "Producer ops/s for oplog_lag scenario")
		topologyLabel          = flag.String("topology-label", "", "Free-form label embedded in every BenchmarkResult.Topology (e.g. cluster-3shards)")
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
	logrus.Infof("Zipfian skew: %.2f", *zipfianSkew)
	logrus.Infof("Synthetic shards: %d", *shards)
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
		hashTagKeys := config.RedisCanvasHashTagKeys
		if *redisCluster {
			addrs := parseAddrs(*redisAddrsStr, config.RedisAddr)
			redisClient := redis.NewRedisClusterConnection(addrs, config.RedisPsw)
			repo = redis_repo.NewCanvasRepository(redisClient, true)
			storageName = "redis_cluster"
		} else {
			redisClient := redis.NewRedisConnection(config.RedisAddr, config.RedisPsw, config.RedisHistory)
			repo = redis_repo.NewCanvasRepository(redisClient, false)
		}

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
		if err := os.Remove(sqlitePath); err != nil && !os.IsNotExist(err) {
			logrus.Fatalf("Failed to cleanup SQLite db: %v", err)
		}
		if err := os.MkdirAll("./sqlite", 0o750); err != nil {
			logrus.Fatalf("Failed to create sqlite directory: %v", err)
		}

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
		ZipfianSkew:       *zipfianSkew,
		Shards:            *shards,
		TargetRates:       rates,
		SustainedDuration: time.Duration(*sustainedSec) * time.Second,
		RUWDuration:       time.Duration(*ruwSec) * time.Second,
		RUWReaders:        *ruwReaders,
	}

	benchmarker := benchmark.NewBenchmarker(repo, storageName, uint(*canvasHeight), uint(*canvasWidth), cfg)
	results := benchmarker.RunAllBenchmarks()

	// Phase 2 extra scenarios (skipped on Phase 1 monolith runs when durations are 0).
	if *concurrentSameCoordSec > 0 {
		logrus.Info("=== Phase 9: Concurrent same-coord (CRDT LWW) ===")
		results = append(results, benchmarker.ConcurrentSameCoord(*concurrentWriters, time.Duration(*concurrentSameCoordSec)*time.Second))
	}
	if *opLogLagSec > 0 && *storageType == "redis" {
		logrus.Info("=== Phase 10: Op-log lag ===")
		rdb := buildRedisCmdableForOpLogLag(*redisCluster, *redisAddrsStr, config)
		results = append(results, benchmarker.OpLogLag(*opLogLagRate, time.Duration(*opLogLagSec)*time.Second, rdb, config.RedisCanvasHashTagKeys))
	}
	if label := strings.TrimSpace(*topologyLabel); label != "" {
		for i := range results {
			results[i].Topology = label
		}
	}

	report := reporter.GenerateReport(results, storageName)
	fmt.Println("\n" + report)

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		logrus.Fatalf("Failed to marshal results: %v", err)
	}

	if err := os.WriteFile(*outputFile, jsonData, 0o600); err != nil {
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

// buildRedisCmdableForOpLogLag returns a Cmdable for the OpLogLag sampler. Reuses the same
// addresses the storage-type=redis branch used so the bench connects to the same shards.
func buildRedisCmdableForOpLogLag(cluster bool, addrsStr string, cfg *utils.Config) redis.Cmdable {
	if cluster {
		addrs := parseAddrs(addrsStr, cfg.RedisClusterAddrs, cfg.RedisAddr)
		return redis_adapter.NewRedisClusterConnection(addrs, cfg.RedisPsw)
	}
	return redis_adapter.NewRedisConnection(cfg.RedisAddr, cfg.RedisPsw, cfg.RedisHistory)
}

func parseAddrs(flagValue, configClusterAddrs, singleAddr string) []string {
	raw := strings.TrimSpace(flagValue)
	if raw == "" {
		raw = strings.TrimSpace(configClusterAddrs)
	}
	addrs := make([]string, 0)
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			addrs = append(addrs, p)
		}
	}
	if len(addrs) == 0 && strings.TrimSpace(singleAddr) != "" {
		addrs = append(addrs, strings.TrimSpace(singleAddr))
	}
	if len(addrs) == 0 {
		addrs = append(addrs, "localhost:6379")
	}
	return addrs
}
