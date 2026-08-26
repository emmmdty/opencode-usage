package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-usage/internal/models"
)

func FormatQuotaTable(results []AccountResult, warningThreshold, dangerThreshold int) string {
	if warningThreshold == 0 {
		warningThreshold = 50
	}
	if dangerThreshold == 0 {
		dangerThreshold = 80
	}

	// 计算账号列的最大宽度
	nameWidth := 5 // 最小宽度（"账号"的长度）
	for _, r := range results {
		if len(r.Name) > nameWidth {
			nameWidth = len(r.Name)
		}
	}
	nameWidth += 2 // 添加一些padding

	// 固定配额列宽度
	quotaWidth := 22

	var b strings.Builder

	// 标题
	b.WriteString("\n  OpenCode Go 配额概览\n\n")

	// 表头
	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s\n", nameWidth, "账号", quotaWidth, "滚动配额", quotaWidth, "周配额", quotaWidth, "月配额")
	b.WriteString(header)
	b.WriteString("  " + strings.Repeat("─", nameWidth+quotaWidth*3+3) + "\n")

	// 数据行
	for _, result := range results {
		if result.Error != "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n", nameWidth, result.Name, "错误: "+result.Error))
		} else {
			rollingReset := formatResetTime(result.Usage.Rolling.ResetsAt)
			weeklyReset := formatResetTime(result.Usage.Weekly.ResetsAt)
			monthlyReset := formatResetTime(result.Usage.Monthly.ResetsAt)

			rollingStr := fmt.Sprintf("%d%% (剩余%s)", result.Usage.Rolling.Percent, rollingReset)
			weeklyStr := fmt.Sprintf("%d%% (剩余%s)", result.Usage.Weekly.Percent, weeklyReset)
			monthlyStr := fmt.Sprintf("%d%% (剩余%s)", result.Usage.Monthly.Percent, monthlyReset)

			rollingColored := colorizeText(rollingStr, result.Usage.Rolling.Percent, warningThreshold, dangerThreshold)
			weeklyColored := colorizeText(weeklyStr, result.Usage.Weekly.Percent, warningThreshold, dangerThreshold)
			monthlyColored := colorizeText(monthlyStr, result.Usage.Monthly.Percent, warningThreshold, dangerThreshold)

			b.WriteString(fmt.Sprintf("  %-*s %s%s %s%s %s%s\n",
				nameWidth, result.Name,
				rollingColored, strings.Repeat(" ", quotaWidth-len(rollingStr)),
				weeklyColored, strings.Repeat(" ", quotaWidth-len(weeklyStr)),
				monthlyColored, strings.Repeat(" ", quotaWidth-len(monthlyStr))))
		}
	}

	return b.String()
}

// colorizeText 返回带颜色的文本
func colorizeText(s string, percent, warningThreshold, dangerThreshold int) string {
	switch {
	case percent >= dangerThreshold:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(s)
	case percent >= warningThreshold:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(s)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(s)
	}
}

func formatResetTime(resetsAt time.Time) string {
	duration := time.Until(resetsAt)
	if duration < 0 {
		return "已过期"
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
	return fmt.Sprintf("%dm", minutes)
}

type AccountResult struct {
	Name  string
	Usage *models.Usage
	Error string
}
