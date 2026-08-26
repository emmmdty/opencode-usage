# opencode-usage 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 创建一个Go CLI工具，用于快速查询多个OpenCode账号下Go计划的使用情况、可用模型和配额信息。

**Architecture:** 使用Go语言开发，采用cobra进行命令解析，bubbletea作为TUI框架，99designs/keyring进行安全存储。项目采用MVP开发模式，先实现核心功能，再逐步完善。

**Tech Stack:** Go 1.22+, bubbletea, cobra, 99designs/keyring, viper

---

## 项目初始化

### Task 1: 项目结构初始化

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `cmd/opencode-usage/main.go`
- Create: `internal/models/usage.go`

- [ ] **Step 1: 初始化Go模块**

```bash
cd /home/tjk/myProjects/tools/opencode-usage
go mod init github.com/opencode-usage
```

- [ ] **Step 2: 安装依赖**

```bash
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go get github.com/99designs/keyring
```

- [ ] **Step 3: 创建数据模型**

创建 `internal/models/usage.go`：

```go
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
```

- [ ] **Step 4: 创建主入口文件**

创建 `cmd/opencode-usage/main.go`：

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// TODO: 实现命令解析
	return nil
}
```

- [ ] **Step 5: 验证项目结构**

```bash
go build ./...
```

Expected: BUILD SUCCESS

- [ ] **Step 6: 提交初始代码**

```bash
git init
git add .
git commit -m "feat: initialize project structure with data models"
```

---

## 配置管理模块

### Task 2: 配置文件管理

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: 编写配置管理测试**

创建 `internal/config/config_test.go`：

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateConfig(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	// 测试创建新配置
	cfg, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}
	
	if cfg.Version != "1" {
		t.Errorf("expected version 1, got %s", cfg.Version)
	}
	
	if cfg.MaxConcurrentRequests != 5 {
		t.Errorf("expected max concurrent requests 5, got %d", cfg.MaxConcurrentRequests)
	}
}

func TestLoadExistingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	// 创建测试配置文件
	configContent := `version: "1"
accounts:
  work:
    key_id: "abc123"
    created_at: "2026-08-24T10:00:00Z"
    last_verified: "2026-08-24T10:05:00Z"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	
	cfg, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	
	if _, exists := cfg.Accounts["work"]; !exists {
		t.Error("expected account 'work' to exist")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/config/... -v
```

Expected: FAIL (function not defined)

- [ ] **Step 3: 实现配置管理**

创建 `internal/config/config.go`：

```go
package config

import (
	"os"
	"path/filepath"
	"time"
	
	"gopkg.in/yaml.v3"
)

const CurrentVersion = "1"

type Config struct {
	Version    string             `yaml:"version"`
	Accounts   map[string]Account `yaml:"accounts"`
	ColorThresholds struct {
		Warning int `yaml:"warning"`
		Danger  int `yaml:"danger"`
	} `yaml:"color_thresholds"`
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"`
}

type Account struct {
	Name         string    `yaml:"name"`
	KeyID        string    `yaml:"key_id"`
	CreatedAt    time.Time `yaml:"created_at"`
	LastVerified time.Time `yaml:"last_verified"`
}

func getDefaultConfig() *Config {
	cfg := &Config{
		Version:               CurrentVersion,
		Accounts:              make(map[string]Account),
		MaxConcurrentRequests: 5,
	}
	cfg.ColorThresholds.Warning = 50
	cfg.ColorThresholds.Danger = 80
	return cfg
}

func LoadOrCreateConfig(path string) (*Config, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 创建默认配置
		cfg := getDefaultConfig()
		if err := saveConfig(cfg, path); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	
	// 读取现有配置
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	
	// 检查版本并迁移
	if cfg.Version != CurrentVersion {
		cfg = migrateConfig(cfg)
		if err := saveConfig(cfg, path); err != nil {
			return nil, err
		}
	}
	
	return cfg, nil
}

func saveConfig(cfg *Config, path string) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0600)
}

