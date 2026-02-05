package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestSanitizeForLoggingMasksSensitiveFields(t *testing.T) {
	data := map[string]interface{}{
		"access_key": "secret",
		"username":   "user",
		"Password":   "123456",
	}

	sanitized := sanitizeForLogging(data)
	require.Equal(t, "***", sanitized["access_key"])
	require.Equal(t, "***", sanitized["Password"])
	require.Equal(t, "user", sanitized["username"])
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		field string
		want  bool
	}{
		{"Access_Key", true},
		{"token", true},
		{"PASSWORD", true},
		{"username", false},
		{"path", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.field, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isSensitiveField(tt.field))
		})
	}
}

func TestLogRequestSafely(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core).Sugar()

	LogRequestSafely(logger, "backup", map[string]interface{}{
		"access_key": "value",
		"path":       "/data",
	})

	entries := recorded.All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	require.Equal(t, "***", fields["access_key"])
	require.Equal(t, "/data", fields["path"])
	require.Contains(t, entries[0].Message, "backup")
}
