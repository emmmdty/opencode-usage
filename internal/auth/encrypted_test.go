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

func TestStoreAndGetEncrypted(t *testing.T) {
	// Skip if master password is not available
	if os.Getenv("OPENCODE_USAGE_MASTER_PASSWORD") == "" {
		t.Skip("OPENCODE_USAGE_MASTER_PASSWORD not set")
	}

	account := "test-account-encrypted"
	apiKey := "sk-test-encrypted-123456"

	// Clean up any existing test data
	path, _ := getEncryptedPath()
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}
	defer os.Remove(path)

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
