package hugegraphhubble

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"go.uber.org/zap"
)

func TestConfigureGraphConnectionCreatesConnection(t *testing.T) {
	var postBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, graphConnectionsPath, r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{})
		case http.MethodPost:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&postBody))
			writeHubbleResponse(t, w, http.StatusOK, graphConnection{ID: 1, Name: "upm_hugegraph_example"})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.NoError(t, err)
	require.Equal(t, "upm_hugegraph_example", postBody["name"])
	require.Equal(t, "hugegraph", postBody["graph"])
	require.Equal(t, "hugegraph-svc.example.svc", postBody["host"])
	require.EqualValues(t, 8080, postBody["port"])
}

func TestConfigureGraphConnectionIsIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{Records: []graphConnection{{
			ID:    1,
			Name:  "upm_hugegraph_example",
			Graph: "hugegraph",
			Host:  "hugegraph-svc.example.svc",
			Port:  8080,
		}}})
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.NoError(t, err)
}

func TestConfigureGraphConnectionRejectsConflictingName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{Records: []graphConnection{{
			Name:  "upm_hugegraph_example",
			Graph: "hugegraph",
			Host:  "different-hugegraph-svc.example.svc",
			Port:  8080,
		}}})
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.EqualError(t, err, "graph connection \"upm_hugegraph_example\" already exists with different graph, host, or port")
}

func TestConfigureGraphConnectionReportsHubbleApplicationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":400,"data":null,"message":"Hubble is unavailable"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.EqualError(t, err, "list graph connections: hubble API returned status 400: Hubble is unavailable")
}

func TestValidateConfigureRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *ConfigureGraphConnectionRequest
	}{
		{name: "nil request", req: nil},
		{name: "missing name", req: &ConfigureGraphConnectionRequest{Graph: "hugegraph", Host: "hugegraph-svc", Port: 8080}},
		{name: "invalid port", req: &ConfigureGraphConnectionRequest{Name: "connection", Graph: "hugegraph", Host: "hugegraph-svc"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, validateConfigureRequest(test.req))
		})
	}
}

func TestBackupCreatesAndUploadsLogicalBackup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, graphConnectionsPath, r.URL.Path)
		writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{Records: []graphConnection{{
			ID: 1, Name: "connection", Graph: "hugegraph", Host: "hugegraph.example", Port: 8080,
		}}})
	}))
	defer server.Close()

	service := newTestService(t, server)
	setTestToolsHome(t)
	storage := &testStorage{storeDir: t.TempDir()}
	service.storageFactory = func(*common.ObjectStorage) (common.ObjectStorageFactory, error) { return storage, nil }
	service.runner = commandRunnerFunc(func(_ context.Context, _ string, args ...string) error {
		require.Contains(t, args, "backup")
		directory := commandValue(t, args, "--directory")
		require.NoError(t, os.WriteFile(filepath.Join(directory, "graph.json"), []byte(`{"vertices":1}`), 0o600))
		return nil
	})

	_, err := service.Backup(context.Background(), backupRequest())
	require.NoError(t, err)
	require.Equal(t, "hugegraph/backup.tar.gz", storage.putObject)

	output := t.TempDir()
	require.NoError(t, extractArchive(storage.putDataPath, output))
	data, err := os.ReadFile(filepath.Join(output, backupDirectoryName, "graph.json"))
	require.NoError(t, err)
	require.Equal(t, `{"vertices":1}`, string(data))
}

func TestRestoreRequiresExplicitOverwriteForNonEmptyGraph(t *testing.T) {
	server := newGraphAndHubbleTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]interface{}{"propertykeys": []interface{}{map[string]string{"name": "name"}}})
	})
	defer server.Close()

	service := newTestService(t, server)
	setTestToolsHome(t)
	storage := &testStorage{getDataPath: createTestBackupArchive(t)}
	service.storageFactory = func(*common.ObjectStorage) (common.ObjectStorageFactory, error) { return storage, nil }
	service.runner = commandRunnerFunc(func(context.Context, string, ...string) error {
		t.Fatal("HugeGraph Tools must not run for a non-empty graph without overwrite")
		return nil
	})

	_, err := service.Restore(context.Background(), restoreRequest(false))
	require.EqualError(t, err, "target graph \"hugegraph\" is not empty; set overwrite=true to clear it before restore")
}

func TestRestoreOverwritesAndResetsGraphMode(t *testing.T) {
	server := newGraphAndHubbleTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		kind := filepath.Base(r.URL.Path)
		writeJSON(t, w, map[string]interface{}{kind: []interface{}{}})
	})
	defer server.Close()

	service := newTestService(t, server)
	setTestToolsHome(t)
	storage := &testStorage{getDataPath: createTestBackupArchive(t)}
	service.storageFactory = func(*common.ObjectStorage) (common.ObjectStorageFactory, error) { return storage, nil }
	var commands [][]string
	service.runner = commandRunnerFunc(func(_ context.Context, _ string, args ...string) error {
		commands = append(commands, append([]string(nil), args...))
		return nil
	})

	_, err := service.Restore(context.Background(), restoreRequest(true))
	require.NoError(t, err)
	require.Len(t, commands, 4)
	require.Contains(t, commands[0], "graph-clear")
	require.Contains(t, commands[1], "graph-mode-set")
	require.Equal(t, "RESTORING", commandValue(t, commands[1], "--graph-mode"))
	require.Contains(t, commands[2], "restore")
	require.Contains(t, commands[3], "graph-mode-set")
	require.Equal(t, "NONE", commandValue(t, commands[3], "--graph-mode"))
}

