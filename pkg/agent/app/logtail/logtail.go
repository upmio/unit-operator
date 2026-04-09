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
	// Log file names produced by supervisord (relative to LOG_MOUNT directory)
	outLogFile = "unit_app.out.log"
	errLogFile = "unit_app.err.log"

	// Scan interval for checking new content
	scanInterval = 500 * time.Millisecond

	// Buffer size for scanner
	maxScanTokenSize = 64 * 1024 // 64KB
)

// stdoutPrefix returns the stable parsing protocol prefix for forwarded stdout logs.
// Downstream Java/UI consumers rely on this exact prefix to identify stdout log lines.
// Format: [<UNIT_TYPE>-STDOUT]
// Example: [MYSQL-STDOUT] , [REDIS-STDOUT] , [MILVUS-STDOUT]
func stdoutPrefix(unitType string) string {
	return fmt.Sprintf("[%s-STDOUT] ", strings.ToUpper(unitType))
}

// stderrPrefix returns the stable parsing protocol prefix for forwarded stderr logs.
// Downstream Java/UI consumers rely on this exact prefix to identify stderr log lines.
// Format: [<UNIT_TYPE>-STDERR]
// Example: [MYSQL-STDERR] , [REDIS-STDERR] , [MILVUS-STDERR]
func stderrPrefix(unitType string) string {
	return fmt.Sprintf("[%s-STDERR] ", strings.ToUpper(unitType))
}

// logtail is a DaemonApp that forwards main container logs to agent stdout/stderr.
// It reads supervisord log files in real-time and outputs with typed prefixes.
type logtail struct {
	logger       *zap.SugaredLogger
	logDir       string // LOG_MOUNT directory
	unitType     string // UNIT_TYPE value
	stdoutPrefix string // computed stdout prefix
	stderrPrefix string // computed stderr prefix
	ctx          context.Context
	cancel       context.CancelFunc
}

// newLogtail creates a new logtail daemon instance.
func newLogtail() *logtail {
	ctx, cancel := context.WithCancel(context.Background())
	return &logtail{
		ctx:    ctx,
		cancel: cancel,
	}
}

// StartDaemon starts the log forwarding daemon
func (lt *logtail) StartDaemon(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	lt.logger.Infof("start logtail daemon for unit type: %s", lt.unitType)

	// Start two goroutines to tail stdout and stderr respectively
	var wgTail sync.WaitGroup
	wgTail.Add(2)

	go func() {
		defer wgTail.Done()
		outPath := filepath.Join(lt.logDir, outLogFile)
		lt.tailFile(outPath, os.Stdout, lt.stdoutPrefix)
	}()

	go func() {
		defer wgTail.Done()
		errPath := filepath.Join(lt.logDir, errLogFile)
		lt.tailFile(errPath, os.Stderr, lt.stderrPrefix)
	}()

	// Wait for context cancellation
	<-ctx.Done()
	lt.logger.Infof("stop logtail daemon for unit type: %s", lt.unitType)

	// Cancel internal context to stop tail goroutines
	lt.cancel()

	// Wait for tail goroutines to finish
	wgTail.Wait()
}

// tailFile reads file content in real-time and outputs to writer
func (lt *logtail) tailFile(path string, output io.Writer, prefix string) {
	lt.logger.Infow("start tailing file", zap.String("path", path))

	file, err := lt.openFileWithRetry(path)
	if err != nil {
		lt.logger.Errorw("failed to open file after retries, stop tailing",
			zap.String("path", path), zap.Error(err))
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Create scanner to read file
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), maxScanTokenSize)

	for {
		select {
		case <-lt.ctx.Done():
			lt.logger.Infow("stop tailing file", zap.String("path", path))
			return
		default:
			if scanner.Scan() {
				line := scanner.Bytes()
				if len(line) > 0 {
					_, _ = output.Write([]byte(prefix + string(line) + "\n"))
				}
			} else {
				err := scanner.Err()
				if err != nil {
					lt.logger.Warnw("scanner error, will retry open",
						zap.String("path", path), zap.Error(err))
					// Wait before retry
					time.Sleep(scanInterval)

					// Try to reopen file (handle log rotation)
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
					// File reached EOF, wait for new content
					time.Sleep(scanInterval)
				}
			}
		}
	}
}

// openFileWithRetry attempts to open file, waits if file does not exist
func (lt *logtail) openFileWithRetry(path string) (*os.File, error) {
	var lastErr error
	for i := 0; i < 60; i++ { // Wait up to 30 seconds
		file, err := os.Open(path)
		if err == nil {
			return file, nil
		}
		lastErr = err

		// File does not exist, wait for creation
		if os.IsNotExist(err) {
			lt.logger.Infow("file not found, waiting...",
				zap.String("path", path), zap.Int("retry", i))
			time.Sleep(scanInterval)
			continue
		}

		// Other errors return immediately
		return nil, err
	}
	return nil, lastErr
}

// Config initializes logtail configuration
func (lt *logtail) Config() error {
	unitType, err := util.IsEnvVarSet(vars.UnitTypeEnvKey)
	if err != nil {
		return err
	}

	lt.unitType = unitType
	lt.stdoutPrefix = stdoutPrefix(unitType)
	lt.stderrPrefix = stderrPrefix(unitType)
	lt.logger = zap.L().Named("logtail-" + unitType).Sugar()

	logDir, err := util.IsEnvVarSet(vars.LogMountEnvKey)
	if err != nil {
		return err
	}

	lt.logDir = logDir
	lt.logger.Infow("logtail configured",
		zap.String("unitType", lt.unitType),
		zap.String("logDir", lt.logDir),
		zap.String("stdoutPrefix", lt.stdoutPrefix),
		zap.String("stderrPrefix", lt.stderrPrefix),
		zap.String("outFile", outLogFile),
		zap.String("errFile", errLogFile))

	return nil
}

// Name returns daemon name
func (lt *logtail) Name() string {
	return "logtail"
}

// Registry registers daemon to app
func (lt *logtail) Registry(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go lt.StartDaemon(ctx, wg)
}

// RegistryDaemonApp registers the generic logtail daemon app
func RegistryDaemonApp() {
	dm := newLogtail()
	app.RegistryDaemonApp(dm)
}

