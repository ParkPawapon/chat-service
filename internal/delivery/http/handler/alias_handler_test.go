package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"chat-service/internal/domain"
	"chat-service/internal/usecase"
	"chat-service/pkg/idgen"
	appvalidator "chat-service/pkg/validator"
)

func TestAliasHandlerGetOrCreate(t *testing.T) {
	t.Parallel()

	t.Run("returns stable alias without exposing raw identifier", func(t *testing.T) {
		t.Parallel()

		repo := newHandlerAliasRepository()
		handler := NewAliasHandler(usecase.NewAliasUseCase(repo), appvalidator.New())

		first := executeAliasRequest(t, handler, `{"identifier":"client-local-storage-id","roomId":"room-a"}`)
		second := executeAliasRequest(t, handler, `{"identifier":"client-local-storage-id","roomId":"room-a"}`)
		otherRoom := executeAliasRequest(t, handler, `{"identifier":"client-local-storage-id","roomId":"room-b"}`)

		if first.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", first.Code, first.Body.String())
		}
		if second.Code != http.StatusOK {
			t.Fatalf("expected second status 200, got %d with body %s", second.Code, second.Body.String())
		}
		if otherRoom.Code != http.StatusOK {
			t.Fatalf("expected other room status 200, got %d with body %s", otherRoom.Code, otherRoom.Body.String())
		}

		firstAlias := decodeAliasResponse(t, first)
		secondAlias := decodeAliasResponse(t, second)
		otherRoomAlias := decodeAliasResponse(t, otherRoom)

		if firstAlias == "" {
			t.Fatal("expected alias in response")
		}
		if firstAlias != secondAlias {
			t.Fatalf("expected same alias for same identifier and room, got %q and %q", firstAlias, secondAlias)
		}
		if firstAlias == otherRoomAlias {
			t.Fatalf("expected room-scoped alias to differ across rooms, got %q", firstAlias)
		}
		if strings.Contains(firstAlias, "client-local-storage-id") {
			t.Fatalf("alias must not expose raw identifier, got %q", firstAlias)
		}

		stored := repo.mustFind(t, "room-a", idgen.HashIdentifier("client-local-storage-id"))
		if stored.IdentifierHash == "client-local-storage-id" {
			t.Fatal("raw identifier must not be stored in alias repository")
		}
	})

	t.Run("returns 400 for invalid request body", func(t *testing.T) {
		t.Parallel()

		handler := NewAliasHandler(usecase.NewAliasUseCase(newHandlerAliasRepository()), appvalidator.New())
		tests := []struct {
			name string
			body string
		}{
			{name: "missing identifier", body: `{"roomId":"room-a"}`},
			{name: "missing room id", body: `{"identifier":"client-local-storage-id"}`},
			{name: "identifier is not string", body: `{"identifier":123,"roomId":"room-a"}`},
			{name: "room id is not string", body: `{"identifier":"client-local-storage-id","roomId":123}`},
			{name: "unknown field", body: `{"identifier":"client-local-storage-id","roomId":"room-a","unexpected":true}`},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				recorder := executeAliasRequest(t, handler, tt.body)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
				}

				var responseBody struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if responseBody.Error.Code != "invalid_request" {
					t.Fatalf("expected invalid_request code, got %q", responseBody.Error.Code)
				}
			})
		}
	})
}

func executeAliasRequest(t *testing.T, handler *AliasHandler, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/client-alias", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.GetOrCreate(recorder, request)
	return recorder
}

func decodeAliasResponse(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var responseBody struct {
		Alias string `json:"alias"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("failed to decode alias response: %v", err)
	}
	return responseBody.Alias
}

type handlerAliasRepository struct {
	mu      sync.Mutex
	records map[string]domain.ClientAlias
}

func newHandlerAliasRepository() *handlerAliasRepository {
	return &handlerAliasRepository{
		records: map[string]domain.ClientAlias{},
	}
}

func (r *handlerAliasRepository) Find(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	alias, ok := r.records[handlerAliasRecordKey(roomID, identifierHash)]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "client alias not found")
	}

	copy := alias
	return &copy, nil
}

func (r *handlerAliasRepository) Create(ctx context.Context, alias *domain.ClientAlias) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := handlerAliasRecordKey(alias.RoomID, alias.IdentifierHash)
	if _, exists := r.records[key]; exists {
		return domain.NewAppError(domain.ErrConflict, "client alias already exists")
	}

	copy := *alias
	r.records[key] = copy
	return nil
}

func (r *handlerAliasRepository) mustFind(t *testing.T, roomID string, identifierHash string) domain.ClientAlias {
	t.Helper()

	alias, err := r.Find(context.Background(), roomID, identifierHash)
	if err != nil {
		t.Fatalf("expected alias to exist: %v", err)
	}
	return *alias
}

func handlerAliasRecordKey(roomID string, identifierHash string) string {
	return roomID + "\x00" + identifierHash
}
