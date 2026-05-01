package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
)

type RoomUseCase struct {
	rooms      domain.RoomRepository
	messages   domain.MessageRepository
	defaultTTL time.Duration
}

type RoomActionInput struct {
	Identifier string
	RoomID     string
}

type JoinRoomOutput struct {
	IsDestroyed bool
	IsOwner     bool
}

type DestroyRoomOutput struct {
	IsDestroyed bool
}

type LeaveRoomOutput struct {
	HasLeft bool
}

type RoomStatusOutput struct {
	ExpiresAt    time.Time
	IsDestroyed  bool
	MessageCount int64
	ServerTime   time.Time
}

func NewRoomUseCase(rooms domain.RoomRepository, messages domain.MessageRepository, defaultTTL time.Duration) *RoomUseCase {
	return &RoomUseCase{
		rooms:      rooms,
		messages:   messages,
		defaultTTL: defaultTTL,
	}
}

func (u *RoomUseCase) JoinRoom(ctx context.Context, input RoomActionInput) (*JoinRoomOutput, error) {
	roomID, identifierHash, err := normalizeRoomActionInput(input)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	room, err := u.rooms.FindByRoomID(ctx, roomID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}

		room = &domain.Room{
			RoomID:              roomID,
			OwnerIdentifierHash: identifierHash,
			IsDestroyed:         false,
			ExpiresAt:           now.Add(u.defaultTTL),
		}
		if err := u.rooms.Create(ctx, room); err != nil {
			return nil, err
		}

		member := &domain.RoomMember{
			RoomID:         roomID,
			IdentifierHash: identifierHash,
			JoinedAt:       now,
		}
		if err := u.rooms.AddMember(ctx, member); err != nil {
			return nil, err
		}

		return &JoinRoomOutput{
			IsDestroyed: false,
			IsOwner:     true,
		}, nil
	}

	if room.IsDestroyed {
		return &JoinRoomOutput{
			IsDestroyed: true,
			IsOwner:     room.OwnerIdentifierHash == identifierHash,
		}, nil
	}
	if room.ExpiresAt.Before(now) {
		return nil, domain.NewAppError(domain.ErrGone, "room has expired")
	}

	if err := u.ensureActiveMember(ctx, roomID, identifierHash, now); err != nil {
		return nil, err
	}

	return &JoinRoomOutput{
		IsDestroyed: false,
		IsOwner:     room.OwnerIdentifierHash == identifierHash,
	}, nil
}

func (u *RoomUseCase) DestroyRoom(ctx context.Context, input RoomActionInput) (*DestroyRoomOutput, error) {
	roomID, identifierHash, err := normalizeRoomActionInput(input)
	if err != nil {
		return nil, err
	}

	room, err := u.rooms.FindByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if room.IsDestroyed {
		return &DestroyRoomOutput{IsDestroyed: true}, nil
	}
	if room.ExpiresAt.Before(time.Now().UTC()) {
		return nil, domain.NewAppError(domain.ErrGone, "room has expired")
	}
	if room.OwnerIdentifierHash != identifierHash {
		return nil, domain.NewAppError(domain.ErrForbidden, "only the room owner can destroy the room")
	}

	room.IsDestroyed = true
	if err := u.rooms.Update(ctx, room); err != nil {
		return nil, err
	}

	return &DestroyRoomOutput{IsDestroyed: true}, nil
}

func (u *RoomUseCase) LeaveRoom(ctx context.Context, input RoomActionInput) (*LeaveRoomOutput, error) {
	roomID, identifierHash, err := normalizeRoomActionInput(input)
	if err != nil {
		return nil, err
	}

	room, err := u.rooms.FindByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.ExpiresAt.Before(time.Now().UTC()) {
		return nil, domain.NewAppError(domain.ErrGone, "room has expired")
	}

	member, err := u.rooms.FindMember(ctx, roomID, identifierHash)
	if err != nil {
		return nil, err
	}
	if member.LeftAt != nil {
		return &LeaveRoomOutput{HasLeft: true}, nil
	}

	leftAt := time.Now().UTC()
	if err := u.rooms.MarkMemberLeft(ctx, roomID, identifierHash, leftAt); err != nil {
		return nil, err
	}

	return &LeaveRoomOutput{HasLeft: true}, nil
}

func (u *RoomUseCase) GetStatus(ctx context.Context, roomID string) (*RoomStatusOutput, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return nil, domain.NewAppError(domain.ErrInvalidInput, "roomId is required")
	}

	room, err := u.rooms.FindByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	messageCount, err := u.messages.CountByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &RoomStatusOutput{
		ExpiresAt:    room.ExpiresAt,
		IsDestroyed:  room.IsDestroyed,
		MessageCount: messageCount,
		ServerTime:   now,
	}, nil
}

func normalizeRoomActionInput(input RoomActionInput) (string, string, error) {
	roomID := strings.TrimSpace(input.RoomID)
	if roomID == "" {
		return "", "", domain.NewAppError(domain.ErrInvalidInput, "roomId is required")
	}

	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" {
		return "", "", domain.NewAppError(domain.ErrInvalidInput, "identifier is required")
	}

	return roomID, idgen.HashIdentifier(identifier), nil
}

func (u *RoomUseCase) ensureActiveMember(ctx context.Context, roomID string, identifierHash string, now time.Time) error {
	member, err := u.rooms.FindMember(ctx, roomID, identifierHash)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		return u.rooms.AddMember(ctx, &domain.RoomMember{
			RoomID:         roomID,
			IdentifierHash: identifierHash,
			JoinedAt:       now,
		})
	}

	if member.LeftAt == nil {
		return nil
	}

	return u.rooms.ReactivateMember(ctx, roomID, identifierHash, now)
}
