package postgresql

import (
	"archive/tar"
	"context"
	"fmt"

	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"golang.org/x/sync/errgroup"

	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/jackc/pgx/v5"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	pgOps PostgresqlOperationServer
	UnimplementedPostgresqlOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer

	dataDir string
}

func (s *service) Config() error {
	s.pgOps = app.GetGrpcApp(appName).(PostgresqlOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)

	dataDir, err := util.IsEnvVarSet(vars.DataDirEnvKey)
	if err != nil {
		return err
	}

	s.dataDir = dataDir

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterPostgresqlOperationServer(server, svr)
}

func (s *service) PhysicalBackup(ctx context.Context, req *PhysicalBackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "postgresql physical backup", map[string]interface{}{
		"username":    req.GetUsername(),
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetObjectStorage().GetBucket(),
		"endpoint":    req.GetObjectStorage().GetEndpoint(),
		"access_key":  req.GetObjectStorage().GetAccessKey(),
		"secret_key":  req.GetObjectStorage().GetSecretKey(),
		"ssl":         req.GetObjectStorage().GetSsl(),
		"type":        req.GetObjectStorage().GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	dir, err := os.MkdirTemp("/tmp", "pgbackup-*")
	if err != nil {
		s.logger.Errorw("failed to create temporary directory", zap.Error(err))
		return nil, err
	}

	defer func() {
		_ = os.RemoveAll(dir)
	}()

	cmd := exec.CommandContext(ctx,
		"pg_basebackup",
		"-D", dir,
		"-U", req.GetUsername(),
		"-h", "127.0.0.1",
		"-P",
		"-F", "t",
		"-X", "stream",
	)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	executor := common.NewCommandExecutor(s.logger)

	factory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}

	if err := executor.ExecuteCommand(cmd, "backup"); err != nil {
		s.logger.Errorw("failed to execute command", zap.Error(err))
		return nil, err
	}

	errGrp := new(errgroup.Group)

	if err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		key := filepath.Join(req.GetBackupFile(), strings.TrimPrefix(path, dir))

		errGrp.Go(func() error {

			if err := factory.PutFile(ctx, req.GetObjectStorage().GetBucket(), key, path); err != nil {
				s.logger.Errorw("failed to put backup file", zap.Error(err))
				return err
			}
			return nil
		})

		return nil
	}); err != nil {
		s.logger.Errorw("failed to prepare backup", zap.Error(err))
		return nil, err
	}

	if err := errGrp.Wait(); err != nil {
		s.logger.Errorw("failed to wait backup", zap.Error(err))
		return nil, err
	}

	s.logger.Info("physical backup postgresql successfully")
	return nil, nil
}

