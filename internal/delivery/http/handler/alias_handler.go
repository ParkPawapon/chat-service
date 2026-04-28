package handler

import (
	"net/http"

	"chat-service/internal/delivery/http/dto"
	"chat-service/internal/delivery/http/response"
	"chat-service/internal/usecase"
	appvalidator "chat-service/pkg/validator"
)

type AliasHandler struct {
	useCase   *usecase.AliasUseCase
	validator *appvalidator.Validator
}

func NewAliasHandler(useCase *usecase.AliasUseCase, validator *appvalidator.Validator) *AliasHandler {
	return &AliasHandler{useCase: useCase, validator: validator}
}

func (h *AliasHandler) GetOrCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.ClientAliasRequest
	if err := response.DecodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.validator.Struct(req); err != nil {
		response.ValidationError(w, err)
		return
	}

	output, err := h.useCase.GetOrCreateAlias(r.Context(), usecase.GetAliasInput{
		Identifier: req.Identifier,
		RoomID:     req.RoomID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ClientAliasResponse{Alias: output.Alias})
}
