package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

// MockClient
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	if args.Get(0) == nil {
		return schema.GroupVersionKind{}, args.Error(1)
	}
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(client.SubResourceClient)
}

// MockEventRecorder 增强的事件记录器模拟
type MockEventRecorder struct {
	mock.Mock
}

func (m *MockEventRecorder) Event(object runtime.Object, eventType, reason, message string) {
	m.Called(object, eventType, reason, message)
}

func (m *MockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Called(object, eventtype, reason, messageFmt, args)
}

func (m *MockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Called(object, annotations, eventtype, reason, messageFmt, args)
}

// MockUnitClient 模拟 Unit 客户端
type MockUnitClient struct {
	mock.Mock
}

func (m *MockUnitClient) Get(ctx context.Context, name, namespace string) (*upmiov1alpha2.Unit, error) {
	args := m.Called(ctx, name, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*upmiov1alpha2.Unit), args.Error(1)
}

type MockConf struct {
	mock.Mock
}

func (m *MockConf) GetClientSet() (*fake.Clientset, error) {
	args := m.Called()
	return args.Get(0).(*fake.Clientset), args.Error(1)
}

// TestEventServiceImplementation 测试事件服务基本实现
func TestEventServiceImplementation(t *testing.T) {
	mockRecorder := &MockEventRecorder{}
	mockClient := &MockClient{}

	// 直接创建一个符合实际结构的 service
	service := &service{
		unitClient:    mockClient,
		EventRecorder: mockRecorder,
		logger:        zaptest.NewLogger(t).Sugar(),
	}

	t.Run("service interface implementation", func(t *testing.T) {
		assert.NotNil(t, service.logger)
		assert.NotNil(t, service.EventRecorder)
		assert.NotNil(t, service.unitClient)
	})
}

// TestSendWarningEventToUnit 测试发送警告事件
func TestSendWarningEventToUnit(t *testing.T) {
	tests := []struct {
		name          string
		unitName      string
		namespace     string
		reason        string
		message       string
		setupMocks    func(*MockClient, *MockEventRecorder)
		expectError   bool
		expectedError string
	}{
		{
			name:      "successful warning event",
			unitName:  "mysql-0",
			namespace: "default",
			reason:    "BackupFailed",
			message:   "Failed to backup database: connection timeout",
			setupMocks: func(client *MockClient, recorder *MockEventRecorder) {
				unit := &upmiov1alpha2.Unit{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mysql-0",
						Namespace: "default",
					},
				}
				// Mock the Get method with correct signature and use MatchedBy to handle the object filling
				client.On("Get", mock.Anything, types.NamespacedName{Name: "mysql-0", Namespace: "default"}, mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
					Run(func(args mock.Arguments) {
						// Fill the Unit object that was passed in
						unitPtr := args.Get(2).(*upmiov1alpha2.Unit)
						*unitPtr = *unit
					}).Return(nil)
				recorder.On("Event", mock.AnythingOfType("*v1alpha2.Unit"), "Warning", "BackupFailed", "Failed to backup database: connection timeout").Return()
			},
			expectError: false,
		},
		{
			name:      "unit not found",
			unitName:  "non-existent-unit",
			namespace: "default",
			reason:    "TestReason",
			message:   "Test message",
			setupMocks: func(client *MockClient, recorder *MockEventRecorder) {
				client.On("Get", mock.Anything, types.NamespacedName{Name: "non-existent-unit", Namespace: "default"}, mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).Return(errors.New("unit not found"))
			},
			expectError:   true,
			expectedError: "failed to fetch unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecorder := &MockEventRecorder{}
			mockClient := &MockClient{}

			service := &service{
				EventRecorder: mockRecorder,
				unitClient:    mockClient,
				logger:        zaptest.NewLogger(t).Sugar(),
			}

			tt.setupMocks(mockClient, mockRecorder)

			err := service.SendWarningEventToUnit(tt.unitName, tt.namespace, tt.reason, tt.message)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRecorder.AssertExpectations(t)
			mockClient.AssertExpectations(t)
		})
	}
}

