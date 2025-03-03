package utils

import (
	"log"
	"strconv"
	"strings"

	// "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds the application configuration values
type Config struct {
	RedisAddr      string   `mapstructure:"REDIS_ADDR"`
	RedisPsw       string   `mapstructure:"REDIS_PSW"`
	RedisHistory   int      `mapstructure:"REDIS_HISTORY"`
	RedisTimer     int      `mapstructure:"REDIS_TIMER"`
	RedisUsers     int      `mapstructure:"REDIS_USERS"`
	RedisBanned    int      `mapstructure:"REDIS_BANNED"`
	CanvasHeight   int      `mapstructure:"CANVAS_HEIGHT"`
	CanvasWidth    int      `mapstructure:"CANVAS_WIDTH"`
	ServiceToken   string   `mapstructure:"SERVICE_TOKEN"`
	AdminIDs       []int    // No `mapstructure` tag to prevent automatic decoding
	APIVersion     string   `mapstructure:"API_VERSION"`
}

// LoadConfig loads configuration from the specified file or environment variables
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)  // Load the specified configuration file
	viper.SetConfigType("env") // Explicitly set file type as ENV
	viper.AutomaticEnv()       // Allow overriding from environment variables

	// Attempt to read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Unable to read config file (%s), relying on environment variables: %v", path, err)
	}

	var config Config

	// Unmarshal configuration except for ADMIN_IDS
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
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

	return &config, nil
}
