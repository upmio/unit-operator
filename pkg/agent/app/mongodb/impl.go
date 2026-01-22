package mongodb

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	milvusOps MongoDBOperationServer
	UnimplementedMongoDBOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

// newMongoDBResponse creates a new MongoDB Response with the given message
func newMongoDBResponse(message string) *Response {
	return &Response{Message: message}
}

// getEnvVarOrError gets environment variable or returns error if not found
func getEnvVarOrError(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}

func (s *service) Config() error {
	s.milvusOps = app.GetGrpcApp(appName).(MongoDBOperationServer)
	s.logger = zap.L().Named("[MONGODB]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterMongoDBOperationServer(server, svr)
}

func (s *service) streamCommandStdoutToS3(ctx context.Context, cmd *exec.Cmd, storageFactory common.S3Storage, bucketName, objectName string) error {
	logDir := os.Getenv(vars.LogMountEnvKey)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", cmd.Args[0], "backup"))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		s.logger.Errorf("failed to write %s stderr to file %s: %v", cmd.Args[0], logPath, err)
		return err
	}
	defer func() {
		_ = logFile.Close()
	}()

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return err
	}

	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
	}()

	err = storageFactory.StreamToS3(ctx, bucketName, objectName, pr)

	_ = pr.Close()

	return err
}

func (s *service) streamCommandStdinFromS3(ctx context.Context, cmd *exec.Cmd, storageFactory common.S3Storage, bucketName, objectName string) error {
	logDir := os.Getenv(vars.LogMountEnvKey)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", cmd.Args[0], "restore"))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		s.logger.Errorf("failed to write %s stderr to file %s: %v", cmd.Args[0], logPath, err)
		return err
	}
	defer func() {
		_ = logFile.Close()
	}()

	pr, pw := io.Pipe()
	cmd.Stdin = pr
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return err
	}

	obj, err := storageFactory.StreamFromS3(ctx, bucketName, objectName)
	if err != nil {
		return err
	}
	defer func() { _ = obj.Close() }()

	go func() {
		_, _ = io.Copy(pw, obj)
		_ = pw.Close()
	}()

	err = cmd.Wait()
	_ = pr.Close()
	return err
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*Response, error) {
	tool := "mongodump"

	_, err := exec.LookPath(tool)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "mongodump command is not installed or not in PATH", err)
	}

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "failed to check service status", err)
	}

	// Example:
	// mongodump --uri="mongodb://admin:xxxxxxx@ \
	//                  demo-mongodb-6rn-0.demo-mongodb-6rn-headless-svc.demo:27017, \
	//                  demo-mongodb-6rn-1.demo-mongodb-6rn-headless-svc.demo:27017, \
	//                  demo-mongodb-6rn-2.demo-mongodb-6rn-headless-svc.demo:27017/ \
	//                  ?replicaSet=demo&readPreference=secondaryPreferred" \
	//                  --oplog --gzip --archive=/backup/mongodb-backup-xxxxxxxx --numParallelCollections=4

	cmd := exec.CommandContext(ctx,
		tool,
		fmt.Sprintf("--uri=\"mongodb://%s:%s@%s?replicaSet=%s&readPreference=secondaryPreferred\"", req.GetUsername(), req.GetPassword(), req.GetUri(), req.GetReplicaSetName()),
		"--oplog",
		"--gzip",
		"--archive",
		"--numParallelCollections=4",
	)

	var storageFactory common.S3Storage
	if req.GetS3Storage() != nil {
		switch req.GetS3Storage().GetType() {
		case S3StorageType_Minio:
			storageFactory, err = common.NewMinioClient(req.GetS3Storage().GetEndpoint(), req.GetS3Storage().GetAccessKey(), req.GetS3Storage().GetSecretKey(), req.GetS3Storage().GetSsl())
			if err != nil {
				return common.LogAndReturnError(s.logger, newMongoDBResponse, "failed to create s3 client", err)
			}
		default:
			return common.LogAndReturnError(s.logger, newMongoDBResponse, "unsupported s3 storage type", nil)
		}
	}

	if err := s.streamCommandStdoutToS3(ctx, cmd, storageFactory, req.GetS3Storage().GetBucket(), req.GetBackupFile()); err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "mongodump command execution failed", err)
	}

	return common.LogAndReturnSuccess(s.logger, newMongoDBResponse, "mongodb backup success")
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*Response, error) {
	path, err := exec.LookPath("mongorestore")
	if err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "mongorestore command is not installed or not in PATH", err)
	}

	// 1. Check if service is stopped
	if _, err := s.slm.CheckServiceStopped(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "service status check failed", err)
	}

	// Use command executor for commands
	executor := common.NewCommandExecutor(ctx, s.logger)

	// Example:
	// mongorestore --uri="mongodb://admin:xxxxxxx@demo-mongodb-6rn-0.demo-mongodb-6rn-headless-svc.demo:27017/?replicaSet=demo" \
	// 					--drop \
	// 					--gzip \
	// 					--archive

	port, err := getEnvVarOrError("MONGODB_PORT")
	if err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "failed to get MONGODB_PORT environment variable", err)
	}

	cmd := exec.CommandContext(ctx,
		path,
		fmt.Sprintf("--uri=\"mongodb://%s:%s@127.0.0.1:%s?replicaSet=%s\"", req.GetUsername(), req.GetPassword(), port, req.GetReplicaSetName()),
		"--drop",
		"--gzip",
		"--archive",
	)

	// Use command executor for single command
	if err := executor.ExecuteCommand(cmd, "restore"); err != nil {
		return common.LogAndReturnError(s.logger, newMongoDBResponse, "mongorestore command execution failed", err)
	}

	return common.LogAndReturnSuccess(s.logger, newMongoDBResponse, "mongodb restore success")
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
