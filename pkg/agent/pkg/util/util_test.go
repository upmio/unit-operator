package util

import (
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"os"
	"path"
	"testing"
)

func TestNodesString(t *testing.T) {
	tests := []struct {
		name     string
		nodes    Nodes
		expected string
	}{
		{
			name:     "Empty slice",
			nodes:    Nodes{},
			expected: "[]",
		},
		{
			name:     "Single element",
			nodes:    Nodes{"node1"},
			expected: "[node1]",
		},
		{
			name:     "Multiple elements",
			nodes:    Nodes{"node1", "node2", "node3"},
			expected: "[node1 node2 node3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.nodes.String(); got != tt.expected {
				t.Errorf("Nodes.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNodes_Set(t *testing.T) {
	tests := []struct {
		name     string
		initial  Nodes
		newNode  string
		expected Nodes
	}{
		{
			name:     "Add to empty list",
			initial:  Nodes{},
			newNode:  "node1",
			expected: Nodes{"node1"},
		},
		{
			name:     "Add to non-empty list",
			initial:  Nodes{"node1"},
			newNode:  "node2",
			expected: Nodes{"node1", "node2"},
		},
		{
			name:     "Add empty node",
			initial:  Nodes{"node1"},
			newNode:  "",
			expected: Nodes{"node1", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := tt.initial
			if err := n.Set(tt.newNode); err != nil {
				t.Errorf("Set() error = %v", err)
			}
			if got := n; !equalNodes(got, tt.expected) {
				t.Errorf("Set() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare Nodes slices
func equalNodes(a, b Nodes) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAppendPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		keys     []string
		expected []string
	}{
		{
			name:     "Empty keys",
			prefix:   "/prefix",
			keys:     []string{},
			expected: []string{},
		},
		{
			name:     "Single key",
			prefix:   "/prefix",
			keys:     []string{"key1"},
			expected: []string{path.Join("/prefix", "key1")},
		},
		{
			name:     "Multiple keys",
			prefix:   "/prefix",
			keys:     []string{"key1", "key2"},
			expected: []string{path.Join("/prefix", "key1"), path.Join("/prefix", "key2")},
		},
		{
			name:     "Prefix with trailing slash",
			prefix:   "/prefix/",
			keys:     []string{"key1", "key2"},
			expected: []string{path.Join("/prefix/", "key1"), path.Join("/prefix/", "key2")},
		},
		{
			name:     "Empty prefix",
			prefix:   "",
			keys:     []string{"key1", "key2"},
			expected: []string{"key1", "key2"},
		},
		{
			name:     "Prefix and key both empty",
			prefix:   "",
			keys:     []string{""},
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppendPrefix(tt.prefix, tt.keys)
			if !equalStringSlices(got, tt.expected) {
				t.Errorf("AppendPrefix() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare slices of strings
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestIsFileExist(t *testing.T) {
	// Define temporary file path
	tempFilePath := "testfile.txt"
	// Remove the file if it exists from previous runs
	_ = os.Remove(tempFilePath)

	tests := []struct {
		name     string
		setup    func() error
		filePath string
		expected bool
	}{
		{
			name:     "File exists",
			setup:    func() error { return os.WriteFile(tempFilePath, []byte("test content"), 0644) },
			filePath: tempFilePath,
			expected: true,
		},
		{
			name:     "File does not exist",
			setup:    func() error { return nil },
			filePath: "nonexistentfile.txt",
			expected: false,
		},
		{
			name:     "File is deleted",
			setup:    func() error { return os.Remove(tempFilePath) },
			filePath: tempFilePath,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			got := IsFileExist(tt.filePath)
			if got != tt.expected {
				t.Errorf("IsFileExist() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsConfigChanged(t *testing.T) {
	file1 := "testfile1.txt"
	file2 := "testfile2.txt"

	// Cleanup files after tests
	defer func() {
		_ = os.Remove(file1)
	}()
	defer func() {
		_ = os.Remove(file2)
	}()

	tests := []struct {
		name     string
		setup    func() error
		src      string
		dest     string
		expected bool
	}{
		{
			name: "Files are identical",
			setup: func() error {
				return os.WriteFile(file1, []byte("content"), 0644)
			},
			src:      file1,
			dest:     file1,
			expected: false,
		},
		{
			name: "Destination file does not exist",
			setup: func() error {
				return os.WriteFile(file1, []byte("content"), 0644)
			},
			src:      file1,
			dest:     file2,
			expected: true,
		},
		{
			name: "Files have different contents",
			setup: func() error {
				if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
					return err
				}
				return nil
			},
			src:      file1,
			dest:     file2,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.setup(); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			got, err := IsConfigChanged(tt.src, tt.dest)
			if err != nil {
				t.Fatalf("IsConfigChanged() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("IsConfigChanged() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Test AES_CTR_Encrypt and AES_CTR_Decrypt methods
func TestAES_CTR_EncryptDecrypt(t *testing.T) {
	aesKey = "bec62eddcb834ece8488c88263a5f248"
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Encrypt and decrypt a simple string",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "Encrypt and decrypt empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Encrypt and decrypt long string",
			input:    "This is a very long test string used to verify that AES-CTR encryption and decryption work correctly!",
			expected: "This is a very long test string used to verify that AES-CTR encryption and decryption work correctly!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := AES_CTR_Encrypt([]byte(tt.input))
			if err != nil {
				t.Errorf("%s: encryption failed: %v", tt.name, err)
				return
			}

			// Decrypt
			decrypted, err := AES_CTR_Decrypt(encrypted)
			if err != nil {
				t.Errorf("%s: decryption failed: %v", tt.name, err)
				return
			}

			// Verify result
			if string(decrypted) != tt.expected {
				t.Errorf("%s: decryption result mismatch\nExpected: %s\nActual: %s", tt.name, tt.expected, string(decrypted))
			}
		})
	}
}

// TestValidateAndSetAESKey 测试 AES 密钥验证和设置
func TestValidateAndSetAESKey(t *testing.T) {
	// 保存原始环境变量和密钥
	originalKey := os.Getenv(vars.AESEnvKey)
	originalAESKey := aesKey
	defer func() {
		if originalKey != "" {
			_ = os.Setenv(vars.AESEnvKey, originalKey)
		} else {
			_ = os.Unsetenv(vars.AESEnvKey)
		}
		aesKey = originalAESKey
	}()

	tests := []struct {
		name        string
		keyValue    string
		setEnv      bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid 32-character key",
			keyValue:    "12345678901234567890123456789012",
			setEnv:      true,
			expectError: false,
		},
		{
			name:        "missing environment variable",
			keyValue:    "",
			setEnv:      false,
			expectError: true,
			errorMsg:    "AES encryption key not found",
		},
		{
			name:        "key too short",
			keyValue:    "short",
			setEnv:      true,
			expectError: true,
			errorMsg:    "invalid AES key length",
		},
		{
			name:        "key too long",
			keyValue:    "123456789012345678901234567890123456789012345",
			setEnv:      true,
			expectError: true,
			errorMsg:    "invalid AES key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置全局 AES 密钥
			aesKey = ""

			// 设置或清除环境变量
			if tt.setEnv {
				_ = os.Setenv(vars.AESEnvKey, tt.keyValue)
			} else {
				_ = os.Unsetenv(vars.AESEnvKey)
			}

			err := ValidateAndSetAESKey()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// 验证全局密钥已设置
				key, err := getAESKey()
				if err != nil {
					t.Errorf("Failed to get AES key: %v", err)
				} else if key != tt.keyValue {
					t.Errorf("Expected key '%s', got '%s'", tt.keyValue, key)
				}
			}
		})
	}
}

// TestAESEncryptionErrors 测试 AES 加密错误情况
func TestAESEncryptionErrors(t *testing.T) {
	// 保存原始状态
	originalAESKey := aesKey
	defer func() {
		aesKey = originalAESKey
	}()

	t.Run("encrypt without key", func(t *testing.T) {
		// 清除 AES 密钥
		aesKey = ""

		_, err := AES_CTR_Encrypt([]byte("test"))
		if err == nil {
			t.Error("Expected error when encrypting without key")
		} else if !containsString(err.Error(), "cannot get AES key") {
			t.Errorf("Expected error to contain 'cannot get AES key', got '%s'", err.Error())
		}
	})

	t.Run("decrypt without key", func(t *testing.T) {
		// 清除 AES 密钥
		aesKey = ""

		_, err := AES_CTR_Decrypt([]byte("test"))
		if err == nil {
			t.Error("Expected error when decrypting without key")
		} else if !containsString(err.Error(), "cannot get AES key") {
			t.Errorf("Expected error to contain 'cannot get AES key', got '%s'", err.Error())
		}
	})

	t.Run("decrypt invalid data", func(t *testing.T) {
		// 设置有效密钥
		aesKey = "12345678901234567890123456789012"

		// 尝试解密太短的数据
		_, err := AES_CTR_Decrypt([]byte("short"))
		if err == nil {
			t.Error("Expected error when decrypting invalid data")
		} else if !containsString(err.Error(), "encrypted data too short") {
			t.Errorf("Expected error to contain 'encrypted data too short', got '%s'", err.Error())
		}
	})
}

// TestFileStat 测试文件状态获取功能
func TestFileStat(t *testing.T) {
	// 创建临时文件
	tempFile := "test_filestat.txt"
	defer func() {
		_ = os.Remove(tempFile)
	}()

	testContent := "test content for file stat"
	if err := os.WriteFile(tempFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("valid file", func(t *testing.T) {
		fi, err := FileStat(tempFile)
		if err != nil {
			t.Errorf("FileStat() error = %v", err)
			return
		}

		if fi.Uid == 0 && fi.Gid == 0 {
			t.Log("Warning: UID and GID are 0, might be running as root")
		}

		if fi.Mode == 0 {
			t.Error("Expected non-zero file mode")
		}

		if fi.Md5 == "" {
			t.Error("Expected non-empty MD5 hash")
		}

		if len(fi.Md5) != 32 {
			t.Errorf("Expected MD5 hash length 32, got %d", len(fi.Md5))
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		_, err := FileStat("non_existing_file.txt")
		if err == nil {
			t.Error("Expected error for non-existing file")
		} else if !containsString(err.Error(), "file not found") {
			t.Errorf("Expected error to contain 'file not found', got '%s'", err.Error())
		}
	})
}

// BenchmarkAppendPrefix 性能测试
func BenchmarkAppendPrefix(b *testing.B) {
	prefix := "/config/app"
	keys := []string{"key1", "key2", "key3", "key4", "key5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AppendPrefix(prefix, keys)
	}
}

// BenchmarkIsFileExist 性能测试
func BenchmarkIsFileExist(b *testing.B) {
	// 创建临时文件
	tempFile := "bench_file.txt"
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		b.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tempFile)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsFileExist(tempFile)
	}
}

// BenchmarkAESEncryption 性能测试
func BenchmarkAESEncryption(b *testing.B) {
	// 设置 AES 密钥
	aesKey = "12345678901234567890123456789012"
	plaintext := []byte("This is a test message for benchmarking AES encryption performance")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := AES_CTR_Encrypt(plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAESDecryption 性能测试
func BenchmarkAESDecryption(b *testing.B) {
	// 设置 AES 密钥并准备加密数据
	aesKey = "12345678901234567890123456789012"
	plaintext := []byte("This is a test message for benchmarking AES decryption performance")
	encrypted, err := AES_CTR_Encrypt(plaintext)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := AES_CTR_Decrypt(encrypted)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// containsString 检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
