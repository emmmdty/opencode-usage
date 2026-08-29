package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProviderInterface(t *testing.T) {
	var _ Provider = (*ClaudeProvider)(nil)
	var _ Provider = (*CodexProvider)(nil)
	var _ Provider = (*OpenCodeProvider)(nil)
}

func TestClaudeProvider_GetUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 OAuth header
		if r.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
			t.Errorf("expected anthropic-beta header, got %s", r.Header.Get("anthropic-beta"))
		}

		// 设置 rate limit headers
		w.Header().Set("anthropic-ratelimit-unified-5h-utilization", "0.35")
		w.Header().Set("anthropic-ratelimit-unified-5h-reset", time.Now().Add(3*time.Hour).Format("1136214245"))
		w.Header().Set("anthropic-ratelimit-unified-5h-status", "active")
		w.Header().Set("anthropic-ratelimit-unified-7d-utilization", "0.12")
		w.Header().Set("anthropic-ratelimit-unified-7d-reset", time.Now().Add(5*24*time.Hour).Format("1136214245"))
		w.Header().Set("anthropic-ratelimit-unified-status", "allowed")
		w.Header().Set("Content-Type", "application/json")

		// 返回一个最小响应
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":          "test",
			"type":        "message",
			"role":        "assistant",
			"content":     []map[string]interface{}{},
			"model":       "claude-haiku-4-5-20251001",
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		})
	}))
	defer server.Close()

	// 创建临时 credentials 文件
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, ".credentials.json")
	creds := map[string]interface{}{
		"claudeAiOauth": map[string]interface{}{
			"accessToken":  "sk-ant-oat01-test-token",
			"refreshToken": "sk-ant-ort01-test-refresh",
			"expiresAt":    time.Now().Add(1 * time.Hour).UnixMilli(),
		},
	}
	data, _ := json.Marshal(creds)
	os.WriteFile(credsPath, data, 0600)

	provider := NewClaudeProviderWithEndpoint(credsPath, server.URL)
	usage, err := provider.GetUsage()
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}

	if usage.Provider != "claude" {
		t.Errorf("expected provider 'claude', got '%s'", usage.Provider)
	}

	if usage.Rolling.Percent != 35 {
		t.Errorf("expected rolling percent 35, got %d", usage.Rolling.Percent)
	}

	if usage.Weekly.Percent != 12 {
		t.Errorf("expected weekly percent 12, got %d", usage.Weekly.Percent)
	}
}

func TestClaudeProvider_IsAvailable(t *testing.T) {
	tmpDir := t.TempDir()

	// 没有凭证文件
	provider := NewClaudeProviderWithEndpoint(filepath.Join(tmpDir, "nonexistent"), "")
	if provider.IsAvailable() {
		t.Error("expected IsAvailable to return false when no credentials")
	}

	// 有凭证文件
	credsPath := filepath.Join(tmpDir, ".credentials.json")
	creds := map[string]interface{}{
		"claudeAiOauth": map[string]interface{}{
			"accessToken":  "test",
			"refreshToken": "test",
			"expiresAt":    time.Now().Add(1 * time.Hour).UnixMilli(),
		},
	}
	data, _ := json.Marshal(creds)
	os.WriteFile(credsPath, data, 0600)

	provider = NewClaudeProviderWithEndpoint(credsPath, "")
	if !provider.IsAvailable() {
		t.Error("expected IsAvailable to return true when credentials exist")
	}
}

func TestCodexProvider_GetUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plan_type": "plus",
			"rate_limit": map[string]interface{}{
				"allowed":       true,
				"limit_reached": false,
				"primary_window": map[string]interface{}{
					"used_percent":        45,
					"reset_after_seconds": 7200,
					"reset_at":            time.Now().Add(2 * time.Hour).Unix(),
				},
				"secondary_window": map[string]interface{}{
					"used_percent":        24,
					"reset_after_seconds": 345600,
					"reset_at":            time.Now().Add(4 * 24 * time.Hour).Unix(),
				},
			},
		})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	authPath := filepath.Join(tmpDir, "auth.json")
	auth := map[string]interface{}{
		"tokens": map[string]interface{}{
			"access_token":  "test-access-token",
			"refresh_token": "test-refresh-token",
		},
	}
	data, _ := json.Marshal(auth)
	os.WriteFile(authPath, data, 0600)

	provider := NewCodexProviderWithEndpoint(authPath, server.URL)
	usage, err := provider.GetUsage()
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}

	if usage.Provider != "codex" {
		t.Errorf("expected provider 'codex', got '%s'", usage.Provider)
	}

	if usage.PlanType != "plus" {
		t.Errorf("expected plan type 'plus', got '%s'", usage.PlanType)
	}

	if usage.Rolling.Percent != 45 {
		t.Errorf("expected rolling percent 45, got %d", usage.Rolling.Percent)
	}

	if usage.Weekly.Percent != 24 {
		t.Errorf("expected weekly percent 24, got %d", usage.Weekly.Percent)
	}
}

func TestOpenCodeProvider_GetUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"usage": map[string]interface{}{
				"rolling": map[string]interface{}{
					"status":   "ok",
					"percent":  60,
					"resetsAt": time.Now().Add(3 * time.Hour).Format(time.RFC3339),
				},
				"weekly": map[string]interface{}{
					"status":   "ok",
					"percent":  25,
					"resetsAt": time.Now().Add(3 * 24 * time.Hour).Format(time.RFC3339),
				},
				"monthly": map[string]interface{}{
					"status":   "ok",
					"percent":  10,
					"resetsAt": time.Now().Add(20 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		})
	}))
	defer server.Close()

	provider := NewOpenCodeProviderWithEndpoint("test-api-key", server.URL)
	usage, err := provider.GetUsage()
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}

	if usage.Provider != "opencode" {
		t.Errorf("expected provider 'opencode', got '%s'", usage.Provider)
	}

	if usage.Rolling.Percent != 60 {
		t.Errorf("expected rolling percent 60, got %d", usage.Rolling.Percent)
	}
}
