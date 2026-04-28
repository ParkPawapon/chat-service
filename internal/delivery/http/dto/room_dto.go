package dto

type RoomActionRequest struct {
	Action     string `json:"action" validate:"required,oneof=join destroy leave"`
	Identifier string `json:"identifier" validate:"required"`
	RoomID     string `json:"roomId" validate:"required"`
}

type JoinRoomResponse struct {
	IsDestroyed bool `json:"isDestroyed"`
	IsOwner     bool `json:"isOwner"`
}

type DestroyRoomResponse struct {
	IsDestroyed bool `json:"isDestroyed"`
}

type LeaveRoomResponse struct {
	HasLeft bool `json:"hasLeft"`
}

type RoomStatusResponse struct {
	ExpiresAt    string `json:"expiresAt"`
	IsDestroyed  bool   `json:"isDestroyed"`
	MessageCount int64  `json:"messageCount"`
	ServerTime   string `json:"serverTime"`
}
