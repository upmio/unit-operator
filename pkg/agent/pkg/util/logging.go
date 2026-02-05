package util

import (
	"regexp"
	"strings"

	"go.uber.org/zap"
)

var (
	// sensitiveFieldPattern matches field names that contain sensitive information
	// Matches fields containing "key" or "password" (case insensitive)
	sensitiveFieldPattern = regexp.MustCompile(`(?i)(access_key|secret_key|token|password)`)
)

// sanitizeForLogging sanitizes sensitive information for logging
func sanitizeForLogging(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for k, v := range data {
		if isSensitiveField(k) {
			sanitized[k] = "***"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// isSensitiveField checks if a field name contains sensitive information
func isSensitiveField(fieldName string) bool {
	// Convert to lowercase for case-insensitive matching
	lowercaseField := strings.ToLower(fieldName)
	return sensitiveFieldPattern.MatchString(lowercaseField)
}

// LogRequestSafely logs request information with sensitive data masked
func LogRequestSafely(logger *zap.SugaredLogger, operation string, data map[string]interface{}) {
	sanitized := sanitizeForLogging(data)
	fields := make([]interface{}, 0, len(sanitized)*2)
	for k, v := range sanitized {
		fields = append(fields, k, v)
	}
	logger.With(fields...).Infof("receive %s request", operation)
}
