package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalAuth(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"valid token", "s3cret", http.StatusOK},
		{"wrong token", "nope", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
		{"prefix of token", "s3c", http.StatusUnauthorized},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/feed/popular", nil)
			if tt.header != "" {
				req.Header.Set("X-Internal-Token", tt.header)
			}
			rec := httptest.NewRecorder()

			InternalAuth("s3cret")(next).ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Errorf("got %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
