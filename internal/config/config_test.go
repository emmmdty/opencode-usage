package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("expected version %s, got %s", CurrentVersion, cfg.Version)
	}

	if cfg.MaxConcurrentRequests != 5 {
		t.Errorf("expected max concurrent requests 5, got %d", cfg.MaxConcurrentRequests)
	}

	if _, ok := cfg.Providers["opencode"]; !ok {
		t.Error("expected opencode preset to exist")
	}
	if _, ok := cfg.Providers["volcengine"]; !ok {
		t.Error("expected volcengine preset to exist")
	}
}

func TestMigrateV2Config(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	v2 := `version: "2"
accounts:
    work:
        name: work
        key_id: "abc123"
        created_at: "2026-08-24T10:00:00Z"
        last_verified: "2026-08-24T10:05:00Z"
providers:
    claude:
        enabled: true
        creds_path: /tmp/creds.json
    codex:
        enabled: true
        auth_path: /tmp/auth.json
    opencode:
        enabled: true
    volcengine:
        enabled: false
color_thresholds:
    warning: 40
    danger: 90
max_concurrent_requests: 3
`
	if err := os.WriteFile(configPath, []byte(v2), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Errorf("expected migrated version %s, got %s", CurrentVersion, cfg.Version)
	}

	oc, ok := cfg.Providers["opencode"]
	if !ok {
		t.Fatal("expected opencode provider")
	}
	acc, exists := oc.Accounts["work"]
	if !exists {
		t.Fatal("expected v2 account 'work' to migrate under providers.opencode")
	}
	if acc.Source != SourceManual || acc.KeyID != "abc123" {
		t.Errorf("unexpected migrated account: %+v", acc)
	}
	if oc.DefaultAccount != "work" {
		t.Errorf("expected default account 'work', got %q", oc.DefaultAccount)
	}

	if cl := cfg.Providers["claude"]; !cl.Enabled || cl.CredsPath != "/tmp/creds.json" {
		t.Errorf("claude provider not migrated correctly: %+v", cl)
	}
	if cx := cfg.Providers["codex"]; !cx.Enabled || cx.AuthPath != "/tmp/auth.json" {
		t.Errorf("codex provider not migrated correctly: %+v", cx)
	}

	if cfg.ColorThresholds.Warning != 40 || cfg.ColorThresholds.Danger != 90 {
		t.Errorf("thresholds not migrated: %+v", cfg.ColorThresholds)
	}
	if cfg.MaxConcurrentRequests != 3 {
		t.Errorf("max concurrent requests not migrated: %d", cfg.MaxConcurrentRequests)
	}
}

func TestFindProvider(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Custom["my-glm"] = CustomProvider{
		QueryType: "zai-glm",
		BaseURL:   "https://api.z.ai",
		Enabled:   true,
		Accounts:  map[string]Account{"main": {Source: SourceManual}},
	}

	if _, enabled, err := cfg.FindProvider("opencode"); err != nil || !enabled {
		t.Errorf("opencode lookup failed: enabled=%v err=%v", enabled, err)
	}
	if _, enabled, err := cfg.FindProvider("my-glm"); err != nil || !enabled {
		t.Errorf("custom lookup failed: enabled=%v err=%v", enabled, err)
	}
	if _, _, err := cfg.FindProvider("nope"); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestLanguageFieldPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	cfg.Language = "zh"
	if err := SaveConfig(cfg, configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Language != "zh" {
		t.Errorf("expected language 'zh', got %q", loaded.Language)
	}
}

func TestLanguageFieldDefaultEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	if cfg.Language != "" {
		t.Errorf("expected default language '', got %q", cfg.Language)
	}
}

func TestAllAccountsOrder(t *testing.T) {
	cfg := getDefaultConfig()
	cfg.Providers["opencode"].Accounts["b"] = Account{Source: SourceManual}
	cfg.Providers["opencode"].Accounts["a"] = Account{Source: SourceManual}
	cfg.Custom["z-provider"] = CustomProvider{Accounts: map[string]Account{"m": {Source: SourceManual}}}
	cfg.Providers["volcengine"].Accounts["agent-plan"] = Account{Source: SourceLocal, Plan: "agent"}

	got := cfg.AllAccounts()
	var ids []string
	for _, pa := range got {
		ids = append(ids, pa.ProviderID+"/"+pa.Account)
	}
	want := []string{"claude/local", "codex/local", "opencode/a", "opencode/b", "volcengine/agent-plan", "volcengine/coding-plan", "z-provider/m"}
	if len(ids) != len(want) {
		t.Fatalf("expected %d accounts, got %d: %v", len(want), len(ids), ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("order mismatch at %d: got %s want %s", i, ids[i], want[i])
		}
	}
}
