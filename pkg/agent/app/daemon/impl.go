package daemon

import (
	"context"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func backupOnce(
	ctx context.Context,
	logger *zap.SugaredLogger,
	namespace, configMapName, filePath, key string,
) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	clientSet, err := conf.GetConf().GetClientSet()
	if err != nil {
		return err
	}

	cmClient := clientSet.CoreV1().ConfigMaps(namespace)

	cm, err := cmClient.Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Infof("configmap %s/%s not found, creating...", namespace, configMapName)
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: namespace,
				},
				Data: map[string]string{
					key: string(data),
				},
			}
			_, err = cmClient.Create(ctx, cm, metav1.CreateOptions{})
			return err
		}
		return err
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data[key] = string(data)

	_, err = cmClient.Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func StartRedisClusterNodesConfBackup(ctx context.Context, wg *sync.WaitGroup, namespace, podName, configDir string) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	logger := zap.L().Named("[REDIS DAEMON]").Sugar()
	key := "nodes.conf"

	configMapName := fmt.Sprintf("%s-config-backup", podName)

	filePath := filepath.Join(configDir, key)

	for {
		select {
		case <-ctx.Done():
			logger.Infof("stop redis cluster backup config daemon, doing final backup...")

			//Upon receiving the exit signal, attempt to perform a final backup (ignore errors and only log them).
			_ = backupOnce(ctx, logger, namespace, configMapName, filePath, key)
			logger.Info("backup loop exited gracefully")
			wg.Done()
		case <-ticker.C:
			if err := backupOnce(ctx, logger, namespace, configMapName, filePath, key); err != nil {
				logger.Errorf("periodic backup failed: %v", err)
			} else {
				logger.Info("periodic backup success")
			}
		}
	}
}