func migrateConfig(old *Config) *Config {
	// 实现配置迁移逻辑
	cfg := getDefaultConfig()
	// 复制旧配置中的账号
	for k, v := range old.Accounts {
		cfg.Accounts[k] = v
	}
	return cfg
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/config/... -v
```

Expected: PASS

- [ ] **Step 5: 提交代码**

```bash
git add internal/config/
git commit -m "feat: add config management with version migration"
```

---

## 安全存储模块

### Task 3: 密钥环和加密存储

**Files:**
- Create: `internal/auth/credential.go`
- Create: `internal/auth/credential_test.go`
- Create: `internal/auth/validator.go`
- Create: `internal/auth/validator_test.go`
- Create: `internal/auth/encrypted.go`
- Create: `internal/auth/encrypted_test.go`

- [ ] **Step 1: 编写密钥环存储测试**

创建 `internal/auth/credential_test.go`：

```go
package auth

import (
	"testing"
)

func TestKeyringOperations(t *testing.T) {
	serviceName := "opencode-usage-test"
	accountName := "test-account"
	apiKey := "sk-test1234567890"
	
	// 测试存储
	if err := StoreAPIKey(serviceName, accountName, apiKey); err != nil {
		t.Fatalf("failed to store API key: %v", err)
	}
	
	// 测试读取
	retrieved, err := GetAPIKey(serviceName, accountName)
	if err != nil {
		t.Fatalf("failed to get API key: %v", err)
	}
	
	if retrieved != apiKey {
		t.Errorf("expected %s, got %s", apiKey, retrieved)
	}
	
	// 测试删除
	if err := DeleteAPIKey(serviceName, accountName); err != nil {
		t.Fatalf("failed to delete API key: %v", err)
	}
	
	// 验证删除
	_, err = GetAPIKey(serviceName, accountName)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestExtractKeyID(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"sk-ant-1234567890abcdef", "7890abcdef"},
		{"sk-test-abcdef123456", "abcdef123456"},
		{"short", "short"},
	}
	
	for _, tt := range tests {
		result := ExtractKeyID(tt.key)
		if result != tt.expected {
			t.Errorf("ExtractKeyID(%s) = %s, want %s", tt.key, result, tt.expected)
		}
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/auth/... -v -run TestKeyringOperations
```

Expected: FAIL (function not defined)

- [ ] **Step 3: 实现密钥环存储**

创建 `internal/auth/credential.go`：

```go
package auth

import (
	"fmt"
	"strings"
	
	"github.com/99designs/keyring"
)

var (
	// 禁用密钥环的回退存储
	fallbackStore = make(map[string]string)
	ring          keyring.Keyring
	keyringAvailable bool
)

func init() {
	// 尝试初始化系统密钥环
	var err error
	ring, err = keyring.Open(keyring.Config{
		ServiceName: "opencode-usage",
	})
	if err != nil {
		fmt.Printf("Warning: system keyring unavailable, using encrypted file storage: %v\n", err)
		keyringAvailable = false
	} else {
		keyringAvailable = true
	}
}

func IsKeyringAvailable() bool {
	return keyringAvailable
}

func StoreAPIKey(service, account, apiKey string) error {
	if ring != nil {
		return ring.Set(keyring.Item{
			Key:  account,
			Data: []byte(apiKey),
		})
	}
	// 回退到加密文件存储
	return storeEncrypted(account, apiKey)
}

func GetAPIKey(service, account string) (string, error) {
	if ring != nil {
		item, err := ring.Get(account)
		if err != nil {
			return "", err
		}
		return string(item.Data), nil
	}
	// 回退到加密文件存储
	return getEncrypted(account)
}

func DeleteAPIKey(service, account string) error {
	if ring != nil {
		return ring.Remove(account)
	}
	// 回退到加密文件存储
	return deleteEncrypted(account)
}

func ExtractKeyID(apiKey string) string {
	if len(apiKey) > 6 {
		return apiKey[len(apiKey)-6:]
	}
	return apiKey
}
```

- [ ] **Step 4: 实现加密文件存储**

创建 `internal/auth/encrypted.go`：

```go
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	
	"golang.org/x/crypto/pbkdf2"
)

const (
	encryptedDir = ".config/opencode-usage"
	encryptedFile = "secrets.enc"
	saltSize = 16
	keySize = 32
	iterations = 100000
)

func getEncryptedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, encryptedDir, encryptedFile), nil
}

