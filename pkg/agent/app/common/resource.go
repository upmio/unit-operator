package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/upmio/compose-operator/api/v1alpha1"
	unitv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type ResourceFactory struct {
	kubeClient     kubernetes.Interface
	ownerReference []metav1.OwnerReference
	logger         *zap.SugaredLogger

	namespace     string
	name          string
	configMapName string
}

func NewResourceFactory(ctx context.Context, logger *zap.SugaredLogger, name, namespace string) (*ResourceFactory, error) {
	kubeClient, err := conf.GetConf().GetClientSet()
	if err != nil {
		return nil, err
	}

	ownerReference, err := fetchUnitOwnerReferences(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	return &ResourceFactory{
		ownerReference: ownerReference,
		kubeClient:     kubeClient,
		logger:         logger,

		namespace:     namespace,
		name:          name,
		configMapName: fmt.Sprintf("%s-config-backup", name),
	}, nil
}

func fetchUnitOwnerReferences(ctx context.Context, namespace, name string) ([]metav1.OwnerReference, error) {
	unitClient, err := conf.GetConf().GetUnitClient()
	if err != nil {
		return nil, err
	}

	instance := &unitv1alpha2.Unit{}
	if err := unitClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, instance); err != nil {
		return nil, err
	}

	return []metav1.OwnerReference{
		*metav1.NewControllerRef(instance, schema.GroupVersionKind{
			Group:   v1alpha1.GroupVersion.Group,
			Version: v1alpha1.GroupVersion.Version,
			Kind:    "Unit",
		}),
	}, nil
}

func (f *ResourceFactory) WriteFileToConfigMap(ctx context.Context, confDir, key string) error {
	configMapName := f.configMapName
	namespace := f.namespace
	filePath := filepath.Join(confDir, key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	instance, err := f.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			f.logger.Infow("config map not found, will create new one", zap.String("namespace", namespace), zap.String("name", configMapName))
			instance = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            configMapName,
					Namespace:       namespace,
					OwnerReferences: f.ownerReference,
				},
				Data: map[string]string{
					key: string(data),
				},
			}

			if _, err = f.kubeClient.CoreV1().ConfigMaps(namespace).Create(ctx, instance, metav1.CreateOptions{}); err != nil {
				return err
			}

		} else {
			return err
		}
	} else if instance.Data == nil {
		instance.Data = map[string]string{}
	}

	if value, ok := instance.Data[key]; ok && value == string(data) {
		f.logger.Infow("config map unchanged, skipping update", zap.String("file", filePath), zap.String("namespace", namespace), zap.String("name", configMapName))

		return nil
	}

	f.logger.Infow("update the config map", zap.String("file", filePath), zap.String("namespace", namespace), zap.String("name", configMapName))
	instance.Data[key] = string(data)
	if _, err = f.kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, instance, metav1.UpdateOptions{}); err != nil {
		return err
	}

	f.logger.Infow("file successfully backup to config map",
		zap.String("file", filePath),
		zap.String("namespace", namespace),
		zap.String("name", configMapName),
		zap.String("key", key))

	return nil
}

func (f *ResourceFactory) WriteConfigMapToFile(ctx context.Context, confDir, key string) error {
	configMapName := f.configMapName
	namespace := f.namespace
	filePath := filepath.Join(confDir, key)

	if _, err := os.Stat(filePath); err == nil {
		f.logger.Infow("local file already exists, skipping restoration",
			zap.String("file", filePath))
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat local file %q: %w", filePath, err)
	}

	f.logger.Infow("local file not found, attempting restoration from config map",
		zap.String("file", filePath),
		zap.String("namespace", namespace),
		zap.String("name", configMapName),
		zap.String("key", key))

	instance, err := f.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			f.logger.Infow("config map not found, restoration skipped",
				zap.String("file", filePath),
				zap.String("namespace", namespace),
				zap.String("name", configMapName),
				zap.String("key", key))
			return nil
		}

		return err
	}

	content, ok := instance.Data[key]
	if !ok {
		f.logger.Infow("key not found in config map, restoration skipped",
			zap.String("file", filePath),
			zap.String("namespace", namespace),
			zap.String("name", configMapName),
			zap.String("key", key))
		return nil
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write restored file %q: %w", filePath, err)
	}

	if err := os.Chown(filePath, 1001, 1001); err != nil {
		return fmt.Errorf("failed to change file ownership, file was still restored")
	}

	f.logger.Infow("file successfully restored from config map",
		zap.String("file", filePath),
		zap.String("namespace", namespace),
		zap.String("name", configMapName),
		zap.String("key", key))

	return nil
}
