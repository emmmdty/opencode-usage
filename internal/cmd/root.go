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
	"github.com/emmmdty/token-usage/internal/auth"
	"github.com/emmmdty/token-usage/internal/config"
	"github.com/emmmdty/token-usage/internal/tui"
	"github.com/emmmdty/token-usage/internal/version"
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
	Use:     "token-usage [provider]",
	Aliases: []string{"tu"},
	Short:   "Multi-provider AI coding tool usage monitor",
	Long: `Token Usage — monitor quota usage across AI coding providers.

Supported providers: OpenCode Go, Claude, Codex, Volcano Engine (Coding/Agent
Plan), and user-defined custom providers (Z.ai GLM, Kimi, MiniMax, DeepSeek,
openai-compatible).`,
	Version: version.Version,
	Args:    cobra.MaximumNArgs(1),
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
		providerFilter := ""
		if len(args) > 0 {
			providerFilter = args[0]
		}
		return runProvidersOverview(providerFilter, jsonOutput, outputFile)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func writeOutput(content string) error {
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(content), 0600)
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
	Long: `Check GitHub Releases for a newer version and self-update.

When a newer release exists, the matching platform binary is downloaded
from GitHub Releases and replaces the running executable. Downgrades are
skipped. If the GitHub API rate limit is hit, install the gh CLI, set
GITHUB_TOKEN, or wait for the limit to reset.`,
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

		if !isNewerVersion(latestVersion, currentVersion) {
			fmt.Printf("Installed version %s is newer than latest release %s; skipping downgrade.\n", currentVersion, latestVersion)
			return nil
		}

		fmt.Printf("New version available: %s (current: %s)\n", latestVersion, currentVersion)

		binaryURL, err := getBinaryURL(release)
		if err != nil {
			if len(release.Assets) == 0 {
				return fmt.Errorf("update is unavailable (no release assets found for tag %s); please install manually from https://github.com/emmmdty/token-usage/releases", release.TagName)
			}
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
		// Resolve symlinks so we replace the real binary, not the link.
		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = resolved
		}

		if err := os.Chmod(tmpFile, 0755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}

		if err := replaceBinary(tmpFile, execPath); err != nil {
			return fmt.Errorf("failed to replace binary: %w", err)
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
	req.Header.Set("User-Agent", "token-usage/"+version.Version)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return client.Do(req)
}

func getLatestRelease() (*githubRelease, error) {
	token := getGitHubToken()

	// Try releases/latest
	resp, err := doGitHubRequest("https://api.github.com/repos/emmmdty/token-usage/releases/latest", token)
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
	resp2, err := doGitHubRequest("https://api.github.com/repos/emmmdty/token-usage/tags?per_page=10", token)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusOK {
		var tags []struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&tags); err == nil && len(tags) > 0 {
			// Tags have no release assets; report the version only.
			return &githubRelease{TagName: tags[0].Name}, nil
		}
	}

	return nil, fmt.Errorf("could not check for updates (HTTP %d)", resp.StatusCode)
}

// isNewerVersion reports whether latest is strictly newer than current,
// comparing dot-separated numeric segments (e.g. 0.3.0 > 0.2.1).
func isNewerVersion(latest, current string) bool {
	parse := func(v string) []int {
		var out []int
		for _, part := range strings.Split(v, ".") {
			n := 0
			for _, r := range part {
				if r < '0' || r > '9' {
					break
				}
				n = n*10 + int(r-'0')
			}
			out = append(out, n)
		}
		return out
	}

	l, c := parse(latest), parse(current)
	maxLen := len(l)
	if len(c) > maxLen {
		maxLen = len(c)
	}
	for i := 0; i < maxLen; i++ {
		var lv, cv int
		if i < len(l) {
			lv = l[i]
		}
		if i < len(c) {
			cv = c[i]
		}
		if lv != cv {
			return lv > cv
		}
	}
	return false
}

// replaceBinary installs tmpFile at execPath. On Windows the running binary
// cannot be overwritten, so the old one is moved aside and removed later.
// On Unix a plain rename works even while running.
func replaceBinary(tmpFile, execPath string) error {
	err := os.Rename(tmpFile, execPath)
	if err == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		// Move the running binary aside, then put the new one in place.
		old := execPath + ".old"
		_ = os.Remove(old)
		if err := os.Rename(execPath, old); err != nil {
			return fmt.Errorf("cannot move current binary: %w", err)
		}
		if err := os.Rename(tmpFile, execPath); err != nil {
			// Try to restore.
			_ = os.Rename(old, execPath)
			return fmt.Errorf("cannot install new binary: %w", err)
		}
		_ = os.Remove(old) // may fail while running; cleaned up on next run
		return nil
	}

	return fmt.Errorf("%w (you may need to run with sudo, or the temp dir and install dir are on different filesystems)", err)
}

func getBinaryURL(release *githubRelease) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var suffix string
	if goos == "windows" {
		suffix = ".exe"
	}

	// Match naming pattern from .goreleaser.yml
	// e.g. token-usage_linux_amd64, token-usage_darwin_arm64, token-usage_windows_amd64.exe
	target := fmt.Sprintf("token-usage_%s_%s%s", goos, goarch, suffix)

	for _, asset := range release.Assets {
		if asset.Name == target {
			return asset.BrowserDownloadURL, nil
		}
	}

	// Fallback: try tar.gz for non-windows
	if goos != "windows" {
		tarName := fmt.Sprintf("token-usage_%s_%s.tar.gz", goos, goarch)
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

	// Prefer a temp file in the executable's directory so the final rename
	// cannot fail with a cross-device link error.
	dir := ""
	if execPath, err := os.Executable(); err == nil {
		dir = filepath.Dir(execPath)
	}
	if dir == "" {
		dir = os.TempDir()
	}

	tmpFile, err := os.CreateTemp(dir, ".token-usage-update-*")
	if err != nil {
		// Fall back to the system temp dir if the install dir is not writable.
		tmpFile, err = os.CreateTemp("", "token-usage-update-*")
		if err != nil {
			return "", err
		}
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
	rootCmd.PersistentFlags().StringVarP(&account, "account", "n", "", "account or provider/account to query")
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
	return homeDir + "/.config/token-usage/config.yaml", nil
}
