package usecase

import (
	"context"
	"sync"
	"testing"
	"time"

	"chat-service/internal/domain"
)

type roomMemoryRoomRepository struct {
	mu      sync.Mutex
	rooms   map[string]domain.Room
	members map[string]domain.RoomMember
}

func newRoomMemoryRoomRepository() *roomMemoryRoomRepository {
	return &roomMemoryRoomRepository{
		rooms:   map[string]domain.Room{},
		members: map[string]domain.RoomMember{},
	}
}

func (r *roomMemoryRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rooms[room.RoomID]; exists {
		return domain.NewAppError(domain.ErrConflict, "room already exists")
	}
	r.rooms[room.RoomID] = *cloneRoomForRoomTest(*room)
	return nil
}

func (r *roomMemoryRoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	room, ok := r.rooms[roomID]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	return cloneRoomForRoomTest(room), nil
}

func (r *roomMemoryRoomRepository) EnsureRoomWithOwnerMember(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.rooms[room.RoomID]; ok {
		*room = *cloneRoomForRoomTest(existing)
		return false, nil
	}

	r.rooms[room.RoomID] = *cloneRoomForRoomTest(*room)
	r.members[roomTestMemberRecordKey(member.RoomID, member.IdentifierHash)] = *cloneRoomMemberForRoomTest(*member)
	return true, nil
}

func (r *roomMemoryRoomRepository) Update(ctx context.Context, room *domain.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.rooms[room.RoomID]; !ok {
		return domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	r.rooms[room.RoomID] = *cloneRoomForRoomTest(*room)
	return nil
}

func (r *roomMemoryRoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := roomTestMemberRecordKey(member.RoomID, member.IdentifierHash)
	if _, exists := r.members[key]; exists {
		return domain.NewAppError(domain.ErrConflict, "room member already exists")
	}
	r.members[key] = *cloneRoomMemberForRoomTest(*member)
	return nil
}

func (r *roomMemoryRoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	member, ok := r.members[roomTestMemberRecordKey(roomID, identifierHash)]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	return cloneRoomMemberForRoomTest(member), nil
}

func (r *roomMemoryRoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := roomTestMemberRecordKey(roomID, identifierHash)
	member, ok := r.members[key]
	if !ok {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	member.JoinedAt = joinedAt
	member.LeftAt = nil
	r.members[key] = member
	return nil
}

func (r *roomMemoryRoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := roomTestMemberRecordKey(roomID, identifierHash)
	member, ok := r.members[key]
	if !ok {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	leftAtCopy := leftAt
	member.LeftAt = &leftAtCopy
	r.members[key] = member
	return nil
}

func (r *roomMemoryRoomRepository) mustFindRoom(t *testing.T, roomID string) domain.Room {
	t.Helper()

	room, err := r.FindByRoomID(context.Background(), roomID)
	if err != nil {
		t.Fatalf("expected room to exist: %v", err)
	}
	return *room
}

func (r *roomMemoryRoomRepository) mustFindMember(t *testing.T, roomID string, identifierHash string) domain.RoomMember {
	t.Helper()

	member, err := r.FindMember(context.Background(), roomID, identifierHash)
	if err != nil {
		t.Fatalf("expected room member to exist: %v", err)
	}
	return *member
}

type memoryMessageRepository struct {
	mu     sync.Mutex
	counts map[string]int64
}

func newMemoryMessageRepository() *memoryMessageRepository {
	return &memoryMessageRepository{
		counts: map[string]int64{},
	}
}

func (r *memoryMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counts[message.RoomID]++
	return nil
}

func (r *memoryMessageRepository) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	return nil, nil
}

func (r *memoryMessageRepository) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.counts[roomID], nil
}

func roomTestMemberRecordKey(roomID string, identifierHash string) string {
	return roomID + "\x00" + identifierHash
}

func cloneRoomForRoomTest(room domain.Room) *domain.Room {
	copy := room
	return &copy
}

func cloneRoomMemberForRoomTest(member domain.RoomMember) *domain.RoomMember {
	copy := member
	return &copy
}

type roomScriptedRoomRepository struct {
	findByRoomIDFunc        func(ctx context.Context, roomID string) (*domain.Room, error)
	ensureRoomWithOwnerFunc func(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error)
	updateFunc              func(ctx context.Context, room *domain.Room) error
	addMemberFunc           func(ctx context.Context, member *domain.RoomMember) error
	findMemberFunc          func(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error)
	reactivateMemberFunc    func(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error
	markMemberLeftFunc      func(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error
}

func (r *roomScriptedRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	return nil
}

func (r *roomScriptedRoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	if r.findByRoomIDFunc == nil {
		return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	return r.findByRoomIDFunc(ctx, roomID)
}

func (r *roomScriptedRoomRepository) EnsureRoomWithOwnerMember(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
	if r.ensureRoomWithOwnerFunc == nil {
		return true, nil
	}
	return r.ensureRoomWithOwnerFunc(ctx, room, member)
}

func (r *roomScriptedRoomRepository) Update(ctx context.Context, room *domain.Room) error {
	if r.updateFunc == nil {
		return nil
	}
	return r.updateFunc(ctx, room)
}

func (r *roomScriptedRoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	if r.addMemberFunc == nil {
		return nil
	}
	return r.addMemberFunc(ctx, member)
}

func (r *roomScriptedRoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	if r.findMemberFunc == nil {
		return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	return r.findMemberFunc(ctx, roomID, identifierHash)
}

func (r *roomScriptedRoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	if r.reactivateMemberFunc == nil {
		return nil
	}
	return r.reactivateMemberFunc(ctx, roomID, identifierHash, joinedAt)
}

func (r *roomScriptedRoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	if r.markMemberLeftFunc == nil {
		return nil
	}
	return r.markMemberLeftFunc(ctx, roomID, identifierHash, leftAt)
}

type scriptedMessageRepository struct {
	countByRoomIDFunc func(ctx context.Context, roomID string) (int64, error)
}

func (r *scriptedMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	return nil
}

func (r *scriptedMessageRepository) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	return nil, nil
}

func (r *scriptedMessageRepository) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	if r.countByRoomIDFunc == nil {
		return 0, nil
	}
	return r.countByRoomIDFunc(ctx, roomID)
}
