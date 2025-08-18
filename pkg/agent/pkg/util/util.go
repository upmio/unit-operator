package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
)

// Nodes is a custom flag Var representing a list of etcd nodes.
type Nodes []string

// String returns the string representation of a node var.
func (n *Nodes) String() string {
	return fmt.Sprintf("%s", *n)
}

// Set appends the node to the etcd node list.
func (n *Nodes) Set(node string) error {
	*n = append(*n, node)
	return nil
}

func AppendPrefix(prefix string, keys []string) []string {
	s := make([]string, len(keys))
	for i, k := range keys {
		s[i] = path.Join(prefix, k)
	}
	return s
}

// isFileExist reports whether path exits.
func IsFileExist(fpath string) bool {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return false
	}
	return true
}

// fileInfo describes a configuration file and is returned by fileStat.
type FileInfo struct {
	Uid  uint32
	Gid  uint32
	Mode os.FileMode
	Md5  string
}

// IsConfigChanged reports whether src and dest config files are equal.
// Two config files are equal when they have the same file contents and
// Unix permissions. The owner, group, and mode must match.
// It return false in other cases.
func IsConfigChanged(src, dest string) (bool, error) {
	if !IsFileExist(dest) {
		return true, nil
	}
	d, err := FileStat(dest)
	if err != nil {
		return true, err
	}
	s, err := FileStat(src)
	if err != nil {
		return true, err
	}
	if d.Uid != s.Uid {
		return true, nil
	}
	if d.Gid != s.Gid {
		return true, nil
	}
	if d.Mode != s.Mode {
		return true, nil
	}
	if d.Md5 != s.Md5 {
		return true, nil
	}
	if d.Uid != s.Uid || d.Gid != s.Gid || d.Mode != s.Mode || d.Md5 != s.Md5 {
		return true, nil
	}
	return false, nil
}

func FileStat(name string) (fi FileInfo, err error) {
	if IsFileExist(name) {
		f, err := os.Open(name)
		if err != nil {
			return fi, err
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Printf("failed to close file: %v\n", err)
			}
		}()
		stats, _ := f.Stat()
		fi.Uid = stats.Sys().(*syscall.Stat_t).Uid
		fi.Gid = stats.Sys().(*syscall.Stat_t).Gid
		fi.Mode = stats.Mode()
		h := md5.New()
		if _, err := io.Copy(h, f); err != nil {
			return fi, fmt.Errorf("failed to copy file data: %v", err)
		}
		fi.Md5 = fmt.Sprintf("%x", h.Sum(nil))
		return fi, nil
	}
	return fi, errors.New("file not found")
}

const (
	// Environment variable name for AES key
	AESKeyEnvVar = "AES_SECRET_KEY"
)

var (
	// Global AES key that should be set during application startup
	aesKey string
)

// ValidateAndSetAESKey validates the AES key from environment variable and sets it for use
// This function should be called during application startup (e.g., in main.go)
// Returns error if key is missing or invalid
func ValidateAndSetAESKey() error {
	key := os.Getenv(AESKeyEnvVar)
	if key == "" {
		return fmt.Errorf("AES encryption key not found in environment variable %s", AESKeyEnvVar)
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
