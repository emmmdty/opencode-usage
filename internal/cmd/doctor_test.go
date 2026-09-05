package cmd

import (
	"errors"
	"path/filepath"
	"testing"
)

// doctorOverrides isolates doctor from the real user environment: the config
// path points at a temp file and the network probe is stubbed. It returns a
// restore func to defer.
func doctorOverrides(t *testing.T, probe func() error) {
	t.Helper()
	origGetConfigPath := getConfigPath
	getConfigPath = func() (string, error) {
		return filepath.Join(t.TempDir(), "config.yaml"), nil
	}
	origProbe := doctorNetworkProbe
	doctorNetworkProbe = probe
	origNoColor := noColor
	noColor = true
	t.Cleanup(func() {
		getConfigPath = origGetConfigPath
		doctorNetworkProbe = origProbe
		noColor = origNoColor
	})
}

// A FAILing check must surface as an error (non-zero exit code), so scripts
// and CI can act on doctor's verdict.
func TestDoctorFailingCheckReturnsError(t *testing.T) {
	doctorOverrides(t, func() error { return errors.New("connection refused") })

	if err := doctorCmd.RunE(doctorCmd, []string{}); err == nil {
		t.Fatal("expected doctor to return an error when a check FAILs")
	}
}

// WARN is informational: a warning-only run must not report failure.
func TestDoctorWarningsDoNotFail(t *testing.T) {
	doctorOverrides(t, func() error { return nil })

	// The keyring check degrades to WARN in sandboxes; that must be
	// tolerated while the network check (and config load) succeed.
	if err := doctorCmd.RunE(doctorCmd, []string{}); err != nil {
		t.Fatalf("expected no error for a warning-only run, got: %v", err)
	}
}
