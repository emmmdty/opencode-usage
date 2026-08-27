package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/emmmdty/opencode-usage/internal/auth"
	"github.com/emmmdty/opencode-usage/internal/client"
	"github.com/emmmdty/opencode-usage/internal/config"
	"github.com/emmmdty/opencode-usage/internal/models"
	"github.com/emmmdty/opencode-usage/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type accountResult struct {
	Name      string        `json:"name"`
	Usage     *models.Usage `json:"quota,omitempty"`
	Error     string        `json:"error,omitempty"`
	IsCurrent bool          `json:"is_current,omitempty"`
}

type quotaResponse struct {
	Version  string          `json:"version"`
	Accounts []accountResult `json:"accounts"`
}

var quotaCmd = &cobra.Command{
	Use:     "quota",
	Aliases: []string{"q"},
	Short:   "View quota usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuotaOverview(account, jsonOutput, outputFile)
	},
}

func runQuotaOverview(accountFilter string, jsonOut bool, outPath string) error {
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
	if accountFilter != "" {
		if acc, exists := cfg.Accounts[accountFilter]; exists {
			accountsToQuery[accountFilter] = acc
		} else {
			return fmt.Errorf("account '%s' not found", accountFilter)
		}
	} else {
		accountsToQuery = cfg.Accounts
	}

	if len(accountsToQuery) == 0 {
		if jsonOut {
			resp := quotaResponse{Version: "1", Accounts: []accountResult{}}
			return printJSON(resp)
		}
		return writeOutput("  No accounts configured. Run 'opencode-usage account add' to get started.\n")
	}

	currentAccount := resolveCurrentAccount(cfg)

	if !jsonOut && !term.IsTerminal(int(os.Stdout.Fd())) {
		// non-TTY: skip spinner
	} else if !jsonOut {
		fmt.Fprintf(os.Stderr, "  Fetching %d accounts...\r", len(accountsToQuery))
	}

	maxConcurrent := cfg.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	sem := make(chan struct{}, maxConcurrent)

	// Resolve credentials for every account up front. This ensures any
	// interactive master-password prompt happens before concurrent workers
	// start, and surfaces a wrong password as a clean per-account error
	// instead of a corrupted read.
	type credResult struct {
		name    string
		apiKey  string
		credErr error
	}
	creds := make([]credResult, 0, len(accountsToQuery))
	for name := range accountsToQuery {
		apiKey, err := auth.GetAPIKey("opencode-usage", name)
		creds = append(creds, credResult{name: name, apiKey: apiKey, credErr: err})
	}

	var wg sync.WaitGroup
	results := make(chan struct {
		name  string
		usage *models.Usage
		err   error
	}, len(creds))

	for _, cred := range creds {
		if cred.credErr != nil {
			results <- struct {
				name  string
				usage *models.Usage
				err   error
			}{cred.name, nil, cred.credErr}
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(name, apiKey string) {
			defer wg.Done()
			defer func() { <-sem }()

			c := client.NewClient(apiKey, "")
			usage, err := c.GetUsage()
			results <- struct {
				name  string
				usage *models.Usage
				err   error
			}{name, usage, err}
		}(cred.name, cred.apiKey)
	}

	wg.Wait()
	close(results)

	var accountResults []accountResult
	for result := range results {
		ar := accountResult{
			Name:      result.name,
			IsCurrent: result.name == currentAccount,
		}
		if result.err != nil {
			ar.Error = result.err.Error()
		} else {
			ar.Usage = result.usage
		}
		accountResults = append(accountResults, ar)
	}

	sort.Slice(accountResults, func(i, j int) bool {
		return accountResults[i].Name < accountResults[j].Name
	})

	if jsonOut {
		resp := quotaResponse{Version: "1", Accounts: accountResults}
		return printJSON(resp)
	}

	return printQuotaTable(accountResults, cfg, currentAccount)
}

func resolveCurrentAccount(cfg *config.Config) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	authPath := homeDir + "/.local/share/opencode/auth.json"

	data, err := os.ReadFile(authPath)
	if err != nil {
		return ""
	}

	var authProviders map[string]struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	if err := json.Unmarshal(data, &authProviders); err != nil {
		return ""
	}

	provider, ok := authProviders["opencode-go"]
	if !ok || provider.Key == "" {
		return ""
	}

	tokenKeyID := auth.ExtractKeyID(provider.Key)
	for name, acc := range cfg.Accounts {
		if acc.KeyID == tokenKeyID {
			return name
		}
	}
	return ""
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

func printQuotaTable(results []accountResult, cfg *config.Config, currentAccount string) error {
	tuiResults := make([]tui.AccountResult, len(results))
	for i, r := range results {
		tuiResults[i] = tui.AccountResult{
			Name:      r.Name,
			Usage:     r.Usage,
			Error:     r.Error,
			IsCurrent: r.IsCurrent,
		}
	}

	style := tui.QuotaStyle{
		WarningThreshold: cfg.ColorThresholds.Warning,
		DangerThreshold:  cfg.ColorThresholds.Danger,
	}
	if style.WarningThreshold == 0 {
		style.WarningThreshold = 50
	}
	if style.DangerThreshold == 0 {
		style.DangerThreshold = 80
	}

	output := tui.FormatQuotaOverview(tuiResults, style, currentAccount)
	return writeOutput(output)
}

func init() {
	rootCmd.AddCommand(quotaCmd)
}
