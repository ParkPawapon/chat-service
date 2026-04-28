package usecase

import (
	"context"

	"chat-service/internal/domain"
)

type MessageUseCase struct {
	messages domain.MessageRepository
	rooms    domain.RoomRepository
	aliases  domain.AliasRepository
	pubSub   domain.MessagePubSub
}

type CreateMessageInput struct {
	Identifier string
	RoomID     string
	Body       string
}

type MessageOutput struct {
	ID         string
	Body       string
	SenderName string
	SentAt     string
}

type ListMessagesOutput struct {
	Messages []MessageOutput
}

func NewMessageUseCase(
	messages domain.MessageRepository,
	rooms domain.RoomRepository,
	aliases domain.AliasRepository,
	pubSub domain.MessagePubSub,
) *MessageUseCase {
	return &MessageUseCase{
		messages: messages,
		rooms:    rooms,
		aliases:  aliases,
		pubSub:   pubSub,
	}
}

func (u *MessageUseCase) CreateMessage(ctx context.Context, input CreateMessageInput) (*MessageOutput, error) {
	_ = ctx
	_ = input

	// TODO: verify room state, resolve alias, persist message, then publish to Redis.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "create message use case is not implemented yet")
}

func (u *MessageUseCase) ListMessages(ctx context.Context, roomID string) (*ListMessagesOutput, error) {
	_ = ctx
	_ = roomID

	// TODO: verify room state and return messages sorted oldest to newest.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "list messages use case is not implemented yet")
}

func (u *MessageUseCase) StreamMessages(ctx context.Context, roomID string) (<-chan domain.Message, error) {
	_ = ctx
	_ = roomID

	// TODO: verify room state and bridge Redis pub/sub messages to the SSE delivery layer.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "message stream use case is not implemented yet")
}
