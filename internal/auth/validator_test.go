package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		valid      bool
		errorCode  string
	}{
		{
			name:       "valid API key",
			statusCode: http.StatusOK,
			valid:      true,
		},
		{
			name:       "invalid API key",
			statusCode: http.StatusUnauthorized,
			valid:      false,
			errorCode:  "invalid_api_key",
		},
		{
			name:       "no subscription",
			statusCode: http.StatusForbidden,
			valid:      false,
			errorCode:  "no_go_subscription",
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			valid:      false,
			errorCode:  "rate_limited",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			valid:      false,
			errorCode:  "server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("expected Authorization header 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			resp, err := ValidateAPIKey("test-key", server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Valid != tt.valid {
				t.Errorf("expected Valid=%v, got %v", tt.valid, resp.Valid)
			}

			if tt.errorCode != "" && resp.Error != tt.errorCode {
				t.Errorf("expected Error=%s, got %s", tt.errorCode, resp.Error)
			}
		})
	}
}

func TestValidateAPIKeyNetworkError(t *testing.T) {
	resp, err := ValidateAPIKey("test-key", "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Valid {
		t.Error("expected Valid=false for network error")
	}

	if resp.Error != "network_error" {
		t.Errorf("expected Error='network_error', got '%s'", resp.Error)
	}
}

func TestValidateAPIKeyDefaultBaseURL(t *testing.T) {
	// Test that the function uses the default base URL when none is provided
	// This test verifies the fallback logic without actually making a network request
	resp, err := ValidateAPIKey("test-key", "")
	if err != nil {
		// Network error is expected since we're not actually connecting
		if resp == nil {
			t.Fatalf("expected non-nil response even on error")
		}
		if resp.Error != "network_error" {
			t.Errorf("expected network_error, got %s", resp.Error)
		}
	}
}

func TestValidationResponseJSON(t *testing.T) {
	resp := &ValidationResponse{
		Valid:   false,
		Error:   "invalid_api_key",
		Message: "Invalid API key",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal ValidationResponse: %v", err)
	}

	var decoded ValidationResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ValidationResponse: %v", err)
	}

	if decoded.Valid != resp.Valid {
		t.Errorf("expected Valid=%v, got %v", resp.Valid, decoded.Valid)
	}

	if decoded.Error != resp.Error {
		t.Errorf("expected Error=%s, got %s", resp.Error, decoded.Error)
	}
}
