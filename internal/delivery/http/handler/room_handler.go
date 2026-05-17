package handler

import (
	"net/http"
	"strings"

	"chat-service/internal/delivery/http/dto"
	"chat-service/internal/delivery/http/response"
	"chat-service/internal/domain"
	"chat-service/internal/usecase"
	appvalidator "chat-service/pkg/validator"
)

type RoomHandler struct {
	useCase   *usecase.RoomUseCase
	validator *appvalidator.Validator
}

func NewRoomHandler(useCase *usecase.RoomUseCase, validator *appvalidator.Validator) *RoomHandler {
	return &RoomHandler{useCase: useCase, validator: validator}
}

func (h *RoomHandler) Action(w http.ResponseWriter, r *http.Request) {
	var req dto.RoomActionRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		response.ValidationError(w, err)
		return
	}

	input := usecase.RoomActionInput{
		Identifier: req.Identifier,
		RoomID:     req.RoomID,
		Force:      req.Force,
	}

	switch req.Action {
	case "join":
		output, err := h.useCase.JoinRoom(r.Context(), input)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, http.StatusOK, dto.JoinRoomResponse{
			IsDestroyed: output.IsDestroyed,
			IsOwner:     output.IsOwner,
		})
	case "destroy":
		output, err := h.useCase.DestroyRoom(r.Context(), input)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, http.StatusOK, dto.DestroyRoomResponse{IsDestroyed: output.IsDestroyed})
	case "leave":
		output, err := h.useCase.LeaveRoom(r.Context(), input)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, http.StatusOK, dto.LeaveRoomResponse{HasLeft: output.HasLeft})
	default:
		response.Error(w, domain.NewAppError(domain.ErrInvalidInput, "unsupported room action"))
	}
}

func (h *RoomHandler) Status(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimSpace(r.URL.Query().Get("roomId"))
	if roomID == "" {
		response.Error(w, domain.NewAppError(domain.ErrInvalidInput, "roomId is required"))
		return
	}

	output, err := h.useCase.GetStatus(r.Context(), roomID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.RoomStatusResponse{
		ExpiresAt:    output.ExpiresAt.UTC().Format(response.ISOTimeLayout),
		IsDestroyed:  output.IsDestroyed,
		MessageCount: output.MessageCount,
		ServerTime:   output.ServerTime.UTC().Format(response.ISOTimeLayout),
	})
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	output, err := h.useCase.ListRooms(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	rooms := make([]dto.RoomSummaryResponse, 0, len(output.Rooms))
	for _, room := range output.Rooms {
		rooms = append(rooms, dto.RoomSummaryResponse{
			RoomID:      room.RoomID,
			ExpiresAt:   room.ExpiresAt.UTC().Format(response.ISOTimeLayout),
			IsDestroyed: room.IsDestroyed,
			CreatedAt:   room.CreatedAt.UTC().Format(response.ISOTimeLayout),
			UpdatedAt:   room.UpdatedAt.UTC().Format(response.ISOTimeLayout),
		})
	}

	response.JSON(w, http.StatusOK, dto.ListRoomsResponse{
		Rooms:      rooms,
		ServerTime: output.ServerTime.UTC().Format(response.ISOTimeLayout),
	})
}
