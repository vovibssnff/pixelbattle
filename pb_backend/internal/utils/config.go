package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// envKeysForViper lists env vars that must be applied into viper before Unmarshal.
// Docker Compose injects env_file into the process environment, but viper.Unmarshal
// only considers keys viper already holds (from the config file). Without this merge,
// a missing on-disk app.env leaves MONGO_URI etc. empty even when set in the environment.
var envKeysForViper = []string{
	"REDIS_ADDR", "REDIS_PSW", "REDIS_HISTORY", "REDIS_TIMER", "REDIS_USERS", "REDIS_BANNED",
	"REDIS_CLUSTER_ADDRS", "REDIS_CANVAS_HASHTAG_KEYS", "REDIS_PSW_FILE",
	"RATE_LIMIT_PIXEL_PER_SEC", "RATE_LIMIT_WS_CONN_PER_MIN",
	"CANVAS_SNAPSHOT_INTERVAL_SEC", "CANVAS_SNAPSHOT_FILE",
	"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
	"SQLITE_PATH",
	"CANVAS_HEIGHT", "CANVAS_WIDTH",
	"MONGO_URI", "MONGO_URI_FILE",
	"API_VERSION", "SERVICE_TOKEN",
	"SESSION_KEY", "SESSION_KEY_FILE", "LOG_LEVEL", "WS_ALLOW_ANONYMOUS",
	"SESSION_SECURE",
	"ADMIN_IDS", "ADMIN_USERNAMES",
	"ADMIN_API_TOKEN", "ADMIN_API_TOKEN_FILE",
	"PIXEL_COOLDOWN_SEC",
	"GATEWAY_INSTANCE_ID", "GATEWAY_GRPC_PORT", "GATEWAY_PEERS",
	"GATEWAY_OPSTREAM_GROUP", "GATEWAY_OPSTREAM_CONSUMER",
}

func mergeProcessEnvIntoViper() {
	for _, k := range envKeysForViper {
		if v, ok := os.LookupEnv(k); ok {
			viper.Set(k, v)
		}
	}
}

