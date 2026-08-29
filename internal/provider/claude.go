package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type ClaudeProvider struct {
	credsPath string
	endpoint  string
}

func NewClaudeProvider(credsPath string) *ClaudeProvider {
	return &ClaudeProvider{
		credsPath: credsPath,
		endpoint:  "https://api.anthropic.com",
	}
}

func NewClaudeProviderWithEndpoint(credsPath, endpoint string) *ClaudeProvider {
	p := NewClaudeProvider(credsPath)
	if endpoint != "" {
		p.endpoint = endpoint
	}
	return p
}

func (p *ClaudeProvider) Name() string {
	return "claude"
}

func (p *ClaudeProvider) IsAvailable() bool {
	_, err := os.Stat(p.credsPath)
	return err == nil
}

type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresAt    int64  `json:"expiresAt"`
	} `json:"claudeAiOauth"`
}

func (p *ClaudeProvider) loadCredentials() (*claudeCredentials, error) {
	data, err := os.ReadFile(p.credsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds claudeCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.ClaudeAiOauth.AccessToken == "" {
		return nil, fmt.Errorf("no access token found")
	}

	return &creds, nil
}

func (p *ClaudeProvider) GetUsage() (*Usage, error) {
	creds, err := p.loadCredentials()
	if err != nil {
		return nil, err
	}

	// 检查 token 是否过期，如果过期则刷新
	if creds.ClaudeAiOauth.ExpiresAt > 0 && time.Now().UnixMilli() > creds.ClaudeAiOauth.ExpiresAt {
		return nil, fmt.Errorf("token expired, please run 'claude' to refresh")
	}

	req, err := http.NewRequest("POST", p.endpoint+"/v1/messages", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.ClaudeAiOauth.AccessToken)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	body := map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "hi"}},
	}
	jsonBody, _ := json.Marshal(body)
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体用于调试
	respBody, _ := io.ReadAll(resp.Body)

	usage := &Usage{
		Provider: "claude",
		PlanType: "subscription",
	}

	// 检查是否成功
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(respBody))
	}

	// Parse 5h window
	if v := resp.Header.Get("anthropic-ratelimit-unified-5h-utilization"); v != "" {
		if percent, err := strconv.ParseFloat(v, 64); err == nil {
			resetTime := parseUnixTimestamp(resp.Header.Get("anthropic-ratelimit-unified-5h-reset"))
			usage.Rolling = QuotaWindow{
				Percent: int(percent * 100),
				Status:  resp.Header.Get("anthropic-ratelimit-unified-5h-status"),
				ResetAt: resetTime,
			}
		}
	}

	// Parse 7d window
	if v := resp.Header.Get("anthropic-ratelimit-unified-7d-utilization"); v != "" {
		if percent, err := strconv.ParseFloat(v, 64); err == nil {
			resetTime := parseUnixTimestamp(resp.Header.Get("anthropic-ratelimit-unified-7d-reset"))
			usage.Weekly = QuotaWindow{
				Percent: int(percent * 100),
				Status:  resp.Header.Get("anthropic-ratelimit-unified-status"),
				ResetAt: resetTime,
			}
		}
	}

	return usage, nil
}

func (p *ClaudeProvider) GetDefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", ".credentials.json")
}

// parseUnixTimestamp 解析 Unix 时间戳字符串
func parseUnixTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// 尝试解析为秒级时间戳
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		// 如果是毫秒级时间戳（大于 10^12），转换为秒
		if ts > 1000000000000 {
			ts = ts / 1000
		}
		return time.Unix(ts, 0)
	}
	return time.Time{}
}
