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
	lt := &logtail{}
	require.Equal(t, "logtail", lt.Name())
}

func TestPrefixGeneration(t *testing.T) {
	tests := []struct {
		unitType       string
		wantStdout     string
		wantStderr     string
	}{
		{"mysql", "[MYSQL-STDOUT] ", "[MYSQL-STDERR] "},
		{"redis", "[REDIS-STDOUT] ", "[REDIS-STDERR] "},
		{"postgresql", "[POSTGRESQL-STDOUT] ", "[POSTGRESQL-STDERR] "},
		{"milvus", "[MILVUS-STDOUT] ", "[MILVUS-STDERR] "},
		{"mongodb", "[MONGODB-STDOUT] ", "[MONGODB-STDERR] "},
		{"proxysql", "[PROXYSQL-STDOUT] ", "[PROXYSQL-STDERR] "},
		{"redis-sentinel", "[REDIS-SENTINEL-STDOUT] ", "[REDIS-SENTINEL-STDERR] "},
	}

	for _, tt := range tests {
		t.Run(tt.unitType, func(t *testing.T) {
			require.Equal(t, tt.wantStdout, stdoutPrefix(tt.unitType))
			require.Equal(t, tt.wantStderr, stderrPrefix(tt.unitType))
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
	require.Equal(t, "[MYSQL-STDOUT] ", lt.stdoutPrefix)
	require.Equal(t, "[MYSQL-STDERR] ", lt.stderrPrefix)
}

func TestTailFile(t *testing.T) {
	unitTypes := []string{"mysql", "redis", "milvus", "postgresql"}

	for _, unitType := range unitTypes {
		t.Run(unitType+"/stdout", func(t *testing.T) {
			testTailFileWithPrefix(t, outLogFile, stdoutPrefix(unitType), "test stdout log message")
		})

		t.Run(unitType+"/stderr", func(t *testing.T) {
			testTailFileWithPrefix(t, errLogFile, stderrPrefix(unitType), "test stderr log message")
		})
	}
}

func testTailFileWithPrefix(t *testing.T, fileName, prefix, message string) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, fileName)

	content := "2026-03-23 10:00:00 INFO " + message + "\n"
	err := os.WriteFile(logFile, []byte(content), 0644)
	require.NoError(t, err)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	var output bytes.Buffer
	lt := &logtail{
		ctx:    ctx,
		logger: zap.L().Named("logtail-test").Sugar(),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		lt.tailFile(logFile, &output, prefix)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	outputStr := output.String()
	require.True(t, strings.Contains(outputStr, prefix))
	require.True(t, strings.Contains(outputStr, message))
}

func TestTailFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "nonexistent.log")

	lt := &logtail{
		ctx:    context.Background(),
		logger: zap.L().Named("logtail-test").Sugar(),
	}

	_, err := lt.openFileWithRetry(logFile)
	require.Error(t, err)
}

func TestTailFileContentAppend(t *testing.T) {
	t.Skip("reserved for future append-behavior verification")

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, outLogFile)

	err := os.WriteFile(logFile, []byte(""), 0644)
	require.NoError(t, err)

	r, w, err := os.Pipe()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	lt := &logtail{
		ctx:    ctx,
		logger: zap.L().Named("logtail-test").Sugar(),
	}

	prefix := stdoutPrefix("mysql")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		lt.tailFile(logFile, w, prefix)
	}()

	time.Sleep(100 * time.Millisecond)

	newContent := "2026-03-23 10:00:00 INFO new log message\n"
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err)
	_, err = f.Write([]byte(newContent))
	require.NoError(t, f.Close())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()
	require.NoError(t, w.Close())
	require.NoError(t, r.Close())
}

