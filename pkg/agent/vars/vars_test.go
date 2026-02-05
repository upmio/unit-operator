package vars

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentKeys(t *testing.T) {
	require.Equal(t, "DATA_DIR", DataDirEnvKey)
	require.Equal(t, "DATA_MOUNT", DataMountEnvKey)
	require.Equal(t, "LOG_MOUNT", LogMountEnvKey)
	require.Equal(t, "SECRET_MOUNT", SecretMountEnvKey)
	require.Equal(t, "CONF_DIR", ConfigDirEnvKey)
	require.Equal(t, "ARCH_MODE", ArchModeEnvKey)
	require.Equal(t, "NAMESPACE", NamespaceEnvKey)
	require.Equal(t, "POD_NAME", PodNameEnvKey)
}
