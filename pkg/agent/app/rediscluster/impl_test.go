package rediscluster

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/vars"
)

func TestDaemonName(t *testing.T) {
	d := &daemon{}
	require.Equal(t, appName, d.Name())
}

func TestConfigMissingEnv(t *testing.T) {
	d := &daemon{}

	t.Setenv(vars.ConfigDirEnvKey, "")
	t.Setenv(vars.NamespaceEnvKey, "")
	t.Setenv(vars.PodNameEnvKey, "")

	err := d.Config()
	require.Error(t, err)
}

func TestConfigMissingNamespace(t *testing.T) {
	d := &daemon{}

	t.Setenv(vars.ConfigDirEnvKey, "/tmp")
	t.Setenv(vars.NamespaceEnvKey, "")
	t.Setenv(vars.PodNameEnvKey, "")

	err := d.Config()
	require.Error(t, err)
}
