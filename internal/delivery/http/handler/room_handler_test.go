package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chat-service/internal/domain"
	"chat-service/internal/usecase"
	"chat-service/pkg/idgen"
	appvalidator "chat-service/pkg/validator"
)

func TestRoomHandlerAction(t *testing.T) {
	t.Parallel()

	t.Run("join creates owner membership without storing raw identifier", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"join","identifier":"client-local-storage-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			IsDestroyed bool `json:"isDestroyed"`
			IsOwner     bool `json:"isOwner"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if responseBody.IsDestroyed {
			t.Fatal("expected room to be active after join")
		}
		if !responseBody.IsOwner {
			t.Fatal("expected first joiner to become owner")
		}

		room := rooms.mustFindRoom(t, "room-a")
		if room.OwnerIdentifierHash == "client-local-storage-id" {
			t.Fatal("raw identifier must not be stored as room owner identifier hash")
		}
		if room.OwnerIdentifierHash != idgen.HashIdentifier("client-local-storage-id") {
			t.Fatalf("unexpected owner identifier hash %q", room.OwnerIdentifierHash)
		}

		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("client-local-storage-id"))
		if member.IdentifierHash == "client-local-storage-id" {
			t.Fatal("raw identifier must not be stored in room members")
		}
	})

	t.Run("join returns non-owner for a later member", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedJoinedRoom(t, rooms, "room-a", "owner-id", false)
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"join","identifier":"member-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			IsDestroyed bool `json:"isDestroyed"`
			IsOwner     bool `json:"isOwner"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if responseBody.IsDestroyed {
			t.Fatal("expected room to remain active")
		}
		if responseBody.IsOwner {
			t.Fatal("expected later joiner to be non-owner")
		}

		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("member-id"))
		if member.IdentifierHash != idgen.HashIdentifier("member-id") {
			t.Fatalf("unexpected member identifier hash %q", member.IdentifierHash)
		}
	})

	t.Run("join returns destroyed status for a destroyed room", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedJoinedRoom(t, rooms, "room-a", "owner-id", true)
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"join","identifier":"member-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			IsDestroyed bool `json:"isDestroyed"`
			IsOwner     bool `json:"isOwner"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if !responseBody.IsDestroyed {
			t.Fatal("expected destroyed room join to report destroyed")
		}
		if responseBody.IsOwner {
			t.Fatal("expected destroyed room join to report non-owner")
		}
	})

	t.Run("join returns 410 for an expired room", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedExpiredRoom(t, rooms, "room-a", "owner-id")
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"join","identifier":"member-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusGone {
			t.Fatalf("expected status 410, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "gone")
	})

	t.Run("destroy marks room as destroyed for the owner", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedJoinedRoom(t, rooms, "room-a", "client-local-storage-id", false)
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"destroy","identifier":"client-local-storage-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			IsDestroyed bool `json:"isDestroyed"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if !responseBody.IsDestroyed {
			t.Fatal("expected destroy response to report destroyed room")
		}
		if !rooms.mustFindRoom(t, "room-a").IsDestroyed {
			t.Fatal("expected room repository state to be destroyed")
		}
	})

	t.Run("destroy returns 403 for a non-owner", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedJoinedRoom(t, rooms, "room-a", "owner-id", false)
		rooms.members[handlerRoomMemberKey("room-a", idgen.HashIdentifier("member-id"))] = domain.RoomMember{
			RoomID:         "room-a",
			IdentifierHash: idgen.HashIdentifier("member-id"),
			JoinedAt:       time.Now().UTC(),
		}
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"destroy","identifier":"member-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "forbidden")
		if rooms.mustFindRoom(t, "room-a").IsDestroyed {
			t.Fatal("expected room to remain active after non-owner destroy")
		}
	})

	t.Run("destroy returns 404 when room does not exist", func(t *testing.T) {
		t.Parallel()

		handler := NewRoomHandler(usecase.NewRoomUseCase(newHandlerRoomRepository(), newHandlerMessageRepository(), 24*time.Hour), appvalidator.New())
		recorder := executeRoomActionRequest(t, handler, `{"action":"destroy","identifier":"owner-id","roomId":"missing-room"}`)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "not_found")
	})

	t.Run("destroy returns 410 for an expired room", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedExpiredRoom(t, rooms, "room-a", "owner-id")
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"destroy","identifier":"owner-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusGone {
			t.Fatalf("expected status 410, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "gone")
	})

	t.Run("leave marks the member as left", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedJoinedRoom(t, rooms, "room-a", "client-local-storage-id", false)
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"leave","identifier":"client-local-storage-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			HasLeft bool `json:"hasLeft"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if !responseBody.HasLeft {
			t.Fatal("expected leave response to report member left")
		}

		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("client-local-storage-id"))
		if member.LeftAt == nil {
			t.Fatal("expected member to be marked as left")
		}
	})

	t.Run("leave returns 404 when room does not exist", func(t *testing.T) {
		t.Parallel()

		handler := NewRoomHandler(usecase.NewRoomUseCase(newHandlerRoomRepository(), newHandlerMessageRepository(), 24*time.Hour), appvalidator.New())
		recorder := executeRoomActionRequest(t, handler, `{"action":"leave","identifier":"member-id","roomId":"missing-room"}`)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "not_found")
	})

	t.Run("leave returns 410 for an expired room", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		seedExpiredRoom(t, rooms, "room-a", "member-id")
		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())

		recorder := executeRoomActionRequest(t, handler, `{"action":"leave","identifier":"member-id","roomId":"room-a"}`)
		if recorder.Code != http.StatusGone {
			t.Fatalf("expected status 410, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "gone")
	})

	t.Run("returns 400 for invalid action requests", func(t *testing.T) {
		t.Parallel()

		handler := NewRoomHandler(usecase.NewRoomUseCase(newHandlerRoomRepository(), newHandlerMessageRepository(), 24*time.Hour), appvalidator.New())
		tests := []struct {
			name string
			body string
		}{
			{name: "missing action", body: `{"identifier":"client-local-storage-id","roomId":"room-a"}`},
			{name: "unsupported action", body: `{"action":"archive","identifier":"client-local-storage-id","roomId":"room-a"}`},
			{name: "missing identifier", body: `{"action":"join","roomId":"room-a"}`},
			{name: "missing room id", body: `{"action":"join","identifier":"client-local-storage-id"}`},
			{name: "action is not string", body: `{"action":123,"identifier":"client-local-storage-id","roomId":"room-a"}`},
			{name: "unknown field", body: `{"action":"join","identifier":"client-local-storage-id","roomId":"room-a","unexpected":true}`},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				recorder := executeRoomActionRequest(t, handler, tt.body)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
				}

				var responseBody struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				decodeRoomResponse(t, recorder, &responseBody)
				if responseBody.Error.Code != "invalid_request" {
					t.Fatalf("expected invalid_request code, got %q", responseBody.Error.Code)
				}
			})
		}
	})
}

func TestRoomHandlerStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns room status with message count", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		expiresAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		rooms.rooms["room-a"] = domain.Room{
			RoomID:              "room-a",
			OwnerIdentifierHash: idgen.HashIdentifier("client-local-storage-id"),
			ExpiresAt:           expiresAt,
		}
		messages.counts["room-a"] = 3

		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())
		recorder := executeRoomStatusRequest(t, handler, "room-a")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			ExpiresAt    string `json:"expiresAt"`
			IsDestroyed  bool   `json:"isDestroyed"`
			MessageCount int64  `json:"messageCount"`
			ServerTime   string `json:"serverTime"`
		}
		decodeRoomResponse(t, recorder, &responseBody)

		if responseBody.ExpiresAt != "2026-01-02T03:04:05.000Z" {
			t.Fatalf("unexpected expiresAt %q", responseBody.ExpiresAt)
		}
		if responseBody.IsDestroyed {
			t.Fatal("expected room to be active")
		}
		if responseBody.MessageCount != 3 {
			t.Fatalf("expected messageCount 3, got %d", responseBody.MessageCount)
		}
		if _, err := time.Parse(time.RFC3339Nano, responseBody.ServerTime); err != nil {
			t.Fatalf("expected serverTime to be RFC3339 timestamp, got %q", responseBody.ServerTime)
		}
	})

	t.Run("returns destroyed room status", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerRoomRepository()
		messages := newHandlerMessageRepository()
		rooms.rooms["room-a"] = domain.Room{
			RoomID:              "room-a",
			OwnerIdentifierHash: idgen.HashIdentifier("client-local-storage-id"),
			IsDestroyed:         true,
			ExpiresAt:           time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		}

		handler := NewRoomHandler(usecase.NewRoomUseCase(rooms, messages, 24*time.Hour), appvalidator.New())
		recorder := executeRoomStatusRequest(t, handler, "room-a")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			IsDestroyed bool `json:"isDestroyed"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if !responseBody.IsDestroyed {
			t.Fatal("expected destroyed room status to report destroyed")
		}
	})

	t.Run("returns 404 when room does not exist", func(t *testing.T) {
		t.Parallel()

		handler := NewRoomHandler(usecase.NewRoomUseCase(newHandlerRoomRepository(), newHandlerMessageRepository(), 24*time.Hour), appvalidator.New())
		recorder := executeRoomStatusRequest(t, handler, "missing-room")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		assertRoomErrorCode(t, recorder, "not_found")
	})

	t.Run("returns 400 when roomId query is missing", func(t *testing.T) {
		t.Parallel()

		handler := NewRoomHandler(usecase.NewRoomUseCase(newHandlerRoomRepository(), newHandlerMessageRepository(), 24*time.Hour), appvalidator.New())
		recorder := executeRoomStatusRequest(t, handler, "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		decodeRoomResponse(t, recorder, &responseBody)
		if responseBody.Error.Code != "invalid_request" {
			t.Fatalf("expected invalid_request code, got %q", responseBody.Error.Code)
		}
	})
}

func executeRoomActionRequest(t *testing.T, handler *RoomHandler, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.Action(recorder, request)
	return recorder
}

func executeRoomStatusRequest(t *testing.T, handler *RoomHandler, roomID string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/api/v1/rooms/status"
	if roomID != "" {
		target += "?roomId=" + roomID
	}

	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	handler.Status(recorder, request)
	return recorder
}

func decodeRoomResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func assertRoomErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()

	var responseBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeRoomResponse(t, recorder, &responseBody)
	if responseBody.Error.Code != want {
		t.Fatalf("expected error code %q, got %q", want, responseBody.Error.Code)
	}
}

func seedJoinedRoom(t *testing.T, rooms *handlerRoomRepository, roomID string, identifier string, isDestroyed bool) {
	t.Helper()

	identifierHash := idgen.HashIdentifier(identifier)
	now := time.Now().UTC()
	rooms.rooms[roomID] = domain.Room{
		RoomID:              roomID,
		OwnerIdentifierHash: identifierHash,
		IsDestroyed:         isDestroyed,
		ExpiresAt:           now.Add(24 * time.Hour),
	}
	rooms.members[handlerRoomMemberKey(roomID, identifierHash)] = domain.RoomMember{
		RoomID:         roomID,
		IdentifierHash: identifierHash,
		JoinedAt:       now,
	}
}

func seedExpiredRoom(t *testing.T, rooms *handlerRoomRepository, roomID string, identifier string) {
	t.Helper()

	identifierHash := idgen.HashIdentifier(identifier)
	now := time.Now().UTC()
	rooms.rooms[roomID] = domain.Room{
		RoomID:              roomID,
		OwnerIdentifierHash: identifierHash,
		ExpiresAt:           now.Add(-time.Minute),
	}
	rooms.members[handlerRoomMemberKey(roomID, identifierHash)] = domain.RoomMember{
		RoomID:         roomID,
		IdentifierHash: identifierHash,
		JoinedAt:       now.Add(-2 * time.Minute),
	}
}

type handlerRoomRepository struct {
	rooms   map[string]domain.Room
	members map[string]domain.RoomMember
}

func newHandlerRoomRepository() *handlerRoomRepository {
	return &handlerRoomRepository{
		rooms:   map[string]domain.Room{},
		members: map[string]domain.RoomMember{},
	}
}

func (r *handlerRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	if _, exists := r.rooms[room.RoomID]; exists {
		return domain.NewAppError(domain.ErrConflict, "room already exists")
	}
	copy := *room
	r.rooms[room.RoomID] = copy
	return nil
}

func (r *handlerRoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	room, ok := r.rooms[roomID]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	copy := room
	return &copy, nil
}

func (r *handlerRoomRepository) EnsureRoomWithOwnerMember(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
	if existing, ok := r.rooms[room.RoomID]; ok {
		copy := existing
		*room = copy
		return false, nil
	}

	roomCopy := *room
	memberCopy := *member
	r.rooms[room.RoomID] = roomCopy
	r.members[handlerRoomMemberKey(member.RoomID, member.IdentifierHash)] = memberCopy
	return true, nil
}

func (r *handlerRoomRepository) Update(ctx context.Context, room *domain.Room) error {
	if _, ok := r.rooms[room.RoomID]; !ok {
		return domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	copy := *room
	r.rooms[room.RoomID] = copy
	return nil
}

func (r *handlerRoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	key := handlerRoomMemberKey(member.RoomID, member.IdentifierHash)
	if _, exists := r.members[key]; exists {
		return domain.NewAppError(domain.ErrConflict, "room member already exists")
	}
	copy := *member
	r.members[key] = copy
	return nil
}

func (r *handlerRoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	member, ok := r.members[handlerRoomMemberKey(roomID, identifierHash)]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	copy := member
	return &copy, nil
}

func (r *handlerRoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	key := handlerRoomMemberKey(roomID, identifierHash)
	member, ok := r.members[key]
	if !ok {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	member.JoinedAt = joinedAt
	member.LeftAt = nil
	r.members[key] = member
	return nil
}

func (r *handlerRoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	key := handlerRoomMemberKey(roomID, identifierHash)
	member, ok := r.members[key]
	if !ok {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	member.LeftAt = &leftAt
	r.members[key] = member
	return nil
}

func (r *handlerRoomRepository) mustFindRoom(t *testing.T, roomID string) domain.Room {
	t.Helper()

	room, err := r.FindByRoomID(context.Background(), roomID)
	if err != nil {
		t.Fatalf("expected room to exist: %v", err)
	}
	return *room
}

func (r *handlerRoomRepository) mustFindMember(t *testing.T, roomID string, identifierHash string) domain.RoomMember {
	t.Helper()

	member, err := r.FindMember(context.Background(), roomID, identifierHash)
	if err != nil {
		t.Fatalf("expected room member to exist: %v", err)
	}
	return *member
}

func handlerRoomMemberKey(roomID string, identifierHash string) string {
	return roomID + "\x00" + identifierHash
}

type handlerMessageRepository struct {
	counts map[string]int64
}

func newHandlerMessageRepository() *handlerMessageRepository {
	return &handlerMessageRepository{
		counts: map[string]int64{},
	}
}

func (r *handlerMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	r.counts[message.RoomID]++
	return nil
}

func (r *handlerMessageRepository) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	return nil, nil
}

func (r *handlerMessageRepository) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	return r.counts[roomID], nil
}
