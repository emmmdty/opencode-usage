package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type ClaudeProvider struct {
	credsPath string
	endpoint  string
	cache     *claudeCache
	mu        sync.RWMutex
}

type claudeCache struct {
	usage     *Usage
	expiresAt time.Time
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
	// 检查缓存（5分钟内有效）
	p.mu.RLock()
	if p.cache != nil && time.Now().Before(p.cache.expiresAt) {
		cached := p.cache.usage
		p.mu.RUnlock()
		return cached, nil
	}
	p.mu.RUnlock()

	creds, err := p.loadCredentials()
	if err != nil {
		return nil, err
	}

	// 检查 token 是否过期
	if creds.ClaudeAiOauth.ExpiresAt > 0 && time.Now().UnixMilli() > creds.ClaudeAiOauth.ExpiresAt {
		return nil, fmt.Errorf("token expired, run 'claude' to refresh")
	}

	// 使用正确的 /api/oauth/usage 端点
	req, err := http.NewRequest("GET", p.endpoint+"/api/oauth/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+creds.ClaudeAiOauth.AccessToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// 处理429 rate limit
	if resp.StatusCode == http.StatusTooManyRequests {
		if cached := p.loadCachedUsage(); cached != nil {
			return cached, nil
		}
		return nil, fmt.Errorf("rate limited (429), try again later")
	}

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		FiveHour struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		} `json:"seven_day"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	usage := &Usage{
		Provider: "claude",
		PlanType: "subscription",
		Rolling: QuotaWindow{
			Percent: int(result.FiveHour.Utilization),
			Status:  "ok",
			ResetAt: parseClaudeTime(result.FiveHour.ResetsAt),
		},
		Weekly: QuotaWindow{
			Percent: int(result.SevenDay.Utilization),
			Status:  "ok",
			ResetAt: parseClaudeTime(result.SevenDay.ResetsAt),
		},
	}

	// 缓存结果
	p.mu.Lock()
	p.cache = &claudeCache{
		usage:     usage,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	p.mu.Unlock()

	p.saveCachedUsage(usage)

	return usage, nil
}

func (p *ClaudeProvider) GetDefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", ".credentials.json")
}

func parseUnixTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		if ts > 1000000000000 {
			ts = ts / 1000
		}
		return time.Unix(ts, 0)
	}
	return time.Time{}
}

func parseClaudeTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Claude 使用 ISO 8601 格式: 2026-08-29T04:19:59.891014+00:00
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (p *ClaudeProvider) getCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "opencode-usage", "claude_cache.json")
}

func (p *ClaudeProvider) loadCachedUsage() *Usage {
	data, err := os.ReadFile(p.getCachePath())
	if err != nil {
		return nil
	}

	var cached struct {
		Usage    *Usage    `json:"usage"`
		CachedAt time.Time `json:"cached_at"`
	}
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}

	// 缓存10分钟内有效
	if time.Since(cached.CachedAt) > 10*time.Minute {
		return nil
	}

	return cached.Usage
}

func (p *ClaudeProvider) saveCachedUsage(usage *Usage) {
	cached := struct {
		Usage    *Usage    `json:"usage"`
		CachedAt time.Time `json:"cached_at"`
	}{
		Usage:    usage,
		CachedAt: time.Now(),
	}

	data, _ := json.Marshal(cached)
	os.MkdirAll(filepath.Dir(p.getCachePath()), 0700)
	os.WriteFile(p.getCachePath(), data, 0600)
}
