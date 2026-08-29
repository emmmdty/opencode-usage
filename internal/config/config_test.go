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

	if cfg.Version != "2" {
		t.Errorf("expected version 2, got %s", cfg.Version)
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
    name: ""
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
