package auth

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestKeyringOperations(t *testing.T) {
	// StoreAPIKey/GetAPIKey/DeleteAPIKey are exercised through the encrypted
	// file fallback with an explicit master password, which is deterministic
	// everywhere (an interactive system keyring is not available in CI and
	// would hang the test).
	t.Setenv("TOKEN_USAGE_MASTER_PASSWORD", "ci-test-master-password")
	setTestSecretsPath(t)
	resetMasterPasswordCache()

	// Force the encrypted file backend so the test never touches a real
	// system keyring (macOS Keychain / Windows Credential Manager).
	origRing := ring
	ring = nil
	defer func() { ring = origRing }()

	serviceName := "token-usage-test"
	accountName := "test-account"
	apiKey := "sk-test1234567890"

	if err := StoreAPIKey(serviceName, accountName, apiKey); err != nil {
		t.Fatalf("failed to store API key: %v", err)
	}

	retrieved, err := GetAPIKey(serviceName, accountName)
	if err != nil {
		t.Fatalf("failed to get API key: %v", err)
	}

	if retrieved != apiKey {
		t.Errorf("expected %s, got %s", apiKey, retrieved)
	}

	if err := DeleteAPIKey(serviceName, accountName); err != nil {
		t.Fatalf("failed to delete API key: %v", err)
	}

	_, err = GetAPIKey(serviceName, accountName)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

// setTestSecretsPath redirects the secrets file to a temp dir and cleans up.
func setTestSecretsPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	testOverridePath = path
	t.Cleanup(func() { testOverridePath = "" })
}

// resetMasterPasswordCache clears the cached master password so tests can
// switch between default and explicit master password modes.
func resetMasterPasswordCache() {
	passwordMu.Lock()
	cachedMasterPassword = ""
	passwordErr = nil
	passwordOnce = sync.Once{}
	passwordMu.Unlock()
}

func TestExtractKeyID(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"sk-ant-1234567890abcdef", "abcdef"},
		{"sk-test-abcdef123456", "123456"},
		{"short", "short"},
	}

	for _, tt := range tests {
		result := ExtractKeyID(tt.key)
		if result != tt.expected {
			t.Errorf("ExtractKeyID(%s) = %s, want %s", tt.key, result, tt.expected)
		}
	}
}