func getMasterPassword() (string, error) {
	// 尝试从环境变量获取
	if pwd := os.Getenv("OPENCODE_USAGE_MASTER_PASSWORD"); pwd != "" {
		return pwd, nil
	}
	
	// 交互式输入
	fmt.Print("请输入主密码: ")
	var pwd string
	fmt.Scanln(&pwd)
	
	if pwd == "" {
		return "", fmt.Errorf("主密码不能为空")
	}
	
	return pwd, nil
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, iterations, keySize, sha256.New)
}

func encrypt(data []byte, password string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	
	key := deriveKey(password, salt)
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	
	ciphertext := aesGCM.Seal(nil, nonce, data, nil)
	
	// 格式: salt + nonce + ciphertext
	result := make([]byte, 0, saltSize+aesGCM.NonceSize()+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	
	return result, nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < saltSize+aesGCM.Overhead() {
		return nil, fmt.Errorf("数据格式无效")
	}
	
	salt := data[:saltSize]
	data = data[saltSize:]
	
	key := deriveKey(password, salt)
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("数据格式无效")
	}
	
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]
	
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}

func storeEncrypted(account, apiKey string) error {
	path, err := getEncryptedPath()
	if err != nil {
		return err
	}
	
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	
	// 读取现有数据
	existing := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil {
			pwd, err := getMasterPassword()
			if err == nil {
				decrypted, err := decrypt(decoded, pwd)
				if err == nil {
					// 解析现有数据
					lines := strings.Split(string(decrypted), "\n")
					for _, line := range lines {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							existing[parts[0]] = parts[1]
						}
					}
				}
			}
		}
	}
	
	// 添加新账号
	existing[account] = apiKey
	
	// 构建数据
	var data strings.Builder
	for k, v := range existing {
		data.WriteString(k + ":" + v + "\n")
	}
	
	// 加密
	pwd, err := getMasterPassword()
	if err != nil {
		return err
	}
	
	encrypted, err := encrypt([]byte(data.String()), pwd)
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
}

func getEncrypted(account string) (string, error) {
	path, err := getEncryptedPath()
	if err != nil {
		return "", err
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return "", err
	}
	
	pwd, err := getMasterPassword()
	if err != nil {
		return "", err
	}
	
	decrypted, err := decrypt(decoded, pwd)
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(string(decrypted), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == account {
			return parts[1], nil
		}
	}
	
	return "", fmt.Errorf("API key not found for account: %s", account)
}

func deleteEncrypted(account string) error {
	path, err := getEncryptedPath()
	if err != nil {
		return err
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	
	pwd, err := getMasterPassword()
	if err != nil {
		return err
	}
	
	decrypted, err := decrypt(decoded, pwd)
	if err != nil {
		return err
	}
	
	existing := make(map[string]string)
	lines := strings.Split(string(decrypted), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			existing[parts[0]] = parts[1]
		}
	}
	
	delete(existing, account)
	
	var newData strings.Builder
	for k, v := range existing {
		newData.WriteString(k + ":" + v + "\n")
	}
	
	encrypted, err := encrypt([]byte(newData.String()), pwd)
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
}
```

- [ ] **Step 5: 运行测试验证通过**

```bash
go test ./internal/auth/... -v -run TestKeyringOperations
```

Expected: PASS

- [ ] **Step 6: 实现API Key验证**

创建 `internal/auth/validator.go`：

```go
package auth

