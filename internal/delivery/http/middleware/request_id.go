package middleware

import (
	"context"
	"net/http"

	"chat-service/pkg/idgen"
)

type requestIDKey struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = idgen.NewUUID()
		}

		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	value, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return ""
	}
	return value
}
