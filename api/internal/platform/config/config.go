package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	Port          string
	EncryptionKey []byte
	JWTSecret     []byte
	LogLevel      slog.Level
	LogFormat     string // "json" or "text"
}

func Load() (*Config, error) {
	// Load .env if present; try current dir first, then parent (for `cd api && ...`).
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	cfg := &Config{
		Port:      getEnv("PORT", "9650"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		// Construct from individual POSTGRES_* variables.
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		db := os.Getenv("POSTGRES_DATABASE")
		user := os.Getenv("POSTGRES_USERNAME")
		pass := os.Getenv("POSTGRES_PASSWORD")
		sslmode := getEnv("POSTGRES_SSLMODE", "disable")

		if host == "" || db == "" || user == "" {
			return nil, fmt.Errorf("DATABASE_URL or POSTGRES_HOST/POSTGRES_DATABASE/POSTGRES_USERNAME are required")
		}
		if port == "" {
			port = "5432"
		}

		cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, pass, host, port, db, sslmode)
	}

	keyHex := os.Getenv("KAPSTAN_ENCRYPTION_KEY")
	if keyHex == "" {
		return nil, fmt.Errorf("KAPSTAN_ENCRYPTION_KEY is required")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("KAPSTAN_ENCRYPTION_KEY must be a valid hex string: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("KAPSTAN_ENCRYPTION_KEY must be exactly 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	cfg.EncryptionKey = key

	// JWT secret: use JWT_SECRET env var if set, otherwise derive from encryption key.
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWTSecret = []byte(jwtSecret)
	} else {
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte("kapstan-jwt-secret"))
		cfg.JWTSecret = mac.Sum(nil)
	}

	levelStr := getEnv("LOG_LEVEL", "info")
	switch strings.ToLower(levelStr) {
	case "debug":
		cfg.LogLevel = slog.LevelDebug
	case "info":
		cfg.LogLevel = slog.LevelInfo
	case "warn":
		cfg.LogLevel = slog.LevelWarn
	case "error":
		cfg.LogLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid LOG_LEVEL %q: must be debug, info, warn, or error", levelStr)
	}

	if cfg.LogFormat != "json" && cfg.LogFormat != "text" {
		return nil, fmt.Errorf("invalid LOG_FORMAT %q: must be json or text", cfg.LogFormat)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
