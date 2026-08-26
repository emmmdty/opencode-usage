package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/emmmdty/opencode-usage/internal/auth"
	"github.com/emmmdty/opencode-usage/internal/config"
	"github.com/emmmdty/opencode-usage/internal/tui"
	"github.com/emmmdty/opencode-usage/internal/version"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	jsonOutput bool
	account    string
	outputFile string
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:     "opencode-usage",
	Aliases: []string{"ou"},
	Short:   "OpenCode Go plan usage query tool",
	Long:    "Query OpenCode Go plan usage across multiple accounts, view available models and quota information.",
	Version: version.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColor || os.Getenv("NO_COLOR") != "" {
			lipgloss.SetColorProfile(termenv.Ascii)
			tui.DisableColor()
		}
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			lipgloss.SetColorProfile(termenv.Ascii)
			tui.DisableColor()
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuotaOverview(account, jsonOutput, outputFile)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func writeOutput(content string) error {
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(content), 0644)
	}
	_, err := os.Stdout.WriteString(content)
	return err
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		return writeOutput(version.GetVersionInfo() + "\n")
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for new releases on GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		var b strings.Builder
		fmt.Fprintf(&b, "Current version: %s\n", version.Version)
		fmt.Fprintln(&b, "Check for updates: https://github.com/emmmdty/opencode-usage/releases")
		fmt.Fprintln(&b, "To update, download the latest binary from the releases page.")
		return writeOutput(b.String())
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "JSON output")
	rootCmd.PersistentFlags().StringVarP(&account, "account", "n", "", "specify account")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "output to file")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}

func configureAuthFromConfig(cfg *config.Config) {
	auth.SetUseMasterPassword(cfg.UseMasterPassword)
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir + "/.config/opencode-usage/config.yaml", nil
}
