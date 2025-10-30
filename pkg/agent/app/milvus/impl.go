package milvus

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/caarlos0/env/v9"
)

//go:embed backup.tmpl
var configTemplate string

type Config struct {
	LogMount    string `env:"LOG_MOUNT,required"`
	SecretMount string `env:"SECRET_MOUNT,required"`

	MilvusPort     string `env:"MILVUS_PORT,required"`
	MilvusUser     string `env:"MILVUS_USER,required"`
	MilvusPassword string

	MinioAddress  string `env:"MINIO_ADDR,required"`
	MinioPort     string `env:"MINIO_PORT,required"`
	MinioUser     string `env:"MINIO_USER,required"`
	MinioPassword string
	MinioBucket   string `env:"MINIO_BUCKET,required"`

	BackupAddress  string
	BackupPort     string
	BackupUser     string
	BackupPassword string
	BackupBucket   string

	BackupRootPath string
}

var (
	// service instance
	svr = &service{}
)

type service struct {
	milvusOps MilvusOperationServer
	UnimplementedMilvusOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

// newMilvusResponse creates a new Mivlus Response with the given message
func newMilvusResponse(message string) *Response {
	return &Response{Message: message}
}

func (s *service) Config() error {
	s.milvusOps = app.GetGrpcApp(appName).(MilvusOperationServer)
	s.logger = zap.L().Named("[MILVUS]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterMilvusOperationServer(server, svr)
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*Response, error) {
	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to check service status", err)
	}

	//2. generate backup.yaml config
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to parse env config, %v", err)
	}

	var err error
	cfg.BackupBucket = req.GetS3Storage().GetBucket()
	cfg.BackupUser = req.GetS3Storage().GetAccessKey()
	cfg.BackupPassword = req.GetS3Storage().GetSecretKey()
	cfg.BackupRootPath = req.GetBackupRootPath()
	cfg.BackupAddress, cfg.BackupPort, err = splitHostPortByEndpoint(req.GetS3Storage().GetEndpoint())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to split host port, %v", err)
	}

	cfg.MinioPassword, err = decryptPassword(cfg.SecretMount, cfg.MinioUser)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to decrypt password, %v", err)
	}

	cfg.MilvusPassword, err = decryptPassword(cfg.SecretMount, cfg.MilvusUser)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to decrypt password, %v", err)
	}

	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to parse config template, %v", err)
	}

	f, _ := os.Create("/tmp/backup.yaml")
	defer func() {
		_ = f.Close()
	}()

	if err := tmpl.Execute(f, cfg); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to generate config file, %v", err)
	}

	//3. execute milvus-backup command
	path, err := exec.LookPath("milvus-backup")
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to found milvus-backup command, %v", err)
	}

	cmd := exec.CommandContext(ctx,
		path,
		"--config",
		"/tmp/backup.yaml",
		"create",
		"-n",
		req.GetBackupFile(),
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.logger.Error(err)
	}

	s.logger.Info("start execute command...")

	if err := cmd.Start(); err != nil {
		s.logger.Error(err)
	}

	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		s.logger.Error(err)
	}

	s.logger.Info("wait command to finish")
	if err := cmd.Wait(); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, fmt.Sprintf("failed to execute milvus-backup: %v %s", err, string(stderrBytes)), nil)
	}

	return common.LogAndReturnSuccess(s.logger, newMilvusResponse, "milvus backup success")
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*Response, error) {
	// 1. Check if service is stopped
	if _, err := s.slm.CheckServiceStopped(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "service status check failed", err)
	}

	//2. generate backup.yaml config
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to parse env config, %v", err)
	}

	var err error
	cfg.BackupBucket = req.GetS3Storage().GetBucket()
	cfg.BackupUser = req.GetS3Storage().GetAccessKey()
	cfg.BackupPassword = req.GetS3Storage().GetSecretKey()
	cfg.BackupRootPath = req.GetBackupRootPath()
	cfg.BackupAddress, cfg.BackupPort, err = splitHostPortByEndpoint(req.GetS3Storage().GetEndpoint())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to split host port, %v", err)
	}

	cfg.MinioPassword, err = decryptPassword(cfg.SecretMount, cfg.MinioUser)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to decrypt password, %v", err)
	}

	cfg.MilvusPassword, err = decryptPassword(cfg.SecretMount, cfg.MilvusUser)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to decrypt password, %v", err)
	}

	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to parse config template, %v", err)
	}

	f, _ := os.Create("/tmp/backup.yaml")
	defer func() {
		_ = f.Close()
	}()

	if err := tmpl.Execute(f, cfg); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to generate config file, %v", err)
	}

	//3. execute milvus-backup command
	path, err := exec.LookPath("milvus-backup")
	if err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, "failed to found milvus-backup command, %v", err)
	}

	var cmd *exec.Cmd

	if req.GetSuffix() == "" {
		cmd = exec.CommandContext(ctx,
			path,
			"--config",
			"/tmp/backup.yaml",
			"restore",
			"--restore_index",
			"-n",
			req.GetBackupFile(),
		)
	} else {
		cmd = exec.CommandContext(ctx,
			path,
			"--config",
			"/tmp/backup.yaml",
			"restore",
			"--restore_index",
			"-n",
			req.GetBackupFile(),
			"-s",
			req.GetSuffix(),
		)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.logger.Error(err)
	}

	s.logger.Info("start execute command...")

	if err := cmd.Start(); err != nil {
		s.logger.Error(err)
	}

	stderrBytes, err := io.ReadAll(stderr)
	if err != nil {
		s.logger.Error(err)
	}

	s.logger.Info("wait command to finish")
	if err := cmd.Wait(); err != nil {
		return common.LogAndReturnError(s.logger, newMilvusResponse, fmt.Sprintf("failed to execute milvus-backup: %v %s", err, string(stderrBytes)), nil)
	}

	return common.LogAndReturnSuccess(s.logger, newMilvusResponse, "milvus restore success")
}

func splitHostPortByEndpoint(url string) (string, string, error) {
	var endpoint string
	if strings.HasPrefix(url, "https://") {
		endpoint = strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "http://") {
		endpoint = strings.TrimPrefix(url, "http://")
	}

	return net.SplitHostPort(endpoint)
}

func decryptPassword(dirName, fileName string) (string, error) {
	content, err := os.ReadFile(filepath.Join(dirName, fileName))
	if err != nil {
		return "", err
	}

	plaintext, err := util.AES_CTR_Decrypt(content)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
