package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
)

func TestRoomUseCaseJoinRoom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("first join creates the room owner and stores only hashed identifiers", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		output, err := useCase.JoinRoom(ctx, RoomActionInput{
			Identifier: "client-local-storage-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("JoinRoom returned error: %v", err)
		}
		if output.IsDestroyed {
			t.Fatal("expected active room after first join")
		}
		if !output.IsOwner {
			t.Fatal("expected first joiner to become owner")
		}

		room := rooms.mustFindRoom(t, "room-a")
		if room.OwnerIdentifierHash == "client-local-storage-id" {
			t.Fatal("raw identifier must not be stored as owner identifier hash")
		}
		if room.OwnerIdentifierHash != idgen.HashIdentifier("client-local-storage-id") {
			t.Fatalf("unexpected owner identifier hash %q", room.OwnerIdentifierHash)
		}

		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("client-local-storage-id"))
		if member.IdentifierHash == "client-local-storage-id" {
			t.Fatal("raw identifier must not be stored in room members")
		}
	})

	t.Run("later join returns non-owner and adds a member", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "owner-id", false, false)
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		output, err := useCase.JoinRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("JoinRoom returned error: %v", err)
		}
		if output.IsDestroyed {
			t.Fatal("expected active room for later join")
		}
		if output.IsOwner {
			t.Fatal("expected later joiner to be non-owner")
		}

		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("member-id"))
		if member.IdentifierHash != idgen.HashIdentifier("member-id") {
			t.Fatalf("unexpected member identifier hash %q", member.IdentifierHash)
		}
	})

	t.Run("destroyed room join returns destroyed state", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "owner-id", true, false)
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		output, err := useCase.JoinRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("JoinRoom returned error: %v", err)
		}
		if !output.IsDestroyed {
			t.Fatal("expected destroyed room join to report destroyed")
		}
		if output.IsOwner {
			t.Fatal("expected destroyed room join to report non-owner")
		}
	})

	t.Run("reactivates a member who had left earlier", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "member-id", false, true)
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		output, err := useCase.JoinRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("JoinRoom returned error: %v", err)
		}
		if output.IsDestroyed {
			t.Fatal("expected rejoined room to remain active")
		}
		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("member-id"))
		if member.LeftAt != nil {
			t.Fatal("expected previously left member to be reactivated")
		}
	})

	t.Run("returns created room after concurrent first join conflict", func(t *testing.T) {
		t.Parallel()

		identifierHash := idgen.HashIdentifier("member-id")
		findCalls := 0
		repo := &roomScriptedRoomRepository{
			findByRoomIDFunc: func(ctx context.Context, roomID string) (*domain.Room, error) {
				findCalls++
				if findCalls == 1 {
					return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
				}
				return &domain.Room{
					RoomID:              roomID,
					OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
					ExpiresAt:           time.Now().UTC().Add(24 * time.Hour),
				}, nil
			},
			ensureRoomWithOwnerFunc: func(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
				*room = domain.Room{
					RoomID:              room.RoomID,
					OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
					ExpiresAt:           time.Now().UTC().Add(24 * time.Hour),
				}
				return false, nil
			},
			findMemberFunc: func(ctx context.Context, roomID string, gotIdentifierHash string) (*domain.RoomMember, error) {
				if gotIdentifierHash != identifierHash {
					t.Fatalf("expected hashed identifier %q, got %q", identifierHash, gotIdentifierHash)
				}
				return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
			},
			addMemberFunc: func(ctx context.Context, member *domain.RoomMember) error {
				return nil
			},
		}

		output, err := NewRoomUseCase(repo, newMemoryMessageRepository(), 24*time.Hour).JoinRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("JoinRoom returned error: %v", err)
		}
		if output.IsDestroyed {
			t.Fatal("expected active room after concurrent first join fallback")
		}
		if output.IsOwner {
			t.Fatal("expected member to remain non-owner after concurrent first join fallback")
		}
	})

	t.Run("returns gone when room has expired", func(t *testing.T) {
		t.Parallel()

		repo := &roomScriptedRoomRepository{
			findByRoomIDFunc: func(ctx context.Context, roomID string) (*domain.Room, error) {
				return &domain.Room{
					RoomID:              roomID,
					OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
					ExpiresAt:           time.Now().UTC().Add(-time.Minute),
				}, nil
			},
		}

		_, err := NewRoomUseCase(repo, newMemoryMessageRepository(), 24*time.Hour).JoinRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if !errors.Is(err, domain.ErrGone) {
			t.Fatalf("expected ErrGone, got %v", err)
		}
	})

	t.Run("propagates room repository dependency errors", func(t *testing.T) {
		t.Parallel()

		dependencyErr := domain.NewAppError(domain.ErrDependency, "room repository unavailable")
		repo := &roomScriptedRoomRepository{
			findByRoomIDFunc: func(ctx context.Context, roomID string) (*domain.Room, error) {
				return nil, dependencyErr
			},
		}

		_, err := NewRoomUseCase(repo, newMemoryMessageRepository(), 24*time.Hour).JoinRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if !errors.Is(err, domain.ErrDependency) {
			t.Fatalf("expected ErrDependency, got %v", err)
		}
	})

	t.Run("validates required input", func(t *testing.T) {
		t.Parallel()

		useCase := NewRoomUseCase(newRoomMemoryRoomRepository(), newMemoryMessageRepository(), 24*time.Hour)
		tests := []struct {
			name  string
			input RoomActionInput
		}{
			{name: "missing identifier", input: RoomActionInput{RoomID: "room-a"}},
			{name: "missing room id", input: RoomActionInput{Identifier: "client-local-storage-id"}},
			{name: "blank identifier", input: RoomActionInput{Identifier: "   ", RoomID: "room-a"}},
			{name: "blank room id", input: RoomActionInput{Identifier: "client-local-storage-id", RoomID: "   "}},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := useCase.JoinRoom(ctx, tt.input)
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
			})
		}
	})
}

