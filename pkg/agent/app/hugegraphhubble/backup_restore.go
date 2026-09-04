package hugegraphhubble

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
)

const (
	hugeGraphToolsHomeEnv  = "HUGEGRAPH_TOOLS_HOME"
	defaultToolsHome       = "/opt/hugegraph-tools"
	defaultTimeoutSeconds  = 3600
	defaultSplitSizeMB     = 64
	maxTimeoutSeconds      = 24 * 60 * 60
	maxSplitSizeMB         = 1024
	backupDirectoryName    = "backup"
	backupArchiveName      = "backup.tar.gz"
	backupLogDirectoryName = "logs"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type commandRunnerFunc func(ctx context.Context, name string, args ...string) error

func (f commandRunnerFunc) Run(ctx context.Context, name string, args ...string) error {
	return f(ctx, name, args...)
}

func runCommand(ctx context.Context, name string, args ...string) error {
	// Do not log args: HugeGraph Tools accepts the GraphConnection password as
	// an argument and command lines must never disclose it in agent logs.
	cmd := exec.CommandContext(ctx, name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("command %q failed: %w", name, err)
		}
		return fmt.Errorf("command %q failed: %w: %s", name, err, message)
	}
	return nil
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*common.Empty, error) {
	if err := validateBackupRequest(req); err != nil {
		return nil, err
	}
	timeout := requestTimeout(req.GetTimeoutSeconds())
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	s.operationMu.Lock()
	defer s.operationMu.Unlock()

	util.LogRequestSafely(s.logger, "hugegraph backup", map[string]interface{}{
		"connection_name": req.GetConnectionName(),
		"backup_file":     req.GetBackupFile(),
		"bucket":          req.GetObjectStorage().GetBucket(),
		"endpoint":        req.GetObjectStorage().GetEndpoint(),
		"access_key":      req.GetObjectStorage().GetAccessKey(),
		"secret_key":      req.GetObjectStorage().GetSecretKey(),
		"ssl":             req.GetObjectStorage().GetSsl(),
	})

	connection, err := s.graphConnection(ctx, req.GetConnectionName())
	if err != nil {
		return nil, fmt.Errorf("get graph connection: %w", err)
	}

	workDir, err := os.MkdirTemp("", "hugegraph-backup-")
	if err != nil {
		return nil, fmt.Errorf("create backup work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	backupDir := filepath.Join(workDir, backupDirectoryName)
	logDir := filepath.Join(workDir, backupLogDirectoryName)
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return nil, fmt.Errorf("create backup log directory: %w", err)
	}

	if err := s.runTools(ctx, connection, timeout, "backup",
		"--directory", backupDir,
		"--log", logDir,
		"--split-size", strconv.FormatInt(int64(requestSplitSize(req.GetSplitSizeMb()))*1024*1024, 10)); err != nil {
		return nil, fmt.Errorf("create HugeGraph logical backup: %w", err)
	}

	archive := filepath.Join(workDir, backupArchiveName)
	if err := archiveDirectory(backupDir, archive); err != nil {
		return nil, fmt.Errorf("archive HugeGraph backup: %w", err)
	}

	factory, err := s.objectStorageFactory(req.GetObjectStorage())
	if err != nil {
		return nil, err
	}
	if err := factory.PutFile(ctx, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), archive); err != nil {
		return nil, fmt.Errorf("upload HugeGraph backup: %w", err)
	}

	s.logger.Infow("HugeGraph backup completed", "connection_name", connection.Name, "backup_file", req.GetBackupFile())
	return &common.Empty{}, nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	if err := validateRestoreRequest(req); err != nil {
		return nil, err
	}
	timeout := requestTimeout(req.GetTimeoutSeconds())
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	s.operationMu.Lock()
	defer s.operationMu.Unlock()

	util.LogRequestSafely(s.logger, "hugegraph restore", map[string]interface{}{
		"connection_name": req.GetConnectionName(),
		"backup_file":     req.GetBackupFile(),
		"overwrite":       req.GetOverwrite(),
		"bucket":          req.GetObjectStorage().GetBucket(),
		"endpoint":        req.GetObjectStorage().GetEndpoint(),
		"access_key":      req.GetObjectStorage().GetAccessKey(),
		"secret_key":      req.GetObjectStorage().GetSecretKey(),
		"ssl":             req.GetObjectStorage().GetSsl(),
	})

	connection, err := s.graphConnection(ctx, req.GetConnectionName())
	if err != nil {
		return nil, fmt.Errorf("get graph connection: %w", err)
	}

	workDir, err := os.MkdirTemp("", "hugegraph-restore-")
	if err != nil {
		return nil, fmt.Errorf("create restore work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	factory, err := s.objectStorageFactory(req.GetObjectStorage())
	if err != nil {
		return nil, err
	}
	archive := filepath.Join(workDir, backupArchiveName)
	if err := factory.GetFile(ctx, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), archive); err != nil {
		return nil, fmt.Errorf("download HugeGraph backup: %w", err)
	}
	if err := extractArchive(archive, workDir); err != nil {
		return nil, fmt.Errorf("extract HugeGraph backup: %w", err)
	}

	backupDir := filepath.Join(workDir, backupDirectoryName)
	if info, err := os.Stat(backupDir); err != nil || !info.IsDir() {
		if err != nil {
			return nil, fmt.Errorf("backup archive does not contain %q: %w", backupDirectoryName, err)
		}
		return nil, fmt.Errorf("backup archive path %q is not a directory", backupDirectoryName)
	}

	if !req.GetOverwrite() {
		empty, err := s.graphIsEmpty(ctx, connection)
		if err != nil {
			return nil, err
		}
		if !empty {
			return nil, fmt.Errorf("target graph %q is not empty; set overwrite=true to clear it before restore", connection.Graph)
		}
	}

	if req.GetOverwrite() {
		if err := s.runTools(ctx, connection, timeout, "graph-clear",
			"--confirm-message", "I'm sure to delete all data"); err != nil {
			return nil, fmt.Errorf("clear target graph before restore: %w", err)
		}
	}

	if err := s.setGraphMode(ctx, connection, timeout, "RESTORING"); err != nil {
		return nil, fmt.Errorf("set target graph to RESTORING mode: %w", err)
	}

	restoreErr := s.runTools(ctx, connection, timeout, "restore",
		"--directory", backupDir,
		"--log", filepath.Join(workDir, backupLogDirectoryName))
	modeErr := s.setGraphMode(ctx, connection, timeout, "NONE")
	if restoreErr != nil {
		if modeErr != nil {
			return nil, fmt.Errorf("restore HugeGraph backup: %w; reset graph mode: %v", restoreErr, modeErr)
		}
		return nil, fmt.Errorf("restore HugeGraph backup: %w", restoreErr)
	}
	if modeErr != nil {
		return nil, fmt.Errorf("restore completed but failed to reset graph mode: %w", modeErr)
	}

	s.logger.Infow("HugeGraph restore completed", "connection_name", connection.Name, "backup_file", req.GetBackupFile(), "overwrite", req.GetOverwrite())
	return &common.Empty{}, nil
}

func (s *service) objectStorageFactory(storage *common.ObjectStorage) (common.ObjectStorageFactory, error) {
	if s.storageFactory == nil {
		return nil, fmt.Errorf("object storage factory is not configured")
	}
	factory, err := s.storageFactory(storage)
	if err != nil {
		return nil, fmt.Errorf("initialize object storage: %w", err)
	}
	return factory, nil
}

func (s *service) runTools(ctx context.Context, connection graphConnection, timeout int32, operation string, operationArgs ...string) error {
	runner := s.runner
	if runner == nil {
		runner = commandRunnerFunc(runCommand)
	}

	tool := filepath.Join(toolsHome(), "bin", "hugegraph")
	if _, err := os.Stat(tool); err != nil {
		return fmt.Errorf("HugeGraph Tools executable %q is unavailable: %w", tool, err)
	}

	args := []string{
		"--url", "http://" + net.JoinHostPort(connection.Host, strconv.Itoa(int(connection.Port))),
		"--graph", connection.Graph,
		"--timeout", strconv.Itoa(int(timeout)),
	}
	if connection.Username != "" || connection.Password != "" {
		args = append(args, "--user", connection.Username, "--password", connection.Password)
	}
	args = append(args, operation)
	args = append(args, operationArgs...)

	s.logger.Infow("execute HugeGraph Tools", "operation", operation, "connection_name", connection.Name)
	return runner.Run(ctx, tool, args...)
}

func (s *service) setGraphMode(ctx context.Context, connection graphConnection, timeout int32, mode string) error {
	return s.runTools(ctx, connection, timeout, "graph-mode-set", "--graph-mode", mode)
}

func (s *service) graphIsEmpty(ctx context.Context, connection graphConnection) (bool, error) {
	baseURL := "http://" + net.JoinHostPort(connection.Host, strconv.Itoa(int(connection.Port)))
	schemaKinds := []string{"propertykeys", "vertexlabels", "edgelabels", "indexlabels"}
	for _, kind := range schemaKinds {
		endpoint := baseURL + "/graphs/" + url.PathEscape(connection.Graph) + "/schema/" + kind
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return false, fmt.Errorf("create schema check request: %w", err)
		}
		response, err := s.httpClient().Do(request)
		if err != nil {
			return false, fmt.Errorf("check target graph schema: %w", err)
		}
		var result map[string]json.RawMessage
		decodeErr := json.NewDecoder(response.Body).Decode(&result)
		closeErr := response.Body.Close()
		if decodeErr != nil {
			return false, fmt.Errorf("decode schema check response: %w", decodeErr)
		}
		if closeErr != nil {
			return false, fmt.Errorf("close schema check response: %w", closeErr)
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return false, fmt.Errorf("check target graph schema: HugeGraph returned HTTP %d", response.StatusCode)
		}
		items, ok := result[kind]
		if !ok {
			return false, fmt.Errorf("check target graph schema: response has no %q field", kind)
		}
		var values []json.RawMessage
		if err := json.Unmarshal(items, &values); err != nil {
			return false, fmt.Errorf("decode target graph %s: %w", kind, err)
		}
		if len(values) != 0 {
			return false, nil
		}
	}
	return true, nil
}

