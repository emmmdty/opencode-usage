package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emmmdty/opencode-usage/internal/auth"
	"github.com/emmmdty/opencode-usage/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose configuration and connectivity issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		theme := newDoctorTheme()
		checks := []doctorCheck{}

		configPath, err := getConfigPath()
		if err != nil {
			checks = append(checks, doctorCheck{name: "Config path", status: "FAIL", detail: err.Error()})
		} else {
			cfg, err := config.LoadOrCreateConfig(configPath)
			if err != nil {
				checks = append(checks, doctorCheck{name: "Config file", status: "FAIL", detail: err.Error()})
			} else {
				configureAuthFromConfig(cfg)
				checks = append(checks, doctorCheck{name: "Config file", status: "OK", detail: configPath})
				checks = append(checks, doctorCheck{
					name:   "Accounts",
					status: "OK",
					detail: fmt.Sprintf("%d configured", len(cfg.Accounts)),
				})
			}
		}

		if auth.IsKeyringAvailable() {
			checks = append(checks, doctorCheck{name: "Keyring", status: "OK", detail: "system keyring available"})
		} else {
			checks = append(checks, doctorCheck{name: "Keyring", status: "WARN", detail: "using encrypted file fallback"})
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("https://opencode.ai/zen/go/v1/usage")
		if err != nil {
			checks = append(checks, doctorCheck{name: "Network", status: "FAIL", detail: "cannot reach opencode.ai"})
		} else {
			resp.Body.Close()
			checks = append(checks, doctorCheck{name: "Network", status: "OK", detail: "opencode.ai reachable"})
		}

		homeDir, _ := os.UserHomeDir()
		authPath := filepath.Join(homeDir, ".local", "share", "opencode", "auth.json")
		if _, err := os.Stat(authPath); err == nil {
			checks = append(checks, doctorCheck{name: "OpenCode auth", status: "OK", detail: "auth.json found"})
		} else {
			checks = append(checks, doctorCheck{name: "OpenCode auth", status: "WARN", detail: "auth.json not found"})
		}

		var out strings.Builder
		for _, c := range checks {
			icon := theme.okIcon
			switch c.status {
			case "WARN":
				icon = theme.warnIcon
			case "FAIL":
				icon = theme.failIcon
			}
			fmt.Fprintf(&out, "  %s %-20s %s\n", icon, c.name, c.detail)
		}

		allOK := true
		for _, c := range checks {
			if c.status == "FAIL" {
				allOK = false
				break
			}
		}
		if allOK {
			fmt.Fprintln(&out, "\n  All checks passed.")
		} else {
			fmt.Fprintln(&out, "\n  Some checks failed. Run 'opencode-usage quota' to see details.")
		}
		return writeOutput(out.String())
	},
}

type doctorCheck struct {
	name   string
	status string
	detail string
}

type doctorTheme struct {
	okIcon   string
	warnIcon string
	failIcon string
}

func newDoctorTheme() doctorTheme {
	if noColor || os.Getenv("NO_COLOR") != "" || !isTerminal() {
		return doctorTheme{okIcon: "[OK]", warnIcon: "[!!]", failIcon: "[XX]"}
	}
	return doctorTheme{okIcon: "\033[32m✓\033[0m", warnIcon: "\033[33m!\033[0m", failIcon: "\033[31m✗\033[0m"}
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
