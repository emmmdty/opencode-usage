package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/emmmdty/opencode-usage/internal/models"
	"github.com/mattn/go-runewidth"
)

type AccountResult struct {
	Name      string
	Usage     *models.Usage
	Error     string
	IsCurrent bool
}

type QuotaStyle struct {
	WarningThreshold int
	DangerThreshold  int
}

func DefaultQuotaStyle() QuotaStyle {
	return QuotaStyle{
		WarningThreshold: 50,
		DangerThreshold:  80,
	}
}

func FormatQuotaOverview(results []AccountResult, style QuotaStyle, currentAccount string) string {
	theme := NewTheme()
	width := GetTerminalWidth()

	if len(results) == 0 {
		return theme.Muted.Render("  No accounts configured. Run 'opencode-usage account add' to get started.\n")
	}

	for i := range results {
		if results[i].Name == currentAccount {
			results[i].IsCurrent = true
		}
	}

	if width < 60 {
		return formatCompact(results, style, theme)
	}
	return formatTable(results, style, theme, width)
}

func formatTable(results []AccountResult, style QuotaStyle, theme Theme, width int) string {
	var b strings.Builder

	nameWidth := computeNameWidth(results)

	usable := width - 2
	if usable < 40 {
		usable = 40
	}

	fixedTotal := nameWidth + 10
	availCols := usable - fixedTotal
	if availCols < 0 {
		availCols = 0
	}
	colWidth := availCols / 3
	if colWidth < 8 {
		colWidth = 8
	}

	barWidth := colWidth - 12
	if barWidth < 0 {
		barWidth = 0
	}

	b.WriteString(theme.Title.Render("  OpenCode Go  ") + theme.Muted.Render(fmt.Sprintf("refreshed %s", time.Now().Format("15:04:05"))) + "\n\n")

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s\n",
		nameWidth, theme.Header.Render("ACCOUNT"),
		colWidth, theme.Header.Render("5H"),
		colWidth, theme.Header.Render("Weekly"),
		colWidth, theme.Header.Render("Monthly"))
	b.WriteString(header)
	sepLen := nameWidth + 3*colWidth + 10
	if sepLen > usable {
		sepLen = usable
	}
	b.WriteString("  " + theme.Border.Render(strings.Repeat("─", sepLen)) + "\n")

	for _, result := range results {
		if result.Error != "" {
			b.WriteString(formatErrorRow(result, nameWidth, theme))
		} else {
			b.WriteString(formatQuotaRow(result, nameWidth, colWidth, barWidth, style, theme))
		}
	}

	b.WriteString("\n")
	summary := computeSummary(results, style)
	b.WriteString("  " + theme.Muted.Render(summary) + "\n")

	for _, r := range results {
		if r.IsCurrent {
			b.WriteString("  " + theme.Muted.Render("Active: ") + theme.Active.Render(r.Name) + "\n")
			break
		}
	}

	best := findBestAccount(results)
	if best != "" {
		b.WriteString("  " + theme.Muted.Render("Best available: ") + theme.Success.Render(best) + "\n")
	}

	nextReset := findNextReset(results)
	if nextReset != "" {
		b.WriteString("  " + theme.Muted.Render("Next reset: ") + nextReset + "\n")
	}

	b.WriteString("\n  " + theme.Success.Render("●") + theme.Muted.Render(" healthy  ") +
		theme.Warning.Render("▲") + theme.Muted.Render(" warning  ") +
		theme.Danger.Render("●") + theme.Muted.Render(" critical  ") +
		theme.Active.Render("→") + theme.Muted.Render(" active") + "\n")

	return b.String()
}

func formatCompact(results []AccountResult, style QuotaStyle, theme Theme) string {
	var b strings.Builder
	b.WriteString("  OpenCode Go\n\n")

	for _, result := range results {
		if result.Error != "" {
			marker := "  "
			if result.IsCurrent {
				marker = theme.Active.Render("→ ")
			}
			b.WriteString(fmt.Sprintf("  %s%s  %s\n", marker, theme.Bold.Render(result.Name), theme.Error.Render("error")))
			b.WriteString(fmt.Sprintf("    %s\n", theme.Muted.Render(truncateError(result.Error, 40))))
		} else {
			marker := "  "
			if result.IsCurrent {
				marker = theme.Active.Render("→ ")
			}
			b.WriteString(fmt.Sprintf("  %s%s\n", marker, theme.Bold.Render(result.Name)))
			b.WriteString(fmt.Sprintf("    5H: %s  W: %s  M: %s\n",
				formatPercentCompact(result.Usage.Rolling.Percent, style, theme),
				formatPercentCompact(result.Usage.Weekly.Percent, style, theme),
				formatPercentCompact(result.Usage.Monthly.Percent, style, theme)))
		}
	}

	summary := computeSummary(results, style)
	b.WriteString("\n  " + theme.Muted.Render(summary) + "\n")

	for _, r := range results {
		if r.IsCurrent {
			b.WriteString("  " + theme.Muted.Render("Active: ") + theme.Active.Render(r.Name) + "\n")
			break
		}
	}

	best := findBestAccount(results)
	if best != "" {
		b.WriteString("\n  " + theme.Muted.Render("Best available: ") + theme.Success.Render(best) + "\n")
	}
	nextReset := findNextReset(results)
	if nextReset != "" {
		b.WriteString("  Reset: " + nextReset + "\n")
	}
	return b.String()
}

func formatPercentCompact(percent int, style QuotaStyle, theme Theme) string {
	s := fmt.Sprintf("%d%%", percent)
	switch {
	case percent >= style.DangerThreshold:
		return theme.Danger.Render(s)
	case percent >= style.WarningThreshold:
		return theme.Warning.Render(s)
	default:
		return theme.Success.Render(s)
	}
}

