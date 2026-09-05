package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emmmdty/token-usage/internal/auth"
	"github.com/emmmdty/token-usage/internal/config"
	"github.com/emmmdty/token-usage/internal/provider"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:     "account",
	Aliases: []string{"a"},
	Short:   "Manage accounts per provider",
}

var (
	acctProvider string
	acctKey      string
	acctUseLocal bool
	acctPlan     string
)

var accountAddCmd = &cobra.Command{
	Use:   "add [provider] [name]",
	Short: "Add an account to a provider",
	Long: `Add an account to a provider.

Presets (claude, codex, volcengine) detect local logins and offer to reuse
them without touching those files. With no arguments this is interactive.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		providerID := acctProvider
		name := ""
		if len(args) > 0 {
			providerID = args[0]
		}
		if len(args) > 1 {
			name = args[1]
		}

		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		reader := bufio.NewReader(os.Stdin)
		if providerID == "" {
			var known []string
			for _, pa := range cfg.AllAccounts() {
				known = appendUnique(known, pa.ProviderID)
			}
			for id := range cfg.Providers {
				known = appendUnique(known, id)
			}
			sort.Strings(known)
			idx, err := promptSelect(reader, "Provider:", known)
			if err != nil {
				return err
			}
			providerID = known[idx]
		}

		if _, _, err := cfg.FindProvider(providerID); err != nil {
			return fmt.Errorf("provider '%s' not found; add it first with 'token-usage provider add'", providerID)
		}

		if isCustom(cfg, providerID) {
			return addCustomAccount(cfg, cfgPath, reader, providerID, name)
		}
		return addPresetProvider(cfg, cfgPath, reader, providerID, addOpts{
			name:     name,
			apiKey:   acctKey,
			useLocal: acctUseLocal,
			plan:     acctPlan,
		})
	},
}

func appendUnique(list []string, v string) []string {
	for _, item := range list {
		if item == v {
			return list
		}
	}
	return append(list, v)
}

func isCustom(cfg *config.Config, id string) bool {
	_, ok := cfg.Custom[id]
	return ok
}

// addCustomAccount stores an additional API key under an existing custom
// provider, validating it live first.
func addCustomAccount(cfg *config.Config, cfgPath string, reader *bufio.Reader, providerID, name string) error {
	custom := cfg.Custom[providerID]
	q, _ := provider.LookupKeyQuery(custom.QueryType)

	apiKey := acctKey
	if apiKey == "" {
		secret, err := promptSecret("  API Key: ")
		if err != nil {
			return err
		}
		apiKey = secret
	}
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	accountName := name
	if accountName == "" {
		v, err := promptInput(reader, "  Account name: ")
		if err != nil {
			return err
		}
		accountName = v
	}
	if strings.ContainsAny(accountName, "/\n\x00:") {
		return fmt.Errorf("account name cannot contain '/', newline, NUL, or ':' characters")
	}
	if _, exists := custom.Accounts[accountName]; exists {
		return fmt.Errorf("account '%s' already exists for provider '%s'", accountName, providerID)
	}

	fmt.Println("  Validating quota query...")
	if _, err := q(apiKey, custom.BaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "\n  %v\n", err)
		return fmt.Errorf("account was NOT saved (validation failed)")
	}

	if err := auth.StoreAPIKey("token-usage", providerID+"/"+accountName, apiKey); err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}
	custom.Accounts[accountName] = config.Account{
		Source:       config.SourceManual,
		KeyID:        auth.ExtractKeyID(apiKey),
		CreatedAt:    timeNow(),
		LastVerified: timeNow(),
	}
	cfg.Custom[providerID] = custom
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		_ = auth.DeleteAPIKey("token-usage", providerID+"/"+accountName)
		return err
	}
	fmt.Printf("\n  Account '%s' added to provider '%s'\n", accountName, providerID)
	return nil
}

var accountListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"al"},
	Short:   "List accounts grouped by provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		return writeOutput(renderAccountList(cfg, acctProvider))
	},
}

func renderAccountList(cfg *config.Config, providerFilter string) string {
	// Group accounts by provider in stable order.
	type group struct {
		id       string
		display  string
		def      string
		accounts []config.ProviderAccount
	}
	groups := map[string]*group{}
	var order []string
	for _, pa := range cfg.AllAccounts() {
		if providerFilter != "" && pa.ProviderID != providerFilter {
			continue
		}
		g, ok := groups[pa.ProviderID]
		if !ok {
			g = &group{
				id:      pa.ProviderID,
				display: displayName(pa.ProviderID, pa.Data.Plan, customPtr(cfg, pa.ProviderID)),
				def:     defaultAccountFor(cfg, pa.ProviderID),
			}
			groups[pa.ProviderID] = g
			order = append(order, pa.ProviderID)
		}
		g.accounts = append(g.accounts, pa)
	}
	if len(order) == 0 {
		return "No accounts configured. Run 'token-usage provider add' to get started.\n"
	}
	sort.Strings(order)

	nameWidth := 7
	for _, id := range order {
		for _, pa := range groups[id].accounts {
			w := runewidth.StringWidth(pa.ProviderID + "/" + pa.Account)
			if w+2 > nameWidth {
				nameWidth = w + 2
			}
		}
	}

	var out strings.Builder
	out.WriteString("\n")
	for _, id := range order {
		g := groups[id]
		fmt.Fprintf(&out, "  %s\n", g.display)
		for _, pa := range g.accounts {
			acc := pa.Data
			status := "unverified"
			lastVerified := "never"
			if !acc.LastVerified.IsZero() {
				lastVerified = formatRelativeTime(acc.LastVerified)
				if time.Since(acc.LastVerified) < 24*time.Hour {
					status = "ok"
				} else {
					status = "stale"
				}
			}
			marker := "  "
			if pa.Account == g.def {
				marker = "-> "
			}
			keyID := ""
			if acc.KeyID != "" {
				keyID = "  Key: sk-..." + acc.KeyID
			}
			padded := pa.ProviderID + "/" + pa.Account
			padded += strings.Repeat(" ", nameWidth-runewidth.StringWidth(padded))
			fmt.Fprintf(&out, "  %s%s  Source: %-5s %s  Status: %-10s  Last verified: %s\n",
				marker, padded, acc.Source, keyID, status, lastVerified)
		}
		out.WriteString("\n")
	}
	return out.String()
}

var accountRemoveCmd = &cobra.Command{
	Use:     "remove <provider> <account>",
	Aliases: []string{"ar"},
	Short:   "Remove an account",
	Args:    cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		providerID, accountName, err := resolveProviderAccount(cfg, args)
		if err != nil {
			return err
		}

		if p, ok := cfg.Providers[providerID]; ok {
			delete(p.Accounts, accountName)
			if p.DefaultAccount == accountName {
				p.DefaultAccount = ""
			}
			cfg.Providers[providerID] = p
		} else if c, ok := cfg.Custom[providerID]; ok {
			delete(c.Accounts, accountName)
			if c.DefaultAccount == accountName {
				c.DefaultAccount = ""
			}
			cfg.Custom[providerID] = c
		}

		if err := config.SaveConfig(cfg, cfgPath); err != nil {
			return err
		}

		if err := auth.DeleteAPIKey("token-usage", providerID+"/"+accountName); err != nil {
			return fmt.Errorf("account removed from config but key deletion failed: %w", err)
		}
		fmt.Printf("Account '%s/%s' removed\n", providerID, accountName)
		return nil
	},
}

// resolveProviderAccount turns CLI args into a (provider, account) pair.
// Accepts "provider account" or just "account" when the name is unique.
func resolveProviderAccount(cfg *config.Config, args []string) (string, string, error) {
	if len(args) == 2 {
		providerID, accountName := args[0], args[1]
		accounts, _, err := cfg.FindProvider(providerID)
		if err != nil {
			return "", "", err
		}
		if _, ok := accounts[accountName]; !ok {
			return "", "", fmt.Errorf("account '%s' not found for provider '%s'", accountName, providerID)
		}
		return providerID, accountName, nil
	}
	if len(args) == 1 {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) == 2 {
			return resolveProviderAccount(cfg, parts)
		}
		var matches []config.ProviderAccount
		for _, pa := range cfg.AllAccounts() {
			if pa.Account == args[0] {
				matches = append(matches, pa)
			}
		}
		switch len(matches) {
		case 0:
			return "", "", fmt.Errorf("account '%s' not found", args[0])
		case 1:
			return matches[0].ProviderID, matches[0].Account, nil
		default:
			var ids []string
			for _, m := range matches {
				ids = append(ids, m.ProviderID+"/"+m.Account)
			}
			return "", "", fmt.Errorf("account '%s' is ambiguous (%s); specify provider", args[0], strings.Join(ids, ", "))
		}
	}
	return "", "", fmt.Errorf("usage: token-usage account remove <provider> <account>")
}

var accountSwitchCmd = &cobra.Command{
	Use:     "switch [provider] [account]",
	Aliases: []string{"sw"},
	Short:   "Mark an account as current",
	Long: `Mark an account as the current one for its provider.

For the opencode provider this also writes the key into opencode's own
auth.json (the classic switch), so the change applies to opencode after
running /connect there. Other providers only record the current marker used
  by 'quota' and 'providers' output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		// A single argument that names an existing provider means "pick an
		// account of that provider interactively".
		if len(args) == 1 && !strings.Contains(args[0], "/") {
			if accounts, _, err := cfg.FindProvider(args[0]); err == nil {
				names := sortedAccountNames(accounts)
				if len(names) == 0 {
					return fmt.Errorf("provider '%s' has no accounts", args[0])
				}
				if len(names) == 1 {
					args = []string{args[0], names[0]}
				} else {
					reader := bufio.NewReader(os.Stdin)
					opts := make([]string, len(names))
					def := defaultAccountFor(cfg, args[0])
					for i, n := range names {
						marker := ""
						if n == def {
							marker = " (current)"
						}
						keyID := accounts[n].KeyID
						if keyID != "" {
							keyID = "  sk-..." + keyID
						}
						opts[i] = n + keyID + marker
					}
					idx, err := promptSelect(reader, "Switch account for '"+args[0]+"':", opts)
					if err != nil {
						return err
					}
					args = []string{args[0], names[idx]}
				}
			}
		}

		providerID, accountName, err := resolveProviderAccount(cfg, args)
		if err != nil {
			return err
		}

		// Record the marker.
		if p, ok := cfg.Providers[providerID]; ok {
			p.DefaultAccount = accountName
			cfg.Providers[providerID] = p
		} else if c, ok := cfg.Custom[providerID]; ok {
			c.DefaultAccount = accountName
			cfg.Custom[providerID] = c
		}
		if err := config.SaveConfig(cfg, cfgPath); err != nil {
			return err
		}

		fmt.Printf("  Current account for '%s': %s\n", providerID, accountName)

		// opencode extra behavior: sync opencode auth.json.
		if providerID == "opencode" {
			apiKey, err := resolveStoredKey(providerID, accountName)
			if err != nil {
				return fmt.Errorf("failed to retrieve API key: %w", err)
			}
			providers, err := readAuthJSON()
			if err != nil {
				return err
			}
			providers["opencode-go"] = authProvider{Type: "api", Key: apiKey}
			if err := writeAuthJSON(providers); err != nil {
				return err
			}
			fmt.Printf("  opencode auth.json updated (sk-...%s)\n", auth.ExtractKeyID(apiKey))
			if switchClipboard {
				if err := copyToClipboard(apiKey); err != nil {
					fmt.Printf("  Could not copy key to clipboard: %v\n", err)
				} else {
					fmt.Println("  API key copied to clipboard!")
					fmt.Println("  WARNING: Clear it after use (run 'token-usage account clear-clipboard').")
				}
			}
			fmt.Println("  Run /connect in opencode to apply the change.")
		}
		return nil
	},
}

