package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emmmdty/token-usage/internal/auth"
	"github.com/emmmdty/token-usage/internal/config"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var accountCmd = &cobra.Command{
	Use:     "account",
	Aliases: []string{"a"},
	Short:   "Manage Token Usage accounts",
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
		if err := auth.StoreAPIKey("token-usage", name, apiKey); err != nil {
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
			if delErr := auth.DeleteAPIKey("token-usage", name); delErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to roll back stored API key for '%s': %v\n", name, delErr)
			}
			return err
		}

		fmt.Printf("Account '%s' added successfully\n", name)
		fmt.Println("Run 'token-usage quota' to view all accounts.")
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
			return writeOutput("No accounts configured. Run 'token-usage account add' to get started.\n")
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

		if err := auth.DeleteAPIKey("token-usage", accountName); err != nil {
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

			if err := auth.StoreAPIKey("token-usage", account.Name, account.APIKey); err != nil {
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
				_ = auth.DeleteAPIKey("token-usage", account.Name)
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

func readAuthJSON() (map[string]authProvider, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	authPath := filepath.Join(homeDir, ".local", "share", "opencode", "auth.json")

	data, err := os.ReadFile(authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth.json: %w", err)
	}

	var providers map[string]authProvider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("failed to parse auth.json: %w", err)
	}
	return providers, nil
}

func writeAuthJSON(providers map[string]authProvider) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	authDir := filepath.Join(homeDir, ".local", "share", "opencode")
	authPath := filepath.Join(authDir, "auth.json")

	if err := os.MkdirAll(authDir, 0700); err != nil {
		return fmt.Errorf("failed to create auth directory: %w", err)
	}

	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth.json: %w", err)
	}

	tmpFile, err := os.CreateTemp(authDir, "auth.json.tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, authPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace auth.json: %w", err)
	}

	return nil
}

var switchClipboard bool

var accountSwitchCmd = &cobra.Command{
	Use:     "switch [account-name]",
	Aliases: []string{"sw"},
	Short:   "Switch active opencode-go account",
	Long: `Switch the opencode-go API key used by opencode.

Without arguments, shows an interactive numbered menu to select an account.
With an argument, switches directly to the named account.

The switch updates ~/.local/share/opencode/auth.json. You still need to
run /connect in opencode for the change to take effect (opencode does not
hot-reload auth.json).

Use --clipboard to copy the API key to the system clipboard for easy
pasting during /connect. Clear the clipboard afterwards with:
  tu account clear-clipboard`,
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
			return fmt.Errorf("no accounts configured. Run 'token-usage account add' to get started")
		}

		names := make([]string, 0, len(cfg.Accounts))
		for name := range cfg.Accounts {
			names = append(names, name)
		}
		sort.Strings(names)

		var targetName string

		if len(args) == 1 {
			targetName = args[0]
			if _, exists := cfg.Accounts[targetName]; !exists {
				return fmt.Errorf("account '%s' not found", targetName)
			}
		} else {
			currentAccount := resolveCurrentAccount(cfg)

			fmt.Println()
			fmt.Println("  Available accounts:")
			fmt.Println()
			for i, name := range names {
				marker := "  "
				if name == currentAccount {
					marker = "-> "
				}
				keyID := "sk-..." + cfg.Accounts[name].KeyID
				fmt.Printf("  %s%d) %-30s %s\n", marker, i+1, name, keyID)
			}
			fmt.Println()

			fmt.Print("  Select account [number]: ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			input = strings.TrimSpace(input)

			var idx int
			if idx, err = strconv.Atoi(input); err != nil || idx < 1 || idx > len(names) {
				return fmt.Errorf("invalid selection: %s", input)
			}
			targetName = names[idx-1]
		}

		apiKey, err := auth.GetAPIKey("token-usage", targetName)
		if err != nil {
			return fmt.Errorf("failed to retrieve API key for '%s': %w", targetName, err)
		}

		providers, err := readAuthJSON()
		if err != nil {
			return err
		}

		providers["opencode-go"] = authProvider{
			Type: "api",
			Key:  apiKey,
		}

		if err := writeAuthJSON(providers); err != nil {
			return err
		}

		keyID := auth.ExtractKeyID(apiKey)
		fmt.Printf("\n  Switched to account: %s (sk-...%s)\n", targetName, keyID)

		if switchClipboard {
			if err := copyToClipboard(apiKey); err != nil {
				fmt.Printf("  Could not copy key to clipboard: %v\n", err)
				fmt.Println("  Run /connect in opencode and enter the key manually.")
			} else {
				fmt.Println("  API key copied to clipboard!")
				fmt.Println("  Run /connect in opencode, then Ctrl+V to paste.")
				fmt.Println("  WARNING: Clear your clipboard after use (e.g. run 'tu account clear-clipboard').")
			}
		} else {
			fmt.Println("  Run /connect in opencode to apply the change.")
			fmt.Println("  Tip: use --clipboard to copy the key for easy pasting.")
		}
		fmt.Println()

		return nil
	},
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("clip.exe"); err == nil {
			cmd = exec.Command("clip.exe")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip, xsel, or run in WSL)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

var clearClipboardCmd = &cobra.Command{
	Use:     "clear-clipboard",
	Aliases: []string{"clc"},
	Short:   "Clear the system clipboard",
	Long:    "Clear the system clipboard to remove any API key that was copied by 'account switch --clipboard'.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := copyToClipboard(""); err != nil {
			return fmt.Errorf("failed to clear clipboard: %w", err)
		}
		fmt.Println("  Clipboard cleared.")
		return nil
	},
}

func init() {
	accountSwitchCmd.Flags().BoolVarP(&switchClipboard, "clipboard", "c", false, "copy API key to clipboard after switching")
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountExportCmd)
	accountCmd.AddCommand(accountImportCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(clearClipboardCmd)
	rootCmd.AddCommand(accountCmd)
}
