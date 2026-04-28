package dto

type CreateMessageRequest struct {
	Identifier string `json:"identifier" validate:"required"`
	RoomID     string `json:"roomId" validate:"required"`
	Body       string `json:"body" validate:"required"`
}

type MessageResponse struct {
	ID         string `json:"id"`
	Body       string `json:"body"`
	SenderName string `json:"senderName"`
	SentAt     string `json:"sentAt"`
}

type ListMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
}
