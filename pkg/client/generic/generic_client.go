package client

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GenericClientset defines a generic client
type GenericClientset struct {
	KubeClient kubeclientset.Interface
}

// newForConfig creates a new Clientset for the given config.
func newForConfig(c *rest.Config) (*GenericClientset, error) {
	if c == nil {
		return nil, fmt.Errorf("config is need")
	}

	cWithProtobuf := rest.CopyConfig(c)
	cWithProtobuf.ContentType = runtime.ContentTypeProtobuf

	kubeClient, err := kubeclientset.NewForConfig(cWithProtobuf)
	if err != nil {
		return nil, err
	}

	return &GenericClientset{
		KubeClient: kubeClient,
	}, nil
}

// newForConfig creates a new Clientset for the given config.
func newForConfigOrDie(c *rest.Config) *GenericClientset {
	gc, err := newForConfig(c)
	if err != nil {
		panic(err)
	}
	return gc
}
