package content

import (
	"testing"
)

func TestRead(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		vars    map[string]string
		wantErr bool
	}{
		{
			name:    "Valid YAML with no variables",
			data:    "key: value",
			vars:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "Valid YAML with variables",
			data:    "key: value\nanother_key: another_value",
			vars:    map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name:    "Empty YAML data",
			data:    "",
			vars:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "Invalid YAML data",
			data:    "key: [unclosed array",
			vars:    map[string]string{},
			wantErr: true,
		},
		{
			name:    "Valid YAML with nested structure",
			data:    "parent:\n  child: value",
			vars:    map[string]string{"parent.child": "value"},
			wantErr: false,
		},
		//{
		//	name:    "Valid YAML with special characters",
		//	data:    "key: value\nspecial: !@#$%^&*()",
		//	vars:    map[string]string{"key": "value"},
		//	wantErr: false,
		//},
		{
			name:    "YAML with multiple lines",
			data:    "line1: value1\nline2: value2\nline3: value3",
			vars:    map[string]string{"line1": "value1", "line2": "value2"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := read(tt.data, tt.vars); (err != nil) != tt.wantErr {
				t.Errorf("read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetValues(t *testing.T) {
	tests := []struct {
		name     string
		contents []string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "No contents",
			contents: []string{},
			expected: map[string]string{},
			wantErr:  false,
		},
		//{
		//	name:     "Single valid YAML",
		//	contents: []string{"key: value"},
		//	expected: map[string]string{"key": "value"},
		//	wantErr:  false,
		//},
		//{
		//	name:     "Multiple valid YAMLs",
		//	contents: []string{"key1: value1", "key2: value2"},
		//	expected: map[string]string{"key1": "value1", "key2": "value2"},
		//	wantErr:  false,
		//},
		//{
		//	name:     "One invalid YAML",
		//	contents: []string{"key1: value1", "invalid"},
		//	expected: map[string]string{"key1": "value1"},
		//	wantErr:  true,
		//},
		{
			name:     "All invalid YAMLs",
			contents: []string{"invalid", "also_invalid"},
			expected: map[string]string{},
			wantErr:  true,
		},
		//{
		//	name:     "Valid YAML with special characters",
		//	contents: []string{"key: value\nspecial: !@#$%^&*()"},
		//	expected: map[string]string{"key": "value", "special": "!@#$%^&*()"},
		//	wantErr:  false,
		//},
		//{
		//{
		//	name:     "Valid and Invalid Mixed",
		//	contents: []string{"key1: value1", "invalid", "key2: value2"},
		//	expected: map[string]string{"key1": "value1"},
		//	wantErr:  true,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{contents: tt.contents}
			got, err := client.GetValues()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !equal(got, tt.expected) {
				t.Errorf("GetValues() = %v, expected %v", got, tt.expected)
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
