package sentinel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	composev1alpha1 "github.com/upmio/compose-operator/api/v1alpha1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type stubClient struct {
	client.Client
	updated bool
}

func (s *stubClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	s.updated = true
	return nil
}

func newRedisReplication(name string) *composev1alpha1.RedisReplication {
	return &composev1alpha1.RedisReplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: composev1alpha1.RedisReplicationSpec{
			Source: &composev1alpha1.CommonNode{
				Host: "master", Port: 6379,
			},
			Replica: composev1alpha1.CommonNodes{
				&composev1alpha1.CommonNode{Host: "replica", Port: 6380},
			},
		},
	}
}

func newSentinelService(t *testing.T) (*service, *stubClient) {
	scheme := runtime.NewScheme()
	require.NoError(t, composev1alpha1.AddToScheme(scheme))

	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	stub := &stubClient{Client: baseClient}

	return &service{
		logger:        zap.NewNop().Sugar(),
		composeClient: stub,
	}, stub
}

func TestEnsureRedisReplicationInstanceNoChange(t *testing.T) {
	svc, stub := newSentinelService(t)
	instance := newRedisReplication("rr")

	err := svc.ensureRedisReplicationInstance(context.Background(), instance, "master", 6379)
	require.NoError(t, err)
	require.False(t, stub.updated)
}

func TestEnsureRedisReplicationInstanceSwapsReplica(t *testing.T) {
	svc, stub := newSentinelService(t)
	instance := newRedisReplication("rr")

	err := svc.ensureRedisReplicationInstance(context.Background(), instance, "replica", 6380)
	require.NoError(t, err)
	require.True(t, stub.updated)
	require.Equal(t, "replica", instance.Spec.Source.Host)
	require.Equal(t, 6380, instance.Spec.Source.Port)
	require.Equal(t, "master", instance.Spec.Replica[0].Host)
}

func TestEnsureRedisReplicationInstanceNotFound(t *testing.T) {
	svc, stub := newSentinelService(t)
	instance := newRedisReplication("rr")

	err := svc.ensureRedisReplicationInstance(context.Background(), instance, "missing", 1234)
	require.Error(t, err)
	require.False(t, stub.updated)
}
