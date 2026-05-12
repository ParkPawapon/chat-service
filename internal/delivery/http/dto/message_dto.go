package dto

type CreateMessageRequest struct {
	Identifier string `json:"identifier" validate:"required,max=512"`
	RoomID     string `json:"roomId" validate:"required,max=128"`
	Body       string `json:"body" validate:"required,max=4096"`
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
