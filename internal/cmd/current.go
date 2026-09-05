package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emmmdty/token-usage/internal/auth"
	"github.com/emmmdty/token-usage/internal/config"
	"github.com/spf13/cobra"
)

type authProvider struct {
	Type    string `json:"type"`
	Key     string `json:"key,omitempty"`
	Refresh string `json:"refresh,omitempty"`
	Access  string `json:"access,omitempty"`
	Expires int    `json:"expires"`
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
			return writeOutput("  No opencode configuration found.\n  Run 'token-usage account add' to get started.\n")
		}

		data, err := os.ReadFile(authPath)
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		var providers map[string]authProvider
		if err := json.Unmarshal(data, &providers); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		configPath, cfgErr := getConfigPath()
		var cfg *config.Config
		if cfgErr == nil {
			cfg, _ = config.LoadOrCreateConfig(configPath)
			if cfg != nil {
				configureAuthFromConfig(cfg)
			}
		}

		var out strings.Builder
		fmt.Fprintln(&out)

		names := make([]string, 0, len(providers))
		for name := range providers {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			p := providers[name]
			marker := "  "
			if name == "opencode-go" {
				marker = "-> "
			}

			fmt.Fprintf(&out, "  %s%s\n", marker, name)

			if p.Type != "" {
				fmt.Fprintf(&out, "       Type: %s\n", p.Type)
			}

			if name == "opencode-go" && p.Key != "" {
				keyID := auth.ExtractKeyID(p.Key)
				fmt.Fprintf(&out, "       Key:  sk-...%s\n", keyID)

				if cfg != nil {
					for accountName, acc := range cfg.Accounts {
						if acc.KeyID == keyID {
							fmt.Fprintf(&out, "       Account: %s\n", accountName)
							break
						}
					}
				}
			} else if p.Key != "" {
				fmt.Fprintf(&out, "       Key:  sk-...%s\n", auth.ExtractKeyID(p.Key))
			}
			fmt.Fprintln(&out)
		}

		return writeOutput(out.String())
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
