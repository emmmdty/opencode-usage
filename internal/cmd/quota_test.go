package cmd

import (
	"testing"

	"github.com/emmmdty/token-usage/internal/config"
)

// A bare quota filter must select a whole provider when it names one
// (preset or custom) — README documents 'token-usage quota volcengine'.
// Anything else is treated as an account name.
func TestSplitQuotaFilter(t *testing.T) {
	cfg := config.DefaultTestConfig()
	cfg.Custom["my-glm"] = config.CustomProvider{QueryType: "zai-glm"}

	tests := []struct {
		arg      string
		provWant string
		acctWant string
	}{
		{"claude", "claude", ""},
		{"volcengine", "volcengine", ""},
		{"my-glm", "my-glm", ""},
		{"work", "", "work"},
		{"nosuchthing", "", "nosuchthing"},
		{"opencode/work", "opencode", "work"},
		{"", "", ""},
	}
	for _, tt := range tests {
		prov, acct := splitQuotaFilter(cfg, tt.arg)
		if prov != tt.provWant || acct != tt.acctWant {
			t.Errorf("splitQuotaFilter(%q) = (%q, %q), want (%q, %q)",
				tt.arg, prov, acct, tt.provWant, tt.acctWant)
		}
	}
}
