package clickhouse

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
)

const (
	expectedBackupSQL  = "BACKUP ALL TO S3('https://s3.example.com/backups/backup-001', 'ak', 'sk')"
	expectedRestoreSQL = "RESTORE ALL FROM S3('https://s3.example.com/backups/backup-001', 'ak', 'sk')"
)

type fakeCommandRunner struct {
	err   error
	args  []string
	env   []string
	stdin string
}

func (f *fakeCommandRunner) ExecuteCommand(cmd *exec.Cmd, _ string) error {
	f.args = append([]string(nil), cmd.Args...)
	f.env = append([]string(nil), cmd.Env...)
	if cmd.Stdin != nil {
		stdin, err := io.ReadAll(cmd.Stdin)
		if err != nil {
			return err
		}
		f.stdin = string(stdin)
	}
	return f.err
}

type fakeSLM struct {
	slm.UnimplementedServiceLifecycleServer

	err     error
	checked int
}

func (f *fakeSLM) CheckProcessStarted(context.Context, *common.Empty) (*common.Empty, error) {
	f.checked++
	if f.err != nil {
		return nil, f.err
	}
	return &common.Empty{}, nil
}

func TestReadClickHouseConnectionDefaults(t *testing.T) {
	t.Setenv(clickHouseHostEnvKey, "")
	t.Setenv(clickHousePortEnvKey, "")
	t.Setenv(clickHouseSecureEnvKey, "")

	conn := readClickHouseConnection()

	require.Equal(t, clickHouseConnection{
		host:   "127.0.0.1",
		port:   "9000",
		secure: false,
	}, conn)
}

func TestReadClickHouseConnectionOverrides(t *testing.T) {
	t.Setenv(clickHouseHostEnvKey, "clickhouse")
	t.Setenv(clickHousePortEnvKey, "9440")
	t.Setenv(clickHouseSecureEnvKey, "TRUE")

	conn := readClickHouseConnection()

	require.Equal(t, clickHouseConnection{
		host:   "clickhouse",
		port:   "9440",
		secure: true,
	}, conn)
}

func TestBuildS3URL(t *testing.T) {
	objectStorage := &common.ObjectStorage{
		Endpoint:  "https://s3.example.com/root",
		Bucket:    "backups",
		AccessKey: "ak",
		SecretKey: "sk",
	}

	url, err := buildS3URL(objectStorage, "/daily/full-001")

	require.NoError(t, err)
	require.Equal(t, "https://s3.example.com/root/backups/daily/full-001", url)
}

func TestBuildS3URLAddsHTTPForBareEndpointWhenSSLDisabled(t *testing.T) {
	objectStorage := &common.ObjectStorage{
		Endpoint:  "s3.example.com:9000/root/",
		Bucket:    "backups",
		AccessKey: "ak",
		SecretKey: "sk",
		Ssl:       false,
	}

	url, err := buildS3URL(objectStorage, "daily/full-001")

	require.NoError(t, err)
	require.Equal(t, "http://s3.example.com:9000/root/backups/daily/full-001", url)
}

func TestBuildS3URLAddsHTTPSForBareEndpointWhenSSLEnabled(t *testing.T) {
	objectStorage := &common.ObjectStorage{
		Endpoint:  "s3.example.com/root",
		Bucket:    "backups",
		AccessKey: "ak",
		SecretKey: "sk",
		Ssl:       true,
	}

	url, err := buildS3URL(objectStorage, "daily/full-001")

	require.NoError(t, err)
	require.Equal(t, "https://s3.example.com/root/backups/daily/full-001", url)
}

func TestBuildS3URLRequiresHost(t *testing.T) {
	objectStorage := &common.ObjectStorage{
		Endpoint:  "https:///root",
		Bucket:    "backups",
		AccessKey: "ak",
		SecretKey: "sk",
	}

	_, err := buildS3URL(objectStorage, "daily/full-001")

	require.EqualError(t, err, "object_storage.endpoint must include a host")
}

