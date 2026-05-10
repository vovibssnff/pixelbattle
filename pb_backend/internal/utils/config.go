package utils

import (
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
	"REDIS_CLUSTER_ADDRS", "REDIS_CANVAS_HASHTAG_KEYS",
	"RATE_LIMIT_PIXEL_PER_SEC", "RATE_LIMIT_WS_CONN_PER_MIN",
	"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
	"SQLITE_PATH",
	"CANVAS_HEIGHT", "CANVAS_WIDTH",
	"MONGO_URI",
	"API_VERSION", "SERVICE_TOKEN",
	"SESSION_KEY", "LOG_LEVEL", "WS_ALLOW_ANONYMOUS",
	"SESSION_SECURE",
	"ADMIN_IDS", "ADMIN_USERNAMES",
	"ADMIN_API_TOKEN",
	"PIXEL_COOLDOWN_SEC",
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
	RedisAddr        string `mapstructure:"REDIS_ADDR"`
	RedisPsw         string `mapstructure:"REDIS_PSW"`
	RedisHistory     int    `mapstructure:"REDIS_HISTORY"`
	RedisTimer       int    `mapstructure:"REDIS_TIMER"`
	// RedisClusterAddrs: comma-separated host:port list for redis.NewClusterClient. When non-empty, canvas + timer use the cluster (DB index is ignored).
	RedisClusterAddrs string `mapstructure:"REDIS_CLUSTER_ADDRS"`
	// RedisCanvasHashTagKeys: use pixel:{y:x} keys (required for Redis Cluster; optional on standalone for migration drills).
	RedisCanvasHashTagKeys bool `mapstructure:"REDIS_CANVAS_HASHTAG_KEYS"`
	RedisUsers       int    `mapstructure:"REDIS_USERS"`
	RedisBanned      int    `mapstructure:"REDIS_BANNED"`
	PostgresHost     string `mapstructure:"POSTGRES_HOST"`
	PostgresPort     string `mapstructure:"POSTGRES_PORT"`
	PostgresUser     string `mapstructure:"POSTGRES_USER"`
	PostgresPassword string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresDB       string `mapstructure:"POSTGRES_DB"`
	SQLitePath       string `mapstructure:"SQLITE_PATH"`
	CanvasHeight     int    `mapstructure:"CANVAS_HEIGHT"`
	CanvasWidth      int    `mapstructure:"CANVAS_WIDTH"`
	MongoURI         string `mapstructure:"MONGO_URI"`
	AdminIDs         []int  // No `mapstructure` tag to prevent automatic decoding
	AdminUsernames   []string
	APIVersion       string `mapstructure:"API_VERSION"`
	ServiceToken     string `mapstructure:"SERVICE_TOKEN"`

	// SESSION_KEY: secret used to sign session cookies (32+ bytes recommended). If empty, a random key is generated per process start.
	SessionKey string `mapstructure:"SESSION_KEY"`
	// LOG_LEVEL: logrus level (debug, info, warn, error). Default: info.
	LogLevel string `mapstructure:"LOG_LEVEL"`
	// SessionSecure: set Secure flag on session cookie (use true behind HTTPS). Default: true if unset.
	SessionSecure bool
	// WSAllowAnonymous: when true, /ws accepts uid+faculty query params without a session (load tests only).
	WSAllowAnonymous bool `mapstructure:"WS_ALLOW_ANONYMOUS"`
	// PixelCooldownSec: minimum seconds between pixel placements per user (Redis TTL). Default 3 if unset/0.
	PixelCooldownSec int `mapstructure:"PIXEL_COOLDOWN_SEC"`
	// ADMIN_API_TOKEN: static bearer for Grafana / automation (X-Admin-Token header). Empty = token auth disabled.
	AdminAPIToken string `mapstructure:"ADMIN_API_TOKEN"`
	// RateLimitPixelPerSec: max pixel WS messages per second per authenticated userid (non-admins). 0 disables. Default 5 when env unset.
	RateLimitPixelPerSec int `mapstructure:"RATE_LIMIT_PIXEL_PER_SEC"`
	// RateLimitWSConnPerMinPerIP: max new /ws handshakes per minute per client IP. 0 disables (recommended for k6 from one loader IP). Set in production (e.g. 50).
	RateLimitWSConnPerMinPerIP int `mapstructure:"RATE_LIMIT_WS_CONN_PER_MIN"`
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

	if config.PixelCooldownSec <= 0 {
		config.PixelCooldownSec = 3
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
