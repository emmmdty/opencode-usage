package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install updates",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", version.Version)
		fmt.Println("Checking for updates...")

		release, err := getLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		currentVersion := strings.TrimPrefix(version.Version, "v")

		if latestVersion == currentVersion {
			fmt.Println("Already up to date.")
			return nil
		}

		fmt.Printf("New version available: %s (current: %s)\n", latestVersion, currentVersion)

		binaryURL, err := getBinaryURL(release)
		if err != nil {
			return fmt.Errorf("no binary found for your platform (%s/%s): %w", runtime.GOOS, runtime.GOARCH, err)
		}

		fmt.Printf("Downloading %s...\n", filepath.Base(binaryURL))

		tmpFile, err := downloadBinary(binaryURL)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		defer os.Remove(tmpFile)

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot determine executable path: %w", err)
		}

		if err := os.Chmod(tmpFile, 0755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}

		if err := os.Rename(tmpFile, execPath); err != nil {
			return fmt.Errorf("failed to replace binary: %w\nYou may need to run with sudo", err)
		}

		fmt.Printf("Updated to %s successfully!\n", latestVersion)
		return nil
	},
}

func getGitHubToken() string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func doGitHubRequest(url string, token string) (*http.Response, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "opencode-usage/"+version.Version)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return client.Do(req)
}

func getLatestRelease() (*githubRelease, error) {
	token := getGitHubToken()

	// Try releases/latest
	resp, err := doGitHubRequest("https://api.github.com/repos/emmmdty/opencode-usage/releases/latest", token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var release githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, err
		}
		return &release, nil
	}

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("GitHub API rate limited. Solutions:\n  1. Install gh CLI: https://cli.github.com\n  2. Set GITHUB_TOKEN environment variable\n  3. Wait an hour for rate limit to reset")
	}

	// Fallback: list tags
	resp2, err := doGitHubRequest("https://api.github.com/repos/emmmdty/opencode-usage/tags?per_page=10", token)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var tags []struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&tags); err == nil && len(tags) > 0 {
			return &githubRelease{TagName: tags[0].Name}, nil
		}
	}

	return nil, fmt.Errorf("could not check for updates (HTTP %d)", resp.StatusCode)
}

func getBinaryURL(release *githubRelease) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var suffix string
	if goos == "windows" {
		suffix = ".exe"
	}

	// Match naming pattern from .goreleaser.yml
	// e.g. opencode-usage_linux_amd64, opencode-usage_darwin_arm64, opencode-usage_windows_amd64.exe
	target := fmt.Sprintf("opencode-usage_%s_%s%s", goos, goarch, suffix)

	for _, asset := range release.Assets {
		if asset.Name == target {
			return asset.BrowserDownloadURL, nil
		}
	}

	// Fallback: try tar.gz for non-windows
	if goos != "windows" {
		tarName := fmt.Sprintf("opencode-usage_%s_%s.tar.gz", goos, goarch)
		for _, asset := range release.Assets {
			if asset.Name == tarName {
				return asset.BrowserDownloadURL, nil
			}
		}
	}

	return "", fmt.Errorf("no binary found for %s/%s in release %s", goos, goarch, release.TagName)
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "opencode-usage-update-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
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
