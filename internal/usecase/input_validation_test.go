package usecase

import (
	"errors"
	"strings"
	"testing"

	"chat-service/internal/domain"
)

func TestInputValidation(t *testing.T) {
	t.Run("rejects oversized room id", func(t *testing.T) {
		_, err := normalizeRoomID(strings.Repeat("r", maxRoomIDLength+1))
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects oversized identifier", func(t *testing.T) {
		_, err := normalizeIdentifier(strings.Repeat("i", maxIdentifierLength+1))
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects oversized message body", func(t *testing.T) {
		_, err := normalizeMessageBody(strings.Repeat("m", maxMessageBodyLength+1))
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})
}
