package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/opencode-usage/internal/auth"
	"github.com/opencode-usage/internal/client"
	"github.com/opencode-usage/internal/config"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"m"},
	Short:   "查看可用模型列表",
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

		var b strings.Builder
		b.WriteString("可用模型:\n")
		for _, model := range models {
			fmt.Fprintf(&b, "  - %s (%s)\n", model.Name, model.ID)
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
		return "", fmt.Errorf("请先添加账号: opencode-usage account add")
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
