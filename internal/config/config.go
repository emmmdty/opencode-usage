package config

import (
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

const CurrentVersion = "1"

type Config struct {
	Version    string             `yaml:"version"`
	Accounts   map[string]Account `yaml:"accounts"`
	ColorThresholds struct {
		Warning int `yaml:"warning"`
		Danger  int `yaml:"danger"`
	} `yaml:"color_thresholds"`
	MaxConcurrentRequests int  `yaml:"max_concurrent_requests"`
	UseMasterPassword     *bool `yaml:"use_master_password,omitempty"`
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
	return cfg
}
