package config

import (
	"context"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// MockKubeConfig
type MockKubeConfig struct {
	mock.Mock
}

func (m *MockKubeConfig) GetClientSet() (kubernetes.Interface, error) {
	args := m.Called()
	return args.Get(0).(kubernetes.Interface), args.Error(1)
}

// MockApp
type MockApp struct {
	mock.Mock
}

func (m *MockApp) GetGrpcApp(name string) interface{} {
	args := m.Called(name)
	return args.Get(0)
}

// MockClientSet
type MockClientSet struct {
	mock.Mock
}

func (m *MockClientSet) CoreV1() kubernetes.Interface {
	return m.Called().Get(0).(kubernetes.Interface)
}

// MockCoreV1
type MockCoreV1 struct {
	mock.Mock
}

func (m *MockCoreV1) ConfigMaps(namespace string) kubernetes.Interface {
	args := m.Called(namespace)
	return args.Get(0).(kubernetes.Interface)
}

// MockConfigMapInterface
type MockConfigMapInterface struct {
	mock.Mock
}

func (m *MockConfigMapInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ConfigMap, error) {
	args := m.Called(ctx, name, opts)
	return args.Get(0).(*v1.ConfigMap), args.Error(1)
}

//func TestSyncConfig_GetConfigMapError(t *testing.T) {
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	mockConfigMap.On("Get", mock.Anything, "test-config", mock.Anything).Return(nil, errors.New("not found"))
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	logger, _ := zap.NewDevelopment()
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//		confdConfig: &confd.Config{
//			TemplateConfig: confd.TemplateConfig{},
//			BackendsConfig: confd.BackendsConfig{},
//		},
//	}
//
//	req := &SyncConfigRequest{
//		Namespace:     "default",
//		ConfigmapName: "test-config",
//	}
//
//	resp, err := s.SyncConfig(context.Background(), req)
//
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "not found")
//	assert.Contains(t, resp.Message, "can't found configMap")
//}

