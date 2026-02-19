package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
// Values are loaded from environment variables with sensible defaults.
type Config struct {
	// Database
	DatabaseURL string

	// API Server
	APIPort string
	APIHost string

	// Auth
	JWTSecret string
	JWTExpiry time.Duration

	// Admin seed user
	AdminEmail    string
	AdminPassword string

	// General
	BaseURL   string
	LogLevel  string
	LogFormat string

	// Discord OAuth
	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURI  string

	// Storage (S3/MinIO)
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageRegion    string
	StorageUseSSL    bool
	MaxUploadSize    int64 // bytes
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(envOrDefault("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("parsing JWT_EXPIRY: %w", err)
	}

	maxUpload := int64(50 * 1024 * 1024) // default 50MB
	if v := os.Getenv("MAX_UPLOAD_SIZE"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			maxUpload = parsed
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(jwtSecret))
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	storageAccessKey := os.Getenv("STORAGE_ACCESS_KEY")
	if storageAccessKey == "" {
		return nil, fmt.Errorf("STORAGE_ACCESS_KEY environment variable is required")
	}

	storageSecretKey := os.Getenv("STORAGE_SECRET_KEY")
	if storageSecretKey == "" {
		return nil, fmt.Errorf("STORAGE_SECRET_KEY environment variable is required")
	}

	cfg := &Config{
		DatabaseURL:         databaseURL,
		APIPort:             envOrDefault("API_PORT", "8080"),
		APIHost:             envOrDefault("API_HOST", "0.0.0.0"),
		JWTSecret:           jwtSecret,
		JWTExpiry:           jwtExpiry,
		AdminEmail:          envOrDefault("ADMIN_EMAIL", ""),
		AdminPassword:       envOrDefault("ADMIN_PASSWORD", ""),
		BaseURL:             envOrDefault("BASE_URL", "http://localhost:3000"),
		LogLevel:            envOrDefault("LOG_LEVEL", "info"),
		LogFormat:           envOrDefault("LOG_FORMAT", "json"),
		DiscordClientID:     envOrDefault("DISCORD_CLIENT_ID", ""),
		DiscordClientSecret: envOrDefault("DISCORD_CLIENT_SECRET", ""),
		DiscordRedirectURI:  envOrDefault("DISCORD_REDIRECT_URI", ""),
		StorageEndpoint:     envOrDefault("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:    storageAccessKey,
		StorageSecretKey:    storageSecretKey,
		StorageBucket:       envOrDefault("STORAGE_BUCKET", "trackforge-attachments"),
		StorageRegion:       envOrDefault("STORAGE_REGION", "us-east-1"),
		StorageUseSSL:       envOrDefault("STORAGE_USE_SSL", "false") == "true",
		MaxUploadSize:       maxUpload,
	}

	return cfg, nil
}

// ListenAddr returns the host:port string for the HTTP server.
func (c *Config) ListenAddr() string {
	return c.APIHost + ":" + c.APIPort
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
