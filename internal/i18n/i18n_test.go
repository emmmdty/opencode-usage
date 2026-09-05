package i18n

import (
	"encoding/json"
	"os"
	"testing"
)

func TestT_English(t *testing.T) {
	SetLanguage("en")
	defer SetLanguage("en")
	result := T("cmd.root.short")
	if result != "Multi-provider AI coding tool usage monitor" {
		t.Errorf("expected English text, got: %s", result)
	}
}

func TestT_Chinese(t *testing.T) {
	SetLanguage("zh")
	defer SetLanguage("en")
	result := T("cmd.root.short")
	if result != "多 Provider AI 编程工具用量监控" {
		t.Errorf("expected Chinese text, got: %s", result)
	}
}

func TestT_Fallback(t *testing.T) {
	SetLanguage("zh")
	defer SetLanguage("en")
	result := T("nonexistent.key.fallback")
	if result != "nonexistent.key.fallback" {
		t.Errorf("expected key as fallback, got: %s", result)
	}
}

func TestT_Args(t *testing.T) {
	SetLanguage("en")
	defer SetLanguage("en")
	result := T("output.provider.add.preset_added", "opencode", "work", "manual")
	expected := "\nProvider 'opencode' account 'work' added (manual)\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestT_UnknownKey(t *testing.T) {
	SetLanguage("en")
	defer SetLanguage("en")
	result := T("totally.made.up.key")
	if result != "totally.made.up.key" {
		t.Errorf("expected key itself for unknown key, got: %s", result)
	}
}

func TestDetectLanguage_PriorityFlag(t *testing.T) {
	result := DetectLanguage("zh", "en", "en", "en_US.UTF-8")
	if result != "zh" {
		t.Errorf("flag should take precedence, got: %s", result)
	}
}

func TestDetectLanguage_PriorityEnv(t *testing.T) {
	result := DetectLanguage("", "zh", "en", "en_US.UTF-8")
	if result != "zh" {
		t.Errorf("env should beat config, got: %s", result)
	}
}

func TestDetectLanguage_PriorityConfig(t *testing.T) {
	result := DetectLanguage("", "", "zh", "en_US.UTF-8")
	if result != "zh" {
		t.Errorf("config should beat system, got: %s", result)
	}
}

func TestDetectLanguage_PrioritySystem(t *testing.T) {
	result := DetectLanguage("", "", "", "zh_CN.UTF-8")
	if result != "zh" {
		t.Errorf("zh system lang should produce zh, got: %s", result)
	}
}

func TestDetectLanguage_DefaultEn(t *testing.T) {
	result := DetectLanguage("", "", "", "en_US.UTF-8")
	if result != "en" {
		t.Errorf("default should be en, got: %s", result)
	}
}

func TestDetectLanguage_EmptySystem(t *testing.T) {
	result := DetectLanguage("", "", "", "")
	if result != "en" {
		t.Errorf("empty system lang should default to en, got: %s", result)
	}
}

func TestKeyConsistency(t *testing.T) {
	enData, err := os.ReadFile("en.json")
	if err != nil {
		t.Fatalf("failed to read en.json: %v", err)
	}
	zhData, err := os.ReadFile("zh.json")
	if err != nil {
		t.Fatalf("failed to read zh.json: %v", err)
	}

	var enKeys, zhKeys map[string]string
	if err := json.Unmarshal(enData, &enKeys); err != nil {
		t.Fatalf("failed to parse en.json: %v", err)
	}
	if err := json.Unmarshal(zhData, &zhKeys); err != nil {
		t.Fatalf("failed to parse zh.json: %v", err)
	}

	for k := range enKeys {
		if _, ok := zhKeys[k]; !ok {
			t.Errorf("key %q exists in en.json but missing from zh.json", k)
		}
	}
	for k := range zhKeys {
		if _, ok := enKeys[k]; !ok {
			t.Errorf("key %q exists in zh.json but missing from en.json", k)
		}
	}
}

func TestSetLanguage_InvalidIgnored(t *testing.T) {
	SetLanguage("en")
	SetLanguage("fr") // should be ignored
	if GetLanguage() != "en" {
		t.Errorf("invalid language should be ignored, got: %s", GetLanguage())
	}
}

func TestSetLanguage_Zh(t *testing.T) {
	SetLanguage("en")
	SetLanguage("zh")
	if GetLanguage() != "zh" {
		t.Errorf("expected zh, got: %s", GetLanguage())
	}
	SetLanguage("en") // cleanup
}
