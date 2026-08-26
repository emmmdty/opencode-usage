package models

import "time"

// Account 表示一个OpenCode Go账号
type Account struct {
	Name         string    `yaml:"name"`
	KeyID        string    `yaml:"key_id"`        // 仅用于显示，存储Key的后6位
	CreatedAt    time.Time `yaml:"created_at"`
	LastVerified time.Time `yaml:"last_verified"`
}

// QuotaWindow 表示一个配额窗口
type QuotaWindow struct {
	Status   string    `json:"status"`
	Percent  int       `json:"percent"`
	ResetsAt time.Time `json:"resetsAt"`
}

// Usage 表示配额使用情况
type Usage struct {
	Rolling QuotaWindow `json:"rolling"`
	Weekly  QuotaWindow `json:"weekly"`
	Monthly QuotaWindow `json:"monthly"`
}

// Model 表示一个可用模型
type Model struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pricing struct {
		Input  float64 `json:"input"`
		Output float64 `json:"output"`
	} `json:"pricing"`
}

// Config 应用配置
type Config struct {
	Version    string             `yaml:"version"`
	Accounts   map[string]Account `yaml:"accounts"`
	ColorThresholds struct {
		Warning int `yaml:"warning"` // 默认50
		Danger  int `yaml:"danger"`  // 默认80
	} `yaml:"color_thresholds"`
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"` // 默认5
}

// ErrorResponse API错误响应
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details"`
	} `json:"error"`
}
