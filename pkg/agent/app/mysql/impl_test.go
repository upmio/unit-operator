package mysql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) *service {
	t.Helper()
	return &service{
		logger: zap.NewNop().Sugar(),
	}
}

func TestGenerateGtidPurgeSql(t *testing.T) {
	svc := newTestService(t)
	dir := t.TempDir()
	svc.dataDir = dir

	content := "mysql-bin.000001 123 uuid:1-5\nbinlog,uuid:6-10\nextra,uuid:11-12\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "xtrabackup_binlog_info"), []byte(content), 0o644))

	gtid, err := svc.generateGtidPurgeSql()
	require.NoError(t, err)
	require.Equal(t, "uuid:1-5,uuid:6-10,uuid:11-12", gtid)
}

func TestGenerateGtidPurgeSqlMissingFile(t *testing.T) {
	svc := newTestService(t)
	svc.dataDir = t.TempDir()

	_, err := svc.generateGtidPurgeSql()
	require.Error(t, err)
}

func TestRemoveContents(t *testing.T) {
	svc := newTestService(t)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1"), []byte("data"), 0o644))
	sub := filepath.Join(dir, "nested")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "file2"), []byte("value"), 0o644))

	require.NoError(t, svc.removeContents(dir))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 0)
}