// Config holds the application configuration values
type Config struct {
	RedisAddr    string `mapstructure:"REDIS_ADDR"`
	RedisPsw     string `mapstructure:"REDIS_PSW"`
	RedisPswFile string `mapstructure:"REDIS_PSW_FILE"`
	RedisHistory int    `mapstructure:"REDIS_HISTORY"`
	RedisTimer   int    `mapstructure:"REDIS_TIMER"`
	// RedisClusterAddrs: comma-separated host:port list for redis.NewClusterClient. When non-empty, canvas + timer use the cluster (DB index is ignored).
	RedisClusterAddrs string `mapstructure:"REDIS_CLUSTER_ADDRS"`
	// RedisCanvasHashTagKeys: use pixel:{y:x} keys (required for Redis Cluster; optional on standalone for migration drills).
	RedisCanvasHashTagKeys bool   `mapstructure:"REDIS_CANVAS_HASHTAG_KEYS"`
	RedisUsers             int    `mapstructure:"REDIS_USERS"`
	RedisBanned            int    `mapstructure:"REDIS_BANNED"`
	PostgresHost           string `mapstructure:"POSTGRES_HOST"`
	PostgresPort           string `mapstructure:"POSTGRES_PORT"`
	PostgresUser           string `mapstructure:"POSTGRES_USER"`
	PostgresPassword       string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresDB             string `mapstructure:"POSTGRES_DB"`
	SQLitePath             string `mapstructure:"SQLITE_PATH"`
	CanvasHeight           int    `mapstructure:"CANVAS_HEIGHT"`
	CanvasWidth            int    `mapstructure:"CANVAS_WIDTH"`
	MongoURI               string `mapstructure:"MONGO_URI"`
	MongoURIFile           string `mapstructure:"MONGO_URI_FILE"`
	AdminIDs               []int  // No `mapstructure` tag to prevent automatic decoding
	AdminUsernames         []string
	APIVersion             string `mapstructure:"API_VERSION"`
	ServiceToken           string `mapstructure:"SERVICE_TOKEN"`

	// SESSION_KEY: secret used to sign session cookies (32+ bytes recommended). If empty, a random key is generated per process start.
	SessionKey     string `mapstructure:"SESSION_KEY"`
	SessionKeyFile string `mapstructure:"SESSION_KEY_FILE"`
	// LOG_LEVEL: logrus level (debug, info, warn, error). Default: info.
	LogLevel string `mapstructure:"LOG_LEVEL"`
	// SessionSecure: set Secure flag on session cookie (use true behind HTTPS). Default: true if unset.
	SessionSecure bool
	// WSAllowAnonymous: when true, /ws accepts uid+faculty query params without a session (load tests only).
	WSAllowAnonymous bool `mapstructure:"WS_ALLOW_ANONYMOUS"`
	// PixelCooldownSec: minimum seconds between pixel placements per user (Redis TTL). Default 3 if unset; 0 disables.
	PixelCooldownSec int `mapstructure:"PIXEL_COOLDOWN_SEC"`
	// ADMIN_API_TOKEN: static bearer for Grafana / automation (X-Admin-Token header). Empty = token auth disabled.
	AdminAPIToken     string `mapstructure:"ADMIN_API_TOKEN"`
	AdminAPITokenFile string `mapstructure:"ADMIN_API_TOKEN_FILE"`
	// RateLimitPixelPerSec: max pixel WS messages per second per authenticated userid (non-admins). 0 disables. Default 5 when env unset.
	RateLimitPixelPerSec int `mapstructure:"RATE_LIMIT_PIXEL_PER_SEC"`
	// RateLimitWSConnPerMinPerIP: max new /ws handshakes per minute per client IP. 0 disables (recommended for k6 from one loader IP). Set in production (e.g. 50).
	RateLimitWSConnPerMinPerIP int `mapstructure:"RATE_LIMIT_WS_CONN_PER_MIN"`
	// CanvasSnapshotIntervalSec: period for background PNG snapshot (GET /api/canvas.png). Default 2 when unset or 0.
	CanvasSnapshotIntervalSec int `mapstructure:"CANVAS_SNAPSHOT_INTERVAL_SEC"`
	// CanvasSnapshotFile: optional path to write the latest PNG atomically (e.g. volume mount for static file server).
	CanvasSnapshotFile string `mapstructure:"CANVAS_SNAPSHOT_FILE"`

	// GatewayInstanceID identifies this gateway in the Phase 2 Swarm (e.g. "gw-1"). When empty,
	// the binary runs in legacy/monolith mode and the gRPC server / op-log consumer stay disabled.
	GatewayInstanceID string `mapstructure:"GATEWAY_INSTANCE_ID"`
	// GatewayGRPCPort: TCP port for the gRPC mesh server (Phase 2). 0 disables the server.
	GatewayGRPCPort int `mapstructure:"GATEWAY_GRPC_PORT"`
	// GatewayPeers: comma-separated host:port peers for inter-gateway gRPC fan-out.
	GatewayPeers string `mapstructure:"GATEWAY_PEERS"`
	// GatewayOpStreamGroup: Redis Streams consumer group name for op-log XREADGROUP fan-out.
	GatewayOpStreamGroup string `mapstructure:"GATEWAY_OPSTREAM_GROUP"`
	// GatewayOpStreamConsumer: this gateway's consumer name within the group (defaults to GATEWAY_INSTANCE_ID).
	GatewayOpStreamConsumer string `mapstructure:"GATEWAY_OPSTREAM_CONSUMER"`
}

