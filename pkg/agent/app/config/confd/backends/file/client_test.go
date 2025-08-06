package file

import (
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		fileName string
		content  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "Valid YAML file",
			fileName: "valid.yaml",
			content:  "key: value\nanother_key: another_value",
			expected: map[string]string{"/key": "value", "/another_key": "another_value"},
			wantErr:  false,
		},
		{
			name:     "Invalid YAML file",
			fileName: "invalid.yaml",
			content:  "key: [unclosed array",
			expected: map[string]string{},
			wantErr:  true,
		},
		//{
		//	name:     "Empty YAML file",
		//	fileName: "empty.yaml",
		//	content:  "",
		//	expected: map[string]string{},
		//	wantErr:  false,
		//},
		{
			name:     "Nested YAML file",
			fileName: "nested.yaml",
			content:  "parent:\n  child: value",
			expected: map[string]string{"/parent/child": "value"},
			wantErr:  false,
		},
		//{
		//	name:     "YAML file with special characters",
		//	fileName: "special.yaml",
		//	content:  "key: value\nspecial: !@#$%^&*()",
		//	expected: map[string]string{"/key": "value", "/special": "!@#$%^&*()"},
		//	wantErr:  false,
		//},
		{
			name:     "Non-existent file",
			fileName: "non_existent.yaml",
			content:  "",
			expected: map[string]string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the test file
			if tt.content != "" {
				filePath := tempDir + "/" + tt.fileName
				err := os.WriteFile(filePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			vars := make(map[string]string)
			err := readFile(tempDir+"/"+tt.fileName, vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("readFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equal(vars, tt.expected) {
				t.Errorf("readFile() = %v, expected %v", vars, tt.expected)
			}
		})
	}
}

func equal(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestGetValues(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filepath string
		content  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "Valid YAML file",
			filepath: "valid.yaml",
			content:  "key: value\nanother_key: another_value",
			expected: map[string]string{"/key": "value", "/another_key": "another_value"},
			wantErr:  false,
		},
		{
			name:     "Invalid YAML file",
			filepath: "invalid.yaml",
			content:  "key: [unclosed array",
			expected: map[string]string{},
			wantErr:  true,
		},
		//{
		//	name:     "Empty YAML file",
		//	filepath: "empty.yaml",
		//	content:  "",
		//	expected: map[string]string{},
		//	wantErr:  false,
		//},
		{
			name:     "Nested YAML file",
			filepath: "nested.yaml",
			content:  "parent:\n  child: value",
			expected: map[string]string{"/parent/child": "value"},
			wantErr:  false,
		},
		//{
		//	name:     "YAML file with special characters",
		//	filepath: "special.yaml",
		//	content:  "key: value\nspecial: !@#$%^&*()",
		//	expected: map[string]string{"/key": "value", "/special": "!@#$%^&*()"},
		//	wantErr:  false,
		//},
		{
			name:     "Non-existent file",
			filepath: "non_existent.yaml",
			content:  "",
			expected: map[string]string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the test file
			if tt.content != "" {
				filePath := tempDir + "/" + tt.filepath
				err := os.WriteFile(filePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
				tt.filepath = filePath
			}

			client := &Client{filepath: tt.filepath}
			got, err := client.GetValues()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equal(got, tt.expected) {
				t.Errorf("GetValues() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestNodeWalk(t *testing.T) {
	tests := []struct {
		name     string
		node     interface{}
		key      string
		expected map[string]string
	}{
		{
			name:     "Single string",
			node:     "value",
			key:      "key",
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "Single int",
			node:     123,
			key:      "key",
			expected: map[string]string{"key": "123"},
		},
		{
			name:     "Single bool",
			node:     true,
			key:      "key",
			expected: map[string]string{"key": "true"},
		},
		{
			name: "Map of strings",
			node: map[interface{}]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			key: "root",
			expected: map[string]string{
				"root/key1": "value1",
				"root/key2": "value2",
			},
		},
		{
			name: "Nested map",
			node: map[interface{}]interface{}{
				"key1": map[interface{}]interface{}{
					"subkey1": "subvalue1",
				},
			},
			key: "root",
			expected: map[string]string{
				"root/key1/subkey1": "subvalue1",
			},
		},
		{
			name: "Array of mixed types",
			node: []interface{}{
				"string",
				123,
				true,
				45.67,
			},
			key: "root",
			expected: map[string]string{
				"root/0": "string",
				"root/1": "123",
				"root/2": "true",
				"root/3": "45.67",
			},
		},
		{
			name: "Complex nested structure",
			node: map[interface{}]interface{}{
				"key1": []interface{}{
					"string",
					123,
					true,
					map[interface{}]interface{}{
						"subkey1": "subvalue1",
					},
				},
			},
			key: "root",
			expected: map[string]string{
				"root/key1/0":         "string",
				"root/key1/1":         "123",
				"root/key1/2":         "true",
				"root/key1/3/subkey1": "subvalue1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := make(map[string]string)
			if err := nodeWalk(tt.node, tt.key, vars); err != nil {
				t.Errorf("nodeWalk() error = %v", err)
			}
			if !equal(vars, tt.expected) {
				t.Errorf("nodeWalk() = %v, expected %v", vars, tt.expected)
			}
		})
	}
}
