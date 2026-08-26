package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	password := "test-password-123"
	data := []byte("test data to encrypt")

	encrypted, err := encrypt(data, password)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if string(encrypted) == string(data) {
		t.Error("encrypted data should differ from original")
	}

	decrypted, err := decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Errorf("expected %s, got %s", data, decrypted)
	}
}

func TestEncryptDecryptWrongPassword(t *testing.T) {
	password := "correct-password"
	wrongPassword := "wrong-password"
	data := []byte("sensitive data")

	encrypted, err := encrypt(data, password)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	_, err = decrypt(encrypted, wrongPassword)
	if err == nil {
		t.Error("expected error when decrypting with wrong password")
	}
}

func TestEncryptDecryptEmptyData(t *testing.T) {
	password := "test-password"
	data := []byte{}

	encrypted, err := encrypt(data, password)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := decrypt(encrypted, password)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(decrypted))
	}
}

func TestDeriveKey(t *testing.T) {
	password := "test-password"
	salt := []byte("1234567890123456") // 16 bytes

	key1 := deriveKey(password, salt)
	key2 := deriveKey(password, salt)

	if len(key1) != keySize {
		t.Errorf("expected key size %d, got %d", keySize, len(key1))
	}

	if string(key1) != string(key2) {
		t.Error("same password and salt should produce same key")
	}

	// Different password should produce different key
	key3 := deriveKey("different-password", salt)
	if string(key1) == string(key3) {
		t.Error("different passwords should produce different keys")
	}

	// Different salt should produce different key
	key4 := deriveKey(password, []byte("6543210987654321"))
	if string(key1) == string(key4) {
		t.Error("different salts should produce different keys")
	}
}

func TestGetEncryptedPath(t *testing.T) {
	path, err := getEncryptedPath()
	if err != nil {
		t.Fatalf("getEncryptedPath failed: %v", err)
	}

	if filepath.Base(path) != encryptedFile {
		t.Errorf("expected filename %s, got %s", encryptedFile, filepath.Base(path))
	}
}

func TestParseSecretsLegacyColonFormat(t *testing.T) {
	// Old builds wrote "account:key" lines; they must still be readable.
	input := "alice@example.com:sk-aaa111222333\nbob@example.com:sk-bbb444555666\n"
	got := parseSecrets(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got["alice@example.com"] != "sk-aaa111222333" {
		t.Errorf("alice key mismatch: %s", got["alice@example.com"])
	}
	if got["bob@example.com"] != "sk-bbb444555666" {
		t.Errorf("bob key mismatch: %s", got["bob@example.com"])
	}
}

func TestParseSecretsNulFormat(t *testing.T) {
	input := "alice@example.com\x00sk-aaa111222333\n"
	got := parseSecrets(input)
	if len(got) != 1 || got["alice@example.com"] != "sk-aaa111222333" {
		t.Errorf("unexpected parse result: %v", got)
	}
}

func TestParseSecretsMixedAndMalformed(t *testing.T) {
	input := "a@x.com:sk-legacy\nc@x.com\x00sk-new\n\nno-separator-line\n:empty-account\nempty-key:\n"
	got := parseSecrets(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
	if got["a@x.com"] != "sk-legacy" || got["c@x.com"] != "sk-new" {
		t.Errorf("unexpected parse result: %v", got)
	}
}

func TestValidateAccountName(t *testing.T) {
	valid := []string{"user@example.com", "work", "a"}
	for _, name := range valid {
		if err := validateAccountName(name); err != nil {
			t.Errorf("validateAccountName(%q) unexpected error: %v", name, err)
		}
	}
	invalid := []string{"", "with\nnewline", "with\x00nul", "with:colon"}
	for _, name := range invalid {
		if err := validateAccountName(name); err == nil {
			t.Errorf("validateAccountName(%q) expected error", name)
		}
	}
}

func TestStoreAndGetEncrypted(t *testing.T) {
	// Skip if master password is not available
	if os.Getenv("OPENCODE_USAGE_MASTER_PASSWORD") == "" {
		t.Skip("OPENCODE_USAGE_MASTER_PASSWORD not set")
	}

	// Use an isolated temp path so the real user config is never touched.
	setTestSecretsPath(t)
	resetMasterPasswordCache()

	account := "test-account-encrypted"
	apiKey := "sk-test-encrypted-123456"

	// Store
	if err := storeEncrypted(account, apiKey); err != nil {
		t.Fatalf("storeEncrypted failed: %v", err)
	}

	// Get
	retrieved, err := getEncrypted(account)
	if err != nil {
		t.Fatalf("getEncrypted failed: %v", err)
	}

	if retrieved != apiKey {
		t.Errorf("expected %s, got %s", apiKey, retrieved)
	}

	// Delete
	if err := deleteEncrypted(account); err != nil {
		t.Fatalf("deleteEncrypted failed: %v", err)
	}

	// Verify deletion
	_, err = getEncrypted(account)
	if err == nil {
		t.Error("expected error after deletion")
	}
}
