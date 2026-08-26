package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/opencode-usage/internal/auth"
	"github.com/opencode-usage/internal/config"
	"github.com/opencode-usage/internal/version"
	"github.com/spf13/cobra"
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
		if noColor {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
	},
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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetVersionInfo())
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for new releases on GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", version.Version)
		fmt.Println("Check for updates: https://github.com/emmmdty/opencode-usage/releases")
		fmt.Println("To update, download the latest binary from the releases page.")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "JSON格式输出")
	rootCmd.PersistentFlags().StringVarP(&account, "account", "n", "", "指定账号")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "输出到文件")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "禁用颜色输出")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}

func configureAuthFromConfig(cfg *config.Config) {
	auth.SetUseMasterPassword(cfg.UseMasterPassword)
}
