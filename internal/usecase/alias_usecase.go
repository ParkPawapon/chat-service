package usecase

import (
	"context"

	"chat-service/internal/domain"
)

type AliasUseCase struct {
	aliases domain.AliasRepository
}

type GetAliasInput struct {
	Identifier string
	RoomID     string
}

type GetAliasOutput struct {
	Alias string
}

func NewAliasUseCase(aliases domain.AliasRepository) *AliasUseCase {
	return &AliasUseCase{aliases: aliases}
}

func (u *AliasUseCase) GetOrCreateAlias(ctx context.Context, input GetAliasInput) (*GetAliasOutput, error) {
	_ = ctx
	_ = input

	// TODO: hash the identifier, load or create a stable alias per room, and persist it.
	return nil, domain.NewAppError(domain.ErrNotImplemented, "client alias use case is not implemented yet")
}
