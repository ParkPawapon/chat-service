package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodyBytes(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader("123456"))
	recorder := httptest.NewRecorder()

	MaxBodyBytes(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err == nil {
			t.Fatal("expected oversized body read to fail")
		}
	})).ServeHTTP(recorder, request)
}
