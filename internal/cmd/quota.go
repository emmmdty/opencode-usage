package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/opencode-usage/internal/auth"
	"github.com/opencode-usage/internal/client"
	"github.com/opencode-usage/internal/config"
	"github.com/opencode-usage/internal/models"
	"github.com/opencode-usage/internal/tui"
)

type accountResult struct {
	Name  string        `json:"name"`
	Usage *models.Usage `json:"quota"`
	Error string        `json:"error,omitempty"`
}

var quotaCmd = &cobra.Command{
	Use:     "quota",
	Aliases: []string{"q"},
	Short:   "查看配额使用情况",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := getConfigPath()
		if err != nil {
			return err
		}

		cfg, err := config.LoadOrCreateConfig(configPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		accountsToQuery := make(map[string]config.Account)
		if account != "" {
			if acc, exists := cfg.Accounts[account]; exists {
				accountsToQuery[account] = acc
			} else {
				return fmt.Errorf("账号 '%s' 不存在", account)
			}
		} else {
			accountsToQuery = cfg.Accounts
		}

		if len(accountsToQuery) == 0 {
			return writeOutput("暂无配置的账号，请先运行 'opencode-usage account add' 添加账号\n")
		}

		maxConcurrent := cfg.MaxConcurrentRequests
		if maxConcurrent <= 0 {
			maxConcurrent = 5
		}
		sem := make(chan struct{}, maxConcurrent)

		var wg sync.WaitGroup
		results := make(chan struct {
			name  string
			usage *models.Usage
			err   error
		}, len(accountsToQuery))

		for name := range accountsToQuery {
			wg.Add(1)
			sem <- struct{}{} // acquire semaphore slot
			go func(name string) {
				defer wg.Done()
				defer func() { <-sem }() // release semaphore slot

				apiKey, err := auth.GetAPIKey("opencode-usage", name)
				if err != nil {
					results <- struct {
						name  string
						usage *models.Usage
						err   error
					}{name, nil, err}
					return
				}

				c := client.NewClient(apiKey, "")
				usage, err := c.GetUsage()
				results <- struct {
					name  string
					usage *models.Usage
					err   error
				}{name, usage, err}
			}(name)
		}

		wg.Wait()
		close(results)

		var accountResults []accountResult
		for result := range results {
			if result.err != nil {
				accountResults = append(accountResults, accountResult{
					Name:  result.name,
					Error: result.err.Error(),
				})
			} else {
				accountResults = append(accountResults, accountResult{
					Name:  result.name,
					Usage: result.usage,
				})
			}
		}

		sort.Slice(accountResults, func(i, j int) bool {
			return accountResults[i].Name < accountResults[j].Name
		})

		if jsonOutput {
			return printJSON(accountResults)
		}

		return printQuotaTable(accountResults)
	},
}

func printJSON(data interface{}) error {
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}
	return writeOutput(buf.String())
}

func printQuotaTable(results []accountResult) error {
	tuiResults := make([]tui.AccountResult, len(results))
	for i, r := range results {
		tuiResults[i] = tui.AccountResult{
			Name:  r.Name,
			Usage: r.Usage,
			Error: r.Error,
		}
	}

	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	cfg, err := config.LoadOrCreateConfig(configPath)
	if err != nil {
		return err
	}

	output := tui.FormatQuotaTable(tuiResults, cfg.ColorThresholds.Warning, cfg.ColorThresholds.Danger)
	return writeOutput(output)
}

func init() {
	rootCmd.AddCommand(quotaCmd)
}
