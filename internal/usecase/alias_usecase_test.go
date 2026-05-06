package usecase

import (
	"context"
	"errors"
	"testing"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
)

func TestAliasUseCaseGetOrCreateAlias(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns existing alias without creating a new record", func(t *testing.T) {
		t.Parallel()

		repo := &scriptedAliasRepository{
			findFunc: func(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
				return &domain.ClientAlias{
					RoomID:         roomID,
					IdentifierHash: identifierHash,
					Alias:          "Client-EXISTING",
				}, nil
			},
			createFunc: func(ctx context.Context, alias *domain.ClientAlias) error {
				t.Fatal("Create must not be called when alias already exists")
				return nil
			},
		}

		output, err := NewAliasUseCase(repo).GetOrCreateAlias(ctx, GetAliasInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
		})

		if err != nil {
			t.Fatalf("GetOrCreateAlias returned error: %v", err)
		}
		if output.Alias != "Client-EXISTING" {
			t.Fatalf("expected existing alias, got %q", output.Alias)
		}
	})

	t.Run("creates deterministic room-scoped alias with hashed identifier", func(t *testing.T) {
		t.Parallel()

		repo := newMemoryAliasRepository()
		useCase := NewAliasUseCase(repo)

		first, err := useCase.GetOrCreateAlias(ctx, GetAliasInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("first GetOrCreateAlias returned error: %v", err)
		}

		second, err := useCase.GetOrCreateAlias(ctx, GetAliasInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("second GetOrCreateAlias returned error: %v", err)
		}

		otherRoom, err := useCase.GetOrCreateAlias(ctx, GetAliasInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-b",
		})
		if err != nil {
			t.Fatalf("other room GetOrCreateAlias returned error: %v", err)
		}

		if first.Alias == "" {
			t.Fatal("expected alias to be generated")
		}
		if first.Alias != second.Alias {
			t.Fatalf("expected stable alias for same identifier and room, got %q and %q", first.Alias, second.Alias)
		}
		if first.Alias == otherRoom.Alias {
			t.Fatalf("expected different room to produce a room-scoped alias, got %q", first.Alias)
		}

		stored := repo.mustFind(t, "room-a", idgen.HashIdentifier("client-local-storage-id"))
		if stored.IdentifierHash == "client-local-storage-id" {
			t.Fatal("raw client identifier must not be stored as identifier hash")
		}
		if stored.IdentifierHash != idgen.HashIdentifier("client-local-storage-id") {
			t.Fatalf("unexpected identifier hash %q", stored.IdentifierHash)
		}
	})

	t.Run("returns created alias after concurrent create conflict", func(t *testing.T) {
		t.Parallel()

		identifierHash := idgen.HashIdentifier("client-local-storage-id")
		findCalls := 0
		repo := &scriptedAliasRepository{
			findFunc: func(ctx context.Context, roomID string, hash string) (*domain.ClientAlias, error) {
				findCalls++
				if findCalls == 1 {
					return nil, domain.NewAppError(domain.ErrNotFound, "client alias not found")
				}
				if hash != identifierHash {
					t.Fatalf("expected hashed identifier %q, got %q", identifierHash, hash)
				}
				return &domain.ClientAlias{
					RoomID:         roomID,
					IdentifierHash: hash,
					Alias:          "Client-RACE",
				}, nil
			},
			createFunc: func(ctx context.Context, alias *domain.ClientAlias) error {
				return domain.NewAppError(domain.ErrConflict, "client alias already exists")
			},
		}

		output, err := NewAliasUseCase(repo).GetOrCreateAlias(ctx, GetAliasInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
		})

		if err != nil {
			t.Fatalf("GetOrCreateAlias returned error: %v", err)
		}
		if output.Alias != "Client-RACE" {
			t.Fatalf("expected alias created by concurrent request, got %q", output.Alias)
		}
	})

	t.Run("validates required input", func(t *testing.T) {
		t.Parallel()

		useCase := NewAliasUseCase(newMemoryAliasRepository())
		tests := []struct {
			name  string
			input GetAliasInput
		}{
			{name: "missing identifier", input: GetAliasInput{RoomID: "room-a"}},
			{name: "missing room id", input: GetAliasInput{Identifier: "client-local-storage-id"}},
			{name: "blank identifier", input: GetAliasInput{Identifier: "   ", RoomID: "room-a"}},
			{name: "blank room id", input: GetAliasInput{Identifier: "client-local-storage-id", RoomID: "   "}},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := useCase.GetOrCreateAlias(ctx, tt.input)
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
			})
		}
	})

	t.Run("propagates repository dependency errors", func(t *testing.T) {
		t.Parallel()

		dependencyErr := domain.NewAppError(domain.ErrDependency, "alias repository unavailable")
		repo := &scriptedAliasRepository{
			findFunc: func(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
				return nil, dependencyErr
			},
		}

		_, err := NewAliasUseCase(repo).GetOrCreateAlias(ctx, GetAliasInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
		})
		if !errors.Is(err, domain.ErrDependency) {
			t.Fatalf("expected ErrDependency, got %v", err)
		}
	})
}

type scriptedAliasRepository struct {
	findFunc   func(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error)
	createFunc func(ctx context.Context, alias *domain.ClientAlias) error
}

func (r *scriptedAliasRepository) Find(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
	if r.findFunc == nil {
		return nil, domain.NewAppError(domain.ErrNotFound, "client alias not found")
	}
	return r.findFunc(ctx, roomID, identifierHash)
}

func (r *scriptedAliasRepository) Create(ctx context.Context, alias *domain.ClientAlias) error {
	if r.createFunc == nil {
		return nil
	}
	return r.createFunc(ctx, alias)
}
