package domain

import "time"

type Room struct {
	ID                  string
	RoomID              string
	OwnerIdentifierHash string
	IsDestroyed         bool
	ExpiresAt           time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type RoomMember struct {
	ID             string
	RoomID         string
	IdentifierHash string
	JoinedAt       time.Time
	LeftAt         *time.Time
}

type RoomStatus struct {
	RoomID       string
	ExpiresAt    time.Time
	IsDestroyed  bool
	MessageCount int64
	ServerTime   time.Time
}
