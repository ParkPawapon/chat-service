package domain

import (
	"context"
	"time"
)

type RoomRepository interface {
	Create(ctx context.Context, room *Room) error
	FindByRoomID(ctx context.Context, roomID string) (*Room, error)
	Update(ctx context.Context, room *Room) error
	AddMember(ctx context.Context, member *RoomMember) error
	FindMember(ctx context.Context, roomID string, identifierHash string) (*RoomMember, error)
	ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error
	MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error
}

type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	ListByRoomID(ctx context.Context, roomID string) ([]Message, error)
	CountByRoomID(ctx context.Context, roomID string) (int64, error)
}

type AliasRepository interface {
	Find(ctx context.Context, roomID string, identifierHash string) (*ClientAlias, error)
	Create(ctx context.Context, alias *ClientAlias) error
}

type MessagePubSub interface {
	PublishMessage(ctx context.Context, roomID string, message Message) error
	SubscribeMessages(ctx context.Context, roomID string) (MessageSubscription, error)
}

type MessageSubscription interface {
	Messages() <-chan Message
	Close() error
}
