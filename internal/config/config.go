package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application, loaded from environment variables.
type Config struct {
	DataDir           string
	StorageLimitBytes int64
	ListenPort        string

	// Ingest (WAL)
	WalRotateInterval time.Duration
	WalRotateBytes    int64

	// Manage (compaction)
	CompactInterval      time.Duration
	CompactMemoryLimitMB int
	MergeTargetBytes     int64
}

// Load reads configuration from environment variables and returns a new Config struct.
// It falls back to default values if environment variables are not set or invalid.
func Load() *Config {
	cfg := &Config{
		DataDir:              getEnv("DATA_DIR", "data"),
		StorageLimitBytes:    getEnvAsInt64("STORAGE_LIMIT_GB", 10) * 1024 * 1024 * 1024,
		ListenPort:           getEnv("LISTEN_PORT", "8080"),
		WalRotateInterval:    time.Duration(getEnvAsInt64("WAL_ROTATE_SECONDS", 60)) * time.Second,
		WalRotateBytes:       getEnvAsInt64("WAL_ROTATE_MB", 8) * 1024 * 1024,
		CompactInterval:      time.Duration(getEnvAsInt64("COMPACT_INTERVAL_SECONDS", 600)) * time.Second,
		CompactMemoryLimitMB: int(getEnvAsInt64("COMPACT_MEMORY_LIMIT_MB", 512)),
		MergeTargetBytes:     getEnvAsInt64("MERGE_TARGET_MB", 128) * 1024 * 1024,
	}

	log.Printf("config: loaded configuration: DataDir=%s StorageLimitBytes=%d ListenPort=%s WalRotateInterval=%s CompactInterval=%s",
		cfg.DataDir, cfg.StorageLimitBytes, cfg.ListenPort, cfg.WalRotateInterval, cfg.CompactInterval)
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