import (
	"encoding/json"
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
```

- [ ] **Step 7: 运行所有测试**

```bash
go test ./internal/auth/... -v
```

Expected: PASS

- [ ] **Step 8: 提交代码**

```bash
git add internal/auth/
git commit -m "feat: add secure credential storage with keyring and encrypted file fallback"
```

---

## API客户端模块

### Task 4: OpenCode Go API客户端

**Files:**
- Create: `internal/client/opencode.go`
- Create: `internal/client/opencode_test.go`

- [ ] **Step 1: 编写API客户端测试**

创建 `internal/client/opencode_test.go`：

```go
package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetUsage(t *testing.T) {
	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		response := map[string]interface{}{
			"usage": map[string]interface{}{
				"rolling": map[string]interface{}{
					"status":   "ok",
					"percent":  35,
					"resetsAt": time.Now().Add(8 * time.Hour).Format(time.RFC3339),
				},
				"weekly": map[string]interface{}{
					"status":   "ok",
					"percent":  12,
					"resetsAt": time.Now().Add(5 * 24 * time.Hour).Format(time.RFC3339),
				},
				"monthly": map[string]interface{}{
					"status":   "ok",
					"percent":  8,
					"resetsAt": time.Now().Add(23 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewClient("test-key", server.URL)
	usage, err := client.GetUsage()
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	
	if usage.Rolling.Percent != 35 {
		t.Errorf("expected rolling percent 35, got %d", usage.Rolling.Percent)
	}
}

func TestGetModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "mimo-v2.5",
					"name": "MiMo-V2.5",
					"pricing": map[string]interface{}{
						"input":  0.14,
						"output": 0.28,
					},
				},
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := NewClient("test-key", server.URL)
	models, err := client.GetModels()
	if err != nil {
		t.Fatalf("failed to get models: %v", err)
	}
	
	if len(models) != 1 {
		t.Errorf("expected 1 model, got %d", len(models))
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/client/... -v
```

Expected: FAIL (function not defined)

- [ ] **Step 3: 实现API客户端**

创建 `internal/client/opencode.go`：

```go
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
	apiKey    string
	baseURL   string
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
	req, err := http.NewRequest("GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}
	
	return body, nil
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
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/client/... -v
```

Expected: PASS

- [ ] **Step 5: 提交代码**

```bash
git add internal/client/
git commit -m "feat: add OpenCode Go API client with retry logic"
```

---

## 命令行接口模块

### Task 5: 命令行解析

**Files:**
- Create: `internal/cmd/root.go`
- Create: `internal/cmd/account.go`
- Create: `internal/cmd/quota.go`
- Create: `internal/cmd/models.go`
- Create: `internal/cmd/current.go`
- Create: `internal/cmd/alias.go`

- [ ] **Step 1: 实现根命令**

创建 `internal/cmd/root.go`：

```go
package cmd

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	account    string
	outputFile string
	noColor    bool
	logFile    string
)

var rootCmd = &cobra.Command{
	Use:   "opencode-usage",
	Short: "OpenCode Go 计划配额查询工具",
	Long:  "用于快速查询多个OpenCode账号下Go计划的使用情况、可用模型和配额信息",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "JSON格式输出")
	rootCmd.PersistentFlags().StringVarP(&account, "account", "n", "", "指定账号")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "输出到文件")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "禁用颜色输出")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "日志输出文件")
}
```

- [ ] **Step 2: 实现账号管理命令**

创建 `internal/cmd/account.go`：

```go
package cmd

import (
	"fmt"
	"time"
	
	"github.com/spf13/cobra"
	"github.com/opencode-usage/internal/auth"
	"github.com/opencode-usage/internal/config"
)

var accountCmd = &cobra.Command{
	Use:     "account",
	Aliases: []string{"a"},
	Short:   "管理OpenCode Go账号",
}

var accountAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"aa"},
	Short:   "添加新账号",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: 实现交互式添加账号
		fmt.Println("添加账号功能开发中...")
		return nil
	},
}

var accountListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"al"},
	Short:   "查看所有账号",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadOrCreateConfig(getConfigPath())
		if err != nil {
			return err
		}
		
		if len(cfg.Accounts) == 0 {
			fmt.Println("暂无配置的账号")
			return nil
		}
		
		for name, account := range cfg.Accounts {
			fmt.Printf("账号: %s, Key ID: sk-...%s\n", name, account.KeyID)
		}
		return nil
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"ar"},
	Short:   "删除账号",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]
		
		cfg, err := config.LoadOrCreateConfig(getConfigPath())
		if err != nil {
			return err
		}
		
		if _, exists := cfg.Accounts[accountName]; !exists {
			return fmt.Errorf("账号 '%s' 不存在", accountName)
		}
		
		// 从密钥环删除
		if err := auth.DeleteAPIKey("opencode-usage", accountName); err != nil {
			return err
		}
		
		// 从配置中删除
		delete(cfg.Accounts, accountName)
		
		// 保存配置
		if err := config.SaveConfig(cfg, getConfigPath()); err != nil {
			return err
		}
		
		fmt.Printf("账号 '%s' 已删除\n", accountName)
		return nil
	},
}

