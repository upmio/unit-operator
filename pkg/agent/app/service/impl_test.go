package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"go.uber.org/zap/zaptest"
)

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

// TestServiceImplementation 测试服务生命周期实现
func TestServiceImplementation(t *testing.T) {
	service := &service{
		logger: zaptest.NewLogger(t).Sugar(),
	}

	// 测试服务基本接口
	t.Run("service interface implementation", func(t *testing.T) {
		assert.Equal(t, appName, service.Name())
		assert.NotNil(t, service.logger)
	})
}

// TestNewServiceResponse 测试响应构造函数
func TestNewServiceResponse(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "success message",
			message:  "Service started successfully",
			expected: "Service started successfully",
		},
		{
			name:     "error message",
			message:  "Service failed to start: permission denied",
			expected: "Service failed to start: permission denied",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := newServiceResponse(tt.message)
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
			key:         "SERVICE_CONFIG_DIR",
			setValue:    "/etc/service",
			expectError: false,
			expected:    "/etc/service",
		},
		{
			name:        "non-existing environment variable",
			key:         "SERVICE_NON_EXISTING",
			setValue:    "",
			expectError: true,
			expected:    "",
		},
		{
			name:        "empty environment variable",
			key:         "SERVICE_EMPTY",
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

// TestValidateServiceRequest 测试服务请求验证
func TestValidateServiceRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *ServiceRequest
		expectError bool
		errorField  string
	}{
		{
			name:        "valid request",
			req:         &ServiceRequest{},
			expectError: false,
		},
		{
			name:        "nil request",
			req:         nil,
			expectError: true,
			errorField:  "request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceRequest(tt.req)

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

// TestServiceStatus 测试服务状态
func TestServiceStatus(t *testing.T) {
	tests := []struct {
		name          string
		state         ProcessState
		expectedValid bool
	}{
		{
			name:          "StateStopped",
			state:         ProcessState_StateStopped,
			expectedValid: true,
		},
		{
			name:          "StateRunning",
			state:         ProcessState_StateRunning,
			expectedValid: true,
		},
		{
			name:          "StateUnknown",
			state:         ProcessState_StateUnknown,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedValid, isValidProcessState(tt.state))
		})
	}
}

// TestServiceResponse 测试服务响应
func TestServiceResponse(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "success response",
			message:  "Service operation completed",
			expected: "Service operation completed",
		},
		{
			name:     "error response",
			message:  "Service operation failed",
			expected: "Service operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &ServiceResponse{Message: tt.message}
			assert.Equal(t, tt.expected, response.Message)
		})
	}
}

// TestProcessStateEnum 测试进程状态枚举
func TestProcessStateEnum(t *testing.T) {
	tests := []struct {
		name           string
		status         ProcessState
		expectedString string
		expectedValid  bool
	}{
		{
			name:           "running status",
			status:         ProcessState_StateRunning,
			expectedString: "StateRunning",
			expectedValid:  true,
		},
		{
			name:           "stopped status",
			status:         ProcessState_StateStopped,
			expectedString: "StateStopped",
			expectedValid:  true,
		},
		{
			name:           "starting status",
			status:         ProcessState_StateStarting,
			expectedString: "StateStarting",
			expectedValid:  true,
		},
		{
			name:           "stopping status",
			status:         ProcessState_StateStopping,
			expectedString: "StateStopping",
			expectedValid:  true,
		},
		{
			name:           "exited status",
			status:         ProcessState_StateExited,
			expectedString: "StateExited",
			expectedValid:  true,
		},
		{
			name:           "unknown status",
			status:         ProcessState_StateUnknown,
			expectedString: "StateUnknown",
			expectedValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedString, tt.status.String())
			assert.Equal(t, tt.expectedValid, isValidProcessState(tt.status))
		})
	}
}

