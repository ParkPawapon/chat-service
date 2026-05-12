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
	"time"

	"chat-service/internal/domain"
	"chat-service/internal/usecase"
	"chat-service/pkg/idgen"
	appvalidator "chat-service/pkg/validator"
)

func TestMessageHandlerCreate(t *testing.T) {
	t.Parallel()

	t.Run("returns created message", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerForTests(t)
		recorder := executeMessageCreateRequest(t, handler, `{"identifier":"client-id","roomId":"room-a","body":"hello world"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			ID         string `json:"id"`
			Body       string `json:"body"`
			SenderName string `json:"senderName"`
			SentAt     string `json:"sentAt"`
		}
		decodeMessageResponse(t, recorder, &responseBody)
		if responseBody.ID == "" {
			t.Fatal("expected id in response")
		}
		if responseBody.Body != "hello world" {
			t.Fatalf("expected body %q, got %q", "hello world", responseBody.Body)
		}
		if responseBody.SenderName == "" {
			t.Fatal("expected senderName in response")
		}
		if _, err := time.Parse(time.RFC3339Nano, responseBody.SentAt); err != nil {
			t.Fatalf("expected sentAt to be RFC3339 timestamp, got %q", responseBody.SentAt)
		}
	})

	t.Run("returns 400 for invalid request body", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerForTests(t)
		tests := []struct {
			name string
			body string
		}{
			{name: "missing identifier", body: `{"roomId":"room-a","body":"hello"}`},
			{name: "missing room id", body: `{"identifier":"client-id","body":"hello"}`},
			{name: "missing body", body: `{"identifier":"client-id","roomId":"room-a"}`},
			{name: "identifier is not string", body: `{"identifier":123,"roomId":"room-a","body":"hello"}`},
			{name: "unknown field", body: `{"identifier":"client-id","roomId":"room-a","body":"hello","unexpected":true}`},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				recorder := executeMessageCreateRequest(t, handler, tt.body)
				if recorder.Code != http.StatusBadRequest {
					t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
				}
				assertMessageErrorCode(t, recorder, "invalid_request")
			})
		}
	})

	t.Run("returns 404 for unknown room", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerWithoutSeededRoom()
		recorder := executeMessageCreateRequest(t, handler, `{"identifier":"client-id","roomId":"missing-room","body":"hello world"}`)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		assertMessageErrorCode(t, recorder, "not_found")
	})

	t.Run("returns 410 for destroyed room", func(t *testing.T) {
		t.Parallel()

		repo := newHandlerMessageRoomRepository()
		seedHandlerMessageRoom(t, repo, "room-a", "owner-id", true, false)
		handler := newMessageHandlerWithDependencies(repo, newHandlerMessageRepository())

		recorder := executeMessageCreateRequest(t, handler, `{"identifier":"owner-id","roomId":"room-a","body":"hello world"}`)
		if recorder.Code != http.StatusGone {
			t.Fatalf("expected status 410, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		assertMessageErrorCode(t, recorder, "gone")
	})
}

func TestMessageHandlerList(t *testing.T) {
	t.Parallel()

	t.Run("returns messages oldest to newest", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerForTests(t)
		_ = executeMessageCreateRequest(t, handler, `{"identifier":"client-id","roomId":"room-a","body":"first"}`)
		_ = executeMessageCreateRequest(t, handler, `{"identifier":"client-id","roomId":"room-a","body":"second"}`)

		recorder := executeMessageListRequest(t, handler, "room-a")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}

		var responseBody struct {
			Messages []struct {
				Body string `json:"body"`
			} `json:"messages"`
		}
		decodeMessageResponse(t, recorder, &responseBody)
		if len(responseBody.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(responseBody.Messages))
		}
		if responseBody.Messages[0].Body != "first" || responseBody.Messages[1].Body != "second" {
			t.Fatalf("expected ordered messages, got %#v", responseBody.Messages)
		}
	})

	t.Run("returns 400 when roomId is missing", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerForTests(t)
		recorder := executeMessageListRequest(t, handler, "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		assertMessageErrorCode(t, recorder, "invalid_request")
	})

	t.Run("returns 404 for unknown room", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerWithoutSeededRoom()
		recorder := executeMessageListRequest(t, handler, "missing-room")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		assertMessageErrorCode(t, recorder, "not_found")
	})
}

func TestMessageHandlerStream(t *testing.T) {
	t.Parallel()

	t.Run("returns 400 when roomId is missing", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerForTests(t)
		recorder := executeMessageStreamRequest(t, handler, "")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		assertMessageErrorCode(t, recorder, "invalid_request")
	})

	t.Run("returns 404 for unknown room", func(t *testing.T) {
		t.Parallel()

		handler := newMessageHandlerWithoutSeededRoom()
		recorder := executeMessageStreamRequest(t, handler, "missing-room")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		assertMessageErrorCode(t, recorder, "not_found")
	})

	t.Run("formats message event as sse payload", func(t *testing.T) {
		t.Parallel()

		rooms := newHandlerMessageRoomRepository()
		seedHandlerMessageRoom(t, rooms, "room-a", "owner-id", false, false)
		pubSub := newHandlerStreamPubSub()
		useCase := usecase.NewMessageUseCase(newHandlerMessageRepository(), rooms, usecase.NewAliasUseCase(newHandlerAliasRepository()), pubSub, nil)
		handler := NewMessageHandler(useCase, appvalidator.New())

		request := httptest.NewRequest(http.MethodGet, "/api/v1/messages/stream?roomId=room-a", nil)
		ctx, cancel := context.WithCancel(request.Context())
		request = request.WithContext(ctx)
		recorder := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			defer close(done)
			handler.Stream(recorder, request)
		}()

		time.Sleep(50 * time.Millisecond)
		_ = pubSub.PublishMessage(context.Background(), "room-a", domain.Message{
			ID:         "message-1",
			RoomID:     "room-a",
			Body:       "hello world",
			SenderName: "Client-ONE",
			SentAt:     time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		})
		time.Sleep(50 * time.Millisecond)
		cancel()
		<-done

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d with body %s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Fatalf("expected text/event-stream, got %q", got)
		}
		body := recorder.Body.String()
		if body == "" {
			t.Fatal("expected SSE body")
		}
		if !bytes.Contains(recorder.Body.Bytes(), []byte("event: message")) {
			t.Fatalf("expected SSE event frame, got %q", body)
		}
		if !bytes.Contains(recorder.Body.Bytes(), []byte(`"body":"hello world"`)) {
			t.Fatalf("expected message payload, got %q", body)
		}
	})
}

func executeMessageCreateRequest(t *testing.T, handler *MessageHandler, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.Create(recorder, request)
	return recorder
}

func executeMessageListRequest(t *testing.T, handler *MessageHandler, roomID string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/api/v1/messages"
	if roomID != "" {
		target += "?roomId=" + roomID
	}

	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	handler.List(recorder, request)
	return recorder
}

func executeMessageStreamRequest(t *testing.T, handler *MessageHandler, roomID string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/api/v1/messages/stream"
	if roomID != "" {
		target += "?roomId=" + roomID
	}

	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	handler.Stream(recorder, request)
	return recorder
}

func decodeMessageResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func assertMessageErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()

	var responseBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeMessageResponse(t, recorder, &responseBody)
	if responseBody.Error.Code != want {
		t.Fatalf("expected error code %q, got %q", want, responseBody.Error.Code)
	}
}

func newMessageHandlerForTests(t *testing.T) *MessageHandler {
	t.Helper()

	roomRepo := newHandlerMessageRoomRepository()
	seedHandlerMessageRoom(t, roomRepo, "room-a", "client-id", false, false)
	return newMessageHandlerWithUseCase(roomRepo, newHandlerMessageRepository(), newHandlerAliasRepository(), newHandlerStreamPubSub())
}

func newMessageHandlerWithoutSeededRoom() *MessageHandler {
	return newMessageHandlerWithUseCase(newHandlerMessageRoomRepository(), newHandlerMessageRepository(), newHandlerAliasRepository(), newHandlerStreamPubSub())
}

func newMessageHandlerWithDependencies(roomRepo *handlerMessageRoomRepository, messageRepo *handlerMessageRepository) *MessageHandler {
	useCase := usecase.NewMessageUseCase(
		messageRepo,
		roomRepo,
		usecase.NewAliasUseCase(newHandlerAliasRepository()),
		newHandlerStreamPubSub(),
		nil,
	)
	return NewMessageHandler(useCase, appvalidator.New())
}

func newMessageHandlerWithUseCase(roomRepo domain.RoomRepository, messageRepo domain.MessageRepository, aliasRepo domain.AliasRepository, pubSub domain.MessagePubSub) *MessageHandler {
	useCase := usecase.NewMessageUseCase(
		messageRepo,
		roomRepo,
		usecase.NewAliasUseCase(aliasRepo),
		pubSub,
		nil,
	)
	return NewMessageHandler(useCase, appvalidator.New())
}

type handlerStreamPubSub struct {
	mu            sync.Mutex
	subscriptions []chan domain.Message
}

func newHandlerStreamPubSub() *handlerStreamPubSub {
	return &handlerStreamPubSub{}
}

func (p *handlerStreamPubSub) PublishMessage(ctx context.Context, roomID string, message domain.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, sub := range p.subscriptions {
		sub <- message
	}
	return nil
}

func (p *handlerStreamPubSub) SubscribeMessages(ctx context.Context, roomID string) (domain.MessageSubscription, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan domain.Message, 4)
	p.subscriptions = append(p.subscriptions, ch)
	return &handlerMessageSubscription{messages: ch}, nil
}

type handlerMessageSubscription struct {
	messages chan domain.Message
	closed   bool
}

func (s *handlerMessageSubscription) Messages() <-chan domain.Message {
	return s.messages
}

func (s *handlerMessageSubscription) Close() error {
	if !s.closed {
		close(s.messages)
		s.closed = true
	}
	return nil
}

type handlerMessageRoomRepository struct {
	mu      sync.Mutex
	rooms   map[string]domain.Room
	members map[string]domain.RoomMember
}

func newHandlerMessageRoomRepository() *handlerMessageRoomRepository {
	return &handlerMessageRoomRepository{
		rooms:   map[string]domain.Room{},
		members: map[string]domain.RoomMember{},
	}
}

func (r *handlerMessageRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	return nil
}

func (r *handlerMessageRoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	room, ok := r.rooms[roomID]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	copy := room
	return &copy, nil
}

func (r *handlerMessageRoomRepository) EnsureRoomWithOwnerMember(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
	return false, nil
}

func (r *handlerMessageRoomRepository) Update(ctx context.Context, room *domain.Room) error {
	return nil
}

func (r *handlerMessageRoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	return nil
}

func (r *handlerMessageRoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	member, ok := r.members[handlerMessageMemberKey(roomID, identifierHash)]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	copy := member
	return &copy, nil
}

func (r *handlerMessageRoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	return nil
}

func (r *handlerMessageRoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	return nil
}

type handlerMessageRepository struct {
	mu       sync.Mutex
	messages map[string][]domain.Message
	nextID   int
}

func newHandlerMessageRepository() *handlerMessageRepository {
	return &handlerMessageRepository{
		messages: map[string][]domain.Message{},
	}
}

func (r *handlerMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	copy := *message
	if copy.ID == "" {
		copy.ID = "message-" + strings.TrimSpace(string(rune('0'+r.nextID)))
		if copy.ID == "message-" {
			copy.ID = "message-1"
		}
	}
	r.messages[copy.RoomID] = append(r.messages[copy.RoomID], copy)
	*message = copy
	return nil
}

func (r *handlerMessageRepository) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored := r.messages[roomID]
	result := make([]domain.Message, 0, len(stored))
	for _, message := range stored {
		copy := message
		result = append(result, copy)
	}
	return result, nil
}

func (r *handlerMessageRepository) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.messages[roomID])), nil
}

func seedHandlerMessageRoom(t *testing.T, repo *handlerMessageRoomRepository, roomID string, identifier string, isDestroyed bool, expired bool) {
	t.Helper()

	repo.mu.Lock()
	defer repo.mu.Unlock()

	identifierHash := idgen.HashIdentifier(identifier)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	if expired {
		expiresAt = time.Now().UTC().Add(-time.Minute)
	}

	repo.rooms[roomID] = domain.Room{
		RoomID:              roomID,
		OwnerIdentifierHash: identifierHash,
		IsDestroyed:         isDestroyed,
		ExpiresAt:           expiresAt,
	}
	repo.members[handlerMessageMemberKey(roomID, identifierHash)] = domain.RoomMember{
		RoomID:         roomID,
		IdentifierHash: identifierHash,
		JoinedAt:       time.Now().UTC(),
	}
}

func handlerMessageMemberKey(roomID string, identifierHash string) string {
	return roomID + "\x00" + identifierHash
}