func TestRoomUseCaseDestroyRoom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("owner can destroy an active room", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "owner-id", false, false)
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		output, err := useCase.DestroyRoom(ctx, RoomActionInput{
			Identifier: "owner-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("DestroyRoom returned error: %v", err)
		}
		if !output.IsDestroyed {
			t.Fatal("expected destroy output to report destroyed room")
		}
		if !rooms.mustFindRoom(t, "room-a").IsDestroyed {
			t.Fatal("expected room to be marked destroyed")
		}
	})

	t.Run("non-owner destroy returns forbidden", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "owner-id", false, false)
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		_, err := useCase.DestroyRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("expired room destroy returns gone", func(t *testing.T) {
		t.Parallel()

		repo := &roomScriptedRoomRepository{
			findByRoomIDFunc: func(ctx context.Context, roomID string) (*domain.Room, error) {
				return &domain.Room{
					RoomID:              roomID,
					OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
					ExpiresAt:           time.Now().UTC().Add(-time.Minute),
				}, nil
			},
		}

		_, err := NewRoomUseCase(repo, newMemoryMessageRepository(), 24*time.Hour).DestroyRoom(ctx, RoomActionInput{
			Identifier: "owner-id",
			RoomID:     "room-a",
		})
		if !errors.Is(err, domain.ErrGone) {
			t.Fatalf("expected ErrGone, got %v", err)
		}
	})

	t.Run("destroy returns success when room is already destroyed", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "owner-id", true, false)

		output, err := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour).DestroyRoom(ctx, RoomActionInput{
			Identifier: "owner-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("DestroyRoom returned error: %v", err)
		}
		if !output.IsDestroyed {
			t.Fatal("expected destroy output to report destroyed room")
		}
	})
}

func TestRoomUseCaseLeaveRoom(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("leave marks the member as left", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "member-id", false, false)
		useCase := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour)

		output, err := useCase.LeaveRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("LeaveRoom returned error: %v", err)
		}
		if !output.HasLeft {
			t.Fatal("expected leave output to report member left")
		}

		member := rooms.mustFindMember(t, "room-a", idgen.HashIdentifier("member-id"))
		if member.LeftAt == nil {
			t.Fatal("expected member to be marked as left")
		}
	})

	t.Run("returns success when member had already left", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		seedRoomUseCaseMember(t, rooms, "room-a", "member-id", false, true)

		output, err := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour).LeaveRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if err != nil {
			t.Fatalf("LeaveRoom returned error: %v", err)
		}
		if !output.HasLeft {
			t.Fatal("expected leave output to report member left")
		}
	})

	t.Run("expired room leave returns gone", func(t *testing.T) {
		t.Parallel()

		repo := &roomScriptedRoomRepository{
			findByRoomIDFunc: func(ctx context.Context, roomID string) (*domain.Room, error) {
				return &domain.Room{
					RoomID:              roomID,
					OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
					ExpiresAt:           time.Now().UTC().Add(-time.Minute),
				}, nil
			},
		}

		_, err := NewRoomUseCase(repo, newMemoryMessageRepository(), 24*time.Hour).LeaveRoom(ctx, RoomActionInput{
			Identifier: "member-id",
			RoomID:     "room-a",
		})
		if !errors.Is(err, domain.ErrGone) {
			t.Fatalf("expected ErrGone, got %v", err)
		}
	})
}

func TestRoomUseCaseGetStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns room status with message count", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		messages := newMemoryMessageRepository()
		expiresAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
		rooms.rooms["room-a"] = domain.Room{
			RoomID:              "room-a",
			OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
			ExpiresAt:           expiresAt,
		}
		messages.counts["room-a"] = 3

		output, err := NewRoomUseCase(rooms, messages, 24*time.Hour).GetStatus(ctx, "room-a")
		if err != nil {
			t.Fatalf("GetStatus returned error: %v", err)
		}
		if output.ExpiresAt != expiresAt {
			t.Fatalf("expected expiresAt %v, got %v", expiresAt, output.ExpiresAt)
		}
		if output.MessageCount != 3 {
			t.Fatalf("expected messageCount 3, got %d", output.MessageCount)
		}
	})

	t.Run("validates required room id", func(t *testing.T) {
		t.Parallel()

		_, err := NewRoomUseCase(newRoomMemoryRoomRepository(), newMemoryMessageRepository(), 24*time.Hour).GetStatus(ctx, "   ")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("returns not found when room does not exist", func(t *testing.T) {
		t.Parallel()

		_, err := NewRoomUseCase(newRoomMemoryRoomRepository(), newMemoryMessageRepository(), 24*time.Hour).GetStatus(ctx, "missing-room")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("propagates message repository dependency errors", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		rooms.rooms["room-a"] = domain.Room{
			RoomID:              "room-a",
			OwnerIdentifierHash: idgen.HashIdentifier("owner-id"),
			ExpiresAt:           time.Now().UTC().Add(24 * time.Hour),
		}
		dependencyErr := domain.NewAppError(domain.ErrDependency, "message repository unavailable")
		messages := &scriptedMessageRepository{
			countByRoomIDFunc: func(ctx context.Context, roomID string) (int64, error) {
				return 0, dependencyErr
			},
		}

		_, err := NewRoomUseCase(rooms, messages, 24*time.Hour).GetStatus(ctx, "room-a")
		if !errors.Is(err, domain.ErrDependency) {
			t.Fatalf("expected ErrDependency, got %v", err)
		}
	})
}

func TestRoomUseCaseListRooms(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("returns all rooms in room id order", func(t *testing.T) {
		t.Parallel()

		rooms := newRoomMemoryRoomRepository()
		rooms.rooms["room-b"] = domain.Room{
			RoomID:              "room-b",
			OwnerIdentifierHash: idgen.HashIdentifier("owner-b"),
			IsDestroyed:         true,
			ExpiresAt:           time.Date(2026, time.January, 3, 3, 4, 5, 0, time.UTC),
			CreatedAt:           time.Date(2026, time.January, 1, 1, 0, 0, 0, time.UTC),
			UpdatedAt:           time.Date(2026, time.January, 1, 2, 0, 0, 0, time.UTC),
		}
		rooms.rooms["room-a"] = domain.Room{
			RoomID:              "room-a",
			OwnerIdentifierHash: idgen.HashIdentifier("owner-a"),
			IsDestroyed:         false,
			ExpiresAt:           time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
			CreatedAt:           time.Date(2025, time.December, 31, 23, 0, 0, 0, time.UTC),
			UpdatedAt:           time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		}

		output, err := NewRoomUseCase(rooms, newMemoryMessageRepository(), 24*time.Hour).ListRooms(ctx)
		if err != nil {
			t.Fatalf("ListRooms returned error: %v", err)
		}

		if len(output.Rooms) != 2 {
			t.Fatalf("expected 2 rooms, got %d", len(output.Rooms))
		}
		if output.Rooms[0].RoomID != "room-a" {
			t.Fatalf("expected first room to be room-a, got %q", output.Rooms[0].RoomID)
		}
		if output.Rooms[1].RoomID != "room-b" {
			t.Fatalf("expected second room to be room-b, got %q", output.Rooms[1].RoomID)
		}
		if !output.Rooms[1].IsDestroyed {
			t.Fatal("expected second room to be destroyed")
		}
		if output.ServerTime.IsZero() {
			t.Fatal("expected serverTime to be set")
		}
	})

	t.Run("propagates repository dependency errors", func(t *testing.T) {
		t.Parallel()

		repo := &roomScriptedRoomRepository{
			listFunc: func(ctx context.Context) ([]domain.Room, error) {
				return nil, domain.NewAppError(domain.ErrDependency, "room repository unavailable")
			},
		}

		_, err := NewRoomUseCase(repo, newMemoryMessageRepository(), 24*time.Hour).ListRooms(ctx)
		if !errors.Is(err, domain.ErrDependency) {
			t.Fatalf("expected ErrDependency, got %v", err)
		}
	})
}

func seedRoomUseCaseMember(t *testing.T, rooms *roomMemoryRoomRepository, roomID string, identifier string, isDestroyed bool, hasLeft bool) {
	t.Helper()

	identifierHash := idgen.HashIdentifier(identifier)
	now := time.Now().UTC()
	room := domain.Room{
		RoomID:              roomID,
		OwnerIdentifierHash: identifierHash,
		IsDestroyed:         isDestroyed,
		ExpiresAt:           now.Add(24 * time.Hour),
	}
	member := domain.RoomMember{
		RoomID:         roomID,
		IdentifierHash: identifierHash,
		JoinedAt:       now,
	}
	if hasLeft {
		leftAt := now.Add(-time.Minute)
		member.LeftAt = &leftAt
	}

	rooms.rooms[roomID] = room
	rooms.members[roomTestMemberRecordKey(roomID, identifierHash)] = member
}
