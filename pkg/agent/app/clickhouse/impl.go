package clickhouse

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	clickHouseHostEnvKey   = "CLICKHOUSE_HOST"
	clickHousePortEnvKey   = "CLICKHOUSE_PORT"
	clickHouseSecureEnvKey = "CLICKHOUSE_SECURE"
)

var (
	svr = &service{}

	identifierRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
)

type commandRunner interface {
	ExecuteCommand(cmd *exec.Cmd, logPrefix string) error
}

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
	s.runner = common.NewCommandExecutor(s.logger)

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

	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	if err := validateIdentifier(req.GetUsername()); err != nil {
		s.logger.Errorw("invalid clickhouse username", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	query, err := buildBackupSQL(req.GetObjectStorage(), req.GetBackupFile())
	if err != nil {
		s.logger.Errorw("failed to build backup query", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	if err := runClickHouseQuery(ctx, s.runner, readClickHouseConnection(), req.GetUsername(), password, query); err != nil {
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

	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	if err := validateIdentifier(req.GetUsername()); err != nil {
		s.logger.Errorw("invalid clickhouse username", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	query, err := buildRestoreSQL(req.GetObjectStorage(), req.GetBackupFile())
	if err != nil {
		s.logger.Errorw("failed to build restore query", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	if err := runClickHouseQuery(ctx, s.runner, readClickHouseConnection(), req.GetUsername(), password, query); err != nil {
		s.logger.Errorw("failed to execute restore", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore clickhouse successfully")
	return nil, nil
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

	return fmt.Sprintf("BACKUP ALL TO S3(%s, %s, %s)",
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

	return fmt.Sprintf("RESTORE ALL FROM S3(%s, %s, %s)",
		quoteSQLString(s3URL),
		quoteSQLString(objectStorage.GetAccessKey()),
		quoteSQLString(objectStorage.GetSecretKey()),
	), nil
}

func buildS3URL(objectStorage *common.ObjectStorage, backupFile string) (string, error) {
	if objectStorage == nil {
		return "", fmt.Errorf("object_storage is required")
	}
	if objectStorage.GetEndpoint() == "" {
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

	parsed, err := url.Parse(objectStorage.GetEndpoint())
	if err != nil {
		return "", err
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
		"--password", password,
	}
	if conn.secure {
		args = append(args, "--secure")
	}
	args = append(args, "--query", query)

	cmd := exec.CommandContext(ctx, "clickhouse-client", args...)
	return runner.ExecuteCommand(cmd, "clickhouse")
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
