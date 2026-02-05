package redis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRedisInfo(t *testing.T) {
	info := `
# Persistence
rdb_bgsave_in_progress:0
loading:0
`
	result := parseRedisInfo(info)
	require.Equal(t, "0", result["rdb_bgsave_in_progress"])
	require.Equal(t, "0", result["loading"])
}

func TestDiscoverRDBPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dump.rdb")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o644))

	rdbPath, err := discoverRDBPath(dir)
	require.NoError(t, err)
	require.Equal(t, path, rdbPath)
}

func TestDiscoverRDBPathMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := discoverRDBPath(dir)
	require.Error(t, err)
}

func TestRenameWithBak(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "dump.rdb")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0o644))

	require.NoError(t, renameWithBak(src))
	require.NoFileExists(t, src)
	require.FileExists(t, src+".bak")
}

func TestRenameWithBakMissing(t *testing.T) {
	dir := t.TempDir()
	err := renameWithBak(filepath.Join(dir, "not-exist.rdb"))
	require.NoError(t, err)
}
