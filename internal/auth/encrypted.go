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
)

const defaultPassword = "opencode-usage-default"

func SetUseMasterPassword(enabled *bool) {
	useMasterPassword = enabled
}

func GetMasterPasswordMode() *bool {
	return useMasterPassword
}

const (
	encryptedDir = ".config/opencode-usage"
	encryptedFile = "secrets.enc"
	saltSize = 16
	keySize = 32
	iterations = 100000
)

func getEncryptedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, encryptedDir, encryptedFile), nil
}

func doInitMasterPassword() {
	// 尝试从环境变量获取
	if pwd := os.Getenv("OPENCODE_USAGE_MASTER_PASSWORD"); pwd != "" {
		cachedMasterPassword = pwd
		return
	}
	
	// 交互式输入 (masked)
	fmt.Print("请输入主密码: ")
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		passwordErr = fmt.Errorf("读取主密码失败: %w", err)
		return
	}
	
	if len(pwd) == 0 {
		passwordErr = fmt.Errorf("主密码不能为空")
		return
	}
	
	cachedMasterPassword = string(pwd)
}

func getMasterPassword() (string, error) {
	if cachedMasterPassword != "" {
		return cachedMasterPassword, nil
	}

	if useMasterPassword != nil && !*useMasterPassword {
		cachedMasterPassword = defaultPassword
		return cachedMasterPassword, nil
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
		return nil, fmt.Errorf("数据格式无效")
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
		return nil, fmt.Errorf("数据格式无效")
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
	
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	pwd, err := getMasterPassword()
	if err != nil {
		return err
	}
	
	// 读取现有数据
	existing := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil {
			decrypted, err := decrypt(decoded, pwd)
			if err == nil {
				// 解析现有数据
				lines := strings.Split(string(decrypted), "\n")
				for _, line := range lines {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						existing[parts[0]] = parts[1]
					}
				}
			}
		}
	}
	
	// 添加新账号
	existing[account] = apiKey
	
	// 构建数据
	var data strings.Builder
	for k, v := range existing {
		data.WriteString(k + ":" + v + "\n")
	}
	
	// 加密
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
		parts := strings.SplitN(line, ":", 2)
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
			return nil // 文件不存在，无需删除
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
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			existing[parts[0]] = parts[1]
		}
	}
	
	delete(existing, account)
	
	var newData strings.Builder
	for k, v := range existing {
		newData.WriteString(k + ":" + v + "\n")
	}
	
	encrypted, err := encrypt([]byte(newData.String()), pwd)
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(encrypted)), 0600)
}
