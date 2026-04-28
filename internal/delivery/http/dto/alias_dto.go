package dto

type ClientAliasRequest struct {
	Identifier string `json:"identifier" validate:"required"`
	RoomID     string `json:"roomId" validate:"required"`
}

type ClientAliasResponse struct {
	Alias string `json:"alias"`
}
