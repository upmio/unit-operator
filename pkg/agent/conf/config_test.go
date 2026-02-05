package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadConfigFromTomlAndAddresses(t *testing.T) {
	content := `
[log]
level = "debug"
dir = "/tmp"

[app]
host = "127.0.0.1"
port = 8080
grpc_host = "127.0.0.1"
grpc_port = 9090

[kube]
kubeConfigPath = ""

[supervisor]
address = "127.0.0.1"
port = 9001
`
	path := writeTempConfig(t, content)

	require.NoError(t, LoadConfigFromToml(path))

	cfg := GetConf()
	require.Equal(t, "127.0.0.1:9090", cfg.App.GrpcAddr())
	require.Equal(t, "127.0.0.1:8080", cfg.App.Addr())
	require.Equal(t, zap.DebugLevel, cfg.Log.GetLogLevel())
}

func TestGetConfPanicsWhenNil(t *testing.T) {
	t.Cleanup(func() {
		config = nil
	})

	config = nil
	require.Panics(t, func() {
		GetConf()
	})
}

func TestLogGetLogLevelDefaults(t *testing.T) {
	t.Run("known levels", func(t *testing.T) {
		require.Equal(t, zap.InfoLevel, (&Log{Level: "INFO"}).GetLogLevel())
		require.Equal(t, zap.WarnLevel, (&Log{Level: "warn"}).GetLogLevel())
		require.Equal(t, zap.ErrorLevel, (&Log{Level: "ERROR"}).GetLogLevel())
	})

	t.Run("unknown level defaults to info", func(t *testing.T) {
		require.Equal(t, zap.InfoLevel, (&Log{Level: "invalid"}).GetLogLevel())
	})
}
