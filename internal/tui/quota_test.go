package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/emmmdty/opencode-usage/internal/models"
)

func TestFormatQuotaOverviewEmpty(t *testing.T) {
	results := []AccountResult{}
	style := DefaultQuotaStyle()
	output := FormatQuotaOverview(results, style, "")
	if !strings.Contains(output, "No accounts") {
		t.Errorf("expected 'No accounts' message, got: %s", output)
	}
}

func TestFormatQuotaOverviewSingleAccount(t *testing.T) {
	results := []AccountResult{
		{
			Name: "work",
			Usage: &models.Usage{
				Rolling: models.QuotaWindow{Percent: 35, ResetsAt: time.Now().Add(8 * time.Hour)},
				Weekly:  models.QuotaWindow{Percent: 12, ResetsAt: time.Now().Add(5 * 24 * time.Hour)},
				Monthly: models.QuotaWindow{Percent: 8, ResetsAt: time.Now().Add(23 * 24 * time.Hour)},
			},
		},
	}
	style := DefaultQuotaStyle()
	output := FormatQuotaOverview(results, style, "")
	if !strings.Contains(output, "work") {
		t.Errorf("expected 'work' in output, got: %s", output)
	}
	if !strings.Contains(output, "35%") {
		t.Errorf("expected '35%%' in output, got: %s", output)
	}
	if !strings.Contains(output, "OpenCode Go") {
		t.Errorf("expected 'OpenCode Go' header, got: %s", output)
	}
}

func TestFormatQuotaOverviewCurrentAccount(t *testing.T) {
	results := []AccountResult{
		{Name: "work", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 35, ResetsAt: time.Now().Add(8 * time.Hour)},
			Weekly:  models.QuotaWindow{Percent: 12, ResetsAt: time.Now().Add(5 * 24 * time.Hour)},
			Monthly: models.QuotaWindow{Percent: 8, ResetsAt: time.Now().Add(23 * 24 * time.Hour)},
		}},
		{Name: "personal", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 67, ResetsAt: time.Now().Add(2 * time.Hour)},
			Weekly:  models.QuotaWindow{Percent: 45, ResetsAt: time.Now().Add(3 * 24 * time.Hour)},
			Monthly: models.QuotaWindow{Percent: 22, ResetsAt: time.Now().Add(18 * 24 * time.Hour)},
		}},
	}
	style := DefaultQuotaStyle()
	output := FormatQuotaOverview(results, style, "work")
	if !strings.Contains(output, "Active:") {
		t.Errorf("expected 'Active:' marker, got: %s", output)
	}
	if !strings.Contains(output, "work") {
		t.Errorf("expected 'work' as active account, got: %s", output)
	}
}

func TestFormatQuotaOverviewPartialFailure(t *testing.T) {
	results := []AccountResult{
		{Name: "work", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 35, ResetsAt: time.Now().Add(8 * time.Hour)},
			Weekly:  models.QuotaWindow{Percent: 12, ResetsAt: time.Now().Add(5 * 24 * time.Hour)},
			Monthly: models.QuotaWindow{Percent: 8, ResetsAt: time.Now().Add(23 * 24 * time.Hour)},
		}},
		{Name: "backup", Error: "API error: HTTP 401"},
	}
	style := DefaultQuotaStyle()
	output := FormatQuotaOverview(results, style, "")
	if !strings.Contains(output, "work") {
		t.Errorf("expected 'work' in output")
	}
	if !strings.Contains(output, "backup") {
		t.Errorf("expected 'backup' in output")
	}
	if !strings.Contains(output, "error") || !strings.Contains(output, "401") {
		t.Errorf("expected error display for backup, got: %s", output)
	}
}

func TestFormatQuotaOverviewCJK(t *testing.T) {
	results := []AccountResult{
		{Name: "生产环境", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 80, ResetsAt: time.Now().Add(4 * time.Hour)},
			Weekly:  models.QuotaWindow{Percent: 50, ResetsAt: time.Now().Add(3 * 24 * time.Hour)},
			Monthly: models.QuotaWindow{Percent: 30, ResetsAt: time.Now().Add(15 * 24 * time.Hour)},
		}},
		{Name: "personal", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 20, ResetsAt: time.Now().Add(6 * time.Hour)},
			Weekly:  models.QuotaWindow{Percent: 10, ResetsAt: time.Now().Add(4 * 24 * time.Hour)},
			Monthly: models.QuotaWindow{Percent: 5, ResetsAt: time.Now().Add(20 * 24 * time.Hour)},
		}},
	}
	style := DefaultQuotaStyle()
	output := FormatQuotaOverview(results, style, "")
	if !strings.Contains(output, "生产环境") {
		t.Errorf("expected '生产环境' in output")
	}
	if !strings.Contains(output, "personal") {
		t.Errorf("expected 'personal' in output")
	}
}

func TestFormatResetTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		contains string
	}{
		{"5 hours", time.Now().Add(5*time.Hour + 30*time.Minute), "5h"},
		{"2 days 3 hours", time.Now().Add(2*24*time.Hour + 3*time.Hour), "2d"},
		{"30 minutes", time.Now().Add(30 * time.Minute), "m"},
		{"expired", time.Now().Add(-1 * time.Hour), "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatResetTime(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatResetTime(%v) = %q, want it to contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		expect string
	}{
		{"abc", 6, "abc   "},
		{"测试", 6, "测试  "},
		{"hello", 3, "hello"},
	}

	for _, tt := range tests {
		result := padRight(tt.input, tt.width)
		runewidth := 0
		for _, r := range result {
			if r == ' ' {
				runewidth++
			} else {
				runewidth += 2
			}
		}
		if len(result) < len(tt.input) {
			t.Errorf("padRight(%q, %d) = %q, shorter than input", tt.input, tt.width, result)
		}
	}
}

func TestComputeSummary(t *testing.T) {
	results := []AccountResult{
		{Name: "a", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 30},
			Weekly:  models.QuotaWindow{Percent: 20},
			Monthly: models.QuotaWindow{Percent: 10},
		}},
		{Name: "b", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 60},
			Weekly:  models.QuotaWindow{Percent: 55},
			Monthly: models.QuotaWindow{Percent: 40},
		}},
		{Name: "c", Error: "timeout"},
	}
	summary := computeSummary(results, DefaultQuotaStyle())
	if !strings.Contains(summary, "3 accounts") {
		t.Errorf("expected '3 accounts' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "1 healthy") {
		t.Errorf("expected '1 healthy' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "1 warning") {
		t.Errorf("expected '1 warning' in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "1 error") {
		t.Errorf("expected '1 error' in summary, got: %s", summary)
	}
}

func TestFindBestAccount(t *testing.T) {
	results := []AccountResult{
		{Name: "heavy", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 80},
			Weekly:  models.QuotaWindow{Percent: 70},
			Monthly: models.QuotaWindow{Percent: 60},
		}},
		{Name: "light", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 10},
			Weekly:  models.QuotaWindow{Percent: 5},
			Monthly: models.QuotaWindow{Percent: 2},
		}},
		{Name: "errored", Error: "fail"},
	}
	best := findBestAccount(results)
	if best != "light" {
		t.Errorf("expected 'light' as best account, got: %q", best)
	}
}

func TestTruncateError(t *testing.T) {
	short := "API error: HTTP 401"
	if truncateError(short, 50) != short {
		t.Errorf("short error should not be truncated")
	}
	long := "This is a very long error message that should definitely be truncated because it exceeds the maximum length"
	truncated := truncateError(long, 30)
	charCount := 0
	for range truncated {
		charCount++
	}
	if charCount > 31 {
		t.Errorf("truncated error should be <= 31 chars, got %d: %q", charCount, truncated)
	}
	if !strings.HasSuffix(truncated, "…") {
		t.Errorf("truncated error should end with ellipsis, got: %q", truncated)
	}
}

func TestFormatQuotaOverviewCompactWidth(t *testing.T) {
	SetTerminalWidth(50)
	defer SetTerminalWidth(80)

	results := []AccountResult{
		{Name: "work", Usage: &models.Usage{
			Rolling: models.QuotaWindow{Percent: 35, ResetsAt: time.Now().Add(8 * time.Hour)},
			Weekly:  models.QuotaWindow{Percent: 12, ResetsAt: time.Now().Add(5 * 24 * time.Hour)},
			Monthly: models.QuotaWindow{Percent: 8, ResetsAt: time.Now().Add(23 * 24 * time.Hour)},
		}},
	}
	style := DefaultQuotaStyle()
	output := FormatQuotaOverview(results, style, "")
	if !strings.Contains(output, "work") {
		t.Errorf("compact mode should still show account name")
	}
}

func TestProgressBars(t *testing.T) {
	DisableColor()
	theme := NewTheme()

	tests := []struct {
		percent int
		filled  int
		empty   int
	}{
		{0, 0, 8},
		{50, 4, 4},
		{100, 8, 0},
	}

	for _, tt := range tests {
		bar := renderBar(tt.percent, 8, DefaultQuotaStyle(), theme)
		filledCount := strings.Count(bar, "█")
		emptyCount := strings.Count(bar, "░")
		if filledCount != tt.filled {
			t.Errorf("renderBar(%d): expected %d filled, got %d", tt.percent, tt.filled, filledCount)
		}
		if emptyCount != tt.empty {
			t.Errorf("renderBar(%d): expected %d empty, got %d", tt.percent, tt.empty, emptyCount)
		}
	}
}