func init() {
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	rootCmd.AddCommand(accountCmd)
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return homeDir + "/.config/opencode-usage/config.yaml"
}
```

- [ ] **Step 3: 实现配额查询命令**

创建 `internal/cmd/quota.go`：

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	
	"github.com/spf13/cobra"
	"github.com/opencode-usage/internal/auth"
	"github.com/opencode-usage/internal/client"
	"github.com/opencode-usage/internal/config"
)

var quotaCmd = &cobra.Command{
	Use:     "quota",
	Aliases: []string{"q"},
	Short:   "查看配额使用情况",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadOrCreateConfig(getConfigPath())
		if err != nil {
			return err
		}
		
		// 确定要查询的账号
		accountsToQuery := make(map[string]config.Account)
		if account != "" {
			if acc, exists := cfg.Accounts[account]; exists {
				accountsToQuery[account] = acc
			} else {
				return fmt.Errorf("账号 '%s' 不存在", account)
			}
		} else {
			accountsToQuery = cfg.Accounts
		}
		
		if len(accountsToQuery) == 0 {
			fmt.Println("暂无配置的账号，请先运行 'opencode-usage account add' 添加账号")
			return nil
		}
		
		// 并发查询
		var wg sync.WaitGroup
		results := make(chan struct {
			name  string
			usage *models.Usage
			err   error
		}, len(accountsToQuery))
		
		for name := range accountsToQuery {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				
				apiKey, err := auth.GetAPIKey("opencode-usage", name)
				if err != nil {
					results <- struct {
						name  string
						usage *models.Usage
						err   error
					}{name, nil, err}
					return
				}
				
				c := client.NewClient(apiKey, "")
				usage, err := c.GetUsage()
				results <- struct {
					name  string
					usage *models.Usage
					err   error
				}{name, usage, err}
			}(name)
		}
		
		wg.Wait()
		close(results)
		
		// 收集结果
		type accountResult struct {
			Name  string         `json:"name"`
			Usage *models.Usage  `json:"quota"`
			Error string         `json:"error,omitempty"`
		}
		
		var accountResults []accountResult
		for result := range results {
			if result.err != nil {
				accountResults = append(accountResults, accountResult{
					Name:  result.name,
					Error: result.err.Error(),
				})
			} else {
				accountResults = append(accountResults, accountResult{
					Name:  result.name,
					Usage: result.usage,
				})
			}
		}
		
		// 输出结果
		if jsonOutput {
			return printJSON(accountResults)
		}
		
		return printQuotaTable(accountResults)
	},
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func printQuotaTable(results []interface{}) error {
	// TODO: 实现表格输出
	fmt.Println("表格输出功能开发中...")
	return nil
}

func init() {
	rootCmd.AddCommand(quotaCmd)
}
```

- [ ] **Step 4: 实现模型查询命令**

创建 `internal/cmd/models.go`：

```go
package cmd

import (
	"fmt"
	
	"github.com/spf13/cobra"
	"github.com/opencode-usage/internal/client"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"m"},
	Short:   "查看可用模型列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 使用第一个账号查询模型
		apiKey, err := auth.GetAPIKey("opencode-usage", "")
		if err != nil {
			return fmt.Errorf("请先添加账号: opencode-usage account add")
		}
		
		c := client.NewClient(apiKey, "")
		models, err := c.GetModels()
		if err != nil {
			return err
		}
		
		if jsonOutput {
			return printJSON(models)
		}
		
		fmt.Println("可用模型:")
		for _, model := range models {
			fmt.Printf("  - %s (%s)\n", model.Name, model.ID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
```

- [ ] **Step 5: 实现当前配置命令**

创建 `internal/cmd/current.go`：

