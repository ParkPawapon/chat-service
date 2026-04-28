package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"chat-service/internal/domain"
	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func Error(w http.ResponseWriter, err error) {
	status, body := errorBody(err)
	JSON(w, status, ErrorResponse{Error: body})
}

func ValidationError(w http.ResponseWriter, err error) {
	fields := map[string]string{}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldError := range validationErrors {
			fields[fieldError.Field()] = validationMessage(fieldError)
		}
	}

	JSON(w, http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{
			Code:    "invalid_request",
			Message: "request validation failed",
			Fields:  fields,
		},
	})
}

func errorBody(err error) (int, ErrorBody) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, ErrorBody{Code: "invalid_request", Message: messageOrDefault(err, "invalid request")}
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, ErrorBody{Code: "forbidden", Message: messageOrDefault(err, "forbidden")}
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, ErrorBody{Code: "not_found", Message: messageOrDefault(err, "resource not found")}
	case errors.Is(err, domain.ErrGone):
		return http.StatusGone, ErrorBody{Code: "gone", Message: messageOrDefault(err, "resource is no longer available")}
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, ErrorBody{Code: "conflict", Message: messageOrDefault(err, "resource conflict")}
	case errors.Is(err, domain.ErrNotImplemented):
		return http.StatusNotImplemented, ErrorBody{Code: "not_implemented", Message: messageOrDefault(err, "not implemented")}
	default:
		return http.StatusInternalServerError, ErrorBody{Code: "internal_error", Message: "internal server error"}
	}
}

func messageOrDefault(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	var appErr *domain.AppError
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return fallback
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "oneof":
		return "has an unsupported value"
	default:
		return "is invalid"
	}
}

func DecodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return domain.WrapAppError(domain.ErrInvalidInput, "invalid JSON request body", err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domain.NewAppError(domain.ErrInvalidInput, "request body must contain a single JSON object")
	}

	return nil
}
