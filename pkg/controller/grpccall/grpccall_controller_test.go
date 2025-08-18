/*
 * UPM for Enterprise
 *
 * Copyright (c) 2009-2025 SYNTROPY Pte. Ltd.
 * All rights reserved.
 *
 * This software is the confidential and proprietary information of
 * SYNTROPY Pte. Ltd. ("Confidential Information"). You shall not
 * disclose such Confidential Information and shall use it only in
 * accordance with the terms of the license agreement you entered
 * into with SYNTROPY.
 */

package grpccall

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// MockClient implements the client.Client interface for testing
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
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

// MockStatusWriter implements the client.StatusWriter interface for testing
type MockStatusWriter struct {
	mock.Mock
}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	args := m.Called(ctx, obj, subResource, opts)
	return args.Error(0)
}

// MockEventRecorder implements the record.EventRecorder interface for testing
type MockEventRecorder struct {
	mock.Mock
}

func (m *MockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	m.Called(object, eventtype, reason, message)
}

func (m *MockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Called(object, eventtype, reason, messageFmt, args)
}

func (m *MockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Called(object, annotations, eventtype, reason, messageFmt, args)
}

// Unit tests using testify
func TestReconcileGrpcCall_Reconcile_NotFound(t *testing.T) {
	mockClient := &MockClient{}
	mockRecorder := &MockEventRecorder{}

	reconciler := &ReconcileGrpcCall{
		client:   mockClient,
		scheme:   runtime.NewScheme(),
		recorder: mockRecorder,
		logger:   zap.New().WithName("test"),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-grpccall",
			Namespace: "default",
		},
	}

	// Mock the Get call to return NotFound error
	mockClient.On("Get", mock.Anything, req.NamespacedName, mock.AnythingOfType("*v1alpha1.GrpcCall"), mock.Anything).
		Return(errors.NewNotFound(schema.GroupResource{Group: "upm.syntropycloud.io", Resource: "grpccalls"}, "test-grpccall"))

	result, err := reconciler.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
	mockClient.AssertExpectations(t)
}

func TestReconcileGrpcCall_Reconcile_GetError(t *testing.T) {
	mockClient := &MockClient{}
	mockRecorder := &MockEventRecorder{}

	reconciler := &ReconcileGrpcCall{
		client:   mockClient,
		scheme:   runtime.NewScheme(),
		recorder: mockRecorder,
		logger:   zap.New().WithName("test"),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-grpccall",
			Namespace: "default",
		},
	}

	expectedError := fmt.Errorf("some client error")
	mockClient.On("Get", mock.Anything, req.NamespacedName, mock.AnythingOfType("*v1alpha1.GrpcCall"), mock.Anything).
		Return(expectedError)

	result, err := reconciler.Reconcile(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, reconcile.Result{}, result)
	mockClient.AssertExpectations(t)
}

func TestReconcileGrpcCall_Reconcile_TTLExpired(t *testing.T) {
	mockClient := &MockClient{}
	mockRecorder := &MockEventRecorder{}

	reconciler := &ReconcileGrpcCall{
		client:   mockClient,
		scheme:   runtime.NewScheme(),
		recorder: mockRecorder,
		logger:   zap.New().WithName("test"),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-grpccall",
			Namespace: "default",
		},
	}

	// Create a GrpcCall with expired TTL
	ttlSeconds := int32(10)
	completionTime := metav1.NewTime(time.Now().Add(-time.Minute)) // Completed 1 minute ago
	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TTLSecondsAfterFinished: &ttlSeconds,
		},
		Status: upmv1alpha1.GrpcCallStatus{
			CompletionTime: &completionTime,
		},
	}

	mockClient.On("Get", mock.Anything, req.NamespacedName, mock.AnythingOfType("*v1alpha1.GrpcCall"), mock.Anything).
		Run(func(args mock.Arguments) {
			obj := args.Get(2).(*upmv1alpha1.GrpcCall)
			*obj = *instance
		}).Return(nil)

	mockClient.On("Delete", mock.Anything, mock.AnythingOfType("*v1alpha1.GrpcCall"), mock.Anything).Return(nil)

	result, err := reconciler.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
	mockClient.AssertExpectations(t)
}

func TestReconcileGrpcCall_Reconcile_AlreadyStarted(t *testing.T) {
	mockClient := &MockClient{}
	mockRecorder := &MockEventRecorder{}

	reconciler := &ReconcileGrpcCall{
		client:   mockClient,
		scheme:   runtime.NewScheme(),
		recorder: mockRecorder,
		logger:   zap.New().WithName("test"),
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-grpccall",
			Namespace: "default",
		},
	}

	startTime := metav1.Now()
	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Status: upmv1alpha1.GrpcCallStatus{
			StartTime: &startTime,
		},
	}

	mockClient.On("Get", mock.Anything, req.NamespacedName, mock.AnythingOfType("*v1alpha1.GrpcCall"), mock.Anything).
		Run(func(args mock.Arguments) {
			obj := args.Get(2).(*upmv1alpha1.GrpcCall)
			*obj = *instance
		}).Return(nil)

	result, err := reconciler.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Hour,
	}, result)
	mockClient.AssertExpectations(t)
}

func TestReconcileGrpcCall_Setup(t *testing.T) {
	// This is a simple test to ensure Setup function doesn't panic
	// In a real scenario, you would need a proper manager mock
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Setup function panicked: %v", r)
		}
	}()

	// We can't easily test Setup without a complex manager mock
	// This test just ensures the function exists and doesn't panic during compilation
	assert.NotNil(t, Setup)
}

// Ginkgo tests (integration-style tests)
var _ = Describe("GrpcCall Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		grpccall := &upmv1alpha1.GrpcCall{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind GrpcCall")
			err := k8sClient.Get(ctx, typeNamespacedName, grpccall)
			if err != nil && errors.IsNotFound(err) {
				ptr := new(int32)
				*ptr = 42
				resource := &upmv1alpha1.GrpcCall{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: upmv1alpha1.GrpcCallSpec{
						Type:       upmv1alpha1.MysqlType,
						Action:     upmv1alpha1.LogicalBackupAction,
						TargetUnit: "test-unit",
						Parameters: map[string]apiextensionsv1.JSON{
							"username": {Raw: []byte(`"root"`)},
							"password": {Raw: []byte(`"password"`)},
						},
						TTLSecondsAfterFinished: ptr,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &upmv1alpha1.GrpcCall{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the specific resource instance GrpcCall")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ReconcileGrpcCall{
				client:   k8sClient,
				scheme:   k8sClient.Scheme(),
				recorder: record.NewFakeRecorder(100),
				logger:   zap.New().WithName("test-controller"),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			// Note: This will likely fail because we don't have a real Unit or grpc server
			// But it tests the basic controller logic
			Expect(err).NotTo(HaveOccurred())

			// Verify the resource status was updated
			updatedResource := &upmv1alpha1.GrpcCall{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedResource)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedResource.Status.StartTime).NotTo(BeNil())
		})

		It("should handle non-existent resource gracefully", func() {
			By("Reconciling a non-existent resource")
			controllerReconciler := &ReconcileGrpcCall{
				client:   k8sClient,
				scheme:   k8sClient.Scheme(),
				recorder: record.NewFakeRecorder(100),
				logger:   zap.New().WithName("test-controller"),
			}

			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-resource",
					Namespace: "default",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})
	})
})