// TestSendNormalEventToUnit 测试发送正常事件
func TestSendNormalEventToUnit(t *testing.T) {
	tests := []struct {
		name          string
		unitName      string
		namespace     string
		reason        string
		message       string
		setupMocks    func(*MockClient, *MockEventRecorder)
		expectError   bool
		expectedError string
	}{
		{
			name:      "successful normal event",
			unitName:  "postgres-0",
			namespace: "default",
			reason:    "BackupCompleted",
			message:   "Database backup completed successfully",
			setupMocks: func(client *MockClient, recorder *MockEventRecorder) {
				unit := &upmiov1alpha2.Unit{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "postgres-0",
						Namespace: "default",
					},
				}
				client.On("Get", mock.Anything, types.NamespacedName{Name: "postgres-0", Namespace: "default"}, mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
					Run(func(args mock.Arguments) {
						unitPtr := args.Get(2).(*upmiov1alpha2.Unit)
						*unitPtr = *unit
					}).Return(nil)
				recorder.On("Event", mock.AnythingOfType("*v1alpha2.Unit"), "Normal", "BackupCompleted", "Database backup completed successfully").Return()
			},
			expectError: false,
		},
		{
			name:      "context timeout",
			unitName:  "slow-unit",
			namespace: "default",
			reason:    "TestReason",
			message:   "Test message",
			setupMocks: func(client *MockClient, recorder *MockEventRecorder) {
				client.On("Get", mock.Anything, types.NamespacedName{Name: "slow-unit", Namespace: "default"}, mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).Return(context.DeadlineExceeded)
			},
			expectError:   true,
			expectedError: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRecorder := &MockEventRecorder{}
			mockClient := &MockClient{}

			service := &service{
				EventRecorder: mockRecorder,
				unitClient:    mockClient,
				logger:        zaptest.NewLogger(t).Sugar(),
			}

			tt.setupMocks(mockClient, mockRecorder)

			err := service.SendNormalEventToUnit(tt.unitName, tt.namespace, tt.reason, tt.message)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRecorder.AssertExpectations(t)
			mockClient.AssertExpectations(t)
		})
	}
}

// TestEventValidation 测试事件参数验证
func TestEventValidation(t *testing.T) {
	tests := []struct {
		name        string
		unitName    string
		namespace   string
		reason      string
		message     string
		expectError bool
		errorField  string
	}{
		{
			name:        "valid event parameters",
			unitName:    "mysql-0",
			namespace:   "default",
			reason:      "BackupCompleted",
			message:     "Backup completed successfully",
			expectError: false,
		},
		{
			name:        "empty unit name",
			unitName:    "",
			namespace:   "default",
			reason:      "TestReason",
			message:     "Test message",
			expectError: true,
			errorField:  "unit name",
		},
		{
			name:        "empty namespace",
			unitName:    "mysql-0",
			namespace:   "",
			reason:      "TestReason",
			message:     "Test message",
			expectError: true,
			errorField:  "namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEventParameters(tt.unitName, tt.namespace, tt.reason, tt.message)

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

// TestConcurrentEventSending 测试并发事件发送
func TestConcurrentEventSending(t *testing.T) {
	mockRecorder := &MockEventRecorder{}
	mockClient := &MockClient{}

	service := &service{
		EventRecorder: mockRecorder,
		unitClient:    mockClient,
		logger:        zaptest.NewLogger(t).Sugar(),
	}

	unit := &upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-unit",
			Namespace: "default",
		},
	}

	mockClient.On("Get", mock.Anything, types.NamespacedName{Name: "test-unit", Namespace: "default"}, mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Run(func(args mock.Arguments) {
			unitPtr := args.Get(2).(*upmiov1alpha2.Unit)
			*unitPtr = *unit
		}).Return(nil)
	mockRecorder.On("Event", mock.AnythingOfType("*v1alpha2.Unit"), "Normal", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return()

	concurrency := 5
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			err := service.SendNormalEventToUnit("test-unit", "default", "ConcurrentTest", "Concurrent event message")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			// 成功完成
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent event sending")
		}
	}

	mockClient.AssertNumberOfCalls(t, "Get", concurrency)
	mockRecorder.AssertNumberOfCalls(t, "Event", concurrency)
}

// 辅助验证函数
func validateEventParameters(unitName, namespace, reason, message string) error {
	if unitName == "" {
		return errors.New("unit name cannot be empty")
	}
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	if reason == "" {
		return errors.New("reason cannot be empty")
	}
	if message == "" {
		return errors.New("message cannot be empty")
	}
	return nil
}

// BenchmarkSendNormalEventToUnit 性能基准测试
func BenchmarkSendNormalEventToUnit(b *testing.B) {
	mockRecorder := &MockEventRecorder{}
	mockClient := &MockClient{}

	service := &service{
		EventRecorder: mockRecorder,
		unitClient:    mockClient,
		logger:        zaptest.NewLogger(b).Sugar(),
	}

	unit := &upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-unit",
			Namespace: "default",
		},
	}

	mockClient.On("Get", mock.Anything, types.NamespacedName{Name: "benchmark-unit", Namespace: "default"}, mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Run(func(args mock.Arguments) {
			unitPtr := args.Get(2).(*upmiov1alpha2.Unit)
			*unitPtr = *unit
		}).Return(nil)
	mockRecorder.On("Event", mock.AnythingOfType("*v1alpha2.Unit"), "Normal", "BenchmarkTest", "Benchmark test message").Return()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.SendNormalEventToUnit("benchmark-unit", "default", "BenchmarkTest", "Benchmark test message")
	}
}
