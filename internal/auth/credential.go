package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/99designs/keyring"
)

var (
	ring             keyring.Keyring
	keyringAvailable bool
)

func init() {
	homeDir, _ := os.UserHomeDir()

	cfg := keyring.Config{
		ServiceName: "token-usage",
		FilePasswordFunc: func(prompt string) (string, error) {
			if pwd := os.Getenv("TOKEN_USAGE_KEYRING_PASSWORD"); pwd != "" {
				return pwd, nil
			}
			return "", fmt.Errorf("keyring password not available in non-interactive mode")
		},
	}

	if homeDir != "" {
		cfg.FileDir = filepath.Join(homeDir, ".config", "token-usage", "keyring")
	}

	var err error
	ring, err = keyring.Open(cfg)
	if err != nil {
		ring = nil
		keyringAvailable = false
		return
	}

	testKey := "__token_usage_test__"
	if err := ring.Set(keyring.Item{Key: testKey, Data: []byte("test")}); err != nil {
		ring = nil
		keyringAvailable = false
		return
	}
	_ = ring.Remove(testKey)
	keyringAvailable = true
}

func IsKeyringAvailable() bool {
	return keyringAvailable
}

// ErrKeyNotFound reports that no key is stored for the account. It is
// returned by GetAPIKey regardless of the backing store so callers can
// distinguish "missing" from a genuine store failure.
var ErrKeyNotFound = errors.New("API key not found")

func StoreAPIKey(service, account, apiKey string) error {
	if ring != nil {
		return ring.Set(keyring.Item{
			Key:  account,
			Data: []byte(apiKey),
		})
	}
	return storeEncrypted(account, apiKey)
}

func GetAPIKey(service, account string) (string, error) {
	if ring != nil {
		item, err := ring.Get(account)
		if err != nil {
			if errors.Is(err, keyring.ErrKeyNotFound) {
				return "", fmt.Errorf("%w for account: %s", ErrKeyNotFound, account)
			}
			return "", fmt.Errorf("keyring error: %w", err)
		}
		return string(item.Data), nil
	}
	key, err := getEncrypted(account)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("%w for account: %s", ErrKeyNotFound, account)
		}
		return "", err
	}
	return key, nil
}

func DeleteAPIKey(service, account string) error {
	if ring != nil {
		err := ring.Remove(account)
		if err != nil && errors.Is(err, keyring.ErrKeyNotFound) {
			return nil
		}
		return err
	}
	return deleteEncrypted(account)
}

func ExtractKeyID(apiKey string) string {
	if len(apiKey) > 6 {
		return apiKey[len(apiKey)-6:]
	}
	return apiKey
}
