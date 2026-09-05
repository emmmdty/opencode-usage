package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Volcengine coding/agent plan endpoints (base of the ark API).
const volcengineAPIBase = "https://ark.cn-beijing.volces.com"

// VolcengineProvider queries Volcano Engine subscription quota.
//
// Two resolution paths:
//  1. arkcli (official CLI) when installed and logged in: `arkcli usage plan
//     --format json` returns the same windows as the console (session/weekly/
//     monthly for coding plan, 5h/weekly/monthly for agent plan).
//  2. API-key probe: a max_tokens=1 chat completion confirms the key works;
//     rate-limit headers are unreliable on Ark (often absent on 200), so the
//     usage is reported as unknown with a note pointing at arkcli.
type VolcengineProvider struct {
	apiKey    string
	plan      string // "coding" | "agent"
	arkcli    string // resolved path, "" = not installed
	probeBase string // override for tests
}

func NewVolcengineProvider(apiKey, plan string) *VolcengineProvider {
	return &VolcengineProvider{
		apiKey: apiKey,
		plan:   plan,
		arkcli: lookPathArkcli(),
	}
}

// lookPathArkcli resolves the arkcli binary once per construction.
func lookPathArkcli() string {
	if path, err := exec.LookPath("arkcli"); err == nil {
		return path
	}
	return ""
}

// ArkcliAvailable reports whether the official ark CLI was found on PATH.
func ArkcliAvailable() bool {
	return lookPathArkcli() != ""
}

func (p *VolcengineProvider) Name() string {
	return "volcengine"
}

func (p *VolcengineProvider) IsAvailable() bool {
	return p.arkcli != "" || p.apiKey != ""
}

func (p *VolcengineProvider) GetUsage() (*Usage, error) {
	if p.arkcli != "" {
		usage, err := p.usageViaArkcli()
		if err == nil {
			return usage, nil
		}
		// Fall through to the probe when a key is available; otherwise
		// surface the arkcli error (e.g. not logged in).
		if p.apiKey == "" {
			return nil, fmt.Errorf("arkcli: %w", err)
		}
	}
	return p.usageViaProbe()
}

// arkcliEnv suppresses update checks and follows the official caller
// attribution protocol for AI agents.
func arkcliEnv() []string {
	return []string{
		"ARKCLI_NO_UPDATE_NOTIFIER=1",
		"ARKCLI_CALLER_TYPE=ai_agent",
		"ARKCLI_CALLER_NAME=token-usage",
	}
}

type arkcliPeriod struct {
	Label   string `json:"label"`
	Percent int    `json:"percent"`
	ResetAt string `json:"reset_at"`
}

type arkcliItem struct {
	Product    string         `json:"product"`
	Edition    string         `json:"edition"`
	Subscribed bool           `json:"subscribed"`
	Periods    []arkcliPeriod `json:"periods"`
	Error      string         `json:"error"`
}

type arkcliOutput struct {
	Items []arkcliItem `json:"items"`
}

// usageViaArkcli shells out to the official CLI. Read-only query; the local
// side effects are arkcli's own caches under ~/.arkcli.
func (p *VolcengineProvider) usageViaArkcli() (*Usage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.arkcli, "usage", "plan", "--format", "json")
	cmd.Env = arkcliEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("usage plan failed: %s", truncateMsg(msg, 200))
	}

	var out arkcliOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("unexpected arkcli output: %w", err)
	}

	usage := &Usage{
		Provider: "volcengine",
		PlanType: p.plan + "-plan",
		Rolling:  QuotaWindow{Status: StatusUnknown},
		Weekly:   QuotaWindow{Status: StatusUnknown},
		Monthly:  QuotaWindow{Status: StatusUnknown},
	}

	// Filter to items matching the configured plan, preferring personal
	// edition over team seats.
	wantPersonal := "agent-plan"
	if p.plan == PlanCoding {
		wantPersonal = "coding-plan"
	}
	var best *arkcliItem
	for i := range out.Items {
		item := &out.Items[i]
		if !item.Subscribed {
			continue
		}
		if item.Product != wantPersonal && item.Product != wantPersonal+"-team" {
			continue
		}
		if best != nil && best.Product == wantPersonal && item.Product != wantPersonal {
			continue // keep personal over team
		}
		best = item
	}
	if best == nil {
		return nil, fmt.Errorf("no active %s subscription found (run 'arkcli usage plan' to check)", wantPersonal)
	}

	usage.PlanType = wantPersonal
	if best.Edition != "" {
		usage.PlanType = wantPersonal + " (" + best.Edition + ")"
	}

	// Label mapping: coding plan uses "session", agent plan uses "5h".
	for _, period := range best.Periods {
		window := QuotaWindow{Status: "ok", Percent: period.Percent}
		if t, err := time.Parse(time.RFC3339, period.ResetAt); err == nil {
			window.ResetAt = t
		}
		switch period.Label {
		case "session", "5h":
			usage.Rolling = window
		case "weekly":
			usage.Weekly = window
		case "monthly":
			usage.Monthly = window
		}
	}
	return usage, nil
}

func truncateMsg(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// usageViaProbe validates the key with a 1-token completion and reads the
// rate-limit headers when Ark returns them. Without headers the windows are
// reported unknown; the note tells the user how to get full quota.
func (p *VolcengineProvider) usageViaProbe() (*Usage, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("no API key and arkcli not available")
	}

	base := p.probeBase
	if base == "" {
		base = volcengineAPIBase
	}

	payload := `{"model":"ark-code-latest","messages":[{"role":"user","content":"hi"}],"max_tokens":1}`
	req, err := http.NewRequest("POST", base+"/api/coding/v3/chat/completions", strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("probe request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		return nil, fmt.Errorf("key rejected: HTTP %d", resp.StatusCode)
	}

	usage := &Usage{
		Provider: "volcengine",
		PlanType: p.plan + "-plan",
		Rolling:  QuotaWindow{Status: StatusUnknown},
		Weekly:   QuotaWindow{Status: StatusUnknown},
		Monthly:  QuotaWindow{Status: StatusUnknown},
	}

	limit := intHeader(resp.Header, "X-Ratelimit-Limit-Requests")
	remaining := intHeader(resp.Header, "X-Ratelimit-Remaining-Requests")
	if limit > 0 && remaining >= 0 {
		usage.Rolling = QuotaWindow{
			Status:  "ok",
			Percent: 100 - (remaining*100)/limit,
		}
		if reset := resp.Header.Get("X-Ratelimit-Reset-Requests"); reset != "" {
			// Ark reports an interval; the exact reset time is not exposed.
			usage.Rolling.ResetAt = time.Time{}
		}
		usage.Note = "rate-limit snapshot (request-based); install arkcli and log in for full quota windows"
	} else {
		usage.Note = "key valid; install arkcli (npm i -g @volcengine/ark-cli) and run 'arkcli auth login' for full quota windows"
	}
	return usage, nil
}

func intHeader(h http.Header, name string) int {
	v := h.Get(name)
	if v == "" {
		return -1
	}
	n := 0
	for _, r := range v {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}
