package config

import (
	"context"
	"errors"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestConfigServiceImplementation 测试配置服务实现
func TestConfigServiceImplementation(t *testing.T) {
	service := &service{
		logger: zaptest.NewLogger(t).Sugar(),
	}

	// 测试服务基本接口
	t.Run("service interface implementation", func(t *testing.T) {
		assert.Equal(t, appName, service.Name())
		assert.NotNil(t, service.logger)
	})
}

// TestNewConfigResponse 测试响应构造函数
func TestNewConfigResponse(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "success message",
			message:  "configuration synced successfully",
			expected: "configuration synced successfully",
		},
		{
			name:     "error message",
			message:  "failed to sync configuration: template not found",
			expected: "failed to sync configuration: template not found",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := newConfigResponse(tt.message)
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
			key:         "CONFIG_PATH",
			setValue:    "/etc/myapp/config.yaml",
			expectError: false,
			expected:    "/etc/myapp/config.yaml",
		},
		{
			name:        "non-existing environment variable",
			key:         "NON_EXISTING_CONFIG",
			setValue:    "",
			expectError: true,
			expected:    "",
		},
		{
			name:        "empty environment variable",
			key:         "EMPTY_CONFIG",
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

// TestSyncConfigValidation 测试配置同步请求验证
func TestSyncConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         *SyncConfigRequest
		expectError bool
		errorField  string
	}{
		{
			name: "valid request",
			req: &SyncConfigRequest{
				Key:                   "config.yaml",
				Namespace:             "default",
				ValueConfigmapName:    "app-config",
				TemplateConfigmapName: "app-template",
				ExtendValueConfigmaps: []string{"extend-config"},
			},
			expectError: false,
		},
		{
			name: "missing key",
			req: &SyncConfigRequest{
				Namespace:             "default",
				ValueConfigmapName:    "app-config",
				TemplateConfigmapName: "app-template",
			},
			expectError: true,
			errorField:  "key",
		},
		{
			name: "missing namespace",
			req: &SyncConfigRequest{
				Key:                   "config.yaml",
				ValueConfigmapName:    "app-config",
				TemplateConfigmapName: "app-template",
			},
			expectError: true,
			errorField:  "namespace",
		},
		{
			name: "missing value configmap",
			req: &SyncConfigRequest{
				Key:                   "config.yaml",
				Namespace:             "default",
				TemplateConfigmapName: "app-template",
			},
			expectError: true,
			errorField:  "value_configmap_name",
		},
		{
			name: "missing template configmap",
			req: &SyncConfigRequest{
				Key:                "config.yaml",
				Namespace:          "default",
				ValueConfigmapName: "app-config",
			},
			expectError: true,
			errorField:  "template_configmap_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSyncConfigRequest(tt.req)

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

// TestRewriteConfigValidation 测试配置重写请求验证
func TestRewriteConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         *RewriteConfigRequest
		expectError bool
		errorField  string
	}{
		{
			name: "valid request",
			req: &RewriteConfigRequest{
				Key:           "database.host",
				Namespace:     "default",
				ConfigmapName: "app-config",
				Value:         "localhost",
			},
			expectError: false,
		},
		{
			name: "missing key",
			req: &RewriteConfigRequest{
				Namespace:     "default",
				ConfigmapName: "app-config",
				Value:         "localhost",
			},
			expectError: true,
			errorField:  "key",
		},
		{
			name: "missing configmap name",
			req: &RewriteConfigRequest{
				Key:       "database.host",
				Namespace: "default",
				Value:     "localhost",
			},
			expectError: true,
			errorField:  "configmap_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRewriteConfigRequest(tt.req)

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

// TestSyncConfigWithMockKubernetes 测试配置同步（使用模拟 Kubernetes 客户端）
func TestSyncConfigWithMockKubernetes(t *testing.T) {

	// 设置环境变量
	t.Setenv("CONFIG_PATH", "/tmp/test-config.yaml")

	ctx := context.Background()

	tests := []struct {
		name          string
		setupMocks    func(clientset *fake.Clientset)
		req           *SyncConfigRequest
		expectError   bool
		expectedError string
	}{
		{
			name: "template configmap not found",
			setupMocks: func(_ *fake.Clientset) {
				// 不创建任何 ConfigMap，模拟找不到的情况
			},
			req: &SyncConfigRequest{
				Key:                   "config.yaml",
				Namespace:             "default",
				ValueConfigmapName:    "app-config",
				TemplateConfigmapName: "app-template",
			},
			expectError:   true,
			expectedError: "failed to fetch template configmap",
		},
		{
			name: "value configmap not found",
			setupMocks: func(clientset *fake.Clientset) {
				// 只创建模板 ConfigMap
				templateConfigMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-template",
						Namespace: "default",
					},
					Data: map[string]string{
						"config.yaml": "database:\n  host: {{ getv \"/database/host\" }}",
					},
				}
				_, _ = clientset.CoreV1().ConfigMaps("default").Create(ctx, templateConfigMap, metav1.CreateOptions{})
			},
			req: &SyncConfigRequest{
				Key:                   "config.yaml",
				Namespace:             "default",
				ValueConfigmapName:    "app-config",
				TemplateConfigmapName: "app-template",
			},
			expectError:   true,
			expectedError: "failed to fetch value configmap",
		},
		{
			name: "successful sync",
			setupMocks: func(clientset *fake.Clientset) {
				// 创建模板 ConfigMap
				templateConfigMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-template",
						Namespace: "default",
					},
					Data: map[string]string{
						"config.yaml": "database:\n  host: localhost\n  port: 5432",
					},
				}
				_, _ = clientset.CoreV1().ConfigMaps("default").Create(ctx, templateConfigMap, metav1.CreateOptions{})

				// 创建值 ConfigMap
				valueConfigMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"config.yaml": "database:\n  host: localhost\n  port: 5432",
					},
				}
				_, _ = clientset.CoreV1().ConfigMaps("default").Create(ctx, valueConfigMap, metav1.CreateOptions{})
			},
			req: &SyncConfigRequest{
				Key:                   "config.yaml",
				Namespace:             "default",
				ValueConfigmapName:    "app-config",
				TemplateConfigmapName: "app-template",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 fake Kubernetes 客户端
			fakeClientset := fake.NewClientset()

			service := &service{
				clientSet: fakeClientset,
				logger:    zaptest.NewLogger(t).Sugar(),
				confdConfig: &confd.Config{
					BackendsConfig: confd.BackendsConfig{
						Backend: "content"},
				},
			}

			// 执行设置操作
			tt.setupMocks(fakeClientset)

			// 执行测试
			response, err := service.SyncConfig(ctx, tt.req)

			if tt.expectError {
				assert.Error(t, err)
				assert.NotNil(t, response)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Contains(t, response.Message, "successfully")
			}
		})
	}
}

