package daemon

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	defer wg.Done()

	logger := zap.L().Named("[REDIS DAEMON]").Sugar()
	logger.Info("start backup config daemon")

	key := "nodes.conf"

	configMapName := fmt.Sprintf("%s-config-backup", podName)

	filePath := filepath.Join(configDir, key)

	//ensure redis node config exists
	if err := ensureRedisClusterNodeConf(ctx, logger, namespace, configMapName, filePath, key); err != nil {
		logger.Error(err)
	}

	var eventsCh <-chan fsnotify.Event
	var errsCh <-chan error

	// 初始化 fsnotify watcher，监听 nodes.conf 所在目录
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Errorf("create fsnotify watcher failed, will only use periodic backup: %v", err)
	}
	// 确保退出时关闭 watcher
	if watcher != nil {
		defer watcher.Close()
	}

	if watcher != nil {
		dir := filepath.Dir(filePath)
		if err := watcher.Add(dir); err != nil {
			logger.Errorf("watch directory %s failed, will only use periodic backup: %v", dir, err)
		} else {
			logger.Infof("watching directory %s for changes of %s", dir, filePath)
			eventsCh = watcher.Events
			errsCh = watcher.Errors
		}
	}

	if err := backupOnce(ctx, logger, namespace, configMapName, filePath, key); err != nil {
		logger.Errorf("initial backup config failed: %v", err)
	} else {
		logger.Info("initial backup config success")
	}

	for {
		select {
		case <-ctx.Done():
			logger.Infof("stop backup config daemon, doing final backup...")

			//Upon receiving the exit signal, attempt to perform a final backup (ignore errors and only log them).
			_ = backupOnce(ctx, logger, namespace, configMapName, filePath, key)
			logger.Info("backup config daemon exited gracefully")
			return
		case <-ticker.C:
			if err := backupOnce(ctx, logger, namespace, configMapName, filePath, key); err != nil {
				logger.Errorf("periodic backup config failed: %v", err)
			} else {
				logger.Info("periodic backup config success")
			}

		case ev, ok := <-eventsCh:
			if !ok {
				eventsCh = nil
				continue
			}

			if filepath.Clean(ev.Name) != filepath.Clean(filePath) {
				continue
			}

			if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				logger.Infof("detected change on %s (op=%s), trigger immediate backup", ev.Name, ev.Op.String())
				if err := backupOnce(ctx, logger, namespace, configMapName, filePath, key); err != nil {
					logger.Errorf("fsnotify backup config failed: %v", err)
				} else {
					logger.Info("fsnotify backup config success")
				}
			}

		case err, ok := <-errsCh:
			if !ok {
				errsCh = nil
				continue
			}
			logger.Errorf("fsnotify watcher error: %v", err)
		}
	}
}

func ensureRedisClusterNodeConf(ctx context.Context, logger *zap.SugaredLogger,
	namespace, configMapName, filePath, key string) error {
	if _, err := os.Stat(filePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat local file %s failed: %v", filePath, err)
		}

		logger.Infof("local file %s not found, try restore from ConfigMap %s/%s",
			filePath, namespace, configMapName)

		clientSet, err := conf.GetConf().GetClientSet()
		if err != nil {
			return err
		}

		cm, err := clientSet.CoreV1().
			ConfigMaps(namespace).
			Get(ctx, configMapName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Infof("configmap %s/%s not found, skip restore", namespace, configMapName)
				return nil
			}

			return fmt.Errorf("get configmap %s/%s failed: %v", namespace, configMapName, err)
		}

		content, ok := cm.Data[key]
		if !ok {
			logger.Infof("configmap %s/%s has no key %q, skip restore", namespace, configMapName, key)
			return nil
		}

		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write local file %s failed: %v", filePath, err)
		}

		if err := os.Chown(filePath, 1001, 1001); err != nil {
			return fmt.Errorf("chown local file %s failed: %v", filePath, err)
		}

		logger.Infof("restore local file %s from configmap %s/%s key %q success",
			filePath, namespace, configMapName, key)
		return nil
	}

	logger.Infof("local file %s exists, skip restore from configmap", filePath)
	return nil
}
