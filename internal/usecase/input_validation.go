package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"chat-service/internal/domain"
)

const (
	maxRoomIDLength      = 128
	maxIdentifierLength  = 512
	maxMessageBodyLength = 4096
)

func normalizeRoomID(value string) (string, error) {
	roomID := strings.TrimSpace(value)
	if roomID == "" {
		return "", domain.NewAppError(domain.ErrInvalidInput, "roomId is required")
	}
	if utf8.RuneCountInString(roomID) > maxRoomIDLength {
		return "", domain.NewAppError(domain.ErrInvalidInput, fmt.Sprintf("roomId must be at most %d characters", maxRoomIDLength))
	}
	return roomID, nil
}

func normalizeIdentifier(value string) (string, error) {
	identifier := strings.TrimSpace(value)
	if identifier == "" {
		return "", domain.NewAppError(domain.ErrInvalidInput, "identifier is required")
	}
	if utf8.RuneCountInString(identifier) > maxIdentifierLength {
		return "", domain.NewAppError(domain.ErrInvalidInput, fmt.Sprintf("identifier must be at most %d characters", maxIdentifierLength))
	}
	return identifier, nil
}

func normalizeMessageBody(value string) (string, error) {
	body := strings.TrimSpace(value)
	if body == "" {
		return "", domain.NewAppError(domain.ErrInvalidInput, "body is required")
	}
	if utf8.RuneCountInString(body) > maxMessageBodyLength {
		return "", domain.NewAppError(domain.ErrInvalidInput, fmt.Sprintf("body must be at most %d characters", maxMessageBodyLength))
	}
	return body, nil
}
