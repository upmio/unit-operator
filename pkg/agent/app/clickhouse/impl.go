package clickhouse

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

const (
	clickHouseHostEnvKey    = "CLICKHOUSE_HOST"
	clickHousePortEnvKey    = "CLICKHOUSE_PORT"
	clickHouseSecureEnvKey  = "CLICKHOUSE_SECURE"
	clickHouseDataDirEnvKey = "CLICKHOUSE_DATA_DIR"

	clickHouseBackupBinary = "clickhouse-backup"
	clickHouseVersionQuery = "SELECT version() FORMAT TSVRaw"
)

var (
	svr = &service{}

	identifierRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

type commandRunner interface {
	ExecuteCommand(cmd *exec.Cmd, logPrefix string) error
}

// commandOutputRunner is intentionally separate from commandRunner. Backup commands
// never expose output because they can contain object-storage paths or credentials;
// the version probe is the only command for which stdout is consumed.
type commandOutputRunner interface {
	ExecuteCommandOutput(cmd *exec.Cmd, logPrefix string) (string, error)
}

type backupOperation string

const (
	backupOperationCreate  backupOperation = "create"
	backupOperationRestore backupOperation = "restore"
)

type backupDriver interface {
	Execute(context.Context, commandRunner, clickHouseConnection, string, string, *common.ObjectStorage, string,
		backupOperation) error
}

type nativeBackupDriver struct{}

type clickHouseBackupDriver struct{}

type clickHouseConnection struct {
	host   string
	port   string
	secure bool
}

type service struct {
	clickHouseOps ClickHouseOperationServer
	UnimplementedClickHouseOperationServer
	logger *zap.SugaredLogger

	slm    slm.ServiceLifecycleServer
	runner commandRunner
}

func (s *service) Config() error {
	s.clickHouseOps = app.GetGrpcApp(appName).(ClickHouseOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)
	s.runner = &safeCommandRunner{logger: s.logger}

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterClickHouseOperationServer(server, svr)
}

func (s *service) LogicalBackup(ctx context.Context, req *LogicalBackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "clickhouse logical backup", map[string]interface{}{
		"username":    req.GetUsername(),
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetObjectStorage().GetBucket(),
		"endpoint":    req.GetObjectStorage().GetEndpoint(),
		"access_key":  req.GetObjectStorage().GetAccessKey(),
		"secret_key":  req.GetObjectStorage().GetSecretKey(),
	})

	if err := s.executeBackupOperation(ctx, req.GetUsername(), req.GetObjectStorage(), req.GetBackupFile(),
		backupOperationCreate); err != nil {
		s.logger.Errorw("failed to execute backup", zap.Error(err))
		return nil, err
	}

	s.logger.Info("logical backup clickhouse successfully")
	return nil, nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "clickhouse restore", map[string]interface{}{
		"username":    req.GetUsername(),
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetObjectStorage().GetBucket(),
		"endpoint":    req.GetObjectStorage().GetEndpoint(),
		"access_key":  req.GetObjectStorage().GetAccessKey(),
		"secret_key":  req.GetObjectStorage().GetSecretKey(),
	})

	if err := s.executeBackupOperation(ctx, req.GetUsername(), req.GetObjectStorage(), req.GetBackupFile(),
		backupOperationRestore); err != nil {
		s.logger.Errorw("failed to execute restore", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore clickhouse successfully")
	return nil, nil
}

func (s *service) executeBackupOperation(ctx context.Context, username string, objectStorage *common.ObjectStorage,
	backupFile string, operation backupOperation) error {
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		return err
	}
	if err := validateIdentifier(username); err != nil {
		return err
	}

	password, err := util.DecryptPlainTextPassword(username)
	if err != nil {
		return err
	}
	conn := readClickHouseConnection()
	version, err := detectClickHouseVersion(ctx, s.runner, conn, username, password)
	if err != nil {
		return err
	}
	driver, err := selectBackupDriver(version)
	if err != nil {
		return err
	}

	s.logger.Infow("selected ClickHouse backup driver", "version", version, "driver", fmt.Sprintf("%T", driver))
	return driver.Execute(ctx, s.runner, conn, username, password, objectStorage, backupFile, operation)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "clickhouse set variable", map[string]interface{}{
		"username": req.GetUsername(),
		"key":      req.GetKey(),
		"value":    req.GetValue(),
	})

	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	if err := validateIdentifier(req.GetUsername()); err != nil {
		s.logger.Errorw("invalid clickhouse username", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	query, err := buildSetVariableSQL(req.GetUsername(), req.GetKey(), req.GetValue())
	if err != nil {
		s.logger.Errorw("failed to build set variable query", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	if err := runClickHouseQuery(ctx, s.runner, readClickHouseConnection(), req.GetUsername(), password, query); err != nil {
		s.logger.Errorw("failed to execute set variable", zap.Error(err))
		return nil, err
	}

	s.logger.Info("set variable clickhouse successfully")
	return nil, nil
}

func detectClickHouseVersion(ctx context.Context, runner commandRunner, conn clickHouseConnection, username,
	password string) (string, error) {
	outputRunner, ok := runner.(commandOutputRunner)
	if !ok {
		return "", fmt.Errorf("clickhouse command runner does not support version detection")
	}

	args := []string{
		"--host", conn.host,
		"--port", conn.port,
		"--user", username,
		"--query", clickHouseVersionQuery,
	}
	if conn.secure {
		args = append(args, "--secure")
	}
	cmd := exec.CommandContext(ctx, "clickhouse-client", args...)
	cmd.Env = append(cmd.Environ(), "CLICKHOUSE_PASSWORD="+password)
	output, err := outputRunner.ExecuteCommandOutput(cmd, "clickhouse-version")
	if err != nil {
		return "", fmt.Errorf("detect ClickHouse version: %w", err)
	}

	version := strings.TrimSpace(output)
	if !regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(version) {
		return "", fmt.Errorf("invalid ClickHouse version returned by server: %q", version)
	}
	return version, nil
}

func selectBackupDriver(version string) (backupDriver, error) {
	switch {
	case strings.HasPrefix(version, "21.8."):
		return clickHouseBackupDriver{}, nil
	case strings.HasPrefix(version, "23.8."), strings.HasPrefix(version, "25.8."), strings.HasPrefix(version, "26.3."):
		return nativeBackupDriver{}, nil
	default:
		return nil, fmt.Errorf("ClickHouse version %s is not certified for backup or restore", version)
	}
}

func (nativeBackupDriver) Execute(ctx context.Context, runner commandRunner, conn clickHouseConnection, username,
	password string, objectStorage *common.ObjectStorage, backupFile string, operation backupOperation) error {
	var (
		query string
		err   error
	)
	switch operation {
	case backupOperationCreate:
		query, err = buildBackupSQL(objectStorage, backupFile)
	case backupOperationRestore:
		query, err = buildRestoreSQL(objectStorage, backupFile)
	default:
		return fmt.Errorf("unsupported backup operation %q", operation)
	}
	if err != nil {
		return err
	}
	return runClickHouseQuery(ctx, runner, conn, username, password, query)
}

func (clickHouseBackupDriver) Execute(ctx context.Context, runner commandRunner, conn clickHouseConnection, username,
	password string, objectStorage *common.ObjectStorage, backupFile string, operation backupOperation) error {
	if err := verifyClickHouseDataDir(); err != nil {
		return err
	}

	configPath, backupName, err := writeClickHouseBackupConfig(conn, username, password, objectStorage, backupFile)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(configPath) }()

	command := "create_remote"
	if operation == backupOperationRestore {
		command = "restore_remote"
	}
	if operation != backupOperationCreate && operation != backupOperationRestore {
		return fmt.Errorf("unsupported backup operation %q", operation)
	}

	cmd := exec.CommandContext(ctx, clickHouseBackupBinary, "--config", configPath, command, backupName)
	return runner.ExecuteCommand(cmd, "clickhouse-backup")
}

func verifyClickHouseDataDir() error {
	dataDir := strings.TrimSpace(os.Getenv(clickHouseDataDirEnvKey))
	if dataDir == "" {
		return fmt.Errorf("%s is required for ClickHouse 21.8 backup", clickHouseDataDirEnvKey)
	}
	info, err := os.Stat(dataDir)
	if err != nil {
		return fmt.Errorf("ClickHouse data directory %q is not accessible: %w", dataDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("ClickHouse data directory %q is not a directory", dataDir)
	}
	return nil
}

type clickHouseBackupConfig struct {
	General struct {
		RemoteStorage      string `yaml:"remote_storage"`
		DisableProgressBar bool   `yaml:"disable_progress_bar"`
	} `yaml:"general"`
	ClickHouse struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Host     string `yaml:"host"`
		Port     uint   `yaml:"port"`
		Secure   bool   `yaml:"secure"`
	} `yaml:"clickhouse"`
	S3 struct {
		Endpoint       string `yaml:"endpoint"`
		AccessKey      string `yaml:"access_key"`
		SecretKey      string `yaml:"secret_key"`
		Bucket         string `yaml:"bucket"`
		Path           string `yaml:"path"`
		ForcePathStyle bool   `yaml:"force_path_style"`
	} `yaml:"s3"`
}

func writeClickHouseBackupConfig(conn clickHouseConnection, username, password string,
	objectStorage *common.ObjectStorage, backupFile string) (string, string, error) {
	endpoint, err := normalizeS3Endpoint(objectStorage)
	if err != nil {
		return "", "", err
	}
	backupDir, backupName, err := splitBackupFile(backupFile)
	if err != nil {
		return "", "", err
	}
	port, err := strconv.ParseUint(conn.port, 10, 16)
	if err != nil || port == 0 {
		return "", "", fmt.Errorf("invalid ClickHouse port %q", conn.port)
	}

	config := clickHouseBackupConfig{}
	config.General.RemoteStorage = "s3"
	config.General.DisableProgressBar = true
	config.ClickHouse.Username = username
	config.ClickHouse.Password = password
	config.ClickHouse.Host = conn.host
	config.ClickHouse.Port = uint(port)
	config.ClickHouse.Secure = conn.secure
	config.S3.Endpoint = endpoint.String()
	config.S3.AccessKey = objectStorage.GetAccessKey()
	config.S3.SecretKey = objectStorage.GetSecretKey()
	config.S3.Bucket = objectStorage.GetBucket()
	config.S3.Path = path.Join(strings.Trim(endpoint.Path, "/"), backupDir)
	if config.S3.Path == "." {
		config.S3.Path = ""
	}
	config.S3.ForcePathStyle = true

	// The endpoint path is represented by s3.path. Sending it in endpoint as
	// well would make MinIO and other S3-compatible gateways double-prefix it.
	endpoint.Path = ""
	config.S3.Endpoint = endpoint.String()

	contents, err := yaml.Marshal(config)
	if err != nil {
		return "", "", fmt.Errorf("marshal clickhouse-backup config: %w", err)
	}
	file, err := os.CreateTemp("", "clickhouse-backup-*.yaml")
	if err != nil {
		return "", "", fmt.Errorf("create clickhouse-backup config: %w", err)
	}
	path := file.Name()
	defer func() {
		if err != nil {
			_ = file.Close()
			_ = os.Remove(path)
		}
	}()
	if err = file.Chmod(0600); err != nil {
		return "", "", fmt.Errorf("set clickhouse-backup config permissions: %w", err)
	}
	if _, err = file.Write(contents); err != nil {
		return "", "", fmt.Errorf("write clickhouse-backup config: %w", err)
	}
	if err = file.Close(); err != nil {
		return "", "", fmt.Errorf("close clickhouse-backup config: %w", err)
	}
	return path, backupName, nil
}

func splitBackupFile(backupFile string) (string, string, error) {
	cleaned := path.Clean(strings.TrimSpace(backupFile))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", "", fmt.Errorf("backup_file must be a relative object-storage path")
	}
	return path.Dir(cleaned), path.Base(cleaned), nil
}

func normalizeS3Endpoint(objectStorage *common.ObjectStorage) (*url.URL, error) {
	if objectStorage == nil {
		return nil, fmt.Errorf("object_storage is required")
	}
	if strings.TrimSpace(objectStorage.GetEndpoint()) == "" {
		return nil, fmt.Errorf("object_storage.endpoint is required")
	}
	if objectStorage.GetBucket() == "" {
		return nil, fmt.Errorf("object_storage.bucket is required")
	}
	if objectStorage.GetAccessKey() == "" {
		return nil, fmt.Errorf("object_storage.access_key is required")
	}
	if objectStorage.GetSecretKey() == "" {
		return nil, fmt.Errorf("object_storage.secret_key is required")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(objectStorage.GetEndpoint()), "/")
	if !strings.Contains(endpoint, "://") {
		if objectStorage.GetSsl() {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse object_storage.endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("object_storage.endpoint must include a host")
	}
	return parsed, nil
}

func readClickHouseConnection() clickHouseConnection {
	host := os.Getenv(clickHouseHostEnvKey)
	if host == "" {
		host = "127.0.0.1"
	}

	port := os.Getenv(clickHousePortEnvKey)
	if port == "" {
		port = "9000"
	}

	return clickHouseConnection{
		host:   host,
		port:   port,
		secure: strings.EqualFold(os.Getenv(clickHouseSecureEnvKey), "true"),
	}
}

func buildBackupSQL(objectStorage *common.ObjectStorage, backupFile string) (string, error) {
	s3URL, err := buildS3URL(objectStorage, backupFile)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("BACKUP ALL EXCEPT DATABASES system, INFORMATION_SCHEMA, information_schema TO S3(%s, %s, %s)",
		quoteSQLString(s3URL),
		quoteSQLString(objectStorage.GetAccessKey()),
		quoteSQLString(objectStorage.GetSecretKey()),
	), nil
}

func buildRestoreSQL(objectStorage *common.ObjectStorage, backupFile string) (string, error) {
	s3URL, err := buildS3URL(objectStorage, backupFile)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("RESTORE ALL EXCEPT DATABASES system, INFORMATION_SCHEMA, information_schema FROM S3(%s, %s, %s)",
		quoteSQLString(s3URL),
		quoteSQLString(objectStorage.GetAccessKey()),
		quoteSQLString(objectStorage.GetSecretKey()),
	), nil
}

func buildS3URL(objectStorage *common.ObjectStorage, backupFile string) (string, error) {
	if objectStorage == nil {
		return "", fmt.Errorf("object_storage is required")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(objectStorage.GetEndpoint()), "/")
	if endpoint == "" {
		return "", fmt.Errorf("object_storage.endpoint is required")
	}
	if objectStorage.GetBucket() == "" {
		return "", fmt.Errorf("object_storage.bucket is required")
	}
	if backupFile == "" {
		return "", fmt.Errorf("backup_file is required")
	}
	if objectStorage.GetAccessKey() == "" {
		return "", fmt.Errorf("object_storage.access_key is required")
	}
	if objectStorage.GetSecretKey() == "" {
		return "", fmt.Errorf("object_storage.secret_key is required")
	}

	if !strings.Contains(endpoint, "://") {
		if objectStorage.GetSsl() {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse object_storage.endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Hostname() == "" {
		return "", fmt.Errorf("object_storage.endpoint must include a host")
	}

	segments := []string{
		strings.TrimRight(parsed.Path, "/"),
		strings.Trim(objectStorage.GetBucket(), "/"),
		strings.TrimLeft(backupFile, "/"),
	}
	parsed.Path = strings.Join(nonEmptySegments(segments), "/")

	return parsed.String(), nil
}

func nonEmptySegments(segments []string) []string {
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			out = append(out, segment)
		}
	}
	return out
}

func buildSetVariableSQL(username, key, value string) (string, error) {
	if err := validateIdentifier(username); err != nil {
		return "", err
	}
	if err := validateIdentifier(key); err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("value is required")
	}

	return fmt.Sprintf("ALTER USER %s SETTINGS %s = %s", username, key, quoteSQLString(value)), nil
}

func validateIdentifier(value string) error {
	if !identifierRE.MatchString(value) {
		return fmt.Errorf("invalid identifier %q", value)
	}

	return nil
}

func quoteSQLString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

func runClickHouseQuery(ctx context.Context, runner commandRunner, conn clickHouseConnection, username, password, query string) error {
	args := []string{
		"--host", conn.host,
		"--port", conn.port,
		"--user", username,
	}
	if conn.secure {
		args = append(args, "--secure")
	}

	cmd := exec.CommandContext(ctx, "clickhouse-client", args...)
	cmd.Env = append(cmd.Environ(), "CLICKHOUSE_PASSWORD="+password)
	cmd.Stdin = strings.NewReader(query)
	return runner.ExecuteCommand(cmd, "clickhouse")
}

type safeCommandRunner struct {
	logger *zap.SugaredLogger
}

func (r *safeCommandRunner) ExecuteCommand(cmd *exec.Cmd, _ string) error {
	_, err := r.execute(cmd, false)
	return err
}

func (r *safeCommandRunner) ExecuteCommandOutput(cmd *exec.Cmd, _ string) (string, error) {
	return r.execute(cmd, true)
}

func (r *safeCommandRunner) execute(cmd *exec.Cmd, captureStdout bool) (string, error) {
	var stdout bytes.Buffer
	cmd.Stderr = io.Discard
	if captureStdout {
		cmd.Stdout = &stdout
	} else {
		cmd.Stdout = io.Discard
	}

	if r.logger != nil {
		r.logger.Infof("starting command: %s", strings.Join(cmd.Args, " "))
	}

	if err := cmd.Run(); err != nil {
		// Do not append process stderr here. ClickHouse includes the submitted SQL
		// in some errors, and that SQL can contain S3 credentials. The command exit
		// status still distinguishes a client/runtime failure from a failed backup.
		return "", fmt.Errorf("command failed: %w", err)
	}
	return stdout.String(), nil
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