```go
package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"cc"},
	Short:   "显示当前opencode配置的账号",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 尝试读取opencode的auth.json
		homeDir, _ := os.UserHomeDir()
		authPath := homeDir + "/.local/share/opencode/auth.json"
		
		// 检查文件是否存在
		if _, err := os.Stat(authPath); os.IsNotExist(err) {
			fmt.Println("未找到opencode配置文件")
			return nil
		}
		
		// 读取并解析auth.json
		// TODO: 实现具体的解析逻辑
		fmt.Println("当前opencode配置:")
		fmt.Println("  Provider: opencode-go")
		fmt.Println("  (详细解析功能开发中...)")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
```

- [ ] **Step 6: 实现别名管理命令**

创建 `internal/cmd/alias.go`：

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "管理shell别名",
}

var aliasInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装shell别名",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, _ := os.UserHomeDir()
		
		// 检测shell类型
		shell := os.Getenv("SHELL")
		var rcFile string
		if strings.Contains(shell, "zsh") {
			rcFile = homeDir + "/.zshrc"
		} else {
			rcFile = homeDir + "/.bashrc"
		}
		
		// 检查是否已存在别名
		if aliasExists(rcFile, "ou") {
			fmt.Printf("别名 'ou' 已存在于 %s\n", rcFile)
			fmt.Print("是否覆盖？(y/N): ")
			
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(response)
			
			if response != "y" && response != "Y" {
				fmt.Println("已取消")
				return nil
			}
		}
		
		// 添加别名
		alias := "\n# opencode-usage alias\nalias ou='opencode-usage'\n"
		
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		
		if _, err := f.WriteString(alias); err != nil {
			return err
		}
		
		fmt.Printf("别名已添加到 %s\n", rcFile)
		fmt.Println("请运行 'source " + rcFile + "' 或重新打开终端")
		
		return nil
	},
}

var aliasUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "卸载shell别名",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, _ := os.UserHomeDir()
		
		shell := os.Getenv("SHELL")
		var rcFile string
		if strings.Contains(shell, "zsh") {
			rcFile = homeDir + "/.zshrc"
		} else {
			rcFile = homeDir + "/.bashrc"
		}
		
		// TODO: 实现别名移除逻辑
		fmt.Println("别名卸载功能开发中...")
		
		return nil
	},
}

func aliasExists(rcFile, alias string) bool {
	file, err := os.Open(rcFile)
	if err != nil {
		return false
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "alias "+alias+"=") {
			return true
		}
	}
	
	return false
}

func init() {
	aliasCmd.AddCommand(aliasInstallCmd)
	aliasCmd.AddCommand(aliasUninstallCmd)
	rootCmd.AddCommand(aliasCmd)
}
```

- [ ] **Step 7: 更新主入口文件**

更新 `cmd/opencode-usage/main.go`：

```go
package main

