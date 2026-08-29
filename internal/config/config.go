package config

import (
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

const CurrentVersion = "2"

type Config struct {
	Version         string                    `yaml:"version"`
	Accounts        map[string]Account        `yaml:"accounts"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
	ColorThresholds struct {
		Warning int `yaml:"warning"`
		Danger  int `yaml:"danger"`
	} `yaml:"color_thresholds"`
	MaxConcurrentRequests int   `yaml:"max_concurrent_requests"`
	UseMasterPassword     *bool `yaml:"use_master_password,omitempty"`
}

type Account struct {
	Name         string    `yaml:"name"`
	KeyID        string    `yaml:"key_id"`
	CreatedAt    time.Time `yaml:"created_at"`
	LastVerified time.Time `yaml:"last_verified"`
}

// ProviderConfig 配置各个 provider
type ProviderConfig struct {
	Enabled   bool   `yaml:"enabled"`
	APIKey    string `yaml:"api_key,omitempty"`    // OpenCode, Volcengine
	CredsPath string `yaml:"creds_path,omitempty"` // Claude
	AuthPath  string `yaml:"auth_path,omitempty"`  // Codex
	Endpoint  string `yaml:"endpoint,omitempty"`   // 自定义端点
}

func getDefaultConfig() *Config {
	cfg := &Config{
		Version:               CurrentVersion,
		Accounts:              make(map[string]Account),
		Providers:             make(map[string]ProviderConfig),
		MaxConcurrentRequests: 5,
	}
	cfg.ColorThresholds.Warning = 50
	cfg.ColorThresholds.Danger = 80

	// 默认启用的 provider
	home, _ := os.UserHomeDir()
	cfg.Providers["opencode"] = ProviderConfig{
		Enabled: true,
	}
	cfg.Providers["claude"] = ProviderConfig{
		Enabled:   true,
		CredsPath: filepath.Join(home, ".claude", ".credentials.json"),
	}
	cfg.Providers["codex"] = ProviderConfig{
		Enabled:  true,
		AuthPath: filepath.Join(home, ".codex", "auth.json"),
	}
	cfg.Providers["volcengine"] = ProviderConfig{
		Enabled: false,
	}

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
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return writeFileAtomic(path, data, 0600)
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by rename, so a crash mid-write never truncates the target.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func SaveConfig(cfg *Config, path string) error {
	return saveConfig(cfg, path)
}

func migrateConfig(old *Config) *Config {
	cfg := getDefaultConfig()
	for k, v := range old.Accounts {
		cfg.Accounts[k] = v
	}
	if old.ColorThresholds.Warning != 0 {
		cfg.ColorThresholds.Warning = old.ColorThresholds.Warning
	}
	if old.ColorThresholds.Danger != 0 {
		cfg.ColorThresholds.Danger = old.ColorThresholds.Danger
	}
	if old.MaxConcurrentRequests != 0 {
		cfg.MaxConcurrentRequests = old.MaxConcurrentRequests
	}
	cfg.UseMasterPassword = old.UseMasterPassword

	// 迁移旧的 provider 配置
	if old.Providers != nil {
		for k, v := range old.Providers {
			cfg.Providers[k] = v
		}
	}

	return cfg
}
