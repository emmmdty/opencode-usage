package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/opencode-usage/internal/models"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://opencode.ai/zen/go/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) doRequest(endpoint string) ([]byte, error) {
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		req, err := http.NewRequest("GET", c.baseURL+endpoint, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries {
				continue
			}
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < maxRetries {
				continue
			}
		}

		return nil, fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("API error: max retries exceeded")
}

func (c *Client) GetUsage() (*models.Usage, error) {
	body, err := c.doRequest("/usage")
	if err != nil {
		return nil, err
	}

	var response struct {
		Usage models.Usage `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.Usage, nil
}

func (c *Client) GetModels() ([]models.Model, error) {
	body, err := c.doRequest("/models")
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []models.Model `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}
