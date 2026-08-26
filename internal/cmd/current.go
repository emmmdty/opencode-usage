package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type authConfig struct {
	Provider string `json:"provider"`
	Token    string `json:"token"`
	BaseURL  string `json:"base_url"`
}

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"cc"},
	Short:   "Show current opencode configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		authPath := filepath.Join(homeDir, ".local", "share", "opencode", "auth.json")

		if _, err := os.Stat(authPath); os.IsNotExist(err) {
			return writeOutput("  No opencode configuration found.\n  Run 'opencode-usage account add' to get started.\n")
		}

		data, err := os.ReadFile(authPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		var authCfg authConfig
		if err := json.Unmarshal(data, &authCfg); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		var out strings.Builder
		fmt.Fprintln(&out)
		fmt.Fprintln(&out, "  Current opencode configuration:")
		fmt.Fprintln(&out)
		if authCfg.Provider != "" {
			fmt.Fprintf(&out, "    Provider: %s\n", authCfg.Provider)
		} else {
			fmt.Fprintln(&out, "    Provider: opencode-go")
		}
		if authCfg.BaseURL != "" {
			fmt.Fprintf(&out, "    Base URL: %s\n", authCfg.BaseURL)
		}
		if authCfg.Token != "" {
			fmt.Fprintln(&out, "    Token:    ***")
		}

		return writeOutput(out.String())
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