// TestServiceOperations 测试服务操作模式
func TestServiceOperations(t *testing.T) {
	testLogger := zaptest.NewLogger(t).Sugar()
	_ = testLogger // 避免未使用变量警告

	tests := []struct {
		name              string
		operation         string
		serviceName       string
		expectedDuration  time.Duration
		expectError       bool
		expectedErrorType string
	}{
		{
			name:             "start mysql service",
			operation:        "start",
			serviceName:      "mysql",
			expectedDuration: 10 * time.Second,
			expectError:      false,
		},
		{
			name:             "stop postgresql service",
			operation:        "stop",
			serviceName:      "postgresql",
			expectedDuration: 5 * time.Second,
			expectError:      false,
		},
		{
			name:              "invalid service name",
			operation:         "start",
			serviceName:       "",
			expectedDuration:  0,
			expectError:       true,
			expectedErrorType: "validation error",
		},
		{
			name:              "unknown operation",
			operation:         "unknown",
			serviceName:       "mysql",
			expectedDuration:  0,
			expectError:       true,
			expectedErrorType: "operation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := estimateOperationDuration(tt.operation, tt.serviceName)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorType != "" {
					assert.Contains(t, err.Error(), "error")
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDuration, duration)
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
			name:         "systemctl command error",
			operation:    "start service",
			err:          errors.New("systemctl: command not found"),
			expectedType: "command error",
		},
		{
			name:         "permission error",
			operation:    "access config",
			err:          errors.New("permission denied"),
			expectedType: "permission error",
		},
		{
			name:         "timeout error",
			operation:    "service start",
			err:          context.DeadlineExceeded,
			expectedType: "timeout error",
		},
		{
			name:         "resource error",
			operation:    "allocate memory",
			err:          errors.New("insufficient resources"),
			expectedType: "resource error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试错误处理和响应创建
			response, err := common.LogAndReturnError(
				testLogger,
				newServiceResponse,
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

// TestServiceConfiguration 测试服务配置
func TestServiceConfiguration(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		configOverrides map[string]interface{}
		expectedValid   bool
		expectedError   string
	}{
		{
			name:        "valid mysql configuration",
			serviceName: "mysql",
			configOverrides: map[string]interface{}{
				"port":            3306,
				"bind_host":       "0.0.0.0",
				"max_connections": 100,
			},
			expectedValid: true,
		},
		{
			name:        "valid postgresql configuration",
			serviceName: "postgresql",
			configOverrides: map[string]interface{}{
				"port":             5432,
				"listen_addresses": "*",
				"max_connections":  200,
			},
			expectedValid: true,
		},
		{
			name:            "empty service name",
			serviceName:     "",
			configOverrides: map[string]interface{}{},
			expectedValid:   false,
			expectedError:   "service name cannot be empty",
		},
		{
			name:        "invalid port configuration",
			serviceName: "mysql",
			configOverrides: map[string]interface{}{
				"port": -1,
			},
			expectedValid: false,
			expectedError: "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceConfig(tt.serviceName, tt.configOverrides)

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

// 辅助验证函数
func validateServiceRequest(req *ServiceRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}
	return nil
}

func isValidProcessState(status ProcessState) bool {
	switch status {
	case ProcessState_StateStopped, ProcessState_StateStarting, ProcessState_StateRunning,
		ProcessState_StateBackoff, ProcessState_StateStopping, ProcessState_StateExited,
		ProcessState_StateFatal, ProcessState_StateUnknown:
		return true
	default:
		return false
	}
}

func estimateOperationDuration(operation, serviceName string) (time.Duration, error) {
	if serviceName == "" {
		return 0, errors.New("validation error: service name cannot be empty")
	}

	switch operation {
	case "start":
		return 10 * time.Second, nil
	case "stop":
		return 5 * time.Second, nil
	case "restart":
		return 15 * time.Second, nil
	case "status":
		return 1 * time.Second, nil
	default:
		return 0, errors.New("operation error: unknown operation")
	}
}

func validateServiceConfig(serviceName string, config map[string]interface{}) error {
	if serviceName == "" {
		return errors.New("service name cannot be empty")
	}

	if port, exists := config["port"]; exists {
		if val, ok := port.(int); ok && (val <= 0 || val > 65535) {
			return errors.New("invalid port range")
		}
	}

	return nil
}

// BenchmarkNewServiceResponse 性能测试
func BenchmarkNewServiceResponse(b *testing.B) {
	message := "Service operation completed successfully"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newServiceResponse(message)
	}
}

// TestServiceEnvironment 测试服务环境设置
func TestServiceEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		cleanupFunc func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid environment setup",
			envVars: map[string]string{
				"SERVICE_HOME":      "/opt/service",
				"SERVICE_LOG_LEVEL": "INFO",
				"SERVICE_PORT":      "8080",
			},
			cleanupFunc: func() {
				_ = os.Unsetenv("SERVICE_HOME")
				_ = os.Unsetenv("SERVICE_LOG_LEVEL")
				_ = os.Unsetenv("SERVICE_PORT")
			},
			expectError: false,
		},
		{
			name: "missing required environment variables",
			envVars: map[string]string{
				"SERVICE_LOG_LEVEL": "DEBUG",
			},
			cleanupFunc: func() {
				_ = os.Unsetenv("SERVICE_LOG_LEVEL")
			},
			expectError: true,
			errorMsg:    "missing required environment variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}
			defer tt.cleanupFunc()

			err := validateServiceEnvironment()

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

// validateServiceEnvironment 验证服务环境设置
func validateServiceEnvironment() error {
	requiredVars := []string{"SERVICE_HOME"}

	for _, envVar := range requiredVars {
		if os.Getenv(envVar) == "" {
			return errors.New("missing required environment variable: " + envVar)
		}
	}

	return nil
}
