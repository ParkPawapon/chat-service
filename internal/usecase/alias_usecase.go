package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
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
	roomID := strings.TrimSpace(input.RoomID)
	if roomID == "" {
		return nil, domain.NewAppError(domain.ErrInvalidInput, "roomId is required")
	}

	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" {
		return nil, domain.NewAppError(domain.ErrInvalidInput, "identifier is required")
	}

	identifierHash := idgen.HashIdentifier(identifier)

	alias, err := u.aliases.Find(ctx, roomID, identifierHash)
	if err == nil {
		return &GetAliasOutput{Alias: alias.Alias}, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	clientAlias := &domain.ClientAlias{
		RoomID:         roomID,
		IdentifierHash: identifierHash,
		Alias:          generateClientAlias(roomID, identifierHash),
	}
	if err := u.aliases.Create(ctx, clientAlias); err != nil {
		alias, findErr := u.aliases.Find(ctx, roomID, identifierHash)
		if findErr == nil {
			return &GetAliasOutput{Alias: alias.Alias}, nil
		}
		return nil, err
	}

	return &GetAliasOutput{Alias: clientAlias.Alias}, nil
}

func generateClientAlias(roomID string, identifierHash string) string {
	const aliasPrefixLength = 8

	aliasSeed := idgen.HashIdentifier(roomID + ":" + identifierHash)
	if len(aliasSeed) < aliasPrefixLength {
		return fmt.Sprintf("Client-%s", strings.ToUpper(aliasSeed))
	}

	return fmt.Sprintf("Client-%s", strings.ToUpper(aliasSeed[:aliasPrefixLength]))
}
