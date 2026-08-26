package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

var (
	cachedMasterPassword string
	passwordOnce         sync.Once
	passwordErr          error
	useMasterPassword    *bool
	passwordMu           sync.RWMutex
)

const defaultPassword = "opencode-usage-default"

func SetUseMasterPassword(enabled *bool) {
	useMasterPassword = enabled
}

func GetMasterPasswordMode() *bool {
	return useMasterPassword
}

const (
	encryptedDir  = ".config/opencode-usage"
	encryptedFile = "secrets.enc"
	saltSize      = 16
	keySize       = 32
	iterations    = 100000
)

func getEncryptedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, encryptedDir, encryptedFile), nil
}

func doInitMasterPassword() {
	// Try to get from environment variable
	if pwd := os.Getenv("OPENCODE_USAGE_MASTER_PASSWORD"); pwd != "" {
		passwordMu.Lock()
		cachedMasterPassword = pwd
		passwordMu.Unlock()
		return
	}

	// Interactive input (masked)
	fmt.Print("Enter master password: ")
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		passwordErr = fmt.Errorf("failed to read master password: %w", err)
		return
	}

	if len(pwd) == 0 {
		passwordErr = fmt.Errorf("master password cannot be empty")
		return
	}

	passwordMu.Lock()
	cachedMasterPassword = string(pwd)
	passwordMu.Unlock()
}

func getMasterPassword() (string, error) {
	passwordMu.RLock()
	if cachedMasterPassword != "" {
		pwd := cachedMasterPassword
		passwordMu.RUnlock()
		return pwd, nil
	}
	passwordMu.RUnlock()

	if useMasterPassword != nil && !*useMasterPassword {
		passwordMu.Lock()
		cachedMasterPassword = defaultPassword
		pwd := cachedMasterPassword
		passwordMu.Unlock()
		return pwd, nil
	}

	passwordOnce.Do(doInitMasterPassword)
	return cachedMasterPassword, passwordErr
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, iterations, keySize, sha256.New)
}

func encrypt(data []byte, password string) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nil, nonce, data, nil)

	// 格式: salt + nonce + ciphertext
	result := make([]byte, 0, saltSize+aesGCM.NonceSize()+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	// GCM overhead is 16 bytes (tag size)
	const gcmOverhead = 16
	if len(data) < saltSize+gcmOverhead {
		return nil, fmt.Errorf("invalid data format")
	}

	salt := data[:saltSize]
	data = data[saltSize:]

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid data format")
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	return aesGCM.Open(nil, nonce, ciphertext, nil)
}

func storeEncrypted(account, apiKey string) error {
	path, err := getEncryptedPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	pwd, err := getMasterPassword()
	if err != nil {
		return err
	}

	// Read existing data
	existing := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil {
			decrypted, err := decrypt(decoded, pwd)
			if err == nil {
				// Parse existing data
				lines := strings.Split(string(decrypted), "\n")
				for _, line := range lines {
					parts := strings.SplitN(line, "\x00", 2)
					if len(parts) == 2 {
						existing[parts[0]] = parts[1]
					}
				}
			}
		}
	}

	// Add new account
	existing[account] = apiKey

	// Build data
	var data strings.Builder
	for k, v := range existing {
		data.WriteString(k + "\x00" + v + "\n")
	}

	// Encrypt
	encrypted, err := encrypt([]byte(data.String()), pwd)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
}

func getEncrypted(account string) (string, error) {
	path, err := getEncryptedPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("API key not found for account: %s", account)
		}
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return "", err
	}

	pwd, err := getMasterPassword()
	if err != nil {
		return "", err
	}

	decrypted, err := decrypt(decoded, pwd)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(decrypted), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) == 2 && parts[0] == account {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("API key not found for account: %s", account)
}

func deleteEncrypted(account string) error {
	path, err := getEncryptedPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}

	pwd, err := getMasterPassword()
	if err != nil {
		return err
	}

	decrypted, err := decrypt(decoded, pwd)
	if err != nil {
		return err
	}

	existing := make(map[string]string)
	lines := strings.Split(string(decrypted), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) == 2 {
			existing[parts[0]] = parts[1]
		}
	}

	delete(existing, account)

	var newData strings.Builder
	for k, v := range existing {
		newData.WriteString(k + "\x00" + v + "\n")
	}

	encrypted, err := encrypt([]byte(newData.String()), pwd)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
}
