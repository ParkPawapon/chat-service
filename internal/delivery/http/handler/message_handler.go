package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"chat-service/internal/delivery/http/dto"
	"chat-service/internal/delivery/http/response"
	"chat-service/internal/domain"
	"chat-service/internal/usecase"
	appvalidator "chat-service/pkg/validator"
)

type MessageHandler struct {
	useCase   *usecase.MessageUseCase
	validator *appvalidator.Validator
}

func NewMessageHandler(useCase *usecase.MessageUseCase, validator *appvalidator.Validator) *MessageHandler {
	return &MessageHandler{useCase: useCase, validator: validator}
}

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMessageRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		response.ValidationError(w, err)
		return
	}

	output, err := h.useCase.CreateMessage(r.Context(), usecase.CreateMessageInput{
		Identifier: req.Identifier,
		RoomID:     req.RoomID,
		Body:       req.Body,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.MessageResponse{
		ID:         output.ID,
		Body:       output.Body,
		SenderName: output.SenderName,
		SentAt:     output.SentAt,
	})
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
	if roomID == "" {
		response.Error(w, domain.NewAppError(domain.ErrInvalidInput, "roomId is required"))
		return
	}

	output, err := h.useCase.ListMessages(r.Context(), roomID)
	if err != nil {
		response.Error(w, err)
		return
	}

	messages := make([]dto.MessageResponse, 0, len(output.Messages))
	for _, message := range output.Messages {
		messages = append(messages, dto.MessageResponse{
			ID:         message.ID,
			Body:       message.Body,
			SenderName: message.SenderName,
			SentAt:     message.SentAt,
		})
	}

	response.JSON(w, http.StatusOK, dto.ListMessagesResponse{Messages: messages})
}

func (h *MessageHandler) Stream(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
	if roomID == "" {
		response.Error(w, domain.NewAppError(domain.ErrInvalidInput, "roomId is required"))
		return
	}

	messages, err := h.useCase.StreamMessages(r.Context(), roomID)
	if err != nil {
		response.Error(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Error(w, domain.NewAppError(domain.ErrDependency, "streaming is not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for {
		select {
		case <-r.Context().Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}

			payload := dto.MessageResponse{
				ID:         message.ID,
				Body:       message.Body,
				SenderName: message.SenderName,
				SentAt:     message.SentAt.UTC().Format(response.ISOTimeLayout),
			}
			data, err := json.Marshal(payload)
			if err != nil {
				return
			}

			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
