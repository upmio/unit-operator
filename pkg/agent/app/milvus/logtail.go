package milvus

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

const (
	// Log file names (relative to LOG_MOUNT directory)
	outLogFile = "unit_app.out.log"
	errLogFile = "unit_app.err.log"

	// Scan interval for checking new content
	scanInterval = 500 * time.Millisecond

	// Buffer size for scanner
	maxScanTokenSize = 64 * 1024 // 64KB
)

// logtail is a DaemonApp that forwards Milvus logs to stdout
// It reads log files in real-time and outputs to stdout/stderr
type logtail struct {
	logger *zap.SugaredLogger
	logDir string // LOG_MOUNT directory
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	// Commented: environment variable control
	// enableEnvKey = "LOG_STDOUT_ENABLE"
}

// Commented: exec.Command tail implementation
// func (lt *logtail) tailFileByCommand(path string, output io.Writer) error {
// 	cmd := exec.Command("tail", "-f", "-n", "+1", path)
// 	cmd.Stdout = output
// 	cmd.Stderr = os.Stderr
// 	return cmd.Run()
// }

// NewLogtail creates a new logtail daemon instance
func NewLogtail() *logtail {
	ctx, cancel := context.WithCancel(context.Background())
	return &logtail{
		ctx:    ctx,
		cancel: cancel,
	}
}

// StartDaemon starts the log forwarding daemon
func (lt *logtail) StartDaemon(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	lt.logger.Info("start milvus logtail daemon")

	// Start two goroutines to tail stdout and stderr respectively
	var wgTail sync.WaitGroup
	wgTail.Add(2)

	go func() {
		defer wgTail.Done()
		outPath := filepath.Join(lt.logDir, outLogFile)
		lt.tailFile(outPath, os.Stdout)
	}()

	go func() {
		defer wgTail.Done()
		errPath := filepath.Join(lt.logDir, errLogFile)
		lt.tailFile(errPath, os.Stderr)
	}()

	// Wait for context cancellation
	<-ctx.Done()
	lt.logger.Info("stop milvus logtail daemon")

	// Wait for tail goroutines to finish
	wgTail.Wait()
}

// tailFile reads file content in real-time and outputs to writer
// Uses native Go implementation with log rotation detection
func (lt *logtail) tailFile(path string, output io.Writer) {
	lt.logger.Infow("start tailing file", zap.String("path", path))

	file, err := lt.openFileWithRetry(path)
	if err != nil {
		lt.logger.Errorw("failed to open file after retries, stop tailing",
			zap.String("path", path), zap.Error(err))
		return
	}
	defer file.Close()

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
					output.Write(line)
					output.Write([]byte("\n"))
				}
			} else {
				err := scanner.Err()
				if err != nil {
					lt.logger.Warnw("scanner error, will retry open",
						zap.String("path", path), zap.Error(err))
					// Wait before retry
					time.Sleep(scanInterval)

					// Try to reopen file (handle log rotation)
					file.Close()
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

// Commented: environment variable control for enable/disable
// func (lt *logtail) isEnabled() bool {
// 	envValue := os.Getenv(lt.enableEnvKey)
// 	return strings.ToLower(envValue) == "true" || envValue == "1"
// }

// Config initializes logtail configuration
func (lt *logtail) Config() error {
	lt.logger = zap.L().Named("milvus-logtail").Sugar()

	// Commented: environment variable control
	// if !lt.isEnabled() {
	// 	lt.logger.Info("logtail is disabled by environment variable")
	// 	return nil
	// }

	logDir, err := util.IsEnvVarSet(vars.LogMountEnvKey)
	if err != nil {
		return err
	}

	lt.logDir = logDir
	lt.logger.Infow("logtail configured",
		zap.String("logDir", lt.logDir),
		zap.String("outFile", outLogFile),
		zap.String("errFile", errLogFile))

	return nil
}

// Name returns daemon name
func (lt *logtail) Name() string {
	return "milvus-logtail"
}

// Registry registers daemon to app
func (lt *logtail) Registry(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go lt.StartDaemon(ctx, wg)
}

// RegistryDaemonApp registers milvus-logtail daemon app
func RegistryDaemonApp() {
	dm := NewLogtail()
	app.RegistryDaemonApp(dm)
}