func (s *service) httpClient() *http.Client {
	if s.client != nil {
		return s.client
	}
	return http.DefaultClient
}

func validateBackupRequest(req *BackupRequest) error {
	if req == nil {
		return fmt.Errorf("backup request must not be nil")
	}
	if err := validateBackupFields(req.GetConnectionName(), req.GetBackupFile(), req.GetObjectStorage()); err != nil {
		return err
	}
	if _, err := validateTimeout(req.GetTimeoutSeconds()); err != nil {
		return err
	}
	if _, err := validateSplitSize(req.GetSplitSizeMb()); err != nil {
		return err
	}
	return nil
}

func validateRestoreRequest(req *RestoreRequest) error {
	if req == nil {
		return fmt.Errorf("restore request must not be nil")
	}
	if err := validateBackupFields(req.GetConnectionName(), req.GetBackupFile(), req.GetObjectStorage()); err != nil {
		return err
	}
	_, err := validateTimeout(req.GetTimeoutSeconds())
	return err
}

func validateBackupFields(connectionName, backupFile string, storage *common.ObjectStorage) error {
	if strings.TrimSpace(connectionName) == "" {
		return fmt.Errorf("connection_name must be set")
	}
	if err := validateObjectName(backupFile); err != nil {
		return err
	}
	if storage == nil {
		return fmt.Errorf("object_storage must be set")
	}
	for field, value := range map[string]string{
		"object_storage.endpoint":   storage.GetEndpoint(),
		"object_storage.bucket":     storage.GetBucket(),
		"object_storage.access_key": storage.GetAccessKey(),
		"object_storage.secret_key": storage.GetSecretKey(),
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be set", field)
		}
	}
	if storage.GetType() != common.ObjectStorageType_Minio {
		return fmt.Errorf("only S3/MinIO object storage is supported")
	}
	return nil
}

func validateObjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("backup_file must be set")
	}
	if path.IsAbs(name) || strings.HasPrefix(path.Clean(name), "../") || path.Clean(name) == ".." {
		return fmt.Errorf("backup_file must be a relative object name")
	}
	return nil
}

func validateTimeout(value int32) (int32, error) {
	if value == 0 {
		return defaultTimeoutSeconds, nil
	}
	if value < 60 || value > maxTimeoutSeconds {
		return 0, fmt.Errorf("timeout_seconds must be in [60, %d]", maxTimeoutSeconds)
	}
	return value, nil
}

func validateSplitSize(value int32) (int32, error) {
	if value == 0 {
		return defaultSplitSizeMB, nil
	}
	if value < 1 || value > maxSplitSizeMB {
		return 0, fmt.Errorf("split_size_mb must be in [1, %d]", maxSplitSizeMB)
	}
	return value, nil
}

func requestTimeout(value int32) int32 {
	timeout, _ := validateTimeout(value)
	return timeout
}

func requestSplitSize(value int32) int32 {
	splitSize, _ := validateSplitSize(value)
	return splitSize
}

func toolsHome() string {
	if home := strings.TrimSpace(os.Getenv(hugeGraphToolsHomeEnv)); home != "" {
		return home
	}
	return defaultToolsHome
}
