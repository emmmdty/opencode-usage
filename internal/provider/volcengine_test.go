package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVolcengineProbe_Ke_valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/coding/v3/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Real Ark 200 responses typically omit rate-limit headers.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": ""}},
			},
		})
	}))
	defer server.Close()

	p := &VolcengineProvider{apiKey: "test-api-key", plan: PlanCoding, probeBase: server.URL}
	usage, err := p.GetUsage()
	if err != nil {
		t.Fatalf("probe should succeed for valid key: %v", err)
	}
	if usage.Rolling.Status != StatusUnknown {
		t.Errorf("expected unknown rolling window without headers, got %q", usage.Rolling.Status)
	}
	if !strings.Contains(usage.Note, "arkcli") {
		t.Errorf("expected note to mention arkcli, got %q", usage.Note)
	}
}

func TestVolcengineProbe_WithRateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Ratelimit-Limit-Requests", "100")
		w.Header().Set("X-Ratelimit-Remaining-Requests", "55")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	p := &VolcengineProvider{apiKey: "k", plan: PlanCoding, probeBase: server.URL}
	usage, err := p.GetUsage()
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if usage.Rolling.Percent != 45 {
		t.Errorf("expected rolling percent 45 (100-55), got %d", usage.Rolling.Percent)
	}
}

func TestVolcengineProbe_InvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	p := &VolcengineProvider{apiKey: "bad", plan: PlanCoding, probeBase: server.URL}
	if _, err := p.GetUsage(); err == nil {
		t.Fatal("expected error for rejected key")
	}
}

func TestVolcengineIsAvailable(t *testing.T) {
	if (&VolcengineProvider{}).IsAvailable() {
		t.Error("expected unavailable with no key and no arkcli")
	}
	if !(&VolcengineProvider{apiKey: "k"}).IsAvailable() {
		t.Error("expected available with key")
	}
	if !(&VolcengineProvider{arkcli: "/usr/bin/arkcli"}).IsAvailable() {
		t.Error("expected available with arkcli path")
	}
}

func TestVolcengineArkcliParsing(t *testing.T) {
	script := `#!/bin/sh
echo '{"items":[{"product":"coding-plan","edition":"personal","subscribed":true,"periods":[{"label":"session","percent":42,"reset_at":"2026-09-05T20:00:00+08:00"},{"label":"weekly","percent":10,"reset_at":"2026-09-08T00:00:00+08:00"},{"label":"monthly","percent":5,"reset_at":"2026-09-29T00:00:00+08:00"}]},{"product":"agent-plan","subscribed":false,"periods":[]}]}'
`
	dir := t.TempDir()
	bin := filepath.Join(dir, "arkcli")
	if err := exec.Command("cp", "/bin/sh", bin).Run(); err != nil {
		t.Skipf("cannot prepare fake arkcli: %v", err)
	}
	if err := exec.Command("chmod", "+x", bin).Run(); err != nil {
		t.Skipf("cannot chmod fake arkcli: %v", err)
	}
	if err := writeFakeArkcli(bin, script); err != nil {
		t.Fatalf("failed to write fake arkcli: %v", err)
	}

	p := &VolcengineProvider{plan: PlanCoding, arkcli: bin}
	usage, err := p.GetUsage()
	if err != nil {
		t.Fatalf("arkcli path failed: %v", err)
	}
	if usage.PlanType != "coding-plan (personal)" {
		t.Errorf("unexpected plan type %q", usage.PlanType)
	}
	if usage.Rolling.Percent != 42 || usage.Weekly.Percent != 10 || usage.Monthly.Percent != 5 {
		t.Errorf("unexpected windows: %+v", usage)
	}
}

func writeFakeArkcli(path, script string) error {
	return exec.Command("/bin/sh", "-c", "cat > "+path+" <<'EOF'\n"+script+"EOF\nchmod +x "+path).Run()
}
