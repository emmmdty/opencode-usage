package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CodexProvider struct {
	authPath string
	endpoint string
	// accessToken, when set, is used directly instead of reading the local
	// auth file (manual accounts).
	accessToken string
}

func NewCodexProvider(authPath string) *CodexProvider {
	return &CodexProvider{
		authPath: authPath,
		endpoint: "https://chatgpt.com",
	}
}

func NewCodexProviderWithEndpoint(authPath, endpoint string) *CodexProvider {
	p := NewCodexProvider(authPath)
	if endpoint != "" {
		p.endpoint = endpoint
	}
	return p
}

// NewCodexProviderWithToken builds a provider that uses a pasted OAuth
// access token instead of the local auth file.
func NewCodexProviderWithToken(accessToken, endpoint string) *CodexProvider {
	p := NewCodexProvider("")
	p.accessToken = accessToken
	if endpoint != "" {
		p.endpoint = endpoint
	}
	return p
}

func (p *CodexProvider) Name() string {
	return "codex"
}

func (p *CodexProvider) IsAvailable() bool {
	if p.accessToken != "" {
		return true
	}
	_, err := os.Stat(p.authPath)
	return err == nil
}

type codexAuth struct {
	Tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"tokens"`
}

func (p *CodexProvider) loadAuth() (*codexAuth, error) {
	if p.accessToken != "" {
		return &codexAuth{}, nil
	}
	data, err := os.ReadFile(p.authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	var auth codexAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	if auth.Tokens.AccessToken == "" {
		return nil, fmt.Errorf("no access token found")
	}

	return &auth, nil
}

func (p *CodexProvider) GetUsage() (*Usage, error) {
	auth, err := p.loadAuth()
	if err != nil {
		return nil, err
	}

	// 使用 wham/usage 端点（codex/usage 有 Cloudflare 保护）
	req, err := http.NewRequest("GET", p.endpoint+"/backend-api/wham/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token := p.accessToken
	if token == "" {
		token = auth.Tokens.AccessToken
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "codex-cli/0.58.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, _ := io.ReadAll(resp.Body)
	respStr := string(respBody)

	// 检查是否返回 HTML（通常是认证失败）
	if strings.Contains(respStr, "<html>") || strings.Contains(respStr, "<!DOCTYPE") {
		return nil, fmt.Errorf("token expired or invalid, run 'codex' to refresh")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, truncateStr(respStr, 100))
	}

	var result struct {
		PlanType  string `json:"plan_type"`
		RateLimit struct {
			PrimaryWindow struct {
				UsedPercent    float64 `json:"used_percent"`
				ResetAfterSecs int     `json:"reset_after_seconds"`
				ResetAt        int64   `json:"reset_at"`
			} `json:"primary_window"`
			SecondaryWindow struct {
				UsedPercent    float64 `json:"used_percent"`
				ResetAfterSecs int     `json:"reset_after_seconds"`
				ResetAt        int64   `json:"reset_at"`
			} `json:"secondary_window"`
		} `json:"rate_limit"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	usage := &Usage{
		Provider: "codex",
		PlanType: result.PlanType,
	}

	// Parse 5h window (primary)
	usage.Rolling = QuotaWindow{
		Percent: int(result.RateLimit.PrimaryWindow.UsedPercent),
		ResetAt: time.Unix(result.RateLimit.PrimaryWindow.ResetAt, 0),
		Status:  "ok",
	}

	// Parse 7d window (secondary)
	usage.Weekly = QuotaWindow{
		Percent: int(result.RateLimit.SecondaryWindow.UsedPercent),
		ResetAt: time.Unix(result.RateLimit.SecondaryWindow.ResetAt, 0),
		Status:  "ok",
	}

	return usage, nil
}

func (p *CodexProvider) GetDefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "auth.json")
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
