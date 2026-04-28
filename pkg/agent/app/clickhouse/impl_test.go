package clickhouse

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
)

type fakeCommandRunner struct {
	err  error
	args []string
}

func (f *fakeCommandRunner) ExecuteCommand(cmd *exec.Cmd, _ string) error {
	f.args = append([]string(nil), cmd.Args...)
	return f.err
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
	require.Equal(t, "BACKUP ALL TO S3('https://s3.example.com/backups/backup-001', 'ak', 'sk')", backupSQL)

	restoreSQL, err := buildRestoreSQL(objectStorage, "backup-001")
	require.NoError(t, err)
	require.Equal(t, "RESTORE ALL FROM S3('https://s3.example.com/backups/backup-001', 'ak', 'sk')", restoreSQL)
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
		"--password", "secret",
		"--secure",
		"--query", "SELECT 1",
	}, runner.args)
}

func TestRunClickHouseQueryReturnsCommandFailure(t *testing.T) {
	runner := &fakeCommandRunner{err: errors.New("failed")}
	conn := clickHouseConnection{host: "127.0.0.1", port: "9000"}

	err := runClickHouseQuery(context.Background(), runner, conn, "admin", "secret", "SELECT 1")

	require.EqualError(t, err, "failed")
}
