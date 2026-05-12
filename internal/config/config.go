package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultMaxRequestBodyBytes int64 = 1 << 20

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
	MaxRequestBodyBytes   int64
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

	maxRequestBodyBytes, err := getenvInt64("MAX_REQUEST_BODY_BYTES", defaultMaxRequestBodyBytes)
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
		MaxRequestBodyBytes:   maxRequestBodyBytes,
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.AppPort == "" {
		return fmt.Errorf("APP_PORT must not be empty")
	}
	port, err := strconv.Atoi(cfg.AppPort)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("APP_PORT must be a valid TCP port")
	}
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL must not be empty")
	}
	if cfg.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR must not be empty")
	}
	if cfg.RedisDB < 0 {
		return fmt.Errorf("REDIS_DB must be greater than or equal to 0")
	}
	if cfg.RoomDefaultTTLMinutes <= 0 {
		return fmt.Errorf("ROOM_DEFAULT_TTL_MINUTES must be greater than 0")
	}
	if cfg.MaxRequestBodyBytes <= 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES must be greater than 0")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must include at least one origin")
	}
	for _, origin := range cfg.CORSAllowedOrigins {
		if origin == "*" {
			if isProductionEnv(cfg.AppEnv) {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain * in production")
			}
			continue
		}
		if err := validateOrigin(origin); err != nil {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS contains invalid origin %q: %w", origin, err)
		}
	}
	return nil
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

func getenvInt64(key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
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

func isProductionEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func validateOrigin(origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("host is required")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return fmt.Errorf("path must be empty")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("query and fragment must be empty")
	}
	return nil
}