// LoadConfig loads configuration from the specified file or environment variables
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)  // Load the specified configuration file
	viper.SetConfigType("env") // Explicit set file type as ENV
	viper.AutomaticEnv()       // Allow overriding from environment variables

	// Attempt to read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Unable to read config file (%s), relying on environment variables: %v", path, err)
	}

	mergeProcessEnvIntoViper()

	var config Config

	// Unmarshal configuration except for ADMIN_IDS
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if config.LogLevel == "" {
		config.LogLevel = "info"
	}

	if !viper.IsSet("PIXEL_COOLDOWN_SEC") {
		config.PixelCooldownSec = 3
	}
	if config.PixelCooldownSec < 0 {
		log.Printf("Warning: PIXEL_COOLDOWN_SEC=%d raised to 0", config.PixelCooldownSec)
		config.PixelCooldownSec = 0
	}
	if config.PixelCooldownSec > 3600 {
		log.Printf("Warning: PIXEL_COOLDOWN_SEC=%d capped to 3600", config.PixelCooldownSec)
		config.PixelCooldownSec = 3600
	}

	if viper.IsSet("SESSION_SECURE") {
		config.SessionSecure = viper.GetBool("SESSION_SECURE")
	} else {
		config.SessionSecure = true
	}

	if !viper.IsSet("RATE_LIMIT_PIXEL_PER_SEC") {
		config.RateLimitPixelPerSec = 5
	}
	if !viper.IsSet("RATE_LIMIT_WS_CONN_PER_MIN") {
		config.RateLimitWSConnPerMinPerIP = 0
	}

	if config.CanvasSnapshotIntervalSec <= 0 {
		config.CanvasSnapshotIntervalSec = 2
	}

	if strings.TrimSpace(config.AdminAPIToken) == "" && strings.TrimSpace(config.AdminAPITokenFile) != "" {
		token, err := readSecretFile("ADMIN_API_TOKEN_FILE", config.AdminAPITokenFile)
		if err != nil {
			return nil, err
		}
		config.AdminAPIToken = token
	}
	if strings.TrimSpace(config.SessionKey) == "" && strings.TrimSpace(config.SessionKeyFile) != "" {
		key, err := readSecretFile("SESSION_KEY_FILE", config.SessionKeyFile)
		if err != nil {
			return nil, err
		}
		config.SessionKey = key
	}
	if strings.TrimSpace(config.RedisPsw) == "" && strings.TrimSpace(config.RedisPswFile) != "" {
		password, err := readSecretFile("REDIS_PSW_FILE", config.RedisPswFile)
		if err != nil {
			return nil, err
		}
		config.RedisPsw = password
	}
	if strings.TrimSpace(config.MongoURI) == "" && strings.TrimSpace(config.MongoURIFile) != "" {
		uri, err := readSecretFile("MONGO_URI_FILE", config.MongoURIFile)
		if err != nil {
			return nil, err
		}
		config.MongoURI = uri
	}

	// Manually parse ADMIN_IDS
	adminIDsStr := viper.GetString("ADMIN_IDS")
	if adminIDsStr != "" {
		adminIDs := strings.Split(adminIDsStr, ",")
		for _, idStr := range adminIDs {
			idStr = strings.TrimSpace(idStr) // Remove any extra spaces
			id, err := strconv.Atoi(idStr)   // Convert to int
			if err != nil {
				log.Printf("Warning: Invalid ADMIN_ID '%s' in config: %v", idStr, err)
				continue
			}
			config.AdminIDs = append(config.AdminIDs, id)
		}
	}

	// ADMIN_USERNAMES: comma-separated local usernames with password auth
	adminNamesStr := viper.GetString("ADMIN_USERNAMES")
	if adminNamesStr != "" {
		for _, name := range strings.Split(adminNamesStr, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				config.AdminUsernames = append(config.AdminUsernames, strings.ToLower(name))
			}
		}
	}

	return &config, nil
}

func readSecretFile(label, path string) (string, error) {
	b, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", label, err)
	}
	return strings.TrimSpace(string(b)), nil
}
