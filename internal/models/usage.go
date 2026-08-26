package models

import "time"

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
