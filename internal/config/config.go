package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application, loaded from environment variables.
type Config struct {
	ArchiveInterval   time.Duration
	StorageLimitBytes int64
	ListenPort        string
}

// Load reads configuration from environment variables and returns a new Config struct.
// It falls back to default values if environment variables are not set or invalid.
func Load() *Config {
	archiveInterval := getEnvAsDuration("ARCHIVE_INTERVAL", 3*time.Hour)
	storageLimitGB := getEnvAsInt64("STORAGE_LIMIT_GB", 10)
	listenPort := getEnv("LISTEN_PORT", "8080")

	cfg := &Config{
		ArchiveInterval:   archiveInterval,
		StorageLimitBytes: storageLimitGB * 1024 * 1024 * 1024,
		ListenPort:        listenPort,
	}

	log.Printf("config: loaded configuration: ArchiveInterval=%v, StorageLimitGB=%d, ListenPort=%s", cfg.ArchiveInterval, storageLimitGB, cfg.ListenPort)
	return cfg
}

// getEnv retrieves a string environment variable or returns a fallback value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// getEnvAsInt64 retrieves an int64 environment variable or returns a fallback value.
func getEnvAsInt64(key string, fallback int64) int64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return fallback
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		log.Printf("config: invalid value for %s: %v. using fallback %d", key, err, fallback)
		return fallback
	}
	return value
}

// getEnvAsDuration retrieves a time.Duration environment variable or returns a fallback value.
func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return fallback
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("config: invalid value for %s: %v. using fallback %v", key, err, fallback)
		return fallback
	}
	return value
}
