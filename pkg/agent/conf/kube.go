package conf

import (
	"fmt"
	composev1alpha1 "github.com/upmio/compose-operator/api/v1alpha1"
	unitv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clientset     kubernetes.Interface
	composeClient client.Client
	unitClient    client.Client
)

func (k *Kube) GetClientSet() (kubernetes.Interface, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if clientset == nil {
		conn, err := k.getClientSet()
		if err != nil {
			return nil, err
		}
		clientset = conn
	}
	return clientset, nil
}

func (k *Kube) getClientSet() (kubernetes.Interface, error) {
	var (
		err    error
		config *rest.Config
	)

	// creates the config
	if len(k.KubeConfig) == 0 {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, fmt.Errorf("create in-cluster config fail, error: %v", err)
		}
	} else {
		if config, err = clientcmd.BuildConfigFromFlags("", k.KubeConfig); err != nil {
			return nil, fmt.Errorf("create out-of-cluster config fail, error: %v", err)
		}
	}

	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset fail, error: %v", err)
	}

	return clientset, nil
}

func (k *Kube) GetComposeClient() (client.Client, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if composeClient == nil {
		conn, err := k.getComposeClient()
		if err != nil {
			return nil, err
		}
		composeClient = conn
	}
	return composeClient, nil
}

func (k *Kube) getComposeClient() (client.Client, error) {
	var (
		err error
		cfg *rest.Config
	)

	if len(k.KubeConfig) == 0 {
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, fmt.Errorf("create in-cluster config fail, error: %v", err)
		}
	} else {
		if cfg, err = clientcmd.BuildConfigFromFlags("", k.KubeConfig); err != nil {
			return nil, fmt.Errorf("create out-of-cluster config fail, error: %v", err)
		}
	}

	scheme, err := composev1alpha1.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (k *Kube) GetUnitClient() (client.Client, error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if unitClient == nil {
		conn, err := k.getUnitClient()
		if err != nil {
			return nil, err
		}
		unitClient = conn
	}
	return unitClient, nil
}

func (k *Kube) getUnitClient() (client.Client, error) {
	var (
		err error
		cfg *rest.Config
	)

	if len(k.KubeConfig) == 0 {
		if cfg, err = rest.InClusterConfig(); err != nil {
			return nil, fmt.Errorf("create in-cluster config fail, error: %v", err)
		}
	} else {
		if cfg, err = clientcmd.BuildConfigFromFlags("", k.KubeConfig); err != nil {
			return nil, fmt.Errorf("create out-of-cluster config fail, error: %v", err)
		}
	}

	scheme, err := unitv1alpha2.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return c, nil
}
