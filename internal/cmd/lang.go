package cmd

import (
	"fmt"

	"github.com/emmmdty/token-usage/internal/config"
	"github.com/emmmdty/token-usage/internal/i18n"
	"github.com/spf13/cobra"
)

var langCmd = &cobra.Command{
	Use:     "lang [zh|en]",
	Aliases: []string{"language"},
	Short:   i18n.T("cmd.lang.short"),
	Long:    i18n.T("cmd.lang.long"),
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			current := i18n.GetLanguage()
			fmt.Printf("Current language: %s\n", current)
			return nil
		}
		newLang := args[0]
		if newLang != "zh" && newLang != "en" {
			return fmt.Errorf(i18n.T("lang.unsupported"), newLang)
		}
		cfgPath, err := getConfigPath()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOrCreateConfig(cfgPath)
		if err != nil {
			return err
		}
		cfg.Language = newLang
		if err := config.SaveConfig(cfg, cfgPath); err != nil {
			return err
		}
		i18n.SetLanguage(newLang)
		if newLang == "zh" {
			fmt.Println(i18n.T("lang.switched_zh"))
			fmt.Println(i18n.T("lang.switched_en_zh"))
		} else {
			fmt.Println(i18n.T("lang.switched_en"))
			fmt.Println(i18n.T("lang.switched_en_zh2"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(langCmd)
}
