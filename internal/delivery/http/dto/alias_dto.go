package dto

type ClientAliasRequest struct {
	Identifier string `json:"identifier" validate:"required,max=512"`
	RoomID     string `json:"roomId" validate:"required,max=128"`
}

type ClientAliasResponse struct {
	Alias string `json:"alias"`
}
