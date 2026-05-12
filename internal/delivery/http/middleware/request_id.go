package middleware

import (
	"context"
	"net/http"
	"strings"

	"chat-service/pkg/idgen"
)

type requestIDKey struct{}

const maxRequestIDLength = 128

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := normalizeRequestID(r.Header.Get("X-Request-ID"))

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

func normalizeRequestID(value string) string {
	requestID := strings.TrimSpace(value)
	if requestID == "" || len(requestID) > maxRequestIDLength {
		return idgen.NewUUID()
	}
	for _, r := range requestID {
		if r < 33 || r > 126 {
			return idgen.NewUUID()
		}
	}
	return requestID
}
