package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultBaseURL = "https://opencode.ai/zen/go/v1"
	timeout        = 10 * time.Second
)

type ValidationResponse struct {
	Valid   bool
	Error   string
	Message string
}

func ValidateAPIKey(apiKey, baseURL string) (*ValidationResponse, error) {
	if baseURL == "" {
		baseURL = os.Getenv("OPENCODE_USAGE_BASE_URL")
		if baseURL == "" {
			baseURL = defaultBaseURL
		}
	}
	
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", baseURL+"/usage", nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+apiKey)
	
	resp, err := client.Do(req)
	if err != nil {
		return &ValidationResponse{
			Valid:   false,
			Error:   "network_error",
			Message: "网络连接失败，请检查网络",
		}, nil
	}
	defer resp.Body.Close()
	
	switch resp.StatusCode {
	case http.StatusOK:
		return &ValidationResponse{
			Valid:   true,
			Message: "API Key有效",
		}, nil
	case http.StatusUnauthorized:
		return &ValidationResponse{
			Valid:   false,
			Error:   "invalid_api_key",
			Message: "请检查您的API Key",
		}, nil
	case http.StatusForbidden:
		return &ValidationResponse{
			Valid:   false,
			Error:   "no_go_subscription",
			Message: "请订阅OpenCode Go计划",
		}, nil
	case http.StatusTooManyRequests:
		return &ValidationResponse{
			Valid:   false,
			Error:   "rate_limited",
			Message: "请求过于频繁，请稍后重试",
		}, nil
	default:
		return &ValidationResponse{
			Valid:   false,
			Error:   "server_error",
			Message: fmt.Sprintf("服务器错误: HTTP %d", resp.StatusCode),
		}, nil
	}
}
