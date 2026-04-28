package domain

import "time"

type Message struct {
	ID                   string
	RoomID               string
	Body                 string
	SenderIdentifierHash string
	SenderName           string
	SentAt               time.Time
}

type MessageEvent struct {
	Message Message
}
