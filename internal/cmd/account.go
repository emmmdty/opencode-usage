package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/emmmdty/opencode-usage/internal/auth"
	"github.com/emmmdty/opencode-usage/internal/config"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var accountCmd = &cobra.Command{
	Use:     "account",
	Aliases: []string{"a"},
	Short:   "Manage OpenCode Go accounts",
}

var accountAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"aa"},
	Short:   "Add a new account",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Account name: ")
		name, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read account name: %w", err)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("account name cannot be empty")
		}
		if strings.ContainsAny(name, "\n\x00:") {
			return fmt.Errorf("account name cannot contain newline, NUL, or ':' characters")
		}

		fmt.Print("API Key: ")
		apiKeyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}
		fmt.Println()
		apiKey := strings.TrimSpace(string(apiKeyBytes))
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		fmt.Println("Validating API key...")
		result, err := auth.ValidateAPIKey(apiKey, "")
		if err != nil {
			return fmt.Errorf("error validating API key: %w", err)
		}
		if !result.Valid {
			return fmt.Errorf("invalid API key: %s", result.Message)
		}

		configPath, err := getConfigPath()
		if err != nil {
			return err
		}

		cfg, err := config.LoadOrCreateConfig(configPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		if _, exists := cfg.Accounts[name]; exists {
			return fmt.Errorf("account '%s' already exists", name)
		}

		// Store the key first: if it fails, config is untouched.
		if err := auth.StoreAPIKey("opencode-usage", name, apiKey); err != nil {
			return fmt.Errorf("failed to store API key: %w", err)
		}

		cfg.Accounts[name] = config.Account{
			Name:         name,
			KeyID:        auth.ExtractKeyID(apiKey),
			CreatedAt:    time.Now(),
			LastVerified: time.Now(),
		}

		if err := config.SaveConfig(cfg, configPath); err != nil {
			delete(cfg.Accounts, name)
			if delErr := auth.DeleteAPIKey("opencode-usage", name); delErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to roll back stored API key for '%s': %v\n", name, delErr)
			}
			return err
		}

		fmt.Printf("Account '%s' added successfully\n", name)
		fmt.Println("Run 'opencode-usage quota' to view all accounts.")
		return nil
	},
}

var accountListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"al"},
	Short:   "List all accounts",
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

		if len(cfg.Accounts) == 0 {
			return writeOutput("No accounts configured. Run 'opencode-usage account add' to get started.\n")
		}

		currentAccount := resolveCurrentAccount(cfg)

		names := make([]string, 0, len(cfg.Accounts))
		for name := range cfg.Accounts {
			names = append(names, name)
		}
		sort.Strings(names)

		nameWidth := 7
		for _, name := range names {
			w := runewidth.StringWidth(name)
			if w+2 > nameWidth {
				nameWidth = w + 2
			}
		}

		var out strings.Builder
		for _, name := range names {
			account := cfg.Accounts[name]
			status := "unverified"
			lastVerified := "never"
			if !account.LastVerified.IsZero() {
				lastVerified = formatRelativeTime(account.LastVerified)
				if time.Since(account.LastVerified) < 24*time.Hour {
					status = "ok"
				} else {
					status = "stale"
				}
			}

			marker := "  "
			if name == currentAccount {
				marker = "-> "
			}

			keyID := "sk-..." + account.KeyID
			paddedName := name + strings.Repeat(" ", nameWidth-runewidth.StringWidth(name))

			fmt.Fprintf(&out, "  %s%s  Key: %-12s  Status: %-10s  Last verified: %s\n",
				marker, paddedName, keyID, status, lastVerified)
		}
		return writeOutput(out.String())
	},
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

var accountRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"ar"},
	Short:   "Remove an account",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountName := args[0]

		configPath, err := getConfigPath()
		if err != nil {
			return err
		}

		cfg, err := config.LoadOrCreateConfig(configPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		if _, exists := cfg.Accounts[accountName]; !exists {
			return fmt.Errorf("account '%s' not found", accountName)
		}

		delete(cfg.Accounts, accountName)

		if err := config.SaveConfig(cfg, configPath); err != nil {
			return err
		}

		if err := auth.DeleteAPIKey("opencode-usage", accountName); err != nil {
			return fmt.Errorf("account removed from config but key deletion failed: %w", err)
		}

		fmt.Printf("Account '%s' removed\n", accountName)
		return nil
	},
}

type ExportAccount struct {
	Name  string `json:"name"`
	KeyID string `json:"key_id"`
}

type ExportData struct {
	Accounts []ExportAccount `json:"accounts"`
}

type ImportAccount struct {
	Name   string `json:"name"`
	APIKey string `json:"api_key"`
}

type ImportData struct {
	Accounts []ImportAccount `json:"accounts"`
}

var accountExportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"ae"},
	Short:   "Export account configuration",
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

		if len(cfg.Accounts) == 0 {
			return fmt.Errorf("no accounts configured")
		}

		var exportData ExportData
		for name, account := range cfg.Accounts {
			exportData.Accounts = append(exportData.Accounts, ExportAccount{
				Name:  name,
				KeyID: account.KeyID,
			})
		}

		sort.Slice(exportData.Accounts, func(i, j int) bool {
			return exportData.Accounts[i].Name < exportData.Accounts[j].Name
		})

		jsonData, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize JSON: %w", err)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, jsonData, 0600); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Printf("Account configuration exported to: %s\n", outputFile)
		} else {
			fmt.Println(string(jsonData))
		}
		return nil
	},
}

var accountImportCmd = &cobra.Command{
	Use:     "import",
	Aliases: []string{"ai"},
	Short:   "Import account configuration",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var importData ImportData
		if err := json.Unmarshal(data, &importData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

		if len(importData.Accounts) == 0 {
			return fmt.Errorf("no importable accounts found")
		}

		configPath, err := getConfigPath()
		if err != nil {
			return err
		}

		cfg, err := config.LoadOrCreateConfig(configPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		var imported, skipped int
		for _, account := range importData.Accounts {
			if account.Name == "" || account.APIKey == "" {
				fmt.Printf("Skipping invalid entry: name=%q\n", account.Name)
				skipped++
				continue
			}

			if strings.ContainsAny(account.Name, "\n\x00:") {
				fmt.Printf("Skipping account '%s': name contains invalid characters\n", account.Name)
				skipped++
				continue
			}

			if _, exists := cfg.Accounts[account.Name]; exists {
				fmt.Printf("Skipping existing account: '%s'\n", account.Name)
				skipped++
				continue
			}

			fmt.Printf("Validating account '%s'...\n", account.Name)
			result, err := auth.ValidateAPIKey(account.APIKey, "")
			if err != nil {
				fmt.Printf("Skipping account '%s': validation error: %v\n", account.Name, err)
				skipped++
				continue
			}
			if !result.Valid {
				fmt.Printf("Skipping account '%s': invalid key: %s\n", account.Name, result.Message)
				skipped++
				continue
			}

			if err := auth.StoreAPIKey("opencode-usage", account.Name, account.APIKey); err != nil {
				fmt.Printf("Skipping account '%s': storage error: %v\n", account.Name, err)
				skipped++
				continue
			}

			cfg.Accounts[account.Name] = config.Account{
				Name:         account.Name,
				KeyID:        auth.ExtractKeyID(account.APIKey),
				CreatedAt:    time.Now(),
				LastVerified: time.Now(),
			}

			if err := config.SaveConfig(cfg, configPath); err != nil {
				_ = auth.DeleteAPIKey("opencode-usage", account.Name)
				fmt.Printf("Skipping account '%s': config save error: %v\n", account.Name, err)
				skipped++
				continue
			}

			fmt.Printf("Imported account '%s'\n", account.Name)
			imported++
		}

		fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)
		return nil
	},
}

func init() {
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountExportCmd)
	accountCmd.AddCommand(accountImportCmd)
	rootCmd.AddCommand(accountCmd)
}
