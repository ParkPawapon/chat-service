package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                string
	AppPort               string
	AppName               string
	DatabaseURL           string
	RedisAddr             string
	RedisPassword         string
	RedisDB               int
	CORSAllowedOrigins    []string
	RoomDefaultTTLMinutes int
	RoomDefaultTTL        time.Duration
}

func Load() (*Config, error) {
	redisDB, err := getenvInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	roomTTLMinutes, err := getenvInt("ROOM_DEFAULT_TTL_MINUTES", 30)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:                getenv("APP_ENV", "development"),
		AppPort:               getenv("APP_PORT", "8080"),
		AppName:               getenv("APP_NAME", "chat-service"),
		DatabaseURL:           getenvRequired("DATABASE_URL"),
		RedisAddr:             getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:         getenv("REDIS_PASSWORD", ""),
		RedisDB:               redisDB,
		CORSAllowedOrigins:    splitCSV(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		RoomDefaultTTLMinutes: roomTTLMinutes,
		RoomDefaultTTL:        time.Duration(roomTTLMinutes) * time.Minute,
	}

	if cfg.AppPort == "" {
		return nil, fmt.Errorf("APP_PORT must not be empty")
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must not be empty")
	}
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR must not be empty")
	}

	return cfg, nil
}

func getenv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getenvRequired(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func getenvInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return value, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
