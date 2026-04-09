package logtail

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

func init() {
	zap.ReplaceGlobals(zap.NewNop())
}

func TestDaemonName(t *testing.T) {
	lt := &logtail{tailing: make(map[string]struct{})}
	require.Equal(t, "logtail", lt.Name())
}

func TestFilePrefix(t *testing.T) {
	tests := []struct {
		unitType string
		filename string
		want     string
	}{
		{"mysql", "unit_app.out.log", "[MYSQL:unit_app.out.log] "},
		{"mysql", "unit_app.err.log", "[MYSQL:unit_app.err.log] "},
		{"mysql", "slow-query.log", "[MYSQL:slow-query.log] "},
		{"redis", "unit_app.out.log", "[REDIS:unit_app.out.log] "},
		{"milvus", "unit_app.out.log", "[MILVUS:unit_app.out.log] "},
		{"postgresql", "postgresql.log", "[POSTGRESQL:postgresql.log] "},
		{"redis-sentinel", "unit_app.out.log", "[REDIS-SENTINEL:unit_app.out.log] "},
	}

	for _, tt := range tests {
		t.Run(tt.unitType+"/"+tt.filename, func(t *testing.T) {
			require.Equal(t, tt.want, filePrefix(tt.unitType, tt.filename))
		})
	}
}

func TestConfigMissingUnitType(t *testing.T) {
	lt := newLogtail()
	t.Setenv(vars.UnitTypeEnvKey, "")
	t.Setenv(vars.LogMountEnvKey, t.TempDir())

	err := lt.Config()
	require.Error(t, err)
}

func TestConfigMissingLogMount(t *testing.T) {
	lt := newLogtail()
	t.Setenv(vars.UnitTypeEnvKey, "mysql")
	t.Setenv(vars.LogMountEnvKey, "")

	err := lt.Config()
	require.Error(t, err)
}

func TestConfigSuccess(t *testing.T) {
	lt := newLogtail()
	tmpDir := t.TempDir()
	t.Setenv(vars.UnitTypeEnvKey, "mysql")
	t.Setenv(vars.LogMountEnvKey, tmpDir)

	err := lt.Config()
	require.NoError(t, err)
	require.Equal(t, tmpDir, lt.logDir)
	require.Equal(t, "mysql", lt.unitType)
}

func TestTailSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "app.log")
	err := os.WriteFile(logFile, []byte("hello world\n"), 0644)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	var output bytes.Buffer
	lt := &logtail{
		ctx:    ctx,
		logger: zap.L().Named("test").Sugar(),
	}

	prefix := filePrefix("mysql", "app.log")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		lt.tailFile(logFile, &output, prefix)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	out := output.String()
	require.Contains(t, out, "[MYSQL:app.log] hello world")
}

func TestScanAndTailMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple log files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.log"), []byte("line from app\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "error.log"), []byte("line from error\n"), 0644))
	// Non-log file should be ignored
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("not a log\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	lt := &logtail{
		ctx:      ctx,
		cancel:   cancel,
		logger:   zap.L().Named("test").Sugar(),
		logDir:   tmpDir,
		unitType: "redis",
		tailing:  make(map[string]struct{}),
	}

	var wg sync.WaitGroup
	lt.scanAndTail(&wg)

	// Should be tailing exactly 2 files
	lt.mu.Lock()
	count := len(lt.tailing)
	lt.mu.Unlock()
	require.Equal(t, 2, count)

	// Second scan should not add duplicates
	lt.scanAndTail(&wg)
	lt.mu.Lock()
	count = len(lt.tailing)
	lt.mu.Unlock()
	require.Equal(t, 2, count)

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestScanPicksUpNewFiles(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "first.log"), []byte("first\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	lt := &logtail{
		ctx:      ctx,
		cancel:   cancel,
		logger:   zap.L().Named("test").Sugar(),
		logDir:   tmpDir,
		unitType: "mysql",
		tailing:  make(map[string]struct{}),
	}

	var wg sync.WaitGroup
	lt.scanAndTail(&wg)

	lt.mu.Lock()
	require.Equal(t, 1, len(lt.tailing))
	lt.mu.Unlock()

	// Add a new file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "second.log"), []byte("second\n"), 0644))

	lt.scanAndTail(&wg)

	lt.mu.Lock()
	require.Equal(t, 2, len(lt.tailing))
	lt.mu.Unlock()

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestTailFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "nonexistent.log")

	lt := &logtail{
		ctx:    context.Background(),
		logger: zap.L().Named("test").Sugar(),
	}

	_, err := lt.openFileWithRetry(logFile)
	require.Error(t, err)
}

func TestNonLogFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "data.csv"), []byte("a,b\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte("[app]\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "real.log"), []byte("real log\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	lt := &logtail{
		ctx:      ctx,
		cancel:   cancel,
		logger:   zap.L().Named("test").Sugar(),
		logDir:   tmpDir,
		unitType: "mysql",
		tailing:  make(map[string]struct{}),
	}

	var wg sync.WaitGroup
	lt.scanAndTail(&wg)

	lt.mu.Lock()
	require.Equal(t, 1, len(lt.tailing))
	_, ok := lt.tailing[filepath.Join(tmpDir, "real.log")]
	require.True(t, ok)
	lt.mu.Unlock()

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestPrefixContainsFilename(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "unit_app.out.log"), []byte("stdout line\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "unit_app.err.log"), []byte("stderr line\n"), 0644))

	ctx, cancel := context.WithCancel(context.Background())
	var output bytes.Buffer
	lt := &logtail{
		ctx:    ctx,
		logger: zap.L().Named("test").Sugar(),
	}

	var wg sync.WaitGroup

	// Tail out log
	wg.Add(1)
	go func() {
		defer wg.Done()
		lt.tailFile(filepath.Join(tmpDir, "unit_app.out.log"), &output, filePrefix("milvus", "unit_app.out.log"))
	}()

	// Tail err log
	wg.Add(1)
	go func() {
		defer wg.Done()
		lt.tailFile(filepath.Join(tmpDir, "unit_app.err.log"), &output, filePrefix("milvus", "unit_app.err.log"))
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	out := output.String()
	require.True(t, strings.Contains(out, "[MILVUS:unit_app.out.log] stdout line"))
	require.True(t, strings.Contains(out, "[MILVUS:unit_app.err.log] stderr line"))
}
