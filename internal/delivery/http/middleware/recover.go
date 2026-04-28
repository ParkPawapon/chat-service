package middleware

import (
	"log/slog"
	"net/http"

	"chat-service/internal/delivery/http/response"
)

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Error("panic recovered", "panic", recovered, "path", r.URL.Path, "request_id", RequestIDFromContext(r.Context()))
					response.JSON(w, http.StatusInternalServerError, response.ErrorResponse{
						Error: response.ErrorBody{
							Code:    "internal_error",
							Message: "internal server error",
						},
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
