package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

func TestNewRegistry_Success(t *testing.T) {
	config := &rest.Config{}
	err := NewRegistry(config)

	assert.NoError(t, err)
	assert.NotNil(t, GetGenericClient())
}

func TestNewRegistry_NilConfig(t *testing.T) {
	err := NewRegistry(nil)

	assert.Error(t, err)
	assert.Nil(t, GetGenericClient())
}

func TestGetGenericClientWithName_Success(t *testing.T) {
	config := &rest.Config{}
	_ = NewRegistry(config)
	clientset := GetGenericClientWithName("test")

	assert.NotNil(t, clientset)
	assert.NotNil(t, clientset.KubeClient)
}

func TestGetGenericClientWithName_NoConfig(t *testing.T) {

	cfg = nil
	clientset := GetGenericClientWithName("test")

	assert.Nil(t, clientset)
}