func formatErrorRow(result AccountResult, nameWidth int, theme Theme) string {
	marker := "  "
	if result.IsCurrent {
		marker = theme.Active.Render("→ ")
	}
	name := theme.Bold.Render(padRight(result.Name, nameWidth))
	errText := theme.Error.Render("✗ " + truncateError(result.Error, 50))
	return fmt.Sprintf("  %s%s  %s\n", marker, name, errText)
}

func formatQuotaRow(result AccountResult, nameWidth, colWidth, barWidth int, style QuotaStyle, theme Theme) string {
	marker := "  "
	if result.IsCurrent {
		marker = theme.Active.Render("→ ")
	}

	name := padRight(result.Name, nameWidth)
	rolling := formatQuotaCell(result.Usage.Rolling.Percent, result.Usage.Rolling.ResetsAt, barWidth, style, theme)
	weekly := formatQuotaCell(result.Usage.Weekly.Percent, result.Usage.Weekly.ResetsAt, barWidth, style, theme)
	monthly := formatQuotaCell(result.Usage.Monthly.Percent, result.Usage.Monthly.ResetsAt, barWidth, style, theme)

	return fmt.Sprintf("  %s%s  %-*s  %-*s  %-*s\n",
		marker, theme.Account.Render(name),
		colWidth, rolling,
		colWidth, weekly,
		colWidth, monthly)
}

func formatQuotaCell(percent int, resetsAt time.Time, barWidth int, style QuotaStyle, theme Theme) string {
	bar := renderBar(percent, barWidth, style, theme)
	pct := formatPercent(percent, style, theme)
	reset := theme.Muted.Render(" " + formatResetTime(resetsAt))
	return bar + " " + pct + reset
}

func renderBar(percent, width int, style QuotaStyle, theme Theme) string {
	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}
	empty := width - filled

	barStyle := theme.Success
	switch {
	case percent >= style.DangerThreshold:
		barStyle = theme.Danger
	case percent >= style.WarningThreshold:
		barStyle = theme.Warning
	}

	filledStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", empty)

	return barStyle.Render(filledStr) + theme.BarEmpty.Render(emptyStr)
}

func formatPercent(percent int, style QuotaStyle, theme Theme) string {
	s := fmt.Sprintf("%3d%%", percent)
	switch {
	case percent >= style.DangerThreshold:
		return theme.Danger.Render(s)
	case percent >= style.WarningThreshold:
		return theme.Warning.Render(s)
	default:
		return theme.Success.Render(s)
	}
}

func formatResetTime(resetsAt time.Time) string {
	duration := time.Until(resetsAt)
	if duration < 0 {
		return "expired"
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
	if minutes == 0 {
		return "<1m"
	}
	return fmt.Sprintf("%dm", minutes)
}

func computeNameWidth(results []AccountResult) int {
	width := 7
	for _, r := range results {
		w := runewidth.StringWidth(r.Name)
		if r.IsCurrent {
			w += 2
		}
		if w > width {
			width = w
		}
	}
	return width + 2
}

func padRight(s string, width int) string {
	sw := runewidth.StringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

func computeSummary(results []AccountResult, style QuotaStyle) string {
	total := len(results)
	healthy := 0
	warnings := 0
	criticals := 0
	errors := 0
	for _, r := range results {
		if r.Error != "" {
			errors++
			continue
		}
		maxPercent := r.Usage.Rolling.Percent
		if r.Usage.Weekly.Percent > maxPercent {
			maxPercent = r.Usage.Weekly.Percent
		}
		if r.Usage.Monthly.Percent > maxPercent {
			maxPercent = r.Usage.Monthly.Percent
		}
		if maxPercent >= style.DangerThreshold {
			criticals++
		} else if maxPercent >= style.WarningThreshold {
			warnings++
		} else {
			healthy++
		}
	}

	parts := []string{}
	if healthy > 0 {
		parts = append(parts, fmt.Sprintf("%d healthy", healthy))
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", warnings))
	}
	if criticals > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", criticals))
	}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d error", errors))
	}

	noun := "accounts"
	if total == 1 {
		noun = "account"
	}
	return fmt.Sprintf("%d %s  %s", total, noun, strings.Join(parts, "  "))
}

func findBestAccount(results []AccountResult) string {
	bestName := ""
	bestPercent := 101

	for _, r := range results {
		if r.Error != "" {
			continue
		}
		maxPercent := r.Usage.Rolling.Percent
		if r.Usage.Weekly.Percent > maxPercent {
			maxPercent = r.Usage.Weekly.Percent
		}
		if r.Usage.Monthly.Percent > maxPercent {
			maxPercent = r.Usage.Monthly.Percent
		}
		if maxPercent < bestPercent {
			bestPercent = maxPercent
			bestName = r.Name
		}
	}
	return bestName
}

func findNextReset(results []AccountResult) string {
	earliest := time.Time{}
	name := ""
	for _, r := range results {
		if r.Error != "" {
			continue
		}
		for _, resetTime := range []time.Time{r.Usage.Rolling.ResetsAt, r.Usage.Weekly.ResetsAt, r.Usage.Monthly.ResetsAt} {
			if resetTime.IsZero() {
				continue
			}
			if earliest.IsZero() || resetTime.Before(earliest) {
				earliest = resetTime
				name = r.Name
			}
		}
	}
	if name == "" {
		return ""
	}
	return fmt.Sprintf("%s · %s", name, formatResetTime(earliest))
}

func truncateError(s string, maxLen int) string {
	w := runewidth.StringWidth(s)
	if w <= maxLen {
		return s
	}
	truncated := ""
	current := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if current+rw+1 > maxLen {
			truncated += "…"
			break
		}
		truncated += string(r)
		current += rw
	}
	return truncated
}
