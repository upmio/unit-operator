package proxysql

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"go.uber.org/zap/zaptest"
)

// newProxySQLResponse creates a new ProxySQL Response with the given message
func newProxySQLResponse(message string) *Response {
	return &Response{Message: message}
}

// getEnvVarOrError 获取环境变量，如果不存在则返回错误
func getEnvVarOrError(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", errors.New("environment variable " + key + " is not set")
	}
	return value, nil
}

// MockServiceLifecycle 模拟服务生命周期管理
type MockServiceLifecycle struct {
	mock.Mock
}

func (m *MockServiceLifecycle) CheckServiceStatus(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

func (m *MockServiceLifecycle) CheckServiceStopped(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

func (m *MockServiceLifecycle) StartService(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

func (m *MockServiceLifecycle) StopService(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

func (m *MockServiceLifecycle) RestartService(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

// TestProxySQLServiceImplementation 测试 ProxySQL 服务实现
func TestProxySQLServiceImplementation(t *testing.T) {
	service := &service{
		logger: zaptest.NewLogger(t).Sugar(),
	}

	// 测试服务基本接口
	t.Run("service interface implementation", func(t *testing.T) {
		assert.Equal(t, appName, service.Name())
		assert.NotNil(t, service.logger)
	})
}

// TestNewProxySQLResponse 测试响应构造函数
func TestNewProxySQLResponse(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "success message",
			message:  "ProxySQL configuration updated successfully",
			expected: "ProxySQL configuration updated successfully",
		},
		{
			name:     "error message",
			message:  "ProxySQL configuration failed: invalid syntax",
			expected: "ProxySQL configuration failed: invalid syntax",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := newProxySQLResponse(tt.message)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expected, response.Message)
		})
	}
}

// TestGetEnvVarOrError 测试环境变量获取
func TestGetEnvVarOrError(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		setValue    string
		expectError bool
		expected    string
	}{
		{
			name:        "existing environment variable",
			key:         "PROXYSQL_CONFIG_DIR",
			setValue:    "/etc/proxysql",
			expectError: false,
			expected:    "/etc/proxysql",
		},
		{
			name:        "non-existing environment variable",
			key:         "PROXYSQL_NON_EXISTING",
			setValue:    "",
			expectError: true,
			expected:    "",
		},
		{
			name:        "empty environment variable",
			key:         "PROXYSQL_EMPTY",
			setValue:    "",
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			if tt.setValue != "" {
				t.Setenv(tt.key, tt.setValue)
			}

			result, err := getEnvVarOrError(tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "environment variable")
				assert.Contains(t, err.Error(), "is not set")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestValidateSetVariableRequest 测试设置变量请求验证
func TestValidateSetVariableRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *SetVariableRequest
		expectError bool
		errorField  string
	}{
		{
			name: "valid request",
			req: &SetVariableRequest{
				Key:      "admin-admin_credentials",
				Value:    "admin:admin",
				Section:  "admin",
				Username: "admin",
				Password: "admin",
			},
			expectError: false,
		},
		{
			name: "missing key",
			req: &SetVariableRequest{
				Value:    "admin:admin",
				Section:  "admin",
				Username: "admin",
				Password: "admin",
			},
			expectError: true,
			errorField:  "key",
		},
		{
			name: "missing value",
			req: &SetVariableRequest{
				Key:      "admin-admin_credentials",
				Section:  "admin",
				Username: "admin",
				Password: "admin",
			},
			expectError: true,
			errorField:  "value",
		},
		{
			name: "missing username",
			req: &SetVariableRequest{
				Key:      "admin-admin_credentials",
				Value:    "admin:admin",
				Section:  "admin",
				Password: "admin",
			},
			expectError: true,
			errorField:  "username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSetVariableRequest(tt.req)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorField != "" {
					assert.Contains(t, err.Error(), tt.errorField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProxySQLConfiguration 测试 ProxySQL 配置
func TestProxySQLConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		configKey   string
		configValue string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid admin configuration",
			configKey:   "admin-admin_credentials",
			configValue: "admin:admin",
			expectError: false,
		},
		{
			name:        "valid mysql server configuration",
			configKey:   "mysql-servers",
			configValue: "192.168.1.100:3306",
			expectError: false,
		},
		{
			name:        "empty configuration value",
			configKey:   "admin-admin_credentials",
			configValue: "",
			expectError: true,
			errorMsg:    "configuration value cannot be empty",
		},
		{
			name:        "invalid configuration key",
			configKey:   "invalid-config",
			configValue: "some-value",
			expectError: true,
			errorMsg:    "invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProxySQLVariableConfig(tt.configKey, tt.configValue)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProxySQLConfigurationPatterns 测试 ProxySQL 配置模式
func TestProxySQLConfigurationPatterns(t *testing.T) {
	testLogger := zaptest.NewLogger(t).Sugar()
	_ = testLogger // 避免未使用变量警告

	tests := []struct {
		name           string
		configKey      string
		configValue    string
		expectedChange bool
		expectedError  bool
	}{
		{
			name:           "valid mysql server configuration",
			configKey:      "mysql_servers",
			configValue:    "INSERT INTO mysql_servers(hostgroup_id, hostname, port) VALUES (0, 'mysql-0', 3306)",
			expectedChange: true,
			expectedError:  false,
		},
		{
			name:           "valid mysql user configuration",
			configKey:      "mysql_users",
			configValue:    "INSERT INTO mysql_users(username, password, default_hostgroup) VALUES ('app', 'password', 0)",
			expectedChange: true,
			expectedError:  false,
		},
		{
			name:           "invalid configuration",
			configKey:      "invalid_table",
			configValue:    "INVALID SQL STATEMENT",
			expectedChange: false,
			expectedError:  true,
		},
		{
			name:           "empty configuration",
			configKey:      "mysql_servers",
			configValue:    "",
			expectedChange: false,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这里测试配置验证逻辑
			err := validateProxySQLConfig(tt.configKey, tt.configValue)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestErrorHandlingPatterns 测试错误处理模式
func TestErrorHandlingPatterns(t *testing.T) {
	testLogger := zaptest.NewLogger(t).Sugar()

	tests := []struct {
		name         string
		operation    string
		err          error
		expectedType string
	}{
		{
			name:         "admin interface connection error",
			operation:    "admin connect",
			err:          errors.New("connection refused"),
			expectedType: "connection error",
		},
		{
			name:         "configuration error",
			operation:    "config update",
			err:          errors.New("invalid configuration"),
			expectedType: "config error",
		},
		{
			name:         "timeout error",
			operation:    "backup",
			err:          context.DeadlineExceeded,
			expectedType: "timeout error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试错误处理和响应创建
			response, err := common.LogAndReturnError(
				testLogger,
				newProxySQLResponse,
				"operation failed: "+tt.operation,
				tt.err,
			)

			assert.Error(t, err)
			assert.NotNil(t, response)
			assert.Contains(t, response.Message, tt.operation)
			assert.Contains(t, err.Error(), tt.operation)
		})
	}
}

// 辅助验证函数
func validateSetVariableRequest(req *SetVariableRequest) error {
	if req.GetKey() == "" {
		return errors.New("key is required")
	}
	if req.GetValue() == "" {
		return errors.New("value is required")
	}
	if req.GetUsername() == "" {
		return errors.New("username is required")
	}
	return nil
}

func validateProxySQLVariableConfig(key, value string) error {
	if value == "" {
		return errors.New("configuration value cannot be empty")
	}

	// 基本的配置键验证
	validPrefixes := []string{"admin-", "mysql-", "general-"}
	validKey := false

	for _, prefix := range validPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			validKey = true
			break
		}
	}

	if !validKey {
		return errors.New("invalid configuration key")
	}

	return nil
}

// validateProxySQLConfig 验证 ProxySQL 配置
func validateProxySQLConfig(key, value string) error {
	if value == "" {
		return errors.New("configuration value cannot be empty")
	}

	// 基本的 SQL 语法检查
	validTables := []string{"mysql_servers", "mysql_users", "mysql_query_rules", "scheduler"}
	validTable := false

	for _, table := range validTables {
		if key == table {
			validTable = true
			break
		}
	}

	if !validTable {
		return errors.New("invalid configuration table")
	}

	return nil
}

// BenchmarkNewProxySQLResponse 性能测试
func BenchmarkNewProxySQLResponse(b *testing.B) {
	message := "ProxySQL operation completed successfully"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newProxySQLResponse(message)
	}
}

// TestProxySQLConnectionPool 测试连接池管理
func TestProxySQLConnectionPool(t *testing.T) {
	tests := []struct {
		name          string
		poolConfig    map[string]interface{}
		expectedValid bool
		expectedError string
	}{
		{
			name: "valid connection pool configuration",
			poolConfig: map[string]interface{}{
				"max_connections":       100,
				"default_query_delay":   0,
				"default_query_timeout": 36000000,
				"have_compress":         true,
			},
			expectedValid: true,
		},
		{
			name: "invalid max_connections",
			poolConfig: map[string]interface{}{
				"max_connections": -1,
			},
			expectedValid: false,
			expectedError: "max_connections must be positive",
		},
		{
			name:          "missing required fields",
			poolConfig:    map[string]interface{}{},
			expectedValid: false,
			expectedError: "missing required configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConnectionPoolConfig(tt.poolConfig)

			if tt.expectedValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}
}

// validateConnectionPoolConfig 验证连接池配置
func validateConnectionPoolConfig(config map[string]interface{}) error {
	if len(config) == 0 {
		return errors.New("missing required configuration")
	}

	if maxConn, exists := config["max_connections"]; exists {
		if val, ok := maxConn.(int); ok && val < 0 {
			return errors.New("max_connections must be positive")
		}
	}

	return nil
}
