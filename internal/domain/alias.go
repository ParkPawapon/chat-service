package domain

import "time"

type ClientAlias struct {
	ID             string
	RoomID         string
	IdentifierHash string
	Alias          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
