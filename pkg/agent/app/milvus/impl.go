package milvus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"

	"github.com/caarlos0/env/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	milvusBackupConfFile = "/tmp/backup.yaml"
)

//go:embed backup.tmpl
var configTemplate string

type OpsConfig struct {
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
	BackupSSL      bool

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

	opsCfg         *OpsConfig
	clientSet      kubernetes.Interface
	namespace      string
	etcdMemberList string
	rootPath       string
	certDir        string
}

func (s *service) Config() error {
	s.milvusOps = app.GetGrpcApp(appName).(MilvusOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)

	var cfg OpsConfig

	if err := env.Parse(&cfg); err != nil {
		return err
	}

	s.opsCfg = &cfg

	clientSet, err := conf.GetConf().Kube.GetClientSet()
	if err != nil {
		return err
	}

	s.clientSet = clientSet

	namespace, err := util.IsEnvVarSet(vars.NamespaceEnvKey)
	if err != nil {
		return err
	}

	etcdMemberList, err := util.IsEnvVarSet(vars.EtcdMemberListEnvKey)
	if err != nil {
		return err
	}

	serviceGroupName, err := util.IsEnvVarSet(vars.ServiceGroupNameEnvKey)
	if err != nil {
		return err
	}

	certDir, err := util.IsEnvVarSet(vars.CertMountEnvKey)
	if err != nil {
		return err

	}

	s.namespace = namespace
	s.etcdMemberList = etcdMemberList
	s.rootPath = serviceGroupName
	s.certDir = certDir

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterMilvusOperationServer(server, svr)
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "milvus backup", map[string]interface{}{
		"backup_root_path": req.GetBackupRootPath(),
		"backup_file":      req.GetBackupFile(),
		"bucket":           req.GetObjectStorage().GetBucket(),
		"endpoint":         req.GetObjectStorage().GetEndpoint(),
		"access_key":       req.GetObjectStorage().GetAccessKey(),
		"secret_key":       req.GetObjectStorage().GetSecretKey(),
		"ssl":              req.GetObjectStorage().GetSsl(),
		"type":             req.GetObjectStorage().GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	if err := s.generateConfig(req.GetObjectStorage(), req.GetBackupRootPath()); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx,
		"milvus-backup",
		"--config",
		milvusBackupConfFile,
		"create",
		"-n",
		req.GetBackupFile(),
	)

	// Use command executor for single command
	executor := common.NewCommandExecutor(s.logger)

	if err := executor.ExecuteCommand(cmd, "backup"); err != nil {
		s.logger.Errorw("failed to execute backup", zap.Error(err))
		return nil, err
	}

	s.logger.Info("backup milvus successfully")
	return nil, nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "milvus restore", map[string]interface{}{
		"suffix":           req.GetSuffix(),
		"backup_root_path": req.GetBackupRootPath(),
		"backup_file":      req.GetBackupFile(),
		"bucket":           req.GetObjectStorage().GetBucket(),
		"endpoint":         req.GetObjectStorage().GetEndpoint(),
		"access_key":       req.GetObjectStorage().GetAccessKey(),
		"secret_key":       req.GetObjectStorage().GetSecretKey(),
		"ssl":              req.GetObjectStorage().GetSsl(),
		"type":             req.GetObjectStorage().GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	if err := s.generateConfig(req.GetObjectStorage(), req.GetBackupRootPath()); err != nil {
		return nil, err
	}

	// Execute milvus-backup command
	var cmd *exec.Cmd

	if req.GetSuffix() == "" {
		cmd = exec.CommandContext(ctx,
			"milvus-backup",
			"--config",
			milvusBackupConfFile,
			"restore",
			"--restore_index",
			"-n",
			req.GetBackupFile(),
		)
	} else {
		cmd = exec.CommandContext(ctx,
			"milvus-backup",
			"--config",
			milvusBackupConfFile,
			"restore",
			"--restore_index",
			"-n",
			req.GetBackupFile(),
			"-s",
			req.GetSuffix(),
		)
	}

	// Use command executor for single command
	executor := common.NewCommandExecutor(s.logger)

	if err := executor.ExecuteCommand(cmd, "restore"); err != nil {
		s.logger.Errorw("failed to execute restore", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore milvus successfully")
	return nil, nil
}

func (s *service) generateConfig(storage *common.ObjectStorage, backupRootPath string) error {
	var err error
	cfg := s.opsCfg
	cfg.BackupBucket = storage.GetBucket()
	cfg.BackupUser = storage.GetAccessKey()
	cfg.BackupPassword = storage.GetSecretKey()
	cfg.BackupRootPath = backupRootPath
	cfg.BackupAddress, cfg.BackupPort, err = net.SplitHostPort(storage.GetEndpoint())

	if err != nil {
		s.logger.Errorw("failed to parse endpoint", zap.Error(err))
		return err
	}

	cfg.MinioPassword, err = util.DecryptPlainTextPassword(cfg.MinioUser)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", cfg.MinioUser))
		return err
	}

	cfg.MilvusPassword, err = util.DecryptPlainTextPassword(cfg.MilvusUser)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", cfg.MilvusUser))
		return err
	}

	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		s.logger.Errorw("failed to parse config template", zap.Error(err))
		return err
	}

	f, err := os.Create(milvusBackupConfFile)
	if err != nil {
		s.logger.Errorw("failed to create backup config file", zap.Error(err))
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	if err := tmpl.Execute(f, cfg); err != nil {
		s.logger.Errorw("failed to execute config template", zap.Error(err))
		return err
	}

	return nil
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "milvus set variable", map[string]interface{}{
		"key":   req.GetKey(),
		"value": req.GetValue(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create etcd connection
	client, err := s.newEtcdClient(ctx)
	if err != nil {
		return nil, err
	}
	defer s.closeEtcdClient(client)

	// Execute set variable
	etcdPrefix := fmt.Sprintf("%s/config/%s", s.rootPath, s.normalizeConfigKeyToPath(req.GetKey()))
	_, err = client.Put(ctx, etcdPrefix, req.GetValue())
	if err != nil {
		s.logger.Errorw("failed to set variable", zap.Error(err))
		return nil, err
	}

	s.logger.Info("set variable successfully")
	return nil, nil
}

// Accept "a.b.c" or "a/b/c" or "a.b/c" and normalize to "a/b/c"
func (s *service) normalizeConfigKeyToPath(k string) string {
	k = strings.TrimSpace(k)
	k = strings.TrimPrefix(k, "/")
	k = strings.TrimSuffix(k, "/")
	// 先把 '.' 替换成 '/'，再把多余的 '//' 压缩
	k = strings.ReplaceAll(k, ".", "/")
	for strings.Contains(k, "//") {
		k = strings.ReplaceAll(k, "//", "/")
	}
	return k
}

func (s *service) newEtcdClient(ctx context.Context) (*clientv3.Client, error) {
	endpoints, err := s.getEtcdEndpoint()
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, errors.New("etcd endpoints is empty")
	}

	tlsCfg, err := s.buildTLSConfig()
	if err != nil {
		return nil, err
	}

	client, err := clientv3.New(clientv3.Config{
		Context:     ctx,
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsCfg,
	})
	if err != nil {
		return nil, fmt.Errorf("create etcd client: %w", err)
	}

	return client, nil
}

func (s *service) closeEtcdClient(client *clientv3.Client) {
	if client == nil {
		return
	}

	if err := client.Close(); err != nil {
		s.logger.Errorw("failed to close etcd connection", zap.Error(err))
	}
}

func (s *service) buildTLSConfig() (*tls.Config, error) {
	// Load client cert/key (mTLS)
	certFile := filepath.Join(s.certDir, "tls.crt")
	keyFile := filepath.Join(s.certDir, "tls.key")
	caFile := filepath.Join(s.certDir, "ca.crt")

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert/key: %w", err)
	}

	// Load CA
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caPEM); !ok {
		return nil, fmt.Errorf("append CA certs failed: %s", caFile)
	}

	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}
	return tlsCfg, nil
}

func (s *service) getEtcdEndpoint() ([]string, error) {

	memList := make([]string, 0)

	err := json.Unmarshal([]byte(s.etcdMemberList), &memList)
	if err != nil {
		return nil, err
	}

	uri := make([]string, 0)
	for _, member := range memList {
		endpoint := net.JoinHostPort(member, "2379")
		uri = append(uri, endpoint)
	}

	return uri, nil
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
