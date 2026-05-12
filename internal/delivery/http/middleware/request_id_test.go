package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestID(t *testing.T) {
	t.Run("keeps a valid request id", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		request.Header.Set("X-Request-ID", "request-123")
		recorder := httptest.NewRecorder()

		RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := RequestIDFromContext(r.Context()); got != "request-123" {
				t.Fatalf("expected request id to be preserved, got %q", got)
			}
		})).ServeHTTP(recorder, request)

		if got := recorder.Header().Get("X-Request-ID"); got != "request-123" {
			t.Fatalf("expected response request id to be preserved, got %q", got)
		}
	})

	t.Run("replaces unsafe request id", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		request.Header.Set("X-Request-ID", strings.Repeat("x", maxRequestIDLength+1))
		recorder := httptest.NewRecorder()

		RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := RequestIDFromContext(r.Context()); got == strings.Repeat("x", maxRequestIDLength+1) {
				t.Fatal("expected unsafe request id to be replaced")
			}
		})).ServeHTTP(recorder, request)

		if got := recorder.Header().Get("X-Request-ID"); got == strings.Repeat("x", maxRequestIDLength+1) || got == "" {
			t.Fatalf("expected generated response request id, got %q", got)
		}
	})
}
