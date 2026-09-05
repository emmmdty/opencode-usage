package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emmmdty/token-usage/internal/client"
	"github.com/emmmdty/token-usage/internal/config"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:     "models [account]",
	Aliases: []string{"m"},
	Short:   "List available models (opencode provider)",
	Long: `List the models available for an OpenCode Go plan.

The optional argument is an opencode account name (or provider/account);
it defaults to the alphabetically first opencode account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := account
		if len(args) > 0 {
			accountName = args[0]
		}
		apiKey, err := getAPIKeyForCommand(accountName)
		if err != nil {
			return err
		}

		c := client.NewClient(apiKey, "")
		models, err := c.GetModels()
		if err != nil {
			return err
		}

		if jsonOutput {
			return printJSON(models)
		}

		if len(models) == 0 {
			return writeOutput("  No models available for your plan.\n")
		}

		var b strings.Builder
		b.WriteString("\n  Available models:\n\n")
		for _, model := range models {
			fmt.Fprintf(&b, "    %-30s  %s\n", model.Name, model.ID)
		}
		return writeOutput(b.String())
	},
}

// getAPIKeyForCommand resolves an opencode account key. Accepts either the
// bare account name or "provider/account"; defaults to the first opencode
// account.
func getAPIKeyForCommand(accountName string) (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	cfg, err := config.LoadOrCreateConfig(configPath)
	if err != nil {
		return "", err
	}
	configureAuthFromConfig(cfg)

	if accountName != "" {
		if idx := strings.Index(accountName, "/"); idx >= 0 {
			return resolveStoredKey(accountName[:idx], accountName[idx+1:])
		}
		return resolveStoredKey("opencode", accountName)
	}

	var names []string
	for _, pa := range cfg.AllAccounts() {
		if pa.ProviderID == "opencode" {
			names = append(names, pa.Account)
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no opencode accounts configured. Run 'token-usage account add opencode' first")
	}
	sort.Strings(names)

	return resolveStoredKey("opencode", names[0])
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
