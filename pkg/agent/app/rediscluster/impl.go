package rediscluster

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

var (
	// daemon instance
	dm = &daemon{}
)

type daemon struct {
	logger  *zap.SugaredLogger
	wg      *sync.WaitGroup
	factory *common.ResourceFactory
	watcher *fsnotify.Watcher

	confDir string
}

func (d *daemon) StartDaemon(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(5 * time.Minute)

	defer func() {
		ticker.Stop()
		wg.Done()
		_ = d.watcher.Close()
	}()

	d.logger.Info("start backup config daemon")

	key := "node.conf"
	if err := d.factory.WriteConfigMapToFile(ctx, d.confDir, key); err != nil {
		d.logger.Warn(err)
	}

	var eventsCh <-chan fsnotify.Event
	var errsCh <-chan error

	if err := d.watcher.Add(d.confDir); err != nil {
		d.logger.Warnw("watch directory failed, will only use periodic backup", zap.Error(err), zap.String("dir", d.confDir))
	} else {
		d.logger.Infow("watch directory for changes", zap.String("dir", d.confDir))
		eventsCh = d.watcher.Events
		errsCh = d.watcher.Errors
	}

	d.logger.Info("initial backup config")
	d.backupOnce(ctx, key)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("stop backup config daemon, doing final backup")
			d.backupOnce(ctx, key)

			return
		case <-ticker.C:
			d.logger.Info("period backup config")
			d.backupOnce(ctx, key)

		case ev, ok := <-eventsCh:
			if !ok {
				eventsCh = nil
				continue
			}

			if filepath.Clean(ev.Name) != filepath.Clean(filepath.Join(d.confDir, key)) {
				continue
			}

			if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				d.logger.Infow("detected file changed, trigger immediate backup", zap.String("file", ev.Name), zap.String("operation", ev.Op.String()))
				d.backupOnce(ctx, key)
			}

		case err, ok := <-errsCh:
			if !ok {
				errsCh = nil
				continue
			}
			d.logger.Errorw("failed to watch fs notify", zap.Error(err))
		}
	}
}

func (d *daemon) backupOnce(ctx context.Context, key string) {
	if err := d.factory.WriteFileToConfigMap(ctx, d.confDir, key); err != nil {
		d.logger.Errorw("failed to write file to config map", zap.Error(err))
	}

}

func (s *daemon) Config() error {
	s.logger = zap.L().Named(appName).Sugar()

	confDir, err := util.IsEnvVarSet(vars.ConfigDirEnvKey)
	if err != nil {
		return err
	}

	namespace, err := util.IsEnvVarSet(vars.NamespaceEnvKey)
	if err != nil {
		return err
	}

	name, err := util.IsEnvVarSet(vars.PodNameEnvKey)
	if err != nil {
		return err
	}

	factory, err := common.NewResourceFactory(context.Background(), s.logger, name, namespace)
	if err != nil {
		return err
	}

	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	s.factory = factory
	s.confDir = confDir
	return nil
}

func (s *daemon) Name() string {
	return appName
}

func (s *daemon) Registry(ctx context.Context, wg *sync.WaitGroup) {
	s.wg = wg

	wg.Add(1)
	go s.StartDaemon(ctx, wg)
}

func RegistryDaemonApp() {
	app.RegistryDaemonApp(dm)
}
