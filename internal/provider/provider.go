package provider

import (
	"time"
)

// QuotaWindow 表示一个配额窗口
type QuotaWindow struct {
	Status  string    `json:"status"`
	Percent int       `json:"percent"`
	ResetAt time.Time `json:"resetAt"`
}

// Usage 表示配额使用情况
type Usage struct {
	Provider string      `json:"provider"`
	PlanType string      `json:"planType"`
	Rolling  QuotaWindow `json:"rolling"`
	Weekly   QuotaWindow `json:"weekly"`
	Monthly  QuotaWindow `json:"monthly"`
	RawData  interface{} `json:"rawData,omitempty"`
}

// Provider 定义了获取用量数据的接口
type Provider interface {
	// Name 返回 provider 名称
	Name() string
	// GetUsage 获取当前配额使用情况
	GetUsage() (*Usage, error)
	// IsAvailable 检查认证信息是否可用
	IsAvailable() bool
}
