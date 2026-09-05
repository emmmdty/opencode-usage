package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIIntegration(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "token-usage-test")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", binary, "../cmd/token-usage/")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	helpCmd := exec.Command(binary, "--help")
	helpCmd.Env = append(os.Environ(), "LANG=en_US.UTF-8")
	output, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run help command: %v", err)
	}

	if !strings.Contains(string(output), "Token Usage") {
		t.Error("help output should contain 'Token Usage'")
	}
}
