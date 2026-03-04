package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Default rate limit constants for authentication endpoints.
const (
	DefaultAuthRateLimit = 10 // requests per minute
	DefaultAuthRateBurst = 5  // max burst size
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

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string

	// GitHub OAuth
	GitHubClientID     string
	GitHubClientSecret string

	// Microsoft OAuth
	MicrosoftClientID     string
	MicrosoftClientSecret string

	// Storage (S3/MinIO)
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageRegion    string
	StorageUseSSL    bool
	MaxUploadSize    int64 // bytes

	// Encryption
	EncryptionKey string // optional, derived from JWTSecret if empty

	// Rate limiting
	AuthRateLimit int // requests per minute for auth endpoints
	AuthRateBurst int // max burst for auth endpoints

	// Worker
	NatsURL        string // NATS server URL (required for worker, e.g. "nats://localhost:4222")
	WorkerPoolSize int    // Number of concurrent worker goroutines (default: 10)
	WorkerDBPool   int    // DB max open conns for worker (default: 10)
	BackfillStats  bool   // If true, backfill historical stats on startup then continue

	// Ollama (optional — semantic search)
	OllamaURL   string // Ollama API URL (e.g. "http://ollama:11434"); empty = disabled
	OllamaModel string // Embedding model name (default: "nomic-embed-text")
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

	authRateLimit := DefaultAuthRateLimit
	if v := os.Getenv("AUTH_RATE_LIMIT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			authRateLimit = parsed
		}
	}

	authRateBurst := DefaultAuthRateBurst
	if v := os.Getenv("AUTH_RATE_BURST"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			authRateBurst = parsed
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

	workerPoolSize := 10
	if v := os.Getenv("WORKER_POOL_SIZE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			workerPoolSize = parsed
		}
	}

	workerDBPool := 10
	if v := os.Getenv("WORKER_DB_POOL"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			workerDBPool = parsed
		}
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
		GoogleClientID:      envOrDefault("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:  envOrDefault("GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:      envOrDefault("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret:  envOrDefault("GITHUB_CLIENT_SECRET", ""),
		MicrosoftClientID:      envOrDefault("MICROSOFT_CLIENT_ID", ""),
		MicrosoftClientSecret:  envOrDefault("MICROSOFT_CLIENT_SECRET", ""),
		StorageEndpoint:     envOrDefault("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:    storageAccessKey,
		StorageSecretKey:    storageSecretKey,
		StorageBucket:       envOrDefault("STORAGE_BUCKET", "taskwondo-attachments"),
		StorageRegion:       envOrDefault("STORAGE_REGION", "us-east-1"),
		StorageUseSSL:       envOrDefault("STORAGE_USE_SSL", "false") == "true",
		MaxUploadSize:       maxUpload,
		EncryptionKey:       envOrDefault("ENCRYPTION_KEY", ""),
		AuthRateLimit:       authRateLimit,
		AuthRateBurst:       authRateBurst,
		NatsURL:             envOrDefault("NATS_URL", ""),
		WorkerPoolSize:      workerPoolSize,
		WorkerDBPool:        workerDBPool,
		BackfillStats:       envOrDefault("BACKFILL_STATS", "") == "true",
		OllamaURL:           envOrDefault("OLLAMA_URL", ""),
		OllamaModel:         envOrDefault("OLLAMA_MODEL", "nomic-embed-text"),
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
