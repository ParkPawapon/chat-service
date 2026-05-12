package usecase

import (
	"context"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
)

type MessageUseCase struct {
	messages     domain.MessageRepository
	rooms        domain.RoomRepository
	aliasUseCase *AliasUseCase
	pubSub       domain.MessagePubSub
	logger       messageLogger
}

type messageLogger interface {
	WarnContext(ctx context.Context, msg string, args ...any)
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

const messageTimeLayout = "2006-01-02T15:04:05.000Z"

func NewMessageUseCase(
	messages domain.MessageRepository,
	rooms domain.RoomRepository,
	aliasUseCase *AliasUseCase,
	pubSub domain.MessagePubSub,
	logger messageLogger,
) *MessageUseCase {
	return &MessageUseCase{
		messages:     messages,
		rooms:        rooms,
		aliasUseCase: aliasUseCase,
		pubSub:       pubSub,
		logger:       logger,
	}
}

func (u *MessageUseCase) CreateMessage(ctx context.Context, input CreateMessageInput) (*MessageOutput, error) {
	roomID, err := normalizeRoomID(input.RoomID)
	if err != nil {
		return nil, err
	}

	identifier, err := normalizeIdentifier(input.Identifier)
	if err != nil {
		return nil, err
	}

	body, err := normalizeMessageBody(input.Body)
	if err != nil {
		return nil, err
	}

	room, err := u.ensureActiveRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	aliasOutput, err := u.aliasUseCase.GetOrCreateAlias(ctx, GetAliasInput{
		Identifier: identifier,
		RoomID:     room.RoomID,
	})
	if err != nil {
		return nil, err
	}

	message := &domain.Message{
		RoomID:               room.RoomID,
		Body:                 body,
		SenderIdentifierHash: idgen.HashIdentifier(identifier),
		SenderName:           aliasOutput.Alias,
		SentAt:               time.Now().UTC(),
	}

	if err := u.messages.Create(ctx, message); err != nil {
		return nil, err
	}
	if err := u.pubSub.PublishMessage(ctx, room.RoomID, *message); err != nil && u.logger != nil {
		u.logger.WarnContext(ctx, "failed to publish message event",
			"room_id", room.RoomID,
			"message_id", message.ID,
			"error", err,
		)
	}

	return toMessageOutput(*message), nil
}

func (u *MessageUseCase) ListMessages(ctx context.Context, roomID string) (*ListMessagesOutput, error) {
	roomID, err := normalizeRoomID(roomID)
	if err != nil {
		return nil, err
	}

	if _, err := u.ensureActiveRoom(ctx, roomID); err != nil {
		return nil, err
	}

	messages, err := u.messages.ListByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	output := &ListMessagesOutput{
		Messages: make([]MessageOutput, 0, len(messages)),
	}
	for _, message := range messages {
		output.Messages = append(output.Messages, *toMessageOutput(message))
	}

	return output, nil
}

func (u *MessageUseCase) StreamMessages(ctx context.Context, roomID string) (<-chan domain.Message, error) {
	roomID, err := normalizeRoomID(roomID)
	if err != nil {
		return nil, err
	}

	if _, err := u.ensureActiveRoom(ctx, roomID); err != nil {
		return nil, err
	}

	subscription, err := u.pubSub.SubscribeMessages(ctx, roomID)
	if err != nil {
		return nil, err
	}

	messages := make(chan domain.Message)
	go func() {
		defer close(messages)
		defer func() {
			_ = subscription.Close()
		}()

		roomStatusTicker := time.NewTicker(5 * time.Second)
		defer roomStatusTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-roomStatusTicker.C:
				if _, err := u.ensureActiveRoom(ctx, roomID); err != nil {
					return
				}
			case message, ok := <-subscription.Messages():
				if !ok {
					return
				}

				select {
				case messages <- message:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return messages, nil
}

func (u *MessageUseCase) ensureActiveRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	room, err := u.rooms.FindByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.IsDestroyed {
		return nil, domain.NewAppError(domain.ErrGone, "room has been destroyed")
	}
	if room.ExpiresAt.Before(time.Now().UTC()) {
		return nil, domain.NewAppError(domain.ErrGone, "room has expired")
	}
	return room, nil
}

func toMessageOutput(message domain.Message) *MessageOutput {
	return &MessageOutput{
		ID:         message.ID,
		Body:       message.Body,
		SenderName: message.SenderName,
		SentAt:     message.SentAt.UTC().Format(messageTimeLayout),
	}
}
