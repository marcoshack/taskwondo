package config

import (
	"fmt"
	"os"
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
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	jwtExpiry, err := time.ParseDuration(envOrDefault("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("parsing JWT_EXPIRY: %w", err)
	}

	cfg := &Config{
		DatabaseURL:   envOrDefault("DATABASE_URL", "postgres://trackforge:trackforge@localhost:5432/trackforge?sslmode=disable"),
		APIPort:       envOrDefault("API_PORT", "8080"),
		APIHost:       envOrDefault("API_HOST", "0.0.0.0"),
		JWTSecret:     envOrDefault("JWT_SECRET", ""),
		JWTExpiry:     jwtExpiry,
		AdminEmail:    envOrDefault("ADMIN_EMAIL", ""),
		AdminPassword: envOrDefault("ADMIN_PASSWORD", ""),
		BaseURL:       envOrDefault("BASE_URL", "http://localhost:3000"),
		LogLevel:      envOrDefault("LOG_LEVEL", "info"),
		LogFormat:     envOrDefault("LOG_FORMAT", "json"),
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
