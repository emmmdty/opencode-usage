package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Only the tool-installed alias line (plus its marker comment) may be
// removed; a user's own "alias tu=..." must survive an uninstall.
func TestRemoveAliasKeepsUserAlias(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	original := "export EDITOR=vim\n" +
		"alias tu='my-custom-thing'\n" +
		"\n" +
		"# token-usage alias\n" +
		"alias tu='token-usage'\n"
	if err := os.WriteFile(rc, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write rc file: %v", err)
	}

	if err := removeAlias(rc, "tu"); err != nil {
		t.Fatalf("removeAlias failed: %v", err)
	}

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("failed to read rc file: %v", err)
	}
	got := string(data)

	if !strings.Contains(got, "alias tu='my-custom-thing'") {
		t.Errorf("user's own alias line was removed; result:\n%s", got)
	}
	if !strings.Contains(got, "export EDITOR=vim") {
		t.Errorf("unrelated line lost; result:\n%s", got)
	}
	if strings.Contains(got, "alias tu='token-usage'") || strings.Contains(got, "# token-usage alias") {
		t.Errorf("tool-installed alias or marker comment remains; result:\n%s", got)
	}
}

// Uninstall modifies a user file: it must leave a backup of the original
// content next to the rc file.
func TestRemoveAliasWritesBackup(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".bashrc")
	original := "# token-usage alias\nalias tu='token-usage'\n"
	if err := os.WriteFile(rc, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write rc file: %v", err)
	}

	if err := removeAlias(rc, "tu"); err != nil {
		t.Fatalf("removeAlias failed: %v", err)
	}

	backup, err := os.ReadFile(rc + ".tu-bak")
	if err != nil {
		t.Fatalf("expected a backup file next to the rc file: %v", err)
	}
	if string(backup) != original {
		t.Errorf("backup content mismatch: %q", string(backup))
	}
}

func TestAliasCommandsUnsupportedOnWindows(t *testing.T) {
	origGOOS := goos
	goos = "windows"
	t.Cleanup(func() { goos = origGOOS })
	t.Setenv("HOME", t.TempDir())

	if err := aliasInstallCmd.RunE(aliasInstallCmd, []string{}); err == nil {
		t.Error("install: expected a not-supported error on windows")
	}
	if err := aliasUninstallCmd.RunE(aliasUninstallCmd, []string{}); err == nil {
		t.Error("uninstall: expected a not-supported error on windows")
	}
}

func TestRcFileForShell(t *testing.T) {
	home := string(filepath.Separator) + filepath.Join("home", "u")
	tests := []struct {
		shell string
		want  string
	}{
		{"/usr/bin/zsh", filepath.Join(home, ".zshrc")},
		{"/bin/zsh", filepath.Join(home, ".zshrc")},
		{"/bin/bash", filepath.Join(home, ".bashrc")},
		{"/usr/bin/fish", filepath.Join(home, ".bashrc")},
		{"", filepath.Join(home, ".bashrc")},
	}
	for _, tt := range tests {
		if got := rcFileFor(tt.shell, home); got != tt.want {
			t.Errorf("rcFileFor(%q) = %q, want %q", tt.shell, got, tt.want)
		}
	}
}
