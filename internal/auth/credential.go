package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

var (
	ring             keyring.Keyring
	keyringAvailable bool
)

func init() {
	homeDir, _ := os.UserHomeDir()

	cfg := keyring.Config{
		ServiceName: "opencode-usage",
		FilePasswordFunc: func(prompt string) (string, error) {
			if pwd := os.Getenv("OPENCODE_USAGE_KEYRING_PASSWORD"); pwd != "" {
				return pwd, nil
			}
			return "", fmt.Errorf("keyring password not available in non-interactive mode")
		},
	}

	if homeDir != "" {
		cfg.FileDir = filepath.Join(homeDir, ".config", "opencode-usage", "keyring")
	}

	var err error
	ring, err = keyring.Open(cfg)
	if err != nil {
		ring = nil
		keyringAvailable = false
		return
	}

	testKey := "__opencode_usage_test__"
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
			return "", err
		}
		return string(item.Data), nil
	}
	return getEncrypted(account)
}

func DeleteAPIKey(service, account string) error {
	if ring != nil {
		return ring.Remove(account)
	}
	return deleteEncrypted(account)
}

func ExtractKeyID(apiKey string) string {
	if len(apiKey) > 6 {
		return apiKey[len(apiKey)-6:]
	}
	return apiKey
}
