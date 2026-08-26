package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emmmdty/opencode-usage/internal/auth"
	"github.com/emmmdty/opencode-usage/internal/client"
	"github.com/emmmdty/opencode-usage/internal/config"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"m"},
	Short:   "List available models",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := getAPIKeyForCommand()
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

func getAPIKeyForCommand() (string, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return "", err
	}

	cfg, err := config.LoadOrCreateConfig(configPath)
	if err != nil {
		return "", err
	}
	configureAuthFromConfig(cfg)

	if account != "" {
		return auth.GetAPIKey("opencode-usage", account)
	}

	if len(cfg.Accounts) == 0 {
		return "", fmt.Errorf("no accounts configured. Run 'opencode-usage account add' first")
	}

	names := make([]string, 0, len(cfg.Accounts))
	for name := range cfg.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)

	return auth.GetAPIKey("opencode-usage", names[0])
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