//func TestSyncConfig_MissingTemplateKey(t *testing.T) {
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	configMapData := map[string]string{
//		"pathKey":  "/path/to/config",
//		"valueKey": "some-value",
//	}
//	configMap := &v1.ConfigMap{Data: configMapData}
//
//	mockConfigMap.On("Get", mock.Anything, "test-config", mock.Anything).Return(configMap, nil)
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	logger, _ := zap.NewDevelopment()
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//		confdConfig: &confd.Config{
//			TemplateConfig: confd.TemplateConfig{},
//			BackendsConfig: confd.BackendsConfig{},
//		},
//	}
//
//	req := &SyncConfigRequest{
//		Namespace:     "default",
//		ConfigmapName: "test-config",
//	}
//
//	resp, err := s.SyncConfig(context.Background(), req)
//
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "does't has key templateKey")
//	assert.Contains(t, resp.Message, "configMap does't has key templateKey")
//}
//
//func TestSyncConfig_WriteTemplateFileError(t *testing.T) {
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	configMapData := map[string]string{
//		"templateKey": "some-template-content",
//		"pathKey":     "/path/to/config",
//		"valueKey":    "some-value",
//	}
//	configMap := &v1.ConfigMap{Data: configMapData}
//
//	mockConfigMap.On("Get", mock.Anything, "test-config", mock.Anything).Return(configMap, nil)
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	tmplFile := "/root/template.tmpl"
//
//	logger, _ := zap.NewDevelopment()
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//		confdConfig: &confd.Config{
//			TemplateConfig: confd.TemplateConfig{
//				TemplateFile: tmplFile,
//			},
//			BackendsConfig: confd.BackendsConfig{},
//		},
//	}
//
//	req := &SyncConfigRequest{
//		Namespace:     "default",
//		ConfigmapName: "test-config",
//	}
//
//	resp, err := s.SyncConfig(context.Background(), req)
//
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "write template file")
//	assert.Contains(t, resp.Message, "write template file")
//}
//
//func TestSyncConfig_MissingPathKey(t *testing.T) {
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	configMapData := map[string]string{
//		"templateKey": "some-template-content",
//		"valueKey":    "some-value",
//	}
//	configMap := &v1.ConfigMap{Data: configMapData}
//
//	mockConfigMap.On("Get", mock.Anything, "test-config", mock.Anything).Return(configMap, nil)
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	logger, _ := zap.NewDevelopment()
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//		confdConfig: &confd.Config{
//			TemplateConfig: confd.TemplateConfig{},
//			BackendsConfig: confd.BackendsConfig{},
//		},
//	}
//
//	req := &SyncConfigRequest{
//		Namespace:     "default",
//		ConfigmapName: "test-config",
//	}
//
//	resp, err := s.SyncConfig(context.Background(), req)
//
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "does't has key pathKey")
//	assert.Contains(t, resp.Message, "configMap does't has key pathKey")
//}
//
//func TestSyncConfig_ExtendConfigMapError(t *testing.T) {
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	configMapData := map[string]string{
//		"templateKey": "some-template-content",
//		"pathKey":     "/path/to/config",
//		"valueKey":    "some-value",
//	}
//	configMap := &v1.ConfigMap{Data: configMapData}
//
//	mockConfigMap.On("Get", mock.Anything, "test-config", mock.Anything).Return(configMap, nil)
//	mockConfigMap.On("Get", mock.Anything, "extend-config", mock.Anything).Return(nil, errors.New("not found"))
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	logger, _ := zap.NewDevelopment()
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//		confdConfig: &confd.Config{
//			TemplateConfig: confd.TemplateConfig{},
//			BackendsConfig: confd.BackendsConfig{},
//			//TemplateFile:   tmplFile,
//		},
//	}
//
//	req := &SyncConfigRequest{
//		Namespace:             "default",
//		ConfigmapName:         "test-config",
//		ExtendValueConfigmaps: []string{"extend-config"},
//	}
//
//	resp, err := s.SyncConfig(context.Background(), req)
//
//	assert.Error(t, err)
//	assert.Contains(t, err.Error(), "can't found configMap")
//	assert.Contains(t, resp.Message, "can't found configMap")
//}
//
//func TestSyncConfig_Success(t *testing.T) {
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	configMapData := map[string]string{
//		"templateKey": "some-template-content",
//		"pathKey":     "/path/to/config",
//		"valueKey":    "some-value",
//	}
//	configMap := &v1.ConfigMap{Data: configMapData}
//
//	mockConfigMap.On("Get", mock.Anything, "test-config", mock.Anything).Return(configMap, nil)
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	logger, _ := zap.NewDevelopment()
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//		confdConfig: &confd.Config{
//			TemplateConfig: confd.TemplateConfig{},
//			BackendsConfig: confd.BackendsConfig{},
//		},
//	}
//
//	req := &SyncConfigRequest{
//		Namespace:     "default",
//		ConfigmapName: "test-config",
//	}
//
//	resp, err := s.SyncConfig(context.Background(), req)
//
//	assert.NoError(t, err)
//	assert.Equal(t, "generate config test-config success.", resp.Message)
//	assert.Contains(t, s.confdConfig.BackendsConfig.Contents, "some-value")
//}

//func TestRewriteConfig(t *testing.T) {
//	logger, _ := zap.NewDevelopment()
//	mockClientSet := new(MockClientSet)
//	mockCoreV1 := new(MockCoreV1)
//	mockConfigMap := new(MockConfigMapInterface)
//
//	mockCoreV1.On("ConfigMaps", "default").Return(mockConfigMap)
//	mockClientSet.On("CoreV1").Return(mockCoreV1)
//
//	s := &service{
//		//clientSet: mockClientSet,
//		logger: logger.Sugar(),
//	}
//
//	tests := []struct {
//		name          string
//		req           *RewriteConfigRequest
//		getResponse   *v1.ConfigMap
//		getError      error
//		updateError   error
//		expectedError bool
//		expectedMsg   string
//	}{
//		{
//			name: "GetConfigMapError",
//			req: &RewriteConfigRequest{
//				Namespace:     "default",
//				ConfigmapName: "test-config",
//				Key:           "some-key",
//				Value:         "some-value",
//			},
//			getError:      errors.New("not found"),
//			expectedError: true,
//			expectedMsg:   "can't found configMap",
//		},
//		{
//			name: "KeyValueAlreadyExists",
//			req: &RewriteConfigRequest{
//				Namespace:     "default",
//				ConfigmapName: "test-config",
//				Key:           "some-key",
//				Value:         "some-value",
//			},
//			getResponse:   &v1.ConfigMap{Data: map[string]string{"some-key": "some-value"}},
//			expectedError: false,
//			expectedMsg:   "no need to update",
//		},
//		{
//			name: "KeyExistsValueDiffers",
//			req: &RewriteConfigRequest{
//				Namespace:     "default",
//				ConfigmapName: "test-config",
//				Key:           "some-key",
//				Value:         "new-value",
//			},
//			getResponse:   &v1.ConfigMap{Data: map[string]string{"some-key": "old-value"}},
//			expectedError: false,
//			expectedMsg:   "generate config test-config success",
//		},
//		{
//			name: "KeyDoesNotExist",
//			req: &RewriteConfigRequest{
//				Namespace:     "default",
//				ConfigmapName: "test-config",
//				Key:           "new-key",
//				Value:         "new-value",
//			},
//			getResponse:   &v1.ConfigMap{Data: map[string]string{}},
//			expectedError: false,
//			expectedMsg:   "generate config test-config success",
//		},
//		{
//			name: "UpdateConfigMapError",
//			req: &RewriteConfigRequest{
//				Namespace:     "default",
//				ConfigmapName: "test-config",
//				Key:           "some-key",
//				Value:         "new-value",
//			},
//			getResponse:   &v1.ConfigMap{Data: map[string]string{"some-key": "old-value"}},
//			updateError:   errors.New("update failed"),
//			expectedError: true,
//			expectedMsg:   "update ConfigMap failed",
//		},
//		{
//			name: "Success",
//			req: &RewriteConfigRequest{
//				Namespace:     "default",
//				ConfigmapName: "test-config",
//				Key:           "some-key",
//				Value:         "new-value",
//			},
//			getResponse:   &v1.ConfigMap{Data: map[string]string{"some-key": "old-value"}},
//			expectedError: false,
//			expectedMsg:   "generate config test-config success",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			mockConfigMap.On("Get", mock.Anything, tt.req.ConfigmapName, mock.Anything).Return(tt.getResponse, tt.getError)
//			if tt.getResponse != nil && tt.updateError == nil {
//				updatedConfigMap := tt.getResponse.DeepCopy()
//				updatedConfigMap.Data[tt.req.Key] = tt.req.Value
//				mockConfigMap.On("Update", mock.Anything, updatedConfigMap, mock.Anything).Return(updatedConfigMap, tt.updateError)
//			} else if tt.updateError != nil {
//				mockConfigMap.On("Update", mock.Anything, tt.getResponse, mock.Anything).Return(nil, tt.updateError)
//			}
//
//			resp, err := s.RewriteConfig(context.Background(), tt.req)
//
//			if tt.expectedError {
//				assert.Error(t, err)
//				assert.Contains(t, resp.Message, tt.expectedMsg)
//			} else {
//				assert.NoError(t, err)
//				assert.Contains(t, resp.Message, tt.expectedMsg)
//			}
//		})
//	}
//}
