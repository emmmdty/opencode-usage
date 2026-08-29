package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVolcengineProvider_GetUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 API Key
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plan_type": "agent_plan_medium",
			"5h": map[string]interface{}{
				"limit":   100,
				"used":    45,
				"percent": 45,
			},
			"weekly": map[string]interface{}{
				"limit":   500,
				"used":    120,
				"percent": 24,
			},
		})
	}))
	defer server.Close()

	provider := NewVolcengineProviderWithEndpoint("test-api-key", server.URL)
	usage, err := provider.GetUsage()
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}

	if usage.Provider != "volcengine" {
		t.Errorf("expected provider 'volcengine', got '%s'", usage.Provider)
	}

	if usage.Rolling.Percent != 45 {
		t.Errorf("expected rolling percent 45, got %d", usage.Rolling.Percent)
	}

	if usage.Weekly.Percent != 24 {
		t.Errorf("expected weekly percent 24, got %d", usage.Weekly.Percent)
	}
}

func TestVolcengineProvider_IsAvailable(t *testing.T) {
	provider := NewVolcengineProvider("")
	if provider.IsAvailable() {
		t.Error("expected IsAvailable to return false when no API key")
	}

	provider = NewVolcengineProvider("test-key")
	if !provider.IsAvailable() {
		t.Error("expected IsAvailable to return true when API key exists")
	}
}
