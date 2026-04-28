package postgres

import (
	"context"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) error {
	model := messageToModel(message)
	if model.ID == "" {
		model.ID = idgen.NewUUID()
	}
	now := time.Now().UTC()
	if model.SentAt.IsZero() {
		model.SentAt = now
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to create message", err)
	}

	*message = modelToMessage(model)
	return nil
}

func (r *MessageRepository) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	var models []MessageModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("sent_at ASC").
		Find(&models).Error; err != nil {
		return nil, domain.WrapAppError(domain.ErrDependency, "failed to list messages", err)
	}

	messages := make([]domain.Message, 0, len(models))
	for _, model := range models {
		messages = append(messages, modelToMessage(model))
	}
	return messages, nil
}

func (r *MessageRepository) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&MessageModel{}).
		Where("room_id = ?", roomID).
		Count(&count).Error; err != nil {
		return 0, domain.WrapAppError(domain.ErrDependency, "failed to count messages", err)
	}
	return count, nil
}

func messageToModel(message *domain.Message) MessageModel {
	if message == nil {
		return MessageModel{}
	}
	return MessageModel{
		ID:                   message.ID,
		RoomID:               message.RoomID,
		Body:                 message.Body,
		SenderIdentifierHash: message.SenderIdentifierHash,
		SenderAlias:          message.SenderName,
		SentAt:               message.SentAt,
	}
}

func modelToMessage(model MessageModel) domain.Message {
	return domain.Message{
		ID:                   model.ID,
		RoomID:               model.RoomID,
		Body:                 model.Body,
		SenderIdentifierHash: model.SenderIdentifierHash,
		SenderName:           model.SenderAlias,
		SentAt:               model.SentAt,
	}
}
