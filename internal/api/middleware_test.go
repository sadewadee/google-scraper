package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAuthentication(t *testing.T) {
	// Helper to create a dummy handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		apiKey         string
		setupRequest   func(req *http.Request)
		expectedStatus int
	}{
		{
			name:   "No API Key set in env - allow access",
			apiKey: "",
			setupRequest: func(req *http.Request) {
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "API Key set - no auth provided",
			apiKey: "secret123",
			setupRequest: func(req *http.Request) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "API Key set - wrong auth provided",
			apiKey: "secret123",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer wrongsecret")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "API Key set - correct Bearer token",
			apiKey: "secret123",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer secret123")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "API Key set - correct X-API-Key header",
			apiKey: "secret123",
			setupRequest: func(req *http.Request) {
				req.Header.Set("X-API-Key", "secret123")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "API Key set - correct query param",
			apiKey: "secret123",
			setupRequest: func(req *http.Request) {
				q := req.URL.Query()
				q.Add("api_key", "secret123")
				req.URL.RawQuery = q.Encode()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current env and restore after test
			oldEnv := os.Getenv("API_KEY")
			defer os.Setenv("API_KEY", oldEnv)

			os.Setenv("API_KEY", tt.apiKey)

			req := httptest.NewRequest("GET", "/", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			// Create middleware chain
			// Auth returns a factory, so we call it with the apiKey
			middleware := Auth(os.Getenv("API_KEY"))
			handler := middleware(nextHandler)
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
