package logtail

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

const (

	// dirScanInterval is how often the daemon re-scans LOG_MOUNT for new log files.
	dirScanInterval = 5 * time.Second

	// scanInterval is how often an individual tail goroutine checks for new content.
	scanInterval = 500 * time.Millisecond

	// maxScanTokenSize is the maximum line length the scanner can handle.
	maxScanTokenSize = 64 * 1024 // 64KB
)

// filePrefix returns the stable parsing protocol prefix for a given log file.
// Downstream Java/UI consumers rely on this prefix to identify the source file.
//
// Format:  [<UNIT_TYPE>:<filename>]
// Example: [MYSQL:unit_app.out.log] , [MILVUS:error.log]
func filePrefix(unitType, filename string) string {
	return fmt.Sprintf("[%s:%s] ", strings.ToUpper(unitType), filename)
}

// logtail is a DaemonApp that forwards all log files under LOG_MOUNT to agent stdout.
type logtail struct {
	logger   *zap.SugaredLogger
	logDir   string // LOG_MOUNT directory
	unitType string // UNIT_TYPE value
	ctx      context.Context
	cancel   context.CancelFunc

	// tailing tracks files that already have a tail goroutine running.
	tailing map[string]struct{}
	mu      sync.Mutex
}

// newLogtail creates a new logtail daemon instance.
func newLogtail() *logtail {
	ctx, cancel := context.WithCancel(context.Background())
	return &logtail{
		ctx:     ctx,
		cancel:  cancel,
		tailing: make(map[string]struct{}),
	}
}

// StartDaemon starts the log forwarding daemon.
// It scans LOG_MOUNT for *.log files and tails each one.
// New files appearing later are picked up by periodic re-scans.
func (lt *logtail) StartDaemon(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	lt.logger.Infof("start logtail daemon for unit type: %s", lt.unitType)

	var wgTail sync.WaitGroup

	// Initial scan
	lt.scanAndTail(&wgTail)

	// Periodic re-scan for new log files
	ticker := time.NewTicker(dirScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			lt.logger.Infof("stop logtail daemon for unit type: %s", lt.unitType)
			lt.cancel()
			wgTail.Wait()
			return
		case <-ticker.C:
			lt.scanAndTail(&wgTail)
		}
	}
}

// scanAndTail reads logDir and starts a tail goroutine for every regular file
// (excluding subdirectories) not already being tailed.
func (lt *logtail) scanAndTail(wg *sync.WaitGroup) {
	entries, err := os.ReadDir(lt.logDir)
	if err != nil {
		lt.logger.Warnw("failed to scan log directory", zap.Error(err))
		return
	}

	lt.mu.Lock()
	defer lt.mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(lt.logDir, entry.Name())
		if _, ok := lt.tailing[path]; ok {
			continue // already tailing
		}
		lt.tailing[path] = struct{}{}

		filename := filepath.Base(path)
		prefix := filePrefix(lt.unitType, filename)

		lt.logger.Infow("discovered log file, start tailing",
			zap.String("path", path), zap.String("prefix", prefix))

		wg.Add(1)
		go func(p, pfx string) {
			defer wg.Done()
			lt.tailFile(p, os.Stdout, pfx)
		}(path, prefix)
	}
}

// tailFile reads file content in real-time and outputs each line to writer with prefix.
func (lt *logtail) tailFile(path string, output io.Writer, prefix string) {
	file, err := lt.openFileWithRetry(path)
	if err != nil {
		lt.logger.Errorw("failed to open file after retries, stop tailing",
			zap.String("path", path), zap.Error(err))
		return
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), maxScanTokenSize)

	for {
		select {
		case <-lt.ctx.Done():
			return
		default:
			if scanner.Scan() {
				line := scanner.Bytes()
				if len(line) > 0 {
					_, _ = output.Write([]byte(prefix + string(line) + "\n"))
				}
			} else {
				if err := scanner.Err(); err != nil {
					lt.logger.Warnw("scanner error, will retry open",
						zap.String("path", path), zap.Error(err))
					time.Sleep(scanInterval)

					_ = file.Close()
					file, err = lt.openFileWithRetry(path)
					if err != nil {
						lt.logger.Errorw("failed to reopen file, stop tailing",
							zap.String("path", path), zap.Error(err))
						return
					}
					scanner = bufio.NewScanner(file)
					scanner.Buffer(make([]byte, 4096), maxScanTokenSize)
				} else {
					// EOF reached. bufio.Scanner sets an internal done flag after
					// EOF, so subsequent Scan() calls always return false even if
					// new data is appended to the file. Rebuild the scanner from
					// the same file handle (whose offset stays at the current
					// position) so new content can be picked up.
					time.Sleep(scanInterval)
					scanner = bufio.NewScanner(file)
					scanner.Buffer(make([]byte, 4096), maxScanTokenSize)
				}
			}
		}
	}
}

// openFileWithRetry attempts to open file, retrying for up to 30 seconds if not found.
func (lt *logtail) openFileWithRetry(path string) (*os.File, error) {
	var lastErr error
	for i := 0; i < 60; i++ {
		file, err := os.Open(path)
		if err == nil {
			return file, nil
		}
		lastErr = err

		if os.IsNotExist(err) {
			lt.logger.Infow("file not found, waiting...",
				zap.String("path", path), zap.Int("retry", i))
			time.Sleep(scanInterval)
			continue
		}

		return nil, err
	}
	return nil, lastErr
}

// Config initializes logtail configuration.
func (lt *logtail) Config() error {
	unitType, err := util.IsEnvVarSet(vars.UnitTypeEnvKey)
	if err != nil {
		return err
	}

	lt.unitType = unitType
	lt.logger = zap.L().Named("logtail-" + unitType).Sugar()

	logDir, err := util.IsEnvVarSet(vars.LogMountEnvKey)
	if err != nil {
		return err
	}

	lt.logDir = logDir
	lt.logger.Infow("logtail configured",
		zap.String("unitType", lt.unitType),
		zap.String("logDir", lt.logDir))

	return nil
}

// Name returns daemon name.
func (lt *logtail) Name() string {
	return "logtail"
}

// Registry registers daemon to app.
func (lt *logtail) Registry(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go lt.StartDaemon(ctx, wg)
}

// RegistryDaemonApp registers the generic logtail daemon app.
func RegistryDaemonApp() {
	dm := newLogtail()
	app.RegistryDaemonApp(dm)
}

