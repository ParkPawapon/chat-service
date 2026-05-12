package usecase

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"chat-service/internal/domain"
	"chat-service/pkg/idgen"
)

type memoryRoomRepository struct {
	mu      sync.Mutex
	rooms   map[string]domain.Room
	members map[string]domain.RoomMember
}

func newMemoryRoomRepository() *memoryRoomRepository {
	return &memoryRoomRepository{
		rooms:   map[string]domain.Room{},
		members: map[string]domain.RoomMember{},
	}
}

func (r *memoryRoomRepository) Create(ctx context.Context, room *domain.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rooms[room.RoomID]; exists {
		return domain.NewAppError(domain.ErrConflict, "room already exists")
	}
	r.rooms[room.RoomID] = *cloneRoom(*room)
	return nil
}

func (r *memoryRoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	room, ok := r.rooms[roomID]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	return cloneRoom(room), nil
}

func (r *memoryRoomRepository) EnsureRoomWithOwnerMember(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.rooms[room.RoomID]; ok {
		*room = *cloneRoom(existing)
		return false, nil
	}

	r.rooms[room.RoomID] = *cloneRoom(*room)
	r.members[roomMemberRecordKey(member.RoomID, member.IdentifierHash)] = *cloneRoomMember(*member)
	return true, nil
}

func (r *memoryRoomRepository) Update(ctx context.Context, room *domain.Room) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.rooms[room.RoomID]; !ok {
		return domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	r.rooms[room.RoomID] = *cloneRoom(*room)
	return nil
}

func (r *memoryRoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := roomMemberRecordKey(member.RoomID, member.IdentifierHash)
	if _, exists := r.members[key]; exists {
		return domain.NewAppError(domain.ErrConflict, "room member already exists")
	}
	r.members[key] = *cloneRoomMember(*member)
	return nil
}