// TestRewriteConfigWithMockKubernetes 测试配置重写（使用模拟 Kubernetes 客户端）
func TestRewriteConfigWithMockKubernetes(t *testing.T) {

	ctx := context.Background()

	tests := []struct {
		name          string
		setupMocks    func(clientset *fake.Clientset)
		req           *RewriteConfigRequest
		expectError   bool
		expectedError string
	}{
		{
			name: "configmap not found",
			setupMocks: func(_ *fake.Clientset) {
				// 不创建任何 ConfigMap
			},
			req: &RewriteConfigRequest{
				Key:           "database.host",
				Namespace:     "default",
				ConfigmapName: "app-config",
				Value:         "new-host",
			},
			expectError:   true,
			expectedError: "failed to fetch configmap",
		},
		{
			name: "successful rewrite - new value",
			setupMocks: func(clientset *fake.Clientset) {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"database.host": "old-host",
					},
				}
				_, _ = clientset.CoreV1().ConfigMaps("default").Create(ctx, configMap, metav1.CreateOptions{})
			},
			req: &RewriteConfigRequest{
				Key:           "database.host",
				Namespace:     "default",
				ConfigmapName: "app-config",
				Value:         "new-host",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 fake Kubernetes 客户端
			fakeClientset := fake.NewClientset()

			service := &service{
				clientSet: fakeClientset,
				logger:    zaptest.NewLogger(t).Sugar(),
				confdConfig: &confd.Config{
					BackendsConfig: confd.BackendsConfig{
						Backend: "content"},
				},
			}

			// 执行设置操作
			tt.setupMocks(fakeClientset)

			// 执行测试
			response, err := service.RewriteConfig(ctx, tt.req)

			if tt.expectError {
				assert.Error(t, err)
				assert.NotNil(t, response)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Contains(t, response.Message, "successfully")
			}
		})
	}
}

// 辅助验证函数
func validateSyncConfigRequest(req *SyncConfigRequest) error {
	if req.GetKey() == "" {
		return errors.New("key is required")
	}
	if req.GetNamespace() == "" {
		return errors.New("namespace is required")
	}
	if req.GetValueConfigmapName() == "" {
		return errors.New("value_configmap_name is required")
	}
	if req.GetTemplateConfigmapName() == "" {
		return errors.New("template_configmap_name is required")
	}
	return nil
}

func validateRewriteConfigRequest(req *RewriteConfigRequest) error {
	if req.GetKey() == "" {
		return errors.New("key is required")
	}
	if req.GetNamespace() == "" {
		return errors.New("namespace is required")
	}
	if req.GetConfigmapName() == "" {
		return errors.New("configmap_name is required")
	}
	return nil
}

// BenchmarkNewConfigResponse 性能测试
func BenchmarkNewConfigResponse(b *testing.B) {
	message := "configuration sync completed successfully"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = newConfigResponse(message)
	}
}

// TestConfigSecurityValidation 测试配置安全验证
func TestConfigSecurityValidation(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		expectError bool
		errorReason string
	}{
		{
			name:        "safe configuration key",
			key:         "database.host",
			value:       "localhost",
			expectError: false,
		},
		{
			name:        "safe configuration value",
			key:         "app.debug",
			value:       "false",
			expectError: false,
		},
		{
			name:        "potentially unsafe key with path traversal",
			key:         "../../../etc/passwd",
			value:       "hacker",
			expectError: true,
			errorReason: "path traversal",
		},
		{
			name:        "potentially unsafe value with script injection",
			key:         "app.script",
			value:       "<script>alert('xss')</script>",
			expectError: true,
			errorReason: "script injection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigSecurity(tt.key, tt.value)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorReason != "" {
					assert.Contains(t, err.Error(), tt.errorReason)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateConfigSecurity 配置安全验证辅助函数
func validateConfigSecurity(key, value string) error {
	// 检查路径遍历攻击
	if containsPathTraversal(key) || containsPathTraversal(value) {
		return errors.New("potential path traversal detected")
	}

	// 检查脚本注入
	if containsScriptInjection(value) {
		return errors.New("potential script injection detected")
	}

	return nil
}

func containsPathTraversal(input string) bool {
	return len(input) > 0 && (input[0] == '/' || input == ".." ||
		len(input) >= 3 && input[:3] == "../")
}

func containsScriptInjection(input string) bool {
	return len(input) > 7 && input[:8] == "<script>"
}
