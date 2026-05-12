package config

import (
	"strings"
	"testing"
)

func TestLoadValidatesProductionConfig(t *testing.T) {
	t.Run("loads valid config with hardened defaults", func(t *testing.T) {
		setValidConfigEnv(t)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected config to load: %v", err)
		}
		if cfg.MaxRequestBodyBytes != defaultMaxRequestBodyBytes {
			t.Fatalf("expected default max request body bytes %d, got %d", defaultMaxRequestBodyBytes, cfg.MaxRequestBodyBytes)
		}
	})

	t.Run("rejects wildcard CORS in production", func(t *testing.T) {
		setValidConfigEnv(t)
		t.Setenv("APP_ENV", "production")
		t.Setenv("CORS_ALLOWED_ORIGINS", "*")

		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "must not contain * in production") {
			t.Fatalf("expected production wildcard CORS error, got %v", err)
		}
	})

	t.Run("rejects invalid port", func(t *testing.T) {
		setValidConfigEnv(t)
		t.Setenv("APP_PORT", "70000")

		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "valid TCP port") {
			t.Fatalf("expected invalid port error, got %v", err)
		}
	})

	t.Run("rejects nonpositive room ttl", func(t *testing.T) {
		setValidConfigEnv(t)
		t.Setenv("ROOM_DEFAULT_TTL_MINUTES", "0")

		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "ROOM_DEFAULT_TTL_MINUTES") {
			t.Fatalf("expected room ttl error, got %v", err)
		}
	})
}

func setValidConfigEnv(t *testing.T) {
	t.Helper()

	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("APP_NAME", "chat-service")
	t.Setenv("DATABASE_URL", "postgres://chat:secret@localhost:5432/chat_service")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "0")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("ROOM_DEFAULT_TTL_MINUTES", "30")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "")
}
