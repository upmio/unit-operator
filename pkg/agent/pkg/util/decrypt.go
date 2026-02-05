package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/upmio/unit-operator/pkg/agent/vars"
)

const (
	key = "7e20c20ea7564231a76dd83ac1cf7013"
	// Initialization vector
	iv = "f8/NeLsJ*s*vygV@"
)

var (
	// Global AES key that should be set during application startup
	aesKey string
)

// ValidateAndSetAESKey validates the AES key from environment variable and sets it for use
// This function should be called during application startup (e.g., in main.go)
// Returns error if key is missing or invalid
func ValidateAndSetAESKey() error {
	key := os.Getenv(vars.AESEnvKey)
	if key == "" {
		return fmt.Errorf("AES encryption key not found in environment variable %s", vars.AESEnvKey)
	}

	// Validate key length (should be 32 characters for AES-256)
	if len(key) != 32 {
		return fmt.Errorf("invalid AES key length: expected 32 characters, got %d. Key: %s", len(key), key)
	}

	aesKey = key
	return nil
}

func getAESKey() (string, error) {
	if aesKey == "" {
		return "", fmt.Errorf("cannot get AES key")
	}
	return aesKey, nil
}

// AES_CTR_Encrypt encrypts plaintext and returns base64 encoded string (for backward compatibility)
func AES_CTR_Encrypt(plainText []byte) ([]byte, error) {
	keyStr, err := getAESKey()
	if err != nil {
		return nil, err
	}

	// Convert key to OpenSSL compatible format
	opensslKey := hex.EncodeToString([]byte(keyStr))
	key, err := hex.DecodeString(opensslKey)
	if err != nil {
		return nil, err
	}

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	// Create CTR mode stream cipher
	stream := cipher.NewCTR(block, iv)

	// Encrypt data
	ciphertext := make([]byte, len(plainText))
	stream.XORKeyStream(ciphertext, plainText)

	// Combine IV and ciphertext
	encryptedData := append(iv, ciphertext...)

	return encryptedData, nil
}

// AES_CTR_Decrypt decrypts base64 encoded string and returns plaintext (for backward compatibility)
func AES_CTR_Decrypt(encryptedData []byte) ([]byte, error) {
	keyStr, err := getAESKey()
	if err != nil {
		return nil, err
	}

	// Convert key to OpenSSL compatible format
	opensslKey := hex.EncodeToString([]byte(keyStr))
	key, err := hex.DecodeString(opensslKey)
	if err != nil {
		return nil, err
	}

	// Check minimum length (at least 16 bytes for IV)
	if len(encryptedData) < aes.BlockSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract IV (first 16 bytes)
	iv := encryptedData[:aes.BlockSize]

	// Extract ciphertext (remaining part)
	ciphertext := encryptedData[aes.BlockSize:]

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create CTR mode stream cipher
	stream := cipher.NewCTR(block, iv)

	// Decrypt data
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

func DecryptPlainTextPassword(username string) (string, error) {
	secretPath, err := IsEnvVarSet(vars.SecretMountEnvKey)
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(secretPath, username)

	if exists := IsFileExist(filePath); !exists {
		return filePath, fmt.Errorf("path %s is not exist", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	plaintext, err := AES_CTR_Decrypt(content)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
