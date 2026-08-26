package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type authConfig struct {
	Provider string `json:"provider"`
	Token    string `json:"token"`
	BaseURL  string `json:"base_url"`
}

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"cc"},
	Short:   "显示当前opencode配置的账号",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}
		authPath := homeDir + "/.local/share/opencode/auth.json"

		if _, err := os.Stat(authPath); os.IsNotExist(err) {
			fmt.Println("未找到opencode配置文件")
			return nil
		}

		data, err := os.ReadFile(authPath)
		if err != nil {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}

		var authCfg authConfig
		if err := json.Unmarshal(data, &authCfg); err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}

		fmt.Println("当前opencode配置:")
		if authCfg.Provider != "" {
			fmt.Printf("  Provider: %s\n", authCfg.Provider)
		} else {
			fmt.Println("  Provider: opencode-go")
		}
		if authCfg.BaseURL != "" {
			fmt.Printf("  Base URL: %s\n", authCfg.BaseURL)
		}
		if authCfg.Token != "" {
			masked := "..." + authCfg.Token[len(authCfg.Token)-min(6, len(authCfg.Token)):]
			fmt.Printf("  Token: %s\n", masked)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
