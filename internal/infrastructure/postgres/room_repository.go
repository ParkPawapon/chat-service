package postgres

import (
	"context"
	"errors"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	model := roomToModel(room)
	if model.ID == "" {
		model.ID = idgen.NewUUID()
	}
	now := time.Now().UTC()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	if model.UpdatedAt.IsZero() {
		model.UpdatedAt = now
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to create room", err)
	}

	*room = modelToRoom(model)
	return nil
}

func (r *RoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	var model RoomModel
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
		}
		return nil, domain.WrapAppError(domain.ErrDependency, "failed to find room", err)
	}

	room := modelToRoom(model)
	return &room, nil
}

func (r *RoomRepository) Update(ctx context.Context, room *domain.Room) error {
	model := roomToModel(room)
	model.UpdatedAt = time.Now().UTC()

	result := r.db.WithContext(ctx).
		Model(&RoomModel{}).
		Where("room_id = ?", room.RoomID).
		Updates(map[string]any{
			"owner_identifier_hash": model.OwnerIdentifierHash,
			"is_destroyed":          model.IsDestroyed,
			"expires_at":            model.ExpiresAt,
			"updated_at":            model.UpdatedAt,
		})
	if result.Error != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to update room", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewAppError(domain.ErrNotFound, "room not found")
	}

	room.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *RoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	model := roomMemberToModel(member)
	if model.ID == "" {
		model.ID = idgen.NewUUID()
	}
	if model.JoinedAt.IsZero() {
		model.JoinedAt = time.Now().UTC()
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to add room member", err)
	}

	*member = modelToRoomMember(model)
	return nil
}

func (r *RoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	var model RoomMemberModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ? AND identifier_hash = ?", roomID, identifierHash).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
		}
		return nil, domain.WrapAppError(domain.ErrDependency, "failed to find room member", err)
	}

	member := modelToRoomMember(model)
	return &member, nil
}

func (r *RoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&RoomMemberModel{}).
		Where("room_id = ? AND identifier_hash = ?", roomID, identifierHash).
		Updates(map[string]any{
			"joined_at": joinedAt.UTC(),
			"left_at":   nil,
		})
	if result.Error != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to reactivate room member", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	return nil
}

func (r *RoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&RoomMemberModel{}).
		Where("room_id = ? AND identifier_hash = ?", roomID, identifierHash).
		Update("left_at", leftAt.UTC())
	if result.Error != nil {
		return domain.WrapAppError(domain.ErrDependency, "failed to mark room member left", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	return nil
}

func roomToModel(room *domain.Room) RoomModel {
	if room == nil {
		return RoomModel{}
	}
	return RoomModel{
		ID:                  room.ID,
		RoomID:              room.RoomID,
		OwnerIdentifierHash: room.OwnerIdentifierHash,
		IsDestroyed:         room.IsDestroyed,
		ExpiresAt:           room.ExpiresAt,
		CreatedAt:           room.CreatedAt,
		UpdatedAt:           room.UpdatedAt,
	}
}

func modelToRoom(model RoomModel) domain.Room {
	return domain.Room{
		ID:                  model.ID,
		RoomID:              model.RoomID,
		OwnerIdentifierHash: model.OwnerIdentifierHash,
		IsDestroyed:         model.IsDestroyed,
		ExpiresAt:           model.ExpiresAt,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
	}
}

func roomMemberToModel(member *domain.RoomMember) RoomMemberModel {
	if member == nil {
		return RoomMemberModel{}
	}
	return RoomMemberModel{
		ID:             member.ID,
		RoomID:         member.RoomID,
		IdentifierHash: member.IdentifierHash,
		JoinedAt:       member.JoinedAt,
		LeftAt:         member.LeftAt,
	}
}

func modelToRoomMember(model RoomMemberModel) domain.RoomMember {
	return domain.RoomMember{
		ID:             model.ID,
		RoomID:         model.RoomID,
		IdentifierHash: model.IdentifierHash,
		JoinedAt:       model.JoinedAt,
		LeftAt:         model.LeftAt,
	}
}
