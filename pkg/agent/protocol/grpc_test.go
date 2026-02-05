package protocol

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/conf"
)

func writeTestConfig(t *testing.T, dir string) string {
	content := fmt.Sprintf(`
[log]
level = "info"
dir = "%s"

[app]
host = "127.0.0.1"
port = 0
grpc_host = "127.0.0.1"
grpc_port = 0

[kube]
kubeConfigPath = ""

[supervisor]
address = "127.0.0.1"
port = 19001
`, dir)
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestGrpcServiceStartStop(t *testing.T) {
	dir := t.TempDir()
	path := writeTestConfig(t, dir)
	require.NoError(t, conf.LoadConfigFromToml(path))

	svc := NewGrpcService()

	done := make(chan struct{})
	go func() {
		svc.Start()
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	svc.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("grpc service did not stop")
	}
}
