package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
)

func TestMessageUseCaseCreateMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("creates a message with stable alias and hashed sender identifier", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "client-local-storage-id", false, false)
		messages := newMemoryMessageStore()
		aliases := newMemoryAliasRepository()
		pubSub := newMemoryMessagePubSub()

		output, err := NewMessageUseCase(
			messages,
			rooms,
			NewAliasUseCase(aliases),
			pubSub,
			nil,
		).CreateMessage(ctx, CreateMessageInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
			Body:       "hello world",
		})
		if err != nil {
			t.Fatalf("CreateMessage returned error: %v", err)
		}
		if output.Body != "hello world" {
			t.Fatalf("expected body %q, got %q", "hello world", output.Body)
		}
		if output.SenderName == "" {
			t.Fatal("expected sender alias in output")
		}

		stored := messages.mustList(t, "room-a")
		if len(stored) != 1 {
			t.Fatalf("expected 1 stored message, got %d", len(stored))
		}
		if stored[0].SenderIdentifierHash != idgen.HashIdentifier("client-local-storage-id") {
			t.Fatalf("unexpected sender identifier hash %q", stored[0].SenderIdentifierHash)
		}
		if stored[0].SenderIdentifierHash == "client-local-storage-id" {
			t.Fatal("raw identifier must not be stored in messages")
		}
		if stored[0].SenderName != output.SenderName {
			t.Fatalf("expected sender alias %q, got %q", output.SenderName, stored[0].SenderName)
		}

		published := pubSub.mustPublished(t, "room-a")
		if len(published) != 1 {
			t.Fatalf("expected 1 published message, got %d", len(published))
		}
	})

	t.Run("returns gone when room has been destroyed", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "owner-id", true, false)

		_, err := NewMessageUseCase(
			newMemoryMessageStore(),
			rooms,
			NewAliasUseCase(newMemoryAliasRepository()),
			newMemoryMessagePubSub(),
			nil,
		).CreateMessage(ctx, CreateMessageInput{
			Identifier: "owner-id",
			RoomID:     "room-a",
			Body:       "hello world",
		})
		if !errors.Is(err, domain.ErrGone) {
			t.Fatalf("expected ErrGone, got %v", err)
		}
	})

	t.Run("returns gone when room has expired", func(t *testing.T) {
		t.Parallel()

		repo := &scriptedRoomRepository{
			findByRoomIDFunc: func(ctx context.Context, roomID string) (*domain.Room, error) {
				return &domain.Room{
					RoomID:              roomID,
					OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
					ExpiresAt:           time.Now().UTC().Add(-time.Minute),
				}, nil
			},
		}

		_, err := NewMessageUseCase(
			newMemoryMessageStore(),
			repo,
			NewAliasUseCase(newMemoryAliasRepository()),
			newMemoryMessagePubSub(),
			nil,
		).CreateMessage(ctx, CreateMessageInput{
			Identifier: "owner-id",
			RoomID:     "room-a",
			Body:       "hello world",
		})
		if !errors.Is(err, domain.ErrGone) {
			t.Fatalf("expected ErrGone, got %v", err)
		}
	})

	t.Run("validates required input", func(t *testing.T) {
		t.Parallel()

		useCase := NewMessageUseCase(
			newMemoryMessageStore(),
			newMemoryRoomRepository(),
			NewAliasUseCase(newMemoryAliasRepository()),
			newMemoryMessagePubSub(),
			nil,
		)
		tests := []struct {
			name  string
			input CreateMessageInput
		}{
			{name: "missing identifier", input: CreateMessageInput{RoomID: "room-a", Body: "hello"}},
			{name: "missing room id", input: CreateMessageInput{Identifier: "client-id", Body: "hello"}},
			{name: "missing body", input: CreateMessageInput{Identifier: "client-id", RoomID: "room-a"}},
			{name: "blank identifier", input: CreateMessageInput{Identifier: "   ", RoomID: "room-a", Body: "hello"}},
			{name: "blank room id", input: CreateMessageInput{Identifier: "client-id", RoomID: "   ", Body: "hello"}},
			{name: "blank body", input: CreateMessageInput{Identifier: "client-id", RoomID: "room-a", Body: "   "}},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := useCase.CreateMessage(ctx, tt.input)
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
			})
		}
	})

	t.Run("propagates message repository dependency errors", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "client-id", false, false)
		dependencyErr := domain.NewAppError(domain.ErrDependency, "message repository unavailable")
		messages := &scriptedMessageStore{
			createFunc: func(ctx context.Context, message *domain.Message) error {
				return dependencyErr
			},
		}

		_, err := NewMessageUseCase(
			messages,
			rooms,
			NewAliasUseCase(newMemoryAliasRepository()),
			newMemoryMessagePubSub(),
			nil,
		).CreateMessage(ctx, CreateMessageInput{
			Identifier: "client-id",
			RoomID:     "room-a",
			Body:       "hello world",
		})
		if !errors.Is(err, domain.ErrDependency) {
			t.Fatalf("expected ErrDependency, got %v", err)
		}
	})

	t.Run("logs publish failures but still returns success", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "client-id", false, false)
		logger := &recordingMessageLogger{}
		pubSub := newMemoryMessagePubSub()
		pubSub.publishErr = domain.NewAppError(domain.ErrDependency, "pubsub unavailable")

		output, err := NewMessageUseCase(
			newMemoryMessageStore(),
			rooms,
			NewAliasUseCase(newMemoryAliasRepository()),
			pubSub,
			logger,
		).CreateMessage(ctx, CreateMessageInput{
			Identifier: "client-id",
			RoomID:     "room-a",
			Body:       "hello world",
		})
		if err != nil {
			t.Fatalf("CreateMessage returned error: %v", err)
		}
		if output == nil {
			t.Fatal("expected message output")
		}
		if logger.warningCount() != 1 {
			t.Fatalf("expected 1 warning log, got %d", logger.warningCount())
		}
	})
}

func TestMessageUseCaseListMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns messages oldest to newest", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "owner-id", false, false)
		sentAtOne := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		sentAtTwo := sentAtOne.Add(time.Minute)
		messages := &scriptedMessageStore{
			listByRoomIDFunc: func(ctx context.Context, roomID string) ([]domain.Message, error) {
				return []domain.Message{
					{ID: "message-1", RoomID: roomID, Body: "first", SenderName: "Client-ONE", SentAt: sentAtOne},
					{ID: "message-2", RoomID: roomID, Body: "second", SenderName: "Client-TWO", SentAt: sentAtTwo},
				}, nil
			},
		}

		output, err := NewMessageUseCase(messages, rooms, NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).ListMessages(ctx, "room-a")
		if err != nil {
			t.Fatalf("ListMessages returned error: %v", err)
		}
		if len(output.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(output.Messages))
		}
		if output.Messages[0].Body != "first" || output.Messages[1].Body != "second" {
			t.Fatalf("expected messages in order, got %#v", output.Messages)
		}
	})

	t.Run("validates required room id", func(t *testing.T) {
		t.Parallel()

		_, err := NewMessageUseCase(newMemoryMessageStore(), newMemoryRoomRepository(), NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).ListMessages(ctx, "   ")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("returns not found for unknown room", func(t *testing.T) {
		t.Parallel()

		_, err := NewMessageUseCase(newMemoryMessageStore(), newMemoryRoomRepository(), NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).ListMessages(ctx, "missing-room")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("returns gone for destroyed room", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "owner-id", true, false)

		_, err := NewMessageUseCase(newMemoryMessageStore(), rooms, NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).ListMessages(ctx, "room-a")
		if !errors.Is(err, domain.ErrGone) {
			t.Fatalf("expected ErrGone, got %v", err)
		}
	})

	t.Run("propagates repository dependency errors", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "owner-id", false, false)
		dependencyErr := domain.NewAppError(domain.ErrDependency, "message repository unavailable")
		messages := &scriptedMessageStore{
			listByRoomIDFunc: func(ctx context.Context, roomID string) ([]domain.Message, error) {
				return nil, dependencyErr
			},
		}

		_, err := NewMessageUseCase(messages, rooms, NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).ListMessages(ctx, "room-a")
		if !errors.Is(err, domain.ErrDependency) {
			t.Fatalf("expected ErrDependency, got %v", err)
		}
	})
}

func TestMessageUseCaseStreamMessages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("validates required room id", func(t *testing.T) {
		t.Parallel()

		_, err := NewMessageUseCase(newMemoryMessageStore(), newMemoryRoomRepository(), NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).StreamMessages(ctx, "   ")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("returns not found for unknown room", func(t *testing.T) {
		t.Parallel()

		_, err := NewMessageUseCase(newMemoryMessageStore(), newMemoryRoomRepository(), NewAliasUseCase(newMemoryAliasRepository()), newMemoryMessagePubSub(), nil).StreamMessages(ctx, "missing-room")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("returns stream subscription", func(t *testing.T) {
		t.Parallel()

		rooms := newMemoryRoomRepository()
		seedRoomMember(t, rooms, "room-a", "owner-id", false, false)
		pubSub := newMemoryMessagePubSub()

		stream, err := NewMessageUseCase(newMemoryMessageStore(), rooms, NewAliasUseCase(newMemoryAliasRepository()), pubSub, nil).StreamMessages(ctx, "room-a")
		if err != nil {
			t.Fatalf("StreamMessages returned error: %v", err)
		}

		want := domain.Message{
			ID:         "message-1",
			RoomID:     "room-a",
			Body:       "hello world",
			SenderName: "Client-ONE",
			SentAt:     time.Now().UTC(),
		}
		if err := pubSub.PublishMessage(ctx, "room-a", want); err != nil {
			t.Fatalf("PublishMessage returned error: %v", err)
		}

		select {
		case got := <-stream:
			if got.ID != want.ID || got.Body != want.Body {
				t.Fatalf("unexpected streamed message %#v", got)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("expected streamed message")
		}
	})
}
