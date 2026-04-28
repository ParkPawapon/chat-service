package postgres

import (
	"context"
	"errors"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
	"gorm.io/gorm"
)

type AliasRepository struct {
	db *gorm.DB
}

func NewAliasRepository(db *gorm.DB) *AliasRepository {
	return &AliasRepository{db: db}
}

func (r *AliasRepository) Find(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
	var model ClientAliasModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ? AND identifier_hash = ?", roomID, identifierHash).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewAppError(domain.ErrNotFound, "client alias not found")
		}
		return nil, domain.WrapAppError(domain.ErrDependency, "failed to find client alias", err)
	}

	alias := modelToAlias(model)
	return &alias, nil
}

func (r *AliasRepository) Create(ctx context.Context, alias *domain.ClientAlias) error {
	model := aliasToModel(alias)
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
		return domain.WrapAppError(domain.ErrDependency, "failed to create client alias", err)
	}

	*alias = modelToAlias(model)
	return nil
}

func aliasToModel(alias *domain.ClientAlias) ClientAliasModel {
	if alias == nil {
		return ClientAliasModel{}
	}
	return ClientAliasModel{
		ID:             alias.ID,
		RoomID:         alias.RoomID,
		IdentifierHash: alias.IdentifierHash,
		Alias:          alias.Alias,
		CreatedAt:      alias.CreatedAt,
		UpdatedAt:      alias.UpdatedAt,
	}
}

func modelToAlias(model ClientAliasModel) domain.ClientAlias {
	return domain.ClientAlias{
		ID:             model.ID,
		RoomID:         model.RoomID,
		IdentifierHash: model.IdentifierHash,
		Alias:          model.Alias,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}
