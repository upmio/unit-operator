package client

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"

	"testing"
)

func TestNewForConfig_Success(t *testing.T) {
	config := &rest.Config{}
	clientset, err := newForConfig(config)

	assert.NoError(t, err)
	assert.NotNil(t, clientset)
	assert.NotNil(t, clientset.KubeClient)
}

func TestNewForConfig_NilConfig(t *testing.T) {
	clientset, err := newForConfig(nil)

	assert.Error(t, err)
	assert.Nil(t, clientset)
}

func TestNewForConfigOrDie_Success(t *testing.T) {
	config := &rest.Config{}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code panicked")
		}
	}()

	clientset := newForConfigOrDie(config)

	assert.NotNil(t, clientset)
	assert.NotNil(t, clientset.KubeClient)
}

func TestNewForConfigOrDie_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	newForConfigOrDie(nil)
}
