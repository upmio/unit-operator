package util

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
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
	os.Remove(tempFilePath)

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
	defer os.Remove(file1)
	defer os.Remove(file2)

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
				return ioutil.WriteFile(file1, []byte("content"), 0644)
			},
			src:      file1,
			dest:     file1,
			expected: false,
		},
		{
			name: "Destination file does not exist",
			setup: func() error {
				return ioutil.WriteFile(file1, []byte("content"), 0644)
			},
			src:      file1,
			dest:     file2,
			expected: true,
		},
		//{
		//	name: "Source file does not exist",
		//	setup: func() error {
		//		return ioutil.WriteFile(file2, []byte("content"), 0644)
		//	},
		//	src:      file1,
		//	dest:     file2,
		//	expected: true,
		//},
		{
			name: "Files have different contents",
			setup: func() error {
				if err := ioutil.WriteFile(file1, []byte("content1"), 0644); err != nil {
					return err
				}
				if err := ioutil.WriteFile(file2, []byte("content2"), 0644); err != nil {
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

func computeMD5(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}

//func TestFileStat(t *testing.T) {
//	fileName := "testfile.txt"
//	defer os.Remove(fileName)
//
//	tests := []struct {
//		name     string
//		setup    func() error
//		expected FileInfo
//		wantErr  bool
//	}{
//		{
//			name: "File exists with correct stats",
//			setup: func() error {
//				return ioutil.WriteFile(fileName, []byte("test content"), 0644)
//			},
//			expected: FileInfo{
//				Mode: 0644,
//				// UID and GID will be set dynamically, so exact values might not be known.
//				// You can use a more sophisticated check if needed.
//			},
//			wantErr: false,
//		},
//		{
//			name:     "File does not exist",
//			setup:    func() error { return nil },
//			expected: FileInfo{},
//			wantErr:  true,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if err := tt.setup(); err != nil {
//				t.Fatalf("Setup failed: %v", err)
//			}
//
//			got, err := FileStat(fileName)
//			if (err != nil) != tt.wantErr {
//				t.Fatalf("FileStat() error = %v, wantErr %v", err, tt.wantErr)
//			}
//			if !tt.wantErr {
//				md5Hash, err := computeMD5(fileName)
//				if err != nil {
//					t.Fatalf("Failed to compute MD5: %v", err)
//				}
//				tt.expected.Md5 = md5Hash
//				if got != tt.expected {
//					t.Errorf("FileStat() = %v, want %v", got, tt.expected)
//				}
//			}
//		})
//	}
//}

func TestPadding(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		blockSize int
		expected  []byte
	}{
		{
			name:      "Exact block size",
			input:     []byte("Hello"),
			blockSize: 8,
			expected:  []byte("Hello\x03\x03\x03"),
		},
		{
			name:      "Block size greater than input",
			input:     []byte("Hello"),
			blockSize: 16,
			expected:  []byte("Hello\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b"),
		},
		//{
		//	name:      "Empty input",
		//	input:     []byte(""),
		//	blockSize: 4,
		//	expected:  []byte("\x04\x04\x04\x04"),
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Padding(tt.input, tt.blockSize)
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("Padding() = %x, want %x", got, tt.expected)
			}
		})
	}
}

//func TestUnPadding(t *testing.T) {
//	tests := []struct {
//		name      string
//		input     []byte
//		expected  []byte
//		expectErr bool
//	}{
//		{
//			name:      "Valid padding",
//			input:     []byte("Hello\x03\x03\x03"),
//			expected:  []byte("Hello"),
//			expectErr: false,
//		},
//		//{
//		//	name:      "Invalid padding length",
//		//	input:     []byte("Hello\x10"),
//		//	expected:  nil,
//		//	expectErr: true,
//		//},
//		{
//			name:      "Empty input",
//			input:     []byte(""),
//			expected:  nil,
//			expectErr: true,
//		},
//		{
//			name:      "No padding",
//			input:     []byte("Hello"),
//			expected:  []byte("Hello"),
//			expectErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got := UnPadding(tt.input)
//			//if (err != nil) != tt.expectErr {
//			//	t.Errorf("UnPadding() error = %v, expectErr %v", err, tt.expectErr)
//			//	return
//			//}
//			if !bytes.Equal(got, tt.expected) {
//				t.Errorf("UnPadding() = %x, want %x", got, tt.expected)
//			}
//		})
//	}
//}

func TestAES_CBC_Encrypt(t *testing.T) {
	tests := []struct {
		name      string
		plainText []byte
		expected  string
		expectErr bool
	}{
		{
			name:      "Encrypt with valid input",
			plainText: []byte("Hello, World!"),
			expected:  "GHlF9VXnLL01nJKk+03uJA==", // Replace with actual expected value
			expectErr: false,
		},
		//{
		//	name: "Encrypt with empty input",
		//	//plainText: []byte(),
		//	expected:  "", // Replace with actual expected value
		//	expectErr: false,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AES_CBC_Encrypt(tt.plainText)
			if (err != nil) != tt.expectErr {
				t.Errorf("AES_CBC_Encrypt() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if got != tt.expected {
				t.Errorf("AES_CBC_Encrypt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAES_CBC_Decrypt(t *testing.T) {
	tests := []struct {
		name       string
		cipherText string
		expected   []byte
		expectErr  bool
	}{
		{
			name:       "Decrypt with valid input",
			cipherText: "GHlF9VXnLL01nJKk+03uJA==", // Replace with actual Base64 encoded ciphertext
			expected:   []byte("Hello, World!"),
			expectErr:  false,
		},
		{
			name:       "Decrypt with invalid input",
			cipherText: "InvalidBase64",
			expected:   nil,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AES_CBC_Decrypt(tt.cipherText)
			if (err != nil) != tt.expectErr {
				t.Errorf("AES_CBC_Decrypt() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !equal(got, tt.expected) {
				t.Errorf("AES_CBC_Decrypt() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to compare byte slices
func equal(a, b []byte) bool {
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

var encryptStr = "/ru+KsOJgjj+JZS11HRh1IDFsQILgnyoqn16XqyoKoo="

type templateTest struct {
	desc     string // description of the test (for helpful errors)
	encrypt  string // encrypt string
	expected string // the value expected
}

var templateTests = []templateTest{
	{
		desc:     "base test",
		encrypt:  "/ru+KsOJgjj+JZS11HRh1IDFsQILgnyoqn16XqyoKoo=",
		expected: "18c6!@nkBNK9P!*d8&1Iq2Qt",
	}}

// TestTemplates runs all tests in templateTests
func TestAES_CBC_Decrypt_V2(t *testing.T) {
	for _, tt := range templateTests {
		ExecuteTestTemplate(tt, t)
	}
}

func ExecuteTestTemplate(tt templateTest, t *testing.T) {
	if s, err := AES_CBC_Decrypt(tt.encrypt); err != nil {
		t.Errorf(tt.desc + ": failed decrypt: " + err.Error())
	} else {
		if !reflect.DeepEqual(string(s), tt.expected) {
			t.Errorf(fmt.Sprintf("%v: Decrypt failed.\nExpected:\n%vActual:\n%v", tt.desc, tt.expected, string(s)))
		}
	}
}