func TestValidateBackupAndRestoreRequests(t *testing.T) {
	request := backupRequest()
	require.NoError(t, validateBackupRequest(request))
	require.NoError(t, validateRestoreRequest(restoreRequest(false)))

	request.BackupFile = "../escape.tar.gz"
	require.EqualError(t, validateBackupRequest(request), "backup_file must be a relative object name")

	request = backupRequest()
	request.TimeoutSeconds = 59
	require.EqualError(t, validateBackupRequest(request), "timeout_seconds must be in [60, 86400]")

	request = backupRequest()
	request.SplitSizeMb = 0
	require.NoError(t, validateBackupRequest(request))
}

func newTestService(t *testing.T, server *httptest.Server) *service {
	t.Helper()
	address := strings.TrimPrefix(server.URL, "http://")
	host, port, err := splitAddress(address)
	require.NoError(t, err)
	return &service{
		logger: zap.NewNop().Sugar(),
		opsCfg: &opsConfig{
			hubbleHost: host,
			hubblePort: port,
		},
		client: server.Client(),
	}
}

func splitAddress(address string) (string, string, error) {
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] != ':' {
			continue
		}
		port, err := strconv.Atoi(address[i+1:])
		if err != nil {
			return "", "", err
		}
		return address[:i], strconv.Itoa(port), nil
	}
	return "", "", strconv.ErrSyntax
}

func writeHubbleResponse(t *testing.T, w http.ResponseWriter, status int, data interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(hubbleResponse{Status: status, Data: mustMarshal(t, data)}))
}

func newGraphAndHubbleTestServer(t *testing.T, graphHandler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == graphConnectionsPath {
			host, port, err := splitAddress(strings.TrimPrefix(r.Host, ""))
			require.NoError(t, err)
			portNumber, err := strconv.Atoi(port)
			require.NoError(t, err)
			writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{Records: []graphConnection{{
				ID: 1, Name: "connection", Graph: "hugegraph", Host: host, Port: int32(portNumber),
			}}})
			return
		}
		graphHandler(w, r)
	}))
}

func writeJSON(t *testing.T, w http.ResponseWriter, value interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}

func backupRequest() *BackupRequest {
	return &BackupRequest{
		ConnectionName: "connection",
		BackupFile:     "hugegraph/backup.tar.gz",
		ObjectStorage: &common.ObjectStorage{
			Endpoint: "minio.example:9000", Bucket: "backups", AccessKey: "access", SecretKey: "secret", Type: common.ObjectStorageType_Minio,
		},
	}
}

func restoreRequest(overwrite bool) *RestoreRequest {
	request := backupRequest()
	return &RestoreRequest{
		ConnectionName: request.ConnectionName,
		BackupFile:     request.BackupFile,
		ObjectStorage:  request.ObjectStorage,
		Overwrite:      overwrite,
	}
}

func setTestToolsHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	tool := filepath.Join(home, "bin", "hugegraph")
	require.NoError(t, os.MkdirAll(filepath.Dir(tool), 0o750))
	require.NoError(t, os.WriteFile(tool, []byte("test"), 0o700))
	t.Setenv(hugeGraphToolsHomeEnv, home)
}

func commandValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for index, value := range args {
		if value == flag && index+1 < len(args) {
			return args[index+1]
		}
	}
	t.Fatalf("flag %q not found in command %v", flag, args)
	return ""
}

func createTestBackupArchive(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	backupDir := filepath.Join(workDir, backupDirectoryName)
	require.NoError(t, os.MkdirAll(backupDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "graph.json"), []byte(`{"vertices":1}`), 0o600))
	archive := filepath.Join(workDir, backupArchiveName)
	require.NoError(t, archiveDirectory(backupDir, archive))
	return archive
}

type testStorage struct {
	putObject   string
	putDataPath string
	getDataPath string
	storeDir    string
}

func (s *testStorage) PutFile(_ context.Context, _ string, object, file string) error {
	s.putObject = object
	s.putDataPath = filepath.Join(s.storeDir, "uploaded.tar.gz")
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return os.WriteFile(s.putDataPath, data, 0o600)
}

func (s *testStorage) GetFile(_ context.Context, _ string, _ string, file string) error {
	data, err := os.ReadFile(s.getDataPath)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o600)
}

func (s *testStorage) PutObject(context.Context, string, string, io.Reader) error {
	return nil
}

func (s *testStorage) GetObject(context.Context, string, string) (io.ReadCloser, error) {
	return nil, os.ErrNotExist
}

func mustMarshal(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}