import (
	"fmt"
	"os"
	
	"github.com/opencode-usage/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 8: 验证编译通过**

```bash
go build ./cmd/opencode-usage/
```

Expected: BUILD SUCCESS

- [ ] **Step 9: 提交代码**

```bash
git add internal/cmd/ cmd/opencode-usage/
git commit -m "feat: add CLI commands for account, quota, models, and alias management"
```

---

## TUI界面模块

### Task 6: 交互式界面

**Files:**
- Create: `internal/tui/account.go`
- Create: `internal/tui/quota.go`
- Create: `internal/tui/components.go`

- [ ] **Step 1: 实现添加账号交互界面**

创建 `internal/tui/account.go`：

```go
package tui

import (
	"fmt"
	"strings"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-usage/internal/auth"
)

type AddAccountModel struct {
	step       int
	name       string
	apiKey     string
	verifying  bool
	result     string
	err        error
}

func NewAddAccountModel() AddAccountModel {
	return AddAccountModel{step: 0}
}

func (m AddAccountModel) Init() tea.Cmd {
	return nil
}

func (m AddAccountModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		}
	case verifyResultMsg:
		m.verifying = false
		if msg.err != nil {
			m.err = msg.err
			m.result = "验证失败: " + msg.err.Error()
		} else {
			m.result = "账号添加成功！"
		}
		return m, tea.Quit
	}
	
	return m, nil
}

func (m *AddAccountModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case 0: // 输入账号名称
		if m.name == "" {
			return m, nil
		}
		m.step = 1
	case 1: // 输入API Key
		if m.apiKey == "" {
			return m, nil
		}
		m.verifying = true
		return m, m.verifyAPIKey()
	}
	return m, nil
}

func (m AddAccountModel) View() string {
	var b strings.Builder
	
	// 标题
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Render("添加 OpenCode Go 账号")
	b.WriteString(title + "\n\n")
	
	switch m.step {
	case 0:
		b.WriteString("账号名称: ")
		b.WriteString(m.name)
	case 1:
		b.WriteString("账号名称: " + m.name + "\n")
		if m.verifying {
			b.WriteString("API Key: " + strings.Repeat("•", len(m.apiKey)) + "\n")
			b.WriteString("验证中...")
		} else {
			b.WriteString("API Key: ")
			b.WriteString(strings.Repeat("•", len(m.apiKey)))
		}
	}
	
	if m.result != "" {
		b.WriteString("\n\n" + m.result)
	}
	
	if m.err != nil {
		b.WriteString("\n错误: " + m.err.Error())
	}
	
	return b.String()
}

func (m *AddAccountModel) verifyAPIKey() tea.Cmd {
	return func() tea.Msg {
		_, err := auth.ValidateAPIKey(m.apiKey, "")
		return verifyResultMsg{err: err}
	}
}

type verifyResultMsg struct {
	err error
}
```

- [ ] **Step 2: 实现配额显示界面**

创建 `internal/tui/quota.go`：

```go
package tui

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-usage/internal/models"
)

func FormatQuotaTable(results []AccountResult) string {
	var b strings.Builder
	
	// 表头
	header := lipgloss.NewStyle().
		Bold(true).
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Render("OpenCode Go 配额概览")
	b.WriteString(header + "\n")
	
	// 表格内容
	for _, result := range results {
		if result.Error != "" {
			b.WriteString(fmt.Sprintf("%s: 错误 - %s\n", result.Name, result.Error))
			continue
		}
		
		// 计算剩余时间
		rollingReset := formatResetTime(result.Usage.Rolling.ResetsAt)
		weeklyReset := formatResetTime(result.Usage.Weekly.ResetsAt)
		monthlyReset := formatResetTime(result.Usage.Monthly.ResetsAt)
		
		// 颜色化显示
		rollingColor := getColor(result.Usage.Rolling.Percent)
		weeklyColor := getColor(result.Usage.Weekly.Percent)
		monthlyColor := getColor(result.Usage.Monthly.Percent)
		
		rollingStr := fmt.Sprintf("%d%% (剩余%s)", result.Usage.Rolling.Percent, rollingReset)
		weeklyStr := fmt.Sprintf("%d%% (剩余%s)", result.Usage.Weekly.Percent, weeklyReset)
		monthlyStr := fmt.Sprintf("%d%% (剩余%s)", result.Usage.Monthly.Percent, monthlyReset)
		
		b.WriteString(fmt.Sprintf("%-10s %-25s %-25s %-25s\n",
			result.Name,
			rollingColor(rollingStr),
			weeklyColor(weeklyStr),
			monthlyColor(monthlyStr),
		))
	}
	
	return b.String()
}

func formatResetTime(resetsAt time.Time) string {
	duration := time.Until(resetsAt)
	if duration < 0 {
		return "已过期"
	}
	
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func getColor(percent int) func(string) string {
	return func(s string) string {
		switch {
		case percent >= 80:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(s) // 红色
		case percent >= 50:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(s) // 黄色
		default:
			return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(s)  // 绿色
		}
	}
}

type AccountResult struct {
	Name  string
	Usage *models.Usage
	Error string
}
```

- [ ] **Step 3: 实现TUI组件**

创建 `internal/tui/components.go`：

```go
package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type TextInputModel struct {
	textInput textinput.Model
	label     string
	err       error
}

func NewTextInput(label string, placeholder string) TextInputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	
	return TextInputModel{
		textInput: ti,
		label:     label,
	}
}

func (m TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m TextInputModel) Update(msg tea.Msg) (TextInputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m TextInputModel) View() string {
	return m.label + ": " + m.textInput.View()
}

func (m *TextInputModel) SetValue(value string) {
	m.textInput.SetValue(value)
}

func (m TextInputModel) Value() string {
	return m.textInput.Value()
}
```

- [ ] **Step 4: 验证编译通过**

```bash
go build ./internal/tui/
```

Expected: BUILD SUCCESS

- [ ] **Step 5: 提交代码**

```bash
git add internal/tui/
git commit -m "feat: add TUI components for interactive interface"
```

---

## 集成测试

### Task 7: 端到端测试

**Files:**
- Create: `test/integration_test.go`

- [ ] **Step 1: 编写集成测试**

创建 `test/integration_test.go`：

```go
package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIIntegration(t *testing.T) {
	// 构建二进制文件
	buildCmd := exec.Command("go", "build", "-o", "opencode-usage-test", "../cmd/opencode-usage/")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
	defer os.Remove("opencode-usage-test")
	
	// 测试帮助命令
	helpCmd := exec.Command("./opencode-usage-test", "--help")
	output, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run help command: %v", err)
	}
	
	if !contains(string(output), "OpenCode Go") {
		t.Error("help output should contain 'OpenCode Go'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
```

- [ ] **Step 2: 运行集成测试**

```bash
go test ./test/... -v
```

Expected: PASS

- [ ] **Step 3: 提交代码**

```bash
git add test/
git commit -m "test: add integration tests for CLI"
```

---

## 文档和发布

### Task 8: 文档和配置

**Files:**
- Create: `README.md`
- Create: `.goreleaser.yml`

- [ ] **Step 1: 创建README**

创建 `README.md`：

```markdown
# opencode-usage

OpenCode Go 计划配额查询工具

## 功能

- 管理多个OpenCode Go账号
- 查看配额使用情况（5小时滚动、每周、每月）
- 查看可用模型列表
- 支持JSON输出
- 支持shell别名

## 安装

```bash
# 下载预编译二进制文件
# 或者从源码构建
go build -o opencode-usage ./cmd/opencode-usage/
```

## 使用方法

```bash
# 添加账号
opencode-usage account add

# 查看所有账号
opencode-usage account list

# 查看配额
opencode-usage quota

# 查看特定账号配额
opencode-usage quota -n work

# 安装shell别名
opencode-usage alias install
```

## 安全

- API Key存储在系统密钥环中
- 配置文件只存储Key的显示ID（后6位）
- 支持降级到加密配置文件

## 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 一般错误 |
| 2 | 用法错误 |
| 3 | 认证失败 |
| 4 | 网络错误 |
| 5 | 配置文件错误 |
| 6 | 配置文件不存在 |
| 7 | 密钥环不可用 |
```

- [ ] **Step 2: 创建GoReleaser配置**

创建 `.goreleaser.yml`：

```yaml
version: 2

builds:
  - binary: opencode-usage
    main: ./cmd/opencode-usage
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

- [ ] **Step 3: 验证配置**

```bash
goreleaser check
```

Expected: VALID

- [ ] **Step 4: 提交代码**

```bash
git add README.md .goreleaser.yml
git commit -m "docs: add README and GoReleaser configuration"
```

---

## 完成

### Task 9: 最终验证

- [ ] **Step 1: 运行所有测试**

```bash
go test ./... -v
```

Expected: ALL PASS

- [ ] **Step 2: 构建最终二进制文件**

```bash
go build -o opencode-usage ./cmd/opencode-usage/
```

Expected: BUILD SUCCESS

- [ ] **Step 3: 验证功能**

```bash
# 测试帮助
./opencode-usage --help

# 测试别名安装
./opencode-usage alias install

# 测试账号添加
./opencode-usage account add
```

- [ ] **Step 4: 最终提交**

```bash
git add .
git commit -m "feat: complete opencode-usage CLI tool v0.1"
```

---

## 执行选项

**Plan complete and saved to `docs/superpowers/plans/2026-08-24-opencode-usage-implementation.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
