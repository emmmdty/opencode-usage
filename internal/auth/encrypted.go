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

	"github.com/emmmdty/token-usage/internal/i18n"

	"errors"
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

const defaultPassword = "token-usage-default"

func SetUseMasterPassword(enabled *bool) {
	useMasterPassword = enabled
}

func GetMasterPasswordMode() *bool {
	return useMasterPassword
}

const (
	encryptedDir  = ".config/token-usage"
	encryptedFile = "secrets.enc"
	saltSize      = 16
	keySize       = 32
	iterations    = 100000
)

// testOverridePath, when non-empty, redirects the secrets file location so
// tests never touch a real user config.
var testOverridePath string

func getEncryptedPath() (string, error) {
	if testOverridePath != "" {
		return testOverridePath, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, encryptedDir, encryptedFile), nil
}

func doInitMasterPassword() {
	// Try to get from environment variable
	if pwd := os.Getenv("TOKEN_USAGE_MASTER_PASSWORD"); pwd != "" {
		passwordMu.Lock()
		cachedMasterPassword = pwd
		passwordMu.Unlock()
		return
	}

	// Interactive input (masked)
	fmt.Print(i18n.T("prompt.master_password"))
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		passwordErr = fmt.Errorf("%s", i18n.T("error.auth.master_password_read", err))
		return
	}

	if len(pwd) == 0 {
		passwordErr = errors.New(i18n.T("error.auth.master_password_empty"))
		return
	}

	passwordMu.Lock()
	cachedMasterPassword = string(pwd)
	passwordMu.Unlock()
}

func getMasterPassword() (string, error) {
	passwordMu.RLock()
	cached := cachedMasterPassword
	err := passwordErr
	passwordMu.RUnlock()
	if cached != "" {
		return cached, nil
	}
	if err != nil {
		return "", err
	}

	passwordMu.Lock()
	if cachedMasterPassword == "" && passwordErr == nil {
		passwordMu.Unlock()

		if useMasterPassword != nil && !*useMasterPassword {
			passwordMu.Lock()
			cachedMasterPassword = defaultPassword
			cached := cachedMasterPassword
			passwordMu.Unlock()
			return cached, nil
		}

		passwordOnce.Do(doInitMasterPassword)
	}

	passwordMu.RLock()
	cached = cachedMasterPassword
	err = passwordErr
	passwordMu.RUnlock()
	return cached, err
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

// parseSecrets parses the secrets blob. Two formats exist:
//   - legacy: "account:key\n" per line
//   - current: "account\x00key\n" per line
//
// Keys cannot contain '\n', so line-based parsing is safe in both formats.
func parseSecrets(decrypted string) map[string]string {
	existing := make(map[string]string)
	for _, line := range strings.Split(decrypted, "\n") {
		if line == "" {
			continue
		}
		var account, key string
		if idx := strings.IndexByte(line, '\x00'); idx >= 0 {
			account, key = line[:idx], line[idx+1:]
		} else if idx := strings.IndexByte(line, ':'); idx >= 0 {
			account, key = line[:idx], line[idx+1:]
		} else {
			continue
		}
		if account == "" || key == "" {
			continue
		}
		existing[account] = key
	}
	return existing
}

// validateAccountName rejects names that would corrupt the secrets file
// format (newline in either format, NUL in the current format, colon in the
// legacy format).
func validateAccountName(account string) error {
	if account == "" {
		return errors.New(i18n.T("error.auth.account_empty"))
	}
	if strings.ContainsAny(account, "\n\x00:") {
		return errors.New(i18n.T("error.auth.account_invalid_chars"))
	}
	return nil
}

// loadEncrypted reads and decrypts the secrets file. Returns (nil, nil) when
// the file does not exist. Any other read/parse/decrypt error is returned:
// callers must NOT silently overwrite the file in that case.
func loadEncrypted() (map[string]string, error) {
	path, err := getEncryptedPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("error.auth.secrets_corrupted", err))
	}

	pwd, err := getMasterPassword()
	if err != nil {
		return nil, err
	}

	decrypted, err := decrypt(decoded, pwd)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("error.auth.decrypt_failed", err))
	}

	return parseSecrets(string(decrypted)), nil
}

// saveEncrypted encrypts and atomically writes the given accounts.
func saveEncrypted(secrets map[string]string) error {
	path, err := getEncryptedPath()
	if err != nil {
		return err
	}

	pwd, err := getMasterPassword()
	if err != nil {
		return err
	}

	var data strings.Builder
	for k, v := range secrets {
		data.WriteString(k + "\x00" + v + "\n")
	}

	encrypted, err := encrypt([]byte(data.String()), pwd)
	if err != nil {
		return err
	}

	return writeFileAtomic(path, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by rename, so a crash mid-write never truncates the target.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func storeEncrypted(account, apiKey string) error {
	if err := validateAccountName(account); err != nil {
		return err
	}

	return withSecretsLock(func() error {
		existing, err := loadEncrypted()
		if err != nil {
			return err
		}
		if existing == nil {
			existing = make(map[string]string)
		}

		existing[account] = apiKey
		return saveEncrypted(existing)
	})
}

func getEncrypted(account string) (string, error) {
	secrets, err := loadEncrypted()
	if err != nil {
		return "", err
	}
	if secrets == nil {
		return "", fmt.Errorf("API key not found for account: %s", account)
	}

	if key, ok := secrets[account]; ok {
		return key, nil
	}
	return "", fmt.Errorf("API key not found for account: %s", account)
}

func deleteEncrypted(account string) error {
	if err := validateAccountName(account); err != nil {
		return err
	}

	return withSecretsLock(func() error {
		secrets, err := loadEncrypted()
		if err != nil {
			return err
		}
		if secrets == nil {
			return nil
		}

		delete(secrets, account)
		return saveEncrypted(secrets)
	})
}
