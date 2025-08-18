package common

import (
	"reflect"
	"testing"
)

func TestSanitizeForLogging(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "basic password and key fields",
			input: map[string]interface{}{
				"username":   "user123",
				"password":   "secret123",
				"access_key": "ak123",
				"secret_key": "sk456",
				"host":       "localhost",
			},
			expected: map[string]interface{}{
				"username":   "user123",
				"password":   "***",
				"access_key": "***",
				"secret_key": "***",
				"host":       "localhost",
			},
		},
		{
			name: "case insensitive matching",
			input: map[string]interface{}{
				"Password":          "secret123",
				"ACCESS_KEY":        "ak123",
				"Secret_Key":        "sk456",
				"API_KEY":           "api123",
				"database_password": "dbpass",
				"SSH_KEY":           "sshkey",
			},
			expected: map[string]interface{}{
				"Password":          "***",
				"ACCESS_KEY":        "***",
				"Secret_Key":        "***",
				"API_KEY":           "***",
				"database_password": "***",
				"SSH_KEY":           "***",
			},
		},
		{
			name: "fields without sensitive information",
			input: map[string]interface{}{
				"username": "user123",
				"host":     "localhost",
				"port":     3306,
				"database": "mydb",
				"timeout":  30,
			},
			expected: map[string]interface{}{
				"username": "user123",
				"host":     "localhost",
				"port":     3306,
				"database": "mydb",
				"timeout":  30,
			},
		},
		{
			name: "mixed sensitive and non-sensitive fields",
			input: map[string]interface{}{
				"source_clone_password": "clonepass",
				"source_host":           "192.168.1.1",
				"source_port":           3306,
				"master_key":            "masterkey123",
				"backup_location":       "/tmp/backup",
			},
			expected: map[string]interface{}{
				"source_clone_password": "***",
				"source_host":           "192.168.1.1",
				"source_port":           3306,
				"master_key":            "***",
				"backup_location":       "/tmp/backup",
			},
		},
		{
			name:     "empty input",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForLogging(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SanitizeForLogging() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  bool
	}{
		// Should be sensitive
		{"password", true},
		{"Password", true},
		{"PASSWORD", true},
		{"secret_key", true},
		{"SECRET_KEY", true},
		{"access_key", true},
		{"api_key", true},
		{"API_KEY", true},
		{"database_password", true},
		{"source_clone_password", true},
		{"master_key", true},
		{"ssh_key", true},
		{"privateKey", true},
		{"publicKey", true},

		// Should not be sensitive
		{"username", false},
		{"host", false},
		{"port", false},
		{"database", false},
		{"timeout", false},
		{"backup_location", false},
		{"socket_file", false},
		{"parallel", false},
		{"storage_type", false},
		{"bucket", false},
		{"endpoint", false},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			result := isSensitiveField(tt.fieldName)
			if result != tt.expected {
				t.Errorf("isSensitiveField(%q) = %v, expected %v", tt.fieldName, result, tt.expected)
			}
		})
	}
}
