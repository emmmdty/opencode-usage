package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/emmmdty/opencode-usage/internal/config"
	"github.com/emmmdty/opencode-usage/internal/models"
	"github.com/emmmdty/opencode-usage/internal/provider"
	"github.com/emmmdty/opencode-usage/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type providerResult struct {
	Name      string        `json:"name"`
	PlanType  string        `json:"plan_type,omitempty"`
	Usage     *models.Usage `json:"usage,omitempty"`
	Error     string        `json:"error,omitempty"`
	IsCurrent bool          `json:"is_current,omitempty"`
}

type providersResponse struct {
	Version   string           `json:"version"`
	Providers []providerResult `json:"providers"`
}

var providersCmd = &cobra.Command{
	Use:     "providers",
	Aliases: []string{"p"},
	Short:   "View usage across all providers (OpenCode, Claude, Codex, Volcengine)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProvidersOverview(jsonOutput, outputFile)
	},
}

func runProvidersOverview(jsonOut bool, outPath string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	cfg, err := config.LoadOrCreateConfig(configPath)
	if err != nil {
		return err
	}

	providers := buildProviders(cfg)
	if len(providers) == 0 {
		if jsonOut {
			resp := providersResponse{Version: "1", Providers: []providerResult{}}
			return printJSON(resp)
		}
		return writeOutput("  No providers configured. Check ~/.config/opencode-usage/config.yaml\n")
	}

	if !jsonOut && !term.IsTerminal(int(os.Stdout.Fd())) {
		// non-TTY: skip spinner
	} else if !jsonOut {
		fmt.Fprintf(os.Stderr, "  Fetching %d providers...\r", len(providers))
	}

	maxConcurrent := cfg.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	sem := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	results := make(chan struct {
		name  string
		usage *provider.Usage
		err   error
	}, len(providers))

	for name, p := range providers {
		wg.Add(1)
		sem <- struct{}{}
		go func(name string, p provider.Provider) {
			defer wg.Done()
			defer func() { <-sem }()

			usage, err := p.GetUsage()
			results <- struct {
				name  string
				usage *provider.Usage
				err   error
			}{name, usage, err}
		}(name, p)
	}

	wg.Wait()
	close(results)

	var providerResults []providerResult
	for result := range results {
		pr := providerResult{
			Name:      result.name,
			IsCurrent: false,
		}
		if result.err != nil {
			pr.Error = result.err.Error()
		} else {
			pr.Usage = models.FromProviderUsage(result.usage)
			pr.PlanType = result.usage.PlanType
		}
		providerResults = append(providerResults, pr)
	}

	sort.Slice(providerResults, func(i, j int) bool {
		return providerResults[i].Name < providerResults[j].Name
	})

	if jsonOut {
		resp := providersResponse{Version: "1", Providers: providerResults}
		return printJSON(resp)
	}

	return printProvidersTable(providerResults, cfg)
}

func buildProviders(cfg *config.Config) map[string]provider.Provider {
	providers := make(map[string]provider.Provider)
	home, _ := os.UserHomeDir()

	// OpenCode - 使用原有的 account 系统
	for name, acc := range cfg.Accounts {
		providers["opencode:"+name] = provider.NewOpenCodeProvider(acc.KeyID)
	}

	// Claude
	if claudeCfg, ok := cfg.Providers["claude"]; ok && claudeCfg.Enabled {
		credsPath := claudeCfg.CredsPath
		if credsPath == "" {
			credsPath = filepath.Join(home, ".claude", ".credentials.json")
		}
		p := provider.NewClaudeProviderWithEndpoint(credsPath, claudeCfg.Endpoint)
		if p.IsAvailable() {
			providers["claude"] = p
		}
	}

	// Codex
	if codexCfg, ok := cfg.Providers["codex"]; ok && codexCfg.Enabled {
		authPath := codexCfg.AuthPath
		if authPath == "" {
			authPath = filepath.Join(home, ".codex", "auth.json")
		}
		p := provider.NewCodexProviderWithEndpoint(authPath, codexCfg.Endpoint)
		if p.IsAvailable() {
			providers["codex"] = p
		}
	}

	// Volcengine
	if volcCfg, ok := cfg.Providers["volcengine"]; ok && volcCfg.Enabled && volcCfg.APIKey != "" {
		p := provider.NewVolcengineProviderWithEndpoint(volcCfg.APIKey, volcCfg.Endpoint)
		if p.IsAvailable() {
			providers["volcengine"] = p
		}
	}

	return providers
}

func printProvidersTable(results []providerResult, cfg *config.Config) error {
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

	output := tui.FormatQuotaOverview(tuiResults, style, "")
	return writeOutput(output)
}

func init() {
	rootCmd.AddCommand(providersCmd)
}
