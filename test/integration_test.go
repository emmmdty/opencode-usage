package test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIIntegration(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "opencode-usage-test")
	buildCmd := exec.Command("go", "build", "-o", binary, "../cmd/opencode-usage/")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	helpCmd := exec.Command(binary, "--help")
	output, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run help command: %v", err)
	}

	if !strings.Contains(string(output), "OpenCode Go") {
		t.Error("help output should contain 'OpenCode Go'")
	}
}
