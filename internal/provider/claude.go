package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ClaudeProvider struct {
	credsPath string
	endpoint  string
	cache     *claudeCache
	mu        sync.RWMutex
	// cachePath 用于测试隔离；为空时使用默认路径 ~/.config/token-usage/claude_cache.json
	cachePath string
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

	// 调试日志：当 TOKEN_USAGE_DEBUG=1 时保存原始响应（含状态码、响应头、响应体）
	// 便于排查 API 返回结构变化（不会发起额外请求，仅落盘本次已有的响应）
	if os.Getenv("TOKEN_USAGE_DEBUG") != "" {
		p.dumpRawResponse(resp, respBody)
	}

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

	// Fallback：若 JSON body 缺少 5h/7d 字段（API 结构变更或返回为空），
	// 退回解析响应头 anthropic-ratelimit-unified-* （旧版 API 在响应头中返回这些值）
	if usage.Rolling.ResetAt.IsZero() {
		if w := parseClaudeHeaderWindow(resp.Header, "5h"); w != nil {
			usage.Rolling = *w
		}
	}
	if usage.Weekly.ResetAt.IsZero() {
		if w := parseClaudeHeaderWindow(resp.Header, "7d"); w != nil {
			usage.Weekly = *w
		}
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

// parseClaudeHeaderWindow 从响应头解析 5h/7d 配额窗口。
// 旧版 API 把利用率放在 anthropic-ratelimit-unified-{5h|7d}-utilization 头里
// （值为 0~1 的小数），重置时间放在 -reset 头里（unix 秒/毫秒）。
// 当 JSON body 没有返回对应字段时用作兜底。
func parseClaudeHeaderWindow(header http.Header, window string) *QuotaWindow {
	v := header.Get("anthropic-ratelimit-unified-" + window + "-utilization")
	if v == "" {
		return nil
	}
	percent, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	resetAt := parseUnixTimestamp(header.Get("anthropic-ratelimit-unified-" + window + "-reset"))
	status := header.Get("anthropic-ratelimit-unified-" + window + "-status")
	if status == "" {
		status = "ok"
	}
	return &QuotaWindow{
		Percent: int(percent * 100),
		Status:  status,
		ResetAt: resetAt,
	}
}

// dumpRawResponse 把原始响应（状态码 + 响应头 + 响应体）落盘到调试日志文件，
// 仅在 TOKEN_USAGE_DEBUG 环境变量非空时启用。文件每次请求会被覆盖。
func (p *ClaudeProvider) dumpRawResponse(resp *http.Response, body []byte) {
	logPath := p.getDebugLogPath()
	var b strings.Builder
	fmt.Fprintf(&b, "Time: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "Status: %s\n", resp.Status)
	fmt.Fprintln(&b, "Headers:")
	for k, vs := range resp.Header {
		for _, v := range vs {
			fmt.Fprintf(&b, "  %s: %s\n", k, v)
		}
	}
	fmt.Fprintln(&b, "Body:")
	b.Write(body)
	_ = os.MkdirAll(filepath.Dir(logPath), 0700)
	_ = os.WriteFile(logPath, []byte(b.String()), 0600)
}

func (p *ClaudeProvider) getDebugLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "token-usage", "claude_debug.log")
}

func (p *ClaudeProvider) getCachePath() string {
	if p.cachePath != "" {
		return p.cachePath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "token-usage", "claude_cache.json")
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
		p.invalidateCache()
		return nil
	}

	// 验证缓存有效性
	if cached.Usage == nil {
		p.invalidateCache()
		return nil
	}

	// 检查 resetAt 是否是零值时间（无效数据）
	if cached.Usage.Rolling.ResetAt.Year() < 2020 || cached.Usage.Weekly.ResetAt.Year() < 2020 {
		p.invalidateCache()
		return nil
	}

	// 缓存5分钟内有效
	if time.Since(cached.CachedAt) > 5*time.Minute {
		return nil
	}

	return cached.Usage
}

func (p *ClaudeProvider) invalidateCache() {
	os.Remove(p.getCachePath())
	p.mu.Lock()
	p.cache = nil
	p.mu.Unlock()
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
