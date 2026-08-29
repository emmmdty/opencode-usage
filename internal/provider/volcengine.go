package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type VolcengineProvider struct {
	apiKey   string
	endpoint string
}

func NewVolcengineProvider(apiKey string) *VolcengineProvider {
	return &VolcengineProvider{
		apiKey:   apiKey,
		endpoint: "https://ark.cn-beijing.volces.com",
	}
}

func NewVolcengineProviderWithEndpoint(apiKey, endpoint string) *VolcengineProvider {
	p := NewVolcengineProvider(apiKey)
	if endpoint != "" {
		p.endpoint = endpoint
	}
	return p
}

func (p *VolcengineProvider) Name() string {
	return "volcengine"
}

func (p *VolcengineProvider) IsAvailable() bool {
	return p.apiKey != ""
}

func (p *VolcengineProvider) GetUsage() (*Usage, error) {
	// 火山引擎的套餐额度查询需要 Access Key 鉴权
	// 使用 API Key 只能查询基础用量
	req, err := http.NewRequest("GET", p.endpoint+"/api/v3/usage", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	usage := &Usage{
		Provider: "volcengine",
		PlanType: "unknown",
	}

	// 尝试从响应中提取配额信息
	// 注意：实际的 API 响应格式可能不同，需要根据实际情况调整
	if planType, ok := result["plan_type"].(string); ok {
		usage.PlanType = planType
	}

	// 提取 5h 配额
	if fiveHour, ok := result["5h"].(map[string]interface{}); ok {
		if percent, ok := fiveHour["percent"].(float64); ok {
			usage.Rolling = QuotaWindow{
				Percent: int(percent),
				Status:  "ok",
			}
		}
	}

	// 提取 weekly 配额
	if weekly, ok := result["weekly"].(map[string]interface{}); ok {
		if percent, ok := weekly["percent"].(float64); ok {
			usage.Weekly = QuotaWindow{
				Percent: int(percent),
				Status:  "ok",
			}
		}
	}

	return usage, nil
}
