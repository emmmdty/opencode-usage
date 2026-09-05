package cmd

import (
	"fmt"

	"github.com/emmmdty/token-usage/internal/config"
	"github.com/emmmdty/token-usage/internal/tui"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:     "providers [provider]",
	Aliases: []string{"p"},
	Short:   "View usage grouped by provider (OpenCode, Claude, Codex, Volcano Engine, custom)",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerFilter := ""
		if len(args) > 0 {
			providerFilter = args[0]
		}
		return runProvidersOverview(providerFilter, jsonOutput, outputFile)
	},
}

func runProvidersOverview(providerFilter string, jsonOut bool, outPath string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	cfg, err := config.LoadOrCreateConfig(configPath)
	if err != nil {
		return err
	}

	configureAuthFromConfig(cfg)

	// Same engine as quota; the only difference is the default filter scope
	// (all providers) and grouped rendering.
	if _, _, err := cfg.FindProvider(providerFilter); providerFilter != "" && err != nil {
		return fmt.Errorf("provider '%s' not found or not configured", providerFilter)
	}

	results, err := fetchAndRender(cfg, providerFilter, "", jsonOut)
	if err != nil {
		return err
	}

	if jsonOut {
		resp := providersResponse{Version: "2", Providers: results}
		return printJSON(resp)
	}
	return printProvidersTable(results, cfg)
}

type providerResult = accountResult

type providersResponse struct {
	Version   string           `json:"version"`
	Providers []providerResult `json:"providers"`
}

func printProvidersTable(results []providerResult, cfg *config.Config) error {
	tuiResults := make([]tui.AccountResult, len(results))
	for i, r := range results {
		var note string
		if r.Usage != nil {
			note = r.Usage.Note
		}
		tuiResults[i] = tui.AccountResult{
			Name:      r.Name,
			Usage:     r.Usage,
			Error:     r.Error,
			Note:      note,
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