func (s *service) LogicalBackup(ctx context.Context, req *LogicalBackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "postgresql logical backup", map[string]interface{}{
		"username":            req.GetUsername(),
		"backup_file":         req.GetBackupFile(),
		"database":            req.GetDatabase(),
		"table":               req.GetTable(),
		"logical_backup_mode": req.GetLogicalBackupMode().String(),
		"bucket":              req.GetObjectStorage().GetBucket(),
		"endpoint":            req.GetObjectStorage().GetEndpoint(),
		"access_key":          req.GetObjectStorage().GetAccessKey(),
		"secret_key":          req.GetObjectStorage().GetSecretKey(),
		"ssl":                 req.GetObjectStorage().GetSsl(),
		"type":                req.GetObjectStorage().GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	var cmd *exec.Cmd
	switch req.GetLogicalBackupMode() {
	case LogicalBackupMode_Full:
		cmd = exec.CommandContext(ctx,
			"pg_dumpall",
			"-U", req.GetUsername(),
			"-h", "127.0.0.1",
		)
	default:
		err = fmt.Errorf("unsupported logical backup mode %s", req.GetLogicalBackupMode().String())
		s.logger.Errorw("failed to logical backup postgresql", zap.Error(err))
		return nil, err
	}
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	executor := common.NewCommandExecutor(s.logger)

	factory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}

	if err := executor.ExecuteCommandStreamToS3(ctx, cmd, factory, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), "backup"); err != nil {
		s.logger.Errorw("failed to execute backup", zap.Error(err))
		return nil, err
	}

	s.logger.Info("logical backup postgresql successfully")
	return nil, nil
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "postgresql set variable", map[string]interface{}{
		"username": req.GetUsername(),
		"key":      req.GetKey(),
		"value":    req.GetValue(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	url := fmt.Sprintf("postgres://%s:%s@localhost:5432/postgres?connect_timeout=5", req.GetUsername(), password)

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		s.logger.Errorw("failed to connect to postgresql", zap.Error(err))
		return nil, err
	}
	defer func() { _ = conn.Close(ctx) }()

	execSQL := fmt.Sprintf("ALTER SYSTEM SET %s = %s", req.GetKey(), req.GetValue())

	if _, err := conn.Exec(ctx, execSQL); err != nil {
		s.logger.Errorw("failed to set variable", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))
		return nil, err
	}

	if _, err := conn.Exec(ctx, "SELECT pg_reload_conf()"); err != nil {
		s.logger.Errorw("failed to reload configuration", zap.Error(err))
		return nil, err
	}

	s.logger.Info("set variable successfully")

	return nil, nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "postgresql restore", map[string]interface{}{
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetObjectStorage().GetBucket(),
		"endpoint":    req.GetObjectStorage().GetEndpoint(),
		"access_key":  req.GetObjectStorage().GetAccessKey(),
		"secret_key":  req.GetObjectStorage().GetSecretKey(),
		"ssl":         req.GetObjectStorage().GetSsl(),
		"type":        req.GetObjectStorage().GetType(),
	})

	// Check process is stopped
	if _, err := s.slm.CheckProcessStopped(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process stopped", zap.Error(err))
		return nil, err
	}

	factory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}

	// Clear data directory
	if err := s.removeContents(s.dataDir); err != nil {
		s.logger.Errorw("failed to remove contents", zap.Error(err), zap.String("dir", s.dataDir))
		return nil, err
	}

	// Extract backup files from S3
	if err = s.extractFileFromS3(ctx, factory, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), "base.tar", s.dataDir); err != nil {
		s.logger.Errorw("failed to extract base.tar", zap.Error(err))
		return nil, err
	}

	if err = s.extractFileFromS3(ctx, factory, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), "pg_wal.tar", filepath.Join(s.dataDir, "pg_wal")); err != nil {
		s.logger.Errorw("failed to extract pg_wal.tar", zap.Error(err))
		return nil, err
	}

	// Create standby signal file
	if _, err = os.Stat(filepath.Join(s.dataDir, "standby.signal")); err != nil {
		s.logger.Info("create standby.signal file")

		if err = os.WriteFile(filepath.Join(s.dataDir, "standby.signal"), []byte{}, 0644); err != nil {
			s.logger.Errorw("failed to create standby.signal file", zap.Error(err))
			return nil, err
		}
	}

	// Set file ownership
	if err = s.recursiveChown(s.dataDir, 1001, 1001); err != nil {
		s.logger.Errorw("failed to recursive chown", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore postgresql successfully")
	return nil, nil
}

func (s *service) removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		if err := d.Close(); err != nil {
			s.logger.Errorw("failed to close directory", zap.Error(err))
		}
	}()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *service) recursiveChown(rootPath string, uid, gid int) error {
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed t o access path %s, %v", path, err)
		}

		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("failed to chown %s to %d:%d, %v", path, uid, gid, err)
		}

		return nil
	})
}

func (s *service) extractFileFromS3(ctx context.Context, storageFactory common.ObjectStorageFactory, bucket, key, filename, targetDir string) error {
	fileKey := filepath.Join(key, filename)

	obj, err := storageFactory.GetObject(ctx, bucket, fileKey)

	if err != nil {
		return fmt.Errorf("download %s failed: %v", fileKey, err)
	}

	defer obj.Close()

	tr := tar.NewReader(obj)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := s.extractTarEntry(tr, hdr, targetDir); err != nil {
			return err
		}
	}
}

func (s *service) safeTarPath(targetDir, tarName string) (string, error) {
	cleanName := filepath.Clean(tarName)

	if cleanName == "." || strings.HasPrefix(cleanName, "..") {
		return "", fmt.Errorf("illegal tar entry path: %s", tarName)
	}

	targetPath := filepath.Join(targetDir, cleanName)

	base := filepath.Clean(targetDir) + string(os.PathSeparator)
	if !strings.HasPrefix(targetPath, base) {
		return "", fmt.Errorf("tar path escapes target dir: %s", tarName)
	}

	return targetPath, nil
}

func (s *service) extractTarEntry(tr *tar.Reader, hdr *tar.Header, targetDir string) error {
	targetPath, err := s.safeTarPath(targetDir, hdr.Name)
	if err != nil {
		return err
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(targetPath, os.FileMode(hdr.Mode))

	case tar.TypeReg:
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		f, err := os.OpenFile(
			targetPath,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			os.FileMode(hdr.Mode),
		)
		if err != nil {
			return err
		}

		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return err
		}

		return f.Close()

	default:
		return nil
	}
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
