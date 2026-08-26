package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

var (
	isColorEnabled = true
	terminalWidth  = 80
)

func init() {
	if os.Getenv("NO_COLOR") != "" {
		isColorEnabled = false
	}
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		terminalWidth = w
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		isColorEnabled = false
	}
}

func DisableColor() {
	isColorEnabled = false
}

func SetTerminalWidth(w int) {
	if w > 0 {
		terminalWidth = w
	}
}

func GetTerminalWidth() int {
	return terminalWidth
}

type Theme struct {
	Primary   lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Danger    lipgloss.Style
	Muted     lipgloss.Style
	Border    lipgloss.Style
	Title     lipgloss.Style
	Header    lipgloss.Style
	Dim       lipgloss.Style
	Bold      lipgloss.Style
	Error     lipgloss.Style
	Account   lipgloss.Style
	Active    lipgloss.Style
	BarFilled lipgloss.Style
	BarEmpty  lipgloss.Style
}

func NewTheme() Theme {
	if !isColorEnabled {
		return Theme{
			Primary:   lipgloss.NewStyle(),
			Success:   lipgloss.NewStyle(),
			Warning:   lipgloss.NewStyle(),
			Danger:    lipgloss.NewStyle(),
			Muted:     lipgloss.NewStyle(),
			Border:    lipgloss.NewStyle(),
			Title:     lipgloss.NewStyle().Bold(true),
			Header:    lipgloss.NewStyle().Bold(true),
			Dim:       lipgloss.NewStyle(),
			Bold:      lipgloss.NewStyle().Bold(true),
			Error:     lipgloss.NewStyle().Bold(true),
			Account:   lipgloss.NewStyle(),
			Active:    lipgloss.NewStyle().Bold(true),
			BarFilled: lipgloss.NewStyle(),
			BarEmpty:  lipgloss.NewStyle(),
		}
	}

	return Theme{
		Primary:   lipgloss.NewStyle().Foreground(lipgloss.Color("62")),
		Success:   lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
		Warning:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Danger:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
		Border:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")),
		Header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
		Dim:       lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
		Bold:      lipgloss.NewStyle().Bold(true),
		Error:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		Account:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		Active:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")),
		BarFilled: lipgloss.NewStyle().Foreground(lipgloss.Color("62")),
		BarEmpty:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
}

func init() {
	if !isColorEnabled {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}
