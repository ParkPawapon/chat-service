package usecase

import (
	"context"
	"sync"
	"testing"

	"chat-service/internal/domain"
)

type memoryAliasRepository struct {
	mu      sync.Mutex
	records map[string]domain.ClientAlias
}

func newMemoryAliasRepository() *memoryAliasRepository {
	return &memoryAliasRepository{
		records: map[string]domain.ClientAlias{},
	}
}

func (r *memoryAliasRepository) Find(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	alias, ok := r.records[aliasRecordKey(roomID, identifierHash)]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "client alias not found")
	}

	return cloneAlias(alias), nil
}

func (r *memoryAliasRepository) Create(ctx context.Context, alias *domain.ClientAlias) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := aliasRecordKey(alias.RoomID, alias.IdentifierHash)
	if _, exists := r.records[key]; exists {
		return domain.NewAppError(domain.ErrConflict, "client alias already exists")
	}

	r.records[key] = *cloneAlias(*alias)
	return nil
}

func (r *memoryAliasRepository) mustFind(t *testing.T, roomID string, identifierHash string) domain.ClientAlias {
	t.Helper()

	alias, err := r.Find(context.Background(), roomID, identifierHash)
	if err != nil {
		t.Fatalf("expected alias to exist: %v", err)
	}
	return *alias
}

func aliasRecordKey(roomID string, identifierHash string) string {
	return roomID + "\x00" + identifierHash
}

func cloneAlias(alias domain.ClientAlias) *domain.ClientAlias {
	copy := alias
	return &copy
}
