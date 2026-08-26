package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage shell aliases",
}

var aliasInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install shell alias",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		shell := os.Getenv("SHELL")
		var rcFile string
		if strings.Contains(shell, "zsh") {
			rcFile = homeDir + "/.zshrc"
		} else {
			rcFile = homeDir + "/.bashrc"
		}

		if aliasExists(rcFile, "ou") {
			fmt.Printf("Alias 'ou' already exists in %s\n", rcFile)
			fmt.Print("Overwrite? (y/N): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(response)

			if response != "y" && response != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		alias := "\n# opencode-usage alias\nalias ou='opencode-usage'\n"

		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString(alias); err != nil {
			return err
		}

		fmt.Printf("Alias added to %s\n", rcFile)
		fmt.Println("Run 'source " + rcFile + "' or restart your terminal")

		return nil
	},
}

var aliasUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall shell alias",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		shell := os.Getenv("SHELL")
		var rcFile string
		if strings.Contains(shell, "zsh") {
			rcFile = homeDir + "/.zshrc"
		} else {
			rcFile = homeDir + "/.bashrc"
		}

		if !aliasExists(rcFile, "ou") {
			fmt.Printf("Alias 'ou' not found in %s\n", rcFile)
			return nil
		}

		if err := removeAlias(rcFile, "ou"); err != nil {
			return err
		}

		fmt.Printf("Alias 'ou' removed from %s\n", rcFile)
		fmt.Println("Run 'source " + rcFile + "' or restart your terminal")
		return nil
	},
}

func aliasExists(rcFile, alias string) bool {
	file, err := os.Open(rcFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "alias "+alias+"=") {
			return true
		}
	}

	return false
}

func removeAlias(rcFile, alias string) error {
	file, err := os.Open(rcFile)
	if err != nil {
		return err
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	file.Close()

	aliasLine := "alias " + alias + "="
	var result []string
	for i, line := range lines {
		if strings.Contains(line, aliasLine) {
			if i > 0 && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "# opencode-usage alias" {
				result = result[:len(result)-1]
			}
			continue
		}
		result = append(result, line)
	}

	return os.WriteFile(rcFile, []byte(strings.Join(result, "\n")+"\n"), 0644)
}

func init() {
	aliasCmd.AddCommand(aliasInstallCmd)
	aliasCmd.AddCommand(aliasUninstallCmd)
	rootCmd.AddCommand(aliasCmd)
}
