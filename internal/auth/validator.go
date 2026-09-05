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
		baseURL = os.Getenv("TOKEN_USAGE_BASE_URL")
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
			Message: "Network connection failed, please check your network",
		}, nil
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return &ValidationResponse{
			Valid:   true,
			Message: "API Key is valid",
		}, nil
	case http.StatusUnauthorized:
		return &ValidationResponse{
			Valid:   false,
			Error:   "invalid_api_key",
			Message: "Please check your API Key",
		}, nil
	case http.StatusForbidden:
		return &ValidationResponse{
			Valid:   false,
			Error:   "no_go_subscription",
			Message: "Please subscribe to the OpenCode Go plan",
		}, nil
	case http.StatusTooManyRequests:
		return &ValidationResponse{
			Valid:   false,
			Error:   "rate_limited",
			Message: "Too many requests, please try again later",
		}, nil
	default:
		return &ValidationResponse{
			Valid:   false,
			Error:   "server_error",
			Message: fmt.Sprintf("Server error: HTTP %d", resp.StatusCode),
		}, nil
	}
}