func (r *memoryRoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	member, ok := r.members[roomMemberRecordKey(roomID, identifierHash)]
	if !ok {
		return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	return cloneRoomMember(member), nil
}

func (r *memoryRoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := roomMemberRecordKey(roomID, identifierHash)
	member, ok := r.members[key]
	if !ok {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	member.JoinedAt = joinedAt
	member.LeftAt = nil
	r.members[key] = member
	return nil
}

func (r *memoryRoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := roomMemberRecordKey(roomID, identifierHash)
	member, ok := r.members[key]
	if !ok {
		return domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	leftAtCopy := leftAt
	member.LeftAt = &leftAtCopy
	r.members[key] = member
	return nil
}

type scriptedRoomRepository struct {
	findByRoomIDFunc        func(ctx context.Context, roomID string) (*domain.Room, error)
	ensureRoomWithOwnerFunc func(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error)
	updateFunc              func(ctx context.Context, room *domain.Room) error
	addMemberFunc           func(ctx context.Context, member *domain.RoomMember) error
	findMemberFunc          func(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error)
	reactivateMemberFunc    func(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error
	markMemberLeftFunc      func(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error
}

func (r *scriptedRoomRepository) Create(ctx context.Context, room *domain.Room) error { return nil }

func (r *scriptedRoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error) {
	if r.findByRoomIDFunc == nil {
		return nil, domain.NewAppError(domain.ErrNotFound, "room not found")
	}
	return r.findByRoomIDFunc(ctx, roomID)
}

func (r *scriptedRoomRepository) EnsureRoomWithOwnerMember(ctx context.Context, room *domain.Room, member *domain.RoomMember) (bool, error) {
	if r.ensureRoomWithOwnerFunc == nil {
		return true, nil
	}
	return r.ensureRoomWithOwnerFunc(ctx, room, member)
}

func (r *scriptedRoomRepository) Update(ctx context.Context, room *domain.Room) error {
	if r.updateFunc == nil {
		return nil
	}
	return r.updateFunc(ctx, room)
}

func (r *scriptedRoomRepository) AddMember(ctx context.Context, member *domain.RoomMember) error {
	if r.addMemberFunc == nil {
		return nil
	}
	return r.addMemberFunc(ctx, member)
}

func (r *scriptedRoomRepository) FindMember(ctx context.Context, roomID string, identifierHash string) (*domain.RoomMember, error) {
	if r.findMemberFunc == nil {
		return nil, domain.NewAppError(domain.ErrNotFound, "room member not found")
	}
	return r.findMemberFunc(ctx, roomID, identifierHash)
}

func (r *scriptedRoomRepository) ReactivateMember(ctx context.Context, roomID string, identifierHash string, joinedAt time.Time) error {
	if r.reactivateMemberFunc == nil {
		return nil
	}
	return r.reactivateMemberFunc(ctx, roomID, identifierHash, joinedAt)
}

func (r *scriptedRoomRepository) MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error {
	if r.markMemberLeftFunc == nil {
		return nil
	}
	return r.markMemberLeftFunc(ctx, roomID, identifierHash, leftAt)
}

type memoryMessageStore struct {
	mu       sync.Mutex
	messages map[string][]domain.Message
	nextID   int
}

func newMemoryMessageStore() *memoryMessageStore {
	return &memoryMessageStore{
		messages: map[string][]domain.Message{},
	}
}

func (r *memoryMessageStore) Create(ctx context.Context, message *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	copy := *message
	if copy.ID == "" {
		copy.ID = messageRecordID(r.nextID)
	}
	r.messages[copy.RoomID] = append(r.messages[copy.RoomID], copy)
	*message = copy
	return nil
}

func (r *memoryMessageStore) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored := r.messages[roomID]
	result := make([]domain.Message, 0, len(stored))
	for _, message := range stored {
		result = append(result, *cloneMessage(message))
	}
	return result, nil
}

func (r *memoryMessageStore) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return int64(len(r.messages[roomID])), nil
}

func (r *memoryMessageStore) mustList(t *testing.T, roomID string) []domain.Message {
	t.Helper()

	messages, err := r.ListByRoomID(context.Background(), roomID)
	if err != nil {
		t.Fatalf("expected messages to exist: %v", err)
	}
	return messages
}

type scriptedMessageStore struct {
	createFunc        func(ctx context.Context, message *domain.Message) error
	listByRoomIDFunc  func(ctx context.Context, roomID string) ([]domain.Message, error)
	countByRoomIDFunc func(ctx context.Context, roomID string) (int64, error)
}

func (r *scriptedMessageStore) Create(ctx context.Context, message *domain.Message) error {
	if r.createFunc == nil {
		return nil
	}
	return r.createFunc(ctx, message)
}

func (r *scriptedMessageStore) ListByRoomID(ctx context.Context, roomID string) ([]domain.Message, error) {
	if r.listByRoomIDFunc == nil {
		return nil, nil
	}
	return r.listByRoomIDFunc(ctx, roomID)
}

func (r *scriptedMessageStore) CountByRoomID(ctx context.Context, roomID string) (int64, error) {
	if r.countByRoomIDFunc == nil {
		return 0, nil
	}
	return r.countByRoomIDFunc(ctx, roomID)
}

type memoryMessagePubSub struct {
	mu            sync.Mutex
	published     map[string][]domain.Message
	subscriptions map[string][]chan domain.Message
	publishErr    error
	subscribeErr  error
}

func newMemoryMessagePubSub() *memoryMessagePubSub {
	return &memoryMessagePubSub{
		published:     map[string][]domain.Message{},
		subscriptions: map[string][]chan domain.Message{},
	}
}

func (p *memoryMessagePubSub) PublishMessage(ctx context.Context, roomID string, message domain.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.publishErr != nil {
		return p.publishErr
	}

	copy := *cloneMessage(message)
	p.published[roomID] = append(p.published[roomID], copy)
	for _, ch := range p.subscriptions[roomID] {
		select {
		case ch <- copy:
		default:
		}
	}
	return nil
}

func (p *memoryMessagePubSub) SubscribeMessages(ctx context.Context, roomID string) (domain.MessageSubscription, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.subscribeErr != nil {
		return nil, p.subscribeErr
	}

	ch := make(chan domain.Message, 4)
	p.subscriptions[roomID] = append(p.subscriptions[roomID], ch)
	return &memoryMessageSubscription{messages: ch}, nil
}

func (p *memoryMessagePubSub) mustPublished(t *testing.T, roomID string) []domain.Message {
	t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()

	stored := p.published[roomID]
	result := make([]domain.Message, 0, len(stored))
	for _, message := range stored {
		result = append(result, *cloneMessage(message))
	}
	return result
}

type memoryMessageSubscription struct {
	messages chan domain.Message
	closed   bool
}

func (s *memoryMessageSubscription) Messages() <-chan domain.Message {
	return s.messages
}

func (s *memoryMessageSubscription) Close() error {
	if !s.closed {
		close(s.messages)
		s.closed = true
	}
	return nil
}

type scriptedMessagePubSub struct {
	publishMessageFunc    func(ctx context.Context, roomID string, message domain.Message) error
	subscribeMessagesFunc func(ctx context.Context, roomID string) (domain.MessageSubscription, error)
}

func (p *scriptedMessagePubSub) PublishMessage(ctx context.Context, roomID string, message domain.Message) error {
	if p.publishMessageFunc == nil {
		return nil
	}
	return p.publishMessageFunc(ctx, roomID, message)
}

func (p *scriptedMessagePubSub) SubscribeMessages(ctx context.Context, roomID string) (domain.MessageSubscription, error) {
	if p.subscribeMessagesFunc == nil {
		return &memoryMessageSubscription{messages: make(chan domain.Message)}, nil
	}
	return p.subscribeMessagesFunc(ctx, roomID)
}

type recordingMessageLogger struct {
	mu       sync.Mutex
	warnings []string
}

func (l *recordingMessageLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnings = append(l.warnings, msg)
}

func (l *recordingMessageLogger) warningCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.warnings)
}

func messageRecordID(n int) string {
	return "message-" + strconv.Itoa(n)
}

func cloneMessage(message domain.Message) *domain.Message {
	copy := message
	return &copy
}

func roomMemberRecordKey(roomID string, identifierHash string) string {
	return roomID + "\x00" + identifierHash
}

func cloneRoom(room domain.Room) *domain.Room {
	copy := room
	return &copy
}

func cloneRoomMember(member domain.RoomMember) *domain.RoomMember {
	copy := member
	return &copy
}

func seedRoomMember(t *testing.T, rooms *memoryRoomRepository, roomID string, identifier string, isDestroyed bool, hasLeft bool) {
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
	rooms.members[roomMemberRecordKey(roomID, identifierHash)] = member
}
