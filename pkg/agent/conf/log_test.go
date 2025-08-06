package conf

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel zapcore.Level
	}{
		{
			name:          "Debug Level",
			level:         "debug",
			expectedLevel: zapcore.DebugLevel,
		},
		{
			name:          "Info Level",
			level:         "info",
			expectedLevel: zapcore.InfoLevel,
		},
		{
			name:          "Warn Level",
			level:         "warn",
			expectedLevel: zapcore.WarnLevel,
		},
		{
			name:          "Error Level",
			level:         "error",
			expectedLevel: zapcore.ErrorLevel,
		},
		{
			name:          "Unknown Level",
			level:         "unknown",
			expectedLevel: zapcore.InfoLevel, // default level
		},
		{
			name:          "Empty Level",
			level:         "",
			expectedLevel: zapcore.InfoLevel, // default level
		},
		{
			name:          "Uppercase Level",
			level:         "DEBUG",
			expectedLevel: zapcore.DebugLevel, // should be case-insensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &Log{
				Level: tt.level,
			}
			actualLevel := log.GetLogLevel()
			assert.Equal(t, tt.expectedLevel, actualLevel)
		})
	}
}
