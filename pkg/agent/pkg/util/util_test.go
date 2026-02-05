package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsEnvVarSet(t *testing.T) {
	t.Setenv("UNIT_AGENT_ENV", "value")

	val, err := IsEnvVarSet("UNIT_AGENT_ENV")
	require.NoError(t, err)
	require.Equal(t, "value", val)

	_, err = IsEnvVarSet("UNIT_AGENT_ENV_MISSING")
	require.Error(t, err)
}

func TestIsFileExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, []byte("content"), 0o644))

	require.True(t, IsFileExist(path))
	require.False(t, IsFileExist(filepath.Join(t.TempDir(), "missing")))
}

func TestIsConfigChanged(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")

	require.NoError(t, os.WriteFile(src, []byte("data"), 0o644))
	require.NoError(t, os.WriteFile(dest, []byte("data"), 0o644))

	changed, err := IsConfigChanged(src, dest)
	require.NoError(t, err)
	require.False(t, changed)

	require.NoError(t, os.WriteFile(dest, []byte("different"), 0o644))
	changed, err = IsConfigChanged(src, dest)
	require.NoError(t, err)
	require.True(t, changed)
}

func TestIsConfigChangedDestMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "missing")

	require.NoError(t, os.WriteFile(src, []byte("data"), 0o644))

	changed, err := IsConfigChanged(src, dest)
	require.NoError(t, err)
	require.True(t, changed)
}

func TestFileStat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(path, []byte("value"), 0o640))

	info, err := FileStat(path)
	require.NoError(t, err)
	require.NotEmpty(t, info.Md5)
	require.NotZero(t, info.Uid)
	require.NotZero(t, info.Gid)
	require.Equal(t, os.FileMode(0o640), info.Mode)

	_, err = FileStat(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}