func TestBuildS3URLRequiresFields(t *testing.T) {
	tests := []struct {
		name          string
		objectStorage *common.ObjectStorage
		backupFile    string
		wantErr       string
	}{
		{
			name:          "object storage",
			objectStorage: nil,
			backupFile:    "backup-001",
			wantErr:       "object_storage is required",
		},
		{
			name:          "endpoint",
			objectStorage: &common.ObjectStorage{Bucket: "bucket", AccessKey: "ak", SecretKey: "sk"},
			backupFile:    "backup-001",
			wantErr:       "object_storage.endpoint is required",
		},
		{
			name:          "bucket",
			objectStorage: &common.ObjectStorage{Endpoint: "https://s3.example.com", AccessKey: "ak", SecretKey: "sk"},
			backupFile:    "backup-001",
			wantErr:       "object_storage.bucket is required",
		},
		{
			name:          "backup file",
			objectStorage: &common.ObjectStorage{Endpoint: "https://s3.example.com", Bucket: "bucket", AccessKey: "ak", SecretKey: "sk"},
			backupFile:    "",
			wantErr:       "backup_file is required",
		},
		{
			name:          "access key",
			objectStorage: &common.ObjectStorage{Endpoint: "https://s3.example.com", Bucket: "bucket", SecretKey: "sk"},
			backupFile:    "backup-001",
			wantErr:       "object_storage.access_key is required",
		},
		{
			name:          "secret key",
			objectStorage: &common.ObjectStorage{Endpoint: "https://s3.example.com", Bucket: "bucket", AccessKey: "ak"},
			backupFile:    "backup-001",
			wantErr:       "object_storage.secret_key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildS3URL(tt.objectStorage, tt.backupFile)

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestBuildBackupAndRestoreSQL(t *testing.T) {
	objectStorage := &common.ObjectStorage{
		Endpoint:  "https://s3.example.com",
		Bucket:    "backups",
		AccessKey: "ak",
		SecretKey: "sk",
	}

	backupSQL, err := buildBackupSQL(objectStorage, "backup-001")
	require.NoError(t, err)
	require.Equal(t, expectedBackupSQL, backupSQL)

	restoreSQL, err := buildRestoreSQL(objectStorage, "backup-001")
	require.NoError(t, err)
	require.Equal(t, expectedRestoreSQL, restoreSQL)
}

func TestQuoteSQLStringEscapesSingleQuotesAndBackslashes(t *testing.T) {
	require.Equal(t, "'a\\'b\\\\c'", quoteSQLString(`a'b\c`))
}

func TestValidateVariableKey(t *testing.T) {
	require.NoError(t, validateIdentifier("max_threads"))
	require.NoError(t, validateIdentifier("profiles.default.max_threads"))
	require.Error(t, validateIdentifier("1max_threads"))
	require.Error(t, validateIdentifier("max-threads"))
	require.Error(t, validateIdentifier(""))
}

func TestBuildSetVariableSQL(t *testing.T) {
	sql, err := buildSetVariableSQL("admin", "max_threads", "8")

	require.NoError(t, err)
	require.Equal(t, "ALTER USER admin SETTINGS max_threads = '8'", sql)
}

func TestBuildSetVariableSQLRequiresValue(t *testing.T) {
	_, err := buildSetVariableSQL("admin", "max_threads", "  ")

	require.EqualError(t, err, "value is required")
}

func TestRunClickHouseQueryPassesConnectionAndQuery(t *testing.T) {
	runner := &fakeCommandRunner{}
	conn := clickHouseConnection{host: "127.0.0.1", port: "9440", secure: true}

	err := runClickHouseQuery(context.Background(), runner, conn, "admin", "secret", "SELECT 1")

	require.NoError(t, err)
	require.Equal(t, []string{
		"clickhouse-client",
		"--host", "127.0.0.1",
		"--port", "9440",
		"--user", "admin",
		"--secure",
	}, runner.args)
	require.Contains(t, runner.env, "CLICKHOUSE_PASSWORD=secret")
	require.Equal(t, "SELECT 1", runner.stdin)
}

func TestRunClickHouseQueryReturnsCommandFailure(t *testing.T) {
	runner := &fakeCommandRunner{err: errors.New("failed")}
	conn := clickHouseConnection{host: "127.0.0.1", port: "9000"}

	err := runClickHouseQuery(context.Background(), runner, conn, "admin", "secret", "SELECT 1")

	require.EqualError(t, err, "failed")
}

func TestLogicalBackupChecksSLMDecryptsPasswordAndRunsSafeCommand(t *testing.T) {
	t.Setenv(clickHouseHostEnvKey, "")
	t.Setenv(clickHousePortEnvKey, "9440")
	t.Setenv(clickHouseSecureEnvKey, "true")
	writeEncryptedPassword(t, "admin", "secret")

	runner := &fakeCommandRunner{}
	lifecycle := &fakeSLM{}
	s := &service{
		logger: zap.NewNop().Sugar(),
		slm:    lifecycle,
		runner: runner,
	}

	_, err := s.LogicalBackup(context.Background(), &LogicalBackupRequest{
		Username:      "admin",
		BackupFile:    "backup-001",
		ObjectStorage: defaultObjectStorage(),
	})

	require.NoError(t, err)
	require.Equal(t, 1, lifecycle.checked)
	requireSafeClickHouseCommand(t, runner, []string{"--secure"})
	require.Equal(t, expectedBackupSQL, runner.stdin)
}

func TestRestoreChecksSLMDecryptsPasswordAndRunsSafeCommand(t *testing.T) {
	t.Setenv(clickHouseHostEnvKey, "")
	t.Setenv(clickHousePortEnvKey, "9440")
	t.Setenv(clickHouseSecureEnvKey, "true")
	writeEncryptedPassword(t, "admin", "secret")

	runner := &fakeCommandRunner{}
	lifecycle := &fakeSLM{}
	s := &service{
		logger: zap.NewNop().Sugar(),
		slm:    lifecycle,
		runner: runner,
	}

	_, err := s.Restore(context.Background(), &RestoreRequest{
		Username:      "admin",
		BackupFile:    "backup-001",
		ObjectStorage: defaultObjectStorage(),
	})

	require.NoError(t, err)
	require.Equal(t, 1, lifecycle.checked)
	requireSafeClickHouseCommand(t, runner, []string{"--secure"})
	require.Equal(t, expectedRestoreSQL, runner.stdin)
}

func TestLogicalBackupSLMFailurePreventsCommandExecution(t *testing.T) {
	runner := &fakeCommandRunner{}
	lifecycle := &fakeSLM{err: errors.New("slm down")}
	s := &service{
		logger: zap.NewNop().Sugar(),
		slm:    lifecycle,
		runner: runner,
	}

	_, err := s.LogicalBackup(context.Background(), &LogicalBackupRequest{
		Username:      "admin",
		BackupFile:    "backup-001",
		ObjectStorage: defaultObjectStorage(),
	})

	require.EqualError(t, err, "slm down")
	require.Equal(t, 1, lifecycle.checked)
	require.Nil(t, runner.args)
}

func TestSetVariableChecksSLMDecryptsPasswordAndRunsSafeCommand(t *testing.T) {
	t.Setenv(clickHouseHostEnvKey, "")
	t.Setenv(clickHousePortEnvKey, "9440")
	t.Setenv(clickHouseSecureEnvKey, "true")
	writeEncryptedPassword(t, "admin", "secret")

	runner := &fakeCommandRunner{}
	lifecycle := &fakeSLM{}
	s := &service{
		logger: zap.NewNop().Sugar(),
		slm:    lifecycle,
		runner: runner,
	}

	_, err := s.SetVariable(context.Background(), &SetVariableRequest{
		Username: "admin",
		Key:      "max_threads",
		Value:    "8",
	})

	require.NoError(t, err)
	require.Equal(t, 1, lifecycle.checked)
	requireSafeClickHouseCommand(t, runner, []string{"--secure"})
	require.Equal(t, "ALTER USER admin SETTINGS max_threads = '8'", runner.stdin)
}

func TestSetVariableSLMFailurePreventsCommandExecution(t *testing.T) {
	runner := &fakeCommandRunner{}
	lifecycle := &fakeSLM{err: errors.New("slm down")}
	s := &service{
		logger: zap.NewNop().Sugar(),
		slm:    lifecycle,
		runner: runner,
	}

	_, err := s.SetVariable(context.Background(), &SetVariableRequest{
		Username: "admin",
		Key:      "max_threads",
		Value:    "8",
	})

	require.EqualError(t, err, "slm down")
	require.Equal(t, 1, lifecycle.checked)
	require.Nil(t, runner.args)
}

func TestSafeCommandRunnerDoesNotReturnOutputOnFailure(t *testing.T) {
	runner := &safeCommandRunner{logger: zap.NewNop().Sugar()}
	cmd := exec.Command("sh", "-c", "printf 'ak sk secret' >&2; exit 7")

	err := runner.ExecuteCommand(cmd, "clickhouse")

	require.Error(t, err)
	require.Contains(t, err.Error(), "command failed")
	require.NotContains(t, err.Error(), "ak")
	require.NotContains(t, err.Error(), "sk")
	require.NotContains(t, err.Error(), "secret")
}

func requireSafeClickHouseCommand(t *testing.T, runner *fakeCommandRunner, extraArgs []string) {
	t.Helper()

	expectedArgs := []string{
		"clickhouse-client",
		"--host", "127.0.0.1",
		"--port", "9440",
		"--user", "admin",
	}
	expectedArgs = append(expectedArgs, extraArgs...)

	require.Equal(t, expectedArgs, runner.args)
	args := strings.Join(runner.args, "\x00")
	require.NotContains(t, runner.args, "--password")
	require.NotContains(t, runner.args, "--query")
	require.NotContains(t, args, "ak")
	require.NotContains(t, args, "sk")
	require.Contains(t, runner.env, "CLICKHOUSE_PASSWORD=secret")
}

func defaultObjectStorage() *common.ObjectStorage {
	return &common.ObjectStorage{
		Endpoint:  "https://s3.example.com",
		Bucket:    "backups",
		AccessKey: "ak",
		SecretKey: "sk",
	}
}

func writeEncryptedPassword(t *testing.T, username, password string) {
	t.Helper()

	secretDir := t.TempDir()
	t.Setenv(vars.SecretMountEnvKey, secretDir)
	t.Setenv(vars.AESEnvKey, "12345678901234567890123456789012")
	require.NoError(t, util.ValidateAndSetAESKey())

	encryptedPassword, err := util.AES_CTR_Encrypt([]byte(password))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, username), encryptedPassword, 0600))
}
