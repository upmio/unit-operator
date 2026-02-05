package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

type fakeStorageFactory struct {
	putErr error
	getErr error

	putBuffer bytes.Buffer
	getBuffer []byte
}

func (f *fakeStorageFactory) PutFile(context.Context, string, string, string) error {
	return f.putErr
}

func (f *fakeStorageFactory) GetFile(context.Context, string, string, string) error {
	return f.getErr
}

func (f *fakeStorageFactory) PutObject(_ context.Context, _ string, _ string, reader io.Reader) error {
	_, err := io.Copy(&f.putBuffer, reader)
	if err != nil {
		return err
	}
	return f.putErr
}

func (f *fakeStorageFactory) GetObject(context.Context, string, string) (io.ReadCloser, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	return io.NopCloser(bytes.NewReader(f.getBuffer)), nil
}

func newCommandExecutorForTest(t *testing.T) *CommandExecutor {
	t.Helper()

	logDir := t.TempDir()
	t.Setenv(vars.LogMountEnvKey, logDir)

	return NewCommandExecutor(zap.NewNop().Sugar())
}

func TestExecuteCommand(t *testing.T) {
	executor := newCommandExecutorForTest(t)
	cmd := exec.Command("sh", "-c", "printf 'ok'")

	err := executor.ExecuteCommand(cmd, "unit")
	require.NoError(t, err)
}

func TestExecuteCommandInvalidBinary(t *testing.T) {
	executor := newCommandExecutorForTest(t)
	cmd := exec.Command("does-not-exist")

	err := executor.ExecuteCommand(cmd, "unit")
	require.Error(t, err)
}

func TestExecuteCommandMissingLogDir(t *testing.T) {
	executor := NewCommandExecutor(zap.NewNop().Sugar())
	cmd := exec.Command("sh", "-c", "printf 'ok'")

	err := executor.ExecuteCommand(cmd, "unit")
	require.Error(t, err)
	require.Contains(t, err.Error(), vars.LogMountEnvKey)
}

func TestExecuteCommandFailure(t *testing.T) {
	executor := newCommandExecutorForTest(t)
	cmd := exec.Command("sh", "-c", "exit 1")

	err := executor.ExecuteCommand(cmd, "unit")
	require.Error(t, err)
	require.Contains(t, err.Error(), "command failed")
}

func TestExecutePipedCommands(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd1 := exec.Command("sh", "-c", "printf 'payload'")
	target := filepath.Join(t.TempDir(), "out")
	cmd2 := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", target))

	err := executor.ExecutePipedCommands(cmd1, cmd2, "pipe")
	require.NoError(t, err)

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "payload", string(data))
}

func TestExecuteCommandStreamToS3(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd := exec.Command("sh", "-c", "printf 'stream-data'")
	factory := &fakeStorageFactory{}

	err := executor.ExecuteCommandStreamToS3(context.Background(), cmd, factory, "bucket", "object", "backup")
	require.NoError(t, err)
	require.Equal(t, "stream-data", factory.putBuffer.String())
}

func TestExecuteCommandStreamFromS3(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	target := filepath.Join(t.TempDir(), "restored")
	cmd := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", target))
	factory := &fakeStorageFactory{
		getBuffer: []byte("from-s3"),
	}

	err := executor.ExecuteCommandStreamFromS3(context.Background(), cmd, factory, "bucket", "object", "restore")
	require.NoError(t, err)

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "from-s3", string(data))
}

func TestExecutePipedCommandsCmdFailure(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd1 := exec.Command("sh", "-c", "exit 1")
	cmd2 := exec.Command("cat")

	err := executor.ExecutePipedCommands(cmd1, cmd2, "pipe")
	require.Error(t, err)
	require.Contains(t, err.Error(), "command sh failed")
}

func TestExecuteCommandStreamToS3UploadError(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd := exec.Command("sh", "-c", "printf 'stream-data'")
	factory := &fakeStorageFactory{
		putErr: errors.New("upload failed"),
	}

	err := executor.ExecuteCommandStreamToS3(context.Background(), cmd, factory, "bucket", "object", "backup")
	require.Error(t, err)
	require.Contains(t, err.Error(), "upload failed")
}

func TestExecuteCommandStreamFromS3GetError(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd := exec.Command("sh", "-c", "cat")
	factory := &fakeStorageFactory{
		getErr: errors.New("missing object"),
	}

	err := executor.ExecuteCommandStreamFromS3(context.Background(), cmd, factory, "bucket", "object", "restore")
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing object")
}

func TestExecuteCommandStreamToS3CommandFailure(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd := exec.Command("sh", "-c", "exit 1")
	factory := &fakeStorageFactory{}

	err := executor.ExecuteCommandStreamToS3(context.Background(), cmd, factory, "bucket", "object", "backup")
	require.Error(t, err)
	require.Contains(t, err.Error(), "command failed")
}

func TestExecuteCommandStreamFromS3CommandFailure(t *testing.T) {
	executor := newCommandExecutorForTest(t)

	cmd := exec.Command("sh", "-c", "cat >/dev/null; exit 1")
	factory := &fakeStorageFactory{
		getBuffer: []byte("payload"),
	}

	err := executor.ExecuteCommandStreamFromS3(context.Background(), cmd, factory, "bucket", "object", "restore")
	require.Error(t, err)
	require.Contains(t, err.Error(), "command failed")
}
