package grpccall

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	upmv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

func TestNewGrpcClient(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		port        string
		expectError bool
	}{
		{
			name:        "valid host and port",
			host:        "localhost",
			port:        "8080",
			expectError: false,
		},
		{
			name:        "empty host",
			host:        "",
			port:        "8080",
			expectError: false, // grpc client allows empty host
		},
		{
			name:        "invalid port",
			host:        "localhost",
			port:        "invalid",
			expectError: false, // grpc client will handle this at connection time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newGrpcClient(tt.host, tt.port)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					assert.NoError(t, client.Close())
				}
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	client, err := newGrpcClient("localhost", "8080")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test that Close() doesn't panic
	err = client.Close()
	assert.NoError(t, err)

	// Test that calling Close() multiple times doesn't panic
	err = client.Close()
	if err != nil {
		assert.Error(t, err)
	}
	// The error might occur on second close, but it shouldn't panic
}

func TestClient_ServiceClients(t *testing.T) {
	client, err := newGrpcClient("localhost", "8080")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	defer func() {
		_ = client.Close()
	}()

	// Test that all service client getters work
	mysqlClient := client.Mysql()
	assert.NotNil(t, mysqlClient)

	postgresqlClient := client.Postgresql()
	assert.NotNil(t, postgresqlClient)

	proxysqlClient := client.Proxysql()
	assert.NotNil(t, proxysqlClient)
}

func TestGatherUnitAgentEndpoint_Success(t *testing.T) {
	mockClient := &MockClient{}
	logger := zap.New().WithName("test")

	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TargetUnit: "test-unit-0",
		},
	}

	unit := &upmv1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-unit-0",
			Namespace: "default",
		},
		Spec: upmv1alpha2.UnitSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "mysql",
							Ports: []corev1.ContainerPort{
								{
									Name:          "mysql",
									ContainerPort: 3306,
								},
							},
						},
						{
							Name: agentName,
							Ports: []corev1.ContainerPort{
								{
									Name:          agentName,
									ContainerPort: 9090,
								},
							},
						},
					},
				},
			},
		},
	}

	mockClient.On("Get", mock.Anything,
		types.NamespacedName{Name: "test-unit-0", Namespace: "default"},
		mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Run(func(args mock.Arguments) {
			obj := args.Get(2).(*upmv1alpha2.Unit)
			*obj = *unit
		}).Return(nil)

	host, port, err := gatherUnitAgentEndpoint(context.Background(), mockClient, instance, logger)

	assert.NoError(t, err)
	assert.Contains(t, host, "test-unit-0")
	assert.Contains(t, host, "svc")
	assert.Equal(t, "9090", port)
	mockClient.AssertExpectations(t)
}

func TestGatherUnitAgentEndpoint_UnitNotFound(t *testing.T) {
	mockClient := &MockClient{}
	logger := zap.New().WithName("test")

	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TargetUnit: "non-existent-unit",
		},
	}

	mockClient.On("Get", mock.Anything,
		types.NamespacedName{Name: "non-existent-unit", Namespace: "default"},
		mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Return(errors.NewNotFound(schema.GroupResource{Group: "upm.syntropycloud.io", Resource: "units"}, "non-existent-unit"))

	host, port, err := gatherUnitAgentEndpoint(context.Background(), mockClient, instance, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch unit")
	assert.Empty(t, host)
	assert.Empty(t, port)
	mockClient.AssertExpectations(t)
}

func TestGatherUnitAgentEndpoint_AgentContainerNotFound(t *testing.T) {
	mockClient := &MockClient{}
	logger := zap.New().WithName("test")

	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TargetUnit: "test-unit-0",
		},
	}

	unit := &upmv1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-unit-0",
			Namespace: "default",
		},
		Spec: upmv1alpha2.UnitSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "mysql", // Only mysql container, no unit-agent
							Ports: []corev1.ContainerPort{
								{
									Name:          "mysql",
									ContainerPort: 3306,
								},
							},
						},
					},
				},
			},
		},
	}

	mockClient.On("Get", mock.Anything,
		types.NamespacedName{Name: "test-unit-0", Namespace: "default"},
		mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Run(func(args mock.Arguments) {
			obj := args.Get(2).(*upmv1alpha2.Unit)
			*obj = *unit
		}).Return(nil)

	host, port, err := gatherUnitAgentEndpoint(context.Background(), mockClient, instance, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("container %q not found", agentName))
	assert.Empty(t, host)
	assert.Empty(t, port)
	mockClient.AssertExpectations(t)
}

func TestGatherUnitAgentEndpoint_AgentPortNotFound(t *testing.T) {
	mockClient := &MockClient{}
	logger := zap.New().WithName("test")

	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TargetUnit: "test-unit-0",
		},
	}

	unit := &upmv1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-unit-0",
			Namespace: "default",
		},
		Spec: upmv1alpha2.UnitSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: agentName,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http", // Wrong port name
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	mockClient.On("Get", mock.Anything,
		types.NamespacedName{Name: "test-unit-0", Namespace: "default"},
		mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Run(func(args mock.Arguments) {
			obj := args.Get(2).(*upmv1alpha2.Unit)
			*obj = *unit
		}).Return(nil)

	host, port, err := gatherUnitAgentEndpoint(context.Background(), mockClient, instance, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("port %s not found", agentName))
	assert.Empty(t, host)
	assert.Empty(t, port)
	mockClient.AssertExpectations(t)
}

func TestGatherUnitAgentEndpoint_GetError(t *testing.T) {
	mockClient := &MockClient{}
	logger := zap.New().WithName("test")

	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "default",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TargetUnit: "test-unit-0",
		},
	}

	expectedError := fmt.Errorf("some client error")
	mockClient.On("Get", mock.Anything,
		types.NamespacedName{Name: "test-unit-0", Namespace: "default"},
		mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Return(expectedError)

	host, port, err := gatherUnitAgentEndpoint(context.Background(), mockClient, instance, logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch unit")
	assert.Empty(t, host)
	assert.Empty(t, port)
	mockClient.AssertExpectations(t)
}

func TestGatherUnitAgentEndpoint_HostFormatting(t *testing.T) {
	mockClient := &MockClient{}
	logger := zap.New().WithName("test")

	instance := &upmv1alpha1.GrpcCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpccall",
			Namespace: "test-namespace",
		},
		Spec: upmv1alpha1.GrpcCallSpec{
			TargetUnit: "mysql-unit-0",
		},
	}

	unit := &upmv1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-unit-0",
			Namespace: "test-namespace",
		},
		Spec: upmv1alpha2.UnitSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: agentName,
							Ports: []corev1.ContainerPort{
								{
									Name:          agentName,
									ContainerPort: 9090,
								},
							},
						},
					},
				},
			},
		},
	}

	mockClient.On("Get", mock.Anything,
		types.NamespacedName{Name: "mysql-unit-0", Namespace: "test-namespace"},
		mock.AnythingOfType("*v1alpha2.Unit"), mock.Anything).
		Run(func(args mock.Arguments) {
			obj := args.Get(2).(*upmv1alpha2.Unit)
			*obj = *unit
		}).Return(nil)

	host, port, err := gatherUnitAgentEndpoint(context.Background(), mockClient, instance, logger)

	assert.NoError(t, err)
	assert.Contains(t, host, "mysql-unit-0")
	assert.Contains(t, host, "test-namespace")
	assert.Contains(t, host, ".svc")
	assert.Equal(t, "9090", port)
	mockClient.AssertExpectations(t)
}
