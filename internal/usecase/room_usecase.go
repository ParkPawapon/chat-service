package usecase

import (
	"context"
	"time"

	"chat-service/internal/domain"
)

type RoomUseCase struct {
	rooms      domain.RoomRepository
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

func NewRoomUseCase(rooms domain.RoomRepository, defaultTTL time.Duration) *RoomUseCase {
	return &RoomUseCase{rooms: rooms, defaultTTL: defaultTTL}
}

func (u *RoomUseCase) JoinRoom(ctx context.Context, input RoomActionInput) (*JoinRoomOutput, error) {
	_ = ctx
	_ = input

	// TODO: create room on first join, assign owner, persist membership, and respect destroyed rooms.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "join room use case is not implemented yet")
}

func (u *RoomUseCase) DestroyRoom(ctx context.Context, input RoomActionInput) (*DestroyRoomOutput, error) {
	_ = ctx
	_ = input

	// TODO: verify owner by identifier hash, mark room destroyed, and publish destroy coordination events.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "destroy room use case is not implemented yet")
}

func (u *RoomUseCase) LeaveRoom(ctx context.Context, input RoomActionInput) (*LeaveRoomOutput, error) {
	_ = ctx
	_ = input

	// TODO: mark membership as left while keeping message history intact.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "leave room use case is not implemented yet")
}

func (u *RoomUseCase) GetStatus(ctx context.Context, roomID string) (*RoomStatusOutput, error) {
	_ = ctx
	_ = roomID

	// TODO: load room status and message count from repositories.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "room status use case is not implemented yet")
}
