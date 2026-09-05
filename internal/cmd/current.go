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
	"github.com/emmmdty/token-usage/internal/i18n"
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
	Long: `Show the providers configured in opencode's own auth.json
(~/.local/share/opencode/auth.json).

The arrow (->) marks the active opencode-go entry; when its key matches a
token-usage account, the matching provider/account is printed as well.
Use 'token-usage account switch opencode' to change it.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		authPath := filepath.Join(homeDir, ".local", "share", "opencode", "auth.json")

		if _, err := os.Stat(authPath); os.IsNotExist(err) {
			return writeOutput("  " + i18n.T("output.current.no_config") + "\n")
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
			loaded, err := config.LoadOrCreateConfig(configPath)
			if err != nil {
				// current works without our own config; surface the problem
				// but keep going.
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("warning.current.config_load", err))
			} else {
				cfg = loaded
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
				fmt.Fprintf(&out, "       "+i18n.T("output.current.type")+"\n", p.Type)
			}

			if name == "opencode-go" && p.Key != "" {
				keyID := auth.ExtractKeyID(p.Key)
				fmt.Fprintf(&out, "       "+i18n.T("output.current.key")+"\n", keyID)

				if cfg != nil {
					for _, pa := range cfg.AllAccounts() {
						if pa.Data.KeyID == keyID {
							fmt.Fprintf(&out, "       "+i18n.T("output.current.account")+"\n", pa.ProviderID, pa.Account)
							break
						}
					}
				}
			} else if p.Key != "" {
				fmt.Fprintf(&out, "       "+i18n.T("output.current.key")+"\n", auth.ExtractKeyID(p.Key))
			}
			fmt.Fprintln(&out)
		}

		return writeOutput(out.String())
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