var accountTestCmd = &cobra.Command{
	Use:     "test [provider] [account]",
	Aliases: []string{"t"},
	Short:   "Validate that quota querying works for an account",
	Args:    cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		providerID, accountName, err := resolveProviderAccount(cfg, args)
		if err != nil {
			return err
		}

		targets, _, err := buildTargets(cfg, providerID, accountName)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return fmt.Errorf("cannot build a query for %s/%s (missing credentials or disabled provider)", providerID, accountName)
		}

		fmt.Printf("  Testing %s/%s...\n", providerID, accountName)
		usage, err := targets[0].query()
		if err != nil {
			return fmt.Errorf("FAILED: %w", err)
		}
		touchLastVerified(cfg, providerID, accountName, cfgPath)
		fmt.Printf("  OK: plan=%s rolling=%s%% weekly=%s%% monthly=%s%%\n",
			usage.PlanType,
			windowPct(usage.Rolling), windowPct(usage.Weekly), windowPct(usage.Monthly))
		if usage.Note != "" {
			fmt.Printf("  %s\n", usage.Note)
		}
		return nil
	},
}

func windowPct(w provider.QuotaWindow) string {
	if w.Status == provider.StatusUnknown {
		return "n/a"
	}
	return strconv.Itoa(w.Percent)
}

