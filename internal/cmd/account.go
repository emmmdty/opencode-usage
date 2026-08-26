package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/opencode-usage/internal/auth"
	"github.com/opencode-usage/internal/config"
	"golang.org/x/term"
)

var accountCmd = &cobra.Command{
	Use:     "account",
	Aliases: []string{"a"},
	Short:   "管理OpenCode Go账号",
}

var accountAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"aa"},
	Short:   "添加新账号",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("请输入账号名称: ")
		name, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取账号名称失败: %w", err)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("账号名称不能为空")
		}

		fmt.Print("请输入API Key: ")
		apiKeyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("读取API Key失败: %w", err)
		}
		fmt.Println()
		apiKey := strings.TrimSpace(string(apiKeyBytes))
		if apiKey == "" {
			return fmt.Errorf("API Key不能为空")
		}

		fmt.Println("正在验证API Key...")
		result, err := auth.ValidateAPIKey(apiKey, "")
		if err != nil {
			return fmt.Errorf("验证API Key时出错: %w", err)
		}
		if !result.Valid {
			return fmt.Errorf("API Key无效: %s", result.Message)
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
			return fmt.Errorf("账号 '%s' 已存在，请使用 'opencode-usage account remove %s' 删除后重新添加", name, name)
		}

		if err := auth.StoreAPIKey("opencode-usage", name, apiKey); err != nil {
			return fmt.Errorf("存储API Key失败: %w", err)
		}

		cfg.Accounts[name] = config.Account{
			Name:         name,
			KeyID:        auth.ExtractKeyID(apiKey),
			CreatedAt:    timeNow(),
			LastVerified: timeNow(),
		}

		if err := config.SaveConfig(cfg, configPath); err != nil {
			return err
		}

		fmt.Printf("账号 '%s' 添加成功\n", name)
		return nil
	},
}

func timeNow() time.Time {
	return time.Now()
}

var accountListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"al"},
	Short:   "查看所有账号",
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
			fmt.Println("暂无配置的账号")
			return nil
		}

		names := make([]string, 0, len(cfg.Accounts))
		for name := range cfg.Accounts {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			account := cfg.Accounts[name]
			status := "未验证"
			lastVerified := "从未验证"
			if !account.LastVerified.IsZero() {
				lastVerified = account.LastVerified.Format("2006-01-02 15:04:05")
				if time.Since(account.LastVerified) < 24*time.Hour {
					status = "正常"
				} else {
					status = "可能过期"
				}
			}
			fmt.Printf("账号: %-12s Key ID: sk-...%-6s 状态: %-8s 上次验证: %s\n",
				name, account.KeyID, status, lastVerified)
		}
		return nil
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:     "remove",
	Aliases: []string{"ar"},
	Short:   "删除账号",
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
			return fmt.Errorf("账号 '%s' 不存在", accountName)
		}

		if err := auth.DeleteAPIKey("opencode-usage", accountName); err != nil {
			return err
		}

		delete(cfg.Accounts, accountName)

		if err := config.SaveConfig(cfg, configPath); err != nil {
			return err
		}

		fmt.Printf("账号 '%s' 已删除\n", accountName)
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
	Short:   "导出账号配置",
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
			return fmt.Errorf("暂无配置的账号")
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
			return fmt.Errorf("序列化JSON失败: %w", err)
		}

		if outputFile != "" {
			if err := os.WriteFile(outputFile, jsonData, 0600); err != nil {
				return fmt.Errorf("写入文件失败: %w", err)
			}
			fmt.Printf("账号配置已导出到: %s\n", outputFile)
		} else {
			fmt.Println(string(jsonData))
		}
		return nil
	},
}

var accountImportCmd = &cobra.Command{
	Use:     "import",
	Aliases: []string{"ai"},
	Short:   "导入账号配置",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取文件失败: %w", err)
		}

		var importData ImportData
		if err := json.Unmarshal(data, &importData); err != nil {
			return fmt.Errorf("解析JSON失败: %w", err)
		}

		if len(importData.Accounts) == 0 {
			return fmt.Errorf("没有找到可导入的账号")
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
				fmt.Printf("跳过无效条目: name=%q\n", account.Name)
				skipped++
				continue
			}

			if _, exists := cfg.Accounts[account.Name]; exists {
				fmt.Printf("跳过已存在的账号: '%s'\n", account.Name)
				skipped++
				continue
			}

			fmt.Printf("正在验证账号 '%s' 的API Key...\n", account.Name)
			result, err := auth.ValidateAPIKey(account.APIKey, "")
			if err != nil {
				fmt.Printf("跳过账号 '%s': 验证出错: %v\n", account.Name, err)
				skipped++
				continue
			}
			if !result.Valid {
				fmt.Printf("跳过账号 '%s': API Key无效: %s\n", account.Name, result.Message)
				skipped++
				continue
			}

			if err := auth.StoreAPIKey("opencode-usage", account.Name, account.APIKey); err != nil {
				fmt.Printf("跳过账号 '%s': 存储失败: %v\n", account.Name, err)
				skipped++
				continue
			}

			cfg.Accounts[account.Name] = config.Account{
				Name:         account.Name,
				KeyID:        auth.ExtractKeyID(account.APIKey),
				CreatedAt:    timeNow(),
				LastVerified: timeNow(),
			}
			fmt.Printf("导入账号 '%s' 成功\n", account.Name)
			imported++
		}

		if err := config.SaveConfig(cfg, configPath); err != nil {
			return err
		}

		fmt.Printf("\n导入完成: 成功 %d, 跳过 %d\n", imported, skipped)
		return nil
	},
}

var accountAddAliasCmd = &cobra.Command{
	Use:   "aa",
	Short: "添加新账号 (account add别名)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return accountAddCmd.RunE(cmd, args)
	},
}

var accountListAliasCmd = &cobra.Command{
	Use:   "al",
	Short: "查看所有账号 (account list别名)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return accountListCmd.RunE(cmd, args)
	},
}

var accountRemoveAliasCmd = &cobra.Command{
	Use:   "ar",
	Short: "删除账号 (account remove别名)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return accountRemoveCmd.RunE(cmd, args)
	},
}

var accountExportAliasCmd = &cobra.Command{
	Use:   "ae",
	Short: "导出账号配置 (account export别名)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return accountExportCmd.RunE(cmd, args)
	},
}

var accountImportAliasCmd = &cobra.Command{
	Use:   "ai",
	Short: "导入账号配置 (account import别名)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return accountImportCmd.RunE(cmd, args)
	},
}

func init() {
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountRemoveCmd)
	accountCmd.AddCommand(accountExportCmd)
	accountCmd.AddCommand(accountImportCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(accountAddAliasCmd)
	rootCmd.AddCommand(accountListAliasCmd)
	rootCmd.AddCommand(accountRemoveAliasCmd)
	rootCmd.AddCommand(accountExportAliasCmd)
	rootCmd.AddCommand(accountImportAliasCmd)
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir + "/.config/opencode-usage/config.yaml", nil
}