type ExportAccount struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	KeyID    string `json:"key_id"`
}

type ExportData struct {
	Accounts []ExportAccount `json:"accounts"`
}

type ImportAccount struct {
	Provider string `json:"provider,omitempty"`
	Name     string `json:"name"`
	APIKey   string `json:"api_key"`
}

type ImportData struct {
	Accounts []ImportAccount `json:"accounts"`
}

var accountExportCmd = &cobra.Command{
	Use:     "export",
	Aliases: []string{"ae"},
	Short:   "Export account metadata (no secrets)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		var exportData ExportData
		for _, pa := range cfg.AllAccounts() {
			exportData.Accounts = append(exportData.Accounts, ExportAccount{
				Provider: pa.ProviderID,
				Name:     pa.Account,
				KeyID:    pa.Data.KeyID,
			})
		}
		if len(exportData.Accounts) == 0 {
			return fmt.Errorf("no accounts configured")
		}

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
	Use:     "import <file>",
	Aliases: []string{"ai"},
	Short:   "Import accounts from a JSON file",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
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

		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		configureAuthFromConfig(cfg)

		var imported, skipped int
		for _, entry := range importData.Accounts {
			providerID := entry.Provider
			if providerID == "" {
				providerID = "opencode" // legacy exports were opencode keys
			}
			if entry.Name == "" || entry.APIKey == "" {
				fmt.Printf("Skipping invalid entry: name=%q\n", entry.Name)
				skipped++
				continue
			}
			if strings.ContainsAny(entry.Name, "/\n\x00:") {
				fmt.Printf("Skipping account '%s': invalid characters\n", entry.Name)
				skipped++
				continue
			}

			accounts, _, err := cfg.FindProvider(providerID)
			if err != nil {
				fmt.Printf("Skipping account '%s': unknown provider '%s'\n", entry.Name, providerID)
				skipped++
				continue
			}
			if _, exists := accounts[entry.Name]; exists {
				fmt.Printf("Skipping existing account: '%s/%s'\n", providerID, entry.Name)
				skipped++
				continue
			}

			if err := auth.StoreAPIKey("token-usage", providerID+"/"+entry.Name, entry.APIKey); err != nil {
				fmt.Printf("Skipping account '%s': storage error: %v\n", entry.Name, err)
				skipped++
				continue
			}

			acc := config.Account{
				Source:    config.SourceManual,
				KeyID:     auth.ExtractKeyID(entry.APIKey),
				CreatedAt: timeNow(),
			}
			if p, ok := cfg.Providers[providerID]; ok {
				if p.Accounts == nil {
					p.Accounts = map[string]config.Account{}
				}
				p.Accounts[entry.Name] = acc
				cfg.Providers[providerID] = p
			} else if c, ok := cfg.Custom[providerID]; ok {
				c.Accounts[entry.Name] = acc
				cfg.Custom[providerID] = c
			}
			if err := config.SaveConfig(cfg, cfgPath); err != nil {
				_ = auth.DeleteAPIKey("token-usage", providerID+"/"+entry.Name)
				fmt.Printf("Skipping account '%s': config save error: %v\n", entry.Name, err)
				skipped++
				continue
			}
			fmt.Printf("Imported account '%s/%s'\n", providerID, entry.Name)
			imported++
		}
		fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)
		return nil
	},
}

func readAuthJSON() (map[string]authProvider, error) {
	data, err := os.ReadFile(opencodeAuthPath())
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
	authDir := filepathDir(opencodeAuthPath())
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
	if err := os.Rename(tmpPath, opencodeAuthPath()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace auth.json: %w", err)
	}
	return nil
}

var switchClipboard bool

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
	accountAddCmd.Flags().StringVarP(&acctProvider, "provider", "p", "", "target provider")
	accountAddCmd.Flags().StringVar(&acctKey, "key", "", "API key (prompts interactively when omitted)")
	accountAddCmd.Flags().BoolVar(&acctUseLocal, "use-local", false, "reuse the locally detected account")
	accountAddCmd.Flags().StringVar(&acctPlan, "plan", "", "volcengine plan (coding|agent)")
	accountListCmd.Flags().StringVarP(&acctProvider, "provider", "p", "", "filter by provider")

	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountTestCmd)
	accountCmd.AddCommand(accountExportCmd)
	accountCmd.AddCommand(accountImportCmd)
	accountCmd.AddCommand(clearClipboardCmd)
	accountSwitchCmd.Flags().BoolVarP(&switchClipboard, "clipboard", "c", false, "copy API key to clipboard after switching (opencode only)")
	rootCmd.AddCommand(accountCmd)
}
