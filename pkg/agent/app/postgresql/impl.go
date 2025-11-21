package postgresql

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app/s3storage"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
)

const (
	dataDirKey   = "DATA_DIR"
	dataMountKey = "DATA_MOUNT"
	logMountKey  = "LOG_MOUNT"
	standbyFile  = "standby.signal"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	postgresqllOps PostgresqlOperationServer
	UnimplementedPostgresqlOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

// Common helper methods

// newPostgresqlResponse creates a new PostgreSQL Response with the given message
func newPostgresqlResponse(message string) *Response {
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
	s.postgresqllOps = app.GetGrpcApp(appName).(PostgresqlOperationServer)
	s.logger = zap.L().Named("[POSTGRESQL]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterPostgresqlOperationServer(server, svr)
}

func (s *service) PhysicalBackup(ctx context.Context, req *PhysicalBackupRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "postgresql physical backup", map[string]interface{}{
		"username":     req.GetUsername(),
		"password":     req.GetPassword(),
		"backup_file":  req.GetBackupFile(),
		"storage_type": req.GetStorageType(),
		"bucket":       req.GetS3Storage().GetBucket(),
		"endpoint":     req.GetS3Storage().GetEndpoint(),
		"access_key":   req.GetS3Storage().GetAccessKey(),
		"secret_key":   req.GetS3Storage().GetSecretKey(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "service status check failed", err)
	}

	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "decrypt password failed", err)
	}

	var cmd *exec.Cmd

	if req.GetS3Storage() != nil {
		_, err := exec.LookPath("pg_basebackup")
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "pg_basebackup command is not installed or not in PATH", nil)
		}

		// 2. Get environment variables
		dataMountValue, err := getEnvVarOrError(dataMountKey)
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to get DATA_MOUNT environment variable", err)
		}

		logMountValue, err := getEnvVarOrError(logMountKey)
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to get LOG_MOUNT environment variable", err)
		}

		backupDirPrefix := filepath.Join(dataMountValue, "backup")
		backupDir := filepath.Join(backupDirPrefix, req.GetBackupFile())

		// 3. Check if backup directory already exists
		if _, err = os.Stat(backupDir); err == nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("backup directory '%s' already exists, please remove it before proceeding", backupDir), nil)
		}

		cmd = exec.CommandContext(ctx,
			"pg_basebackup",
			"-D",
			backupDir,
			"-U",
			req.GetUsername(),
			"-h",
			"127.0.0.1",
			"-P",
			"-Ft",
			"-v",
			"-X",
			"stream",
		)
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

		s.logger.Info("starting pg_basebackup command...")
		output, err := cmd.Output()

		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to start pg_basebackup command", err)
		}

		pgBaseBackupLogPath := fmt.Sprintf("%s/pg-basebackup-backup.log", logMountValue)
		// 4. Write backup log
		err = os.WriteFile(pgBaseBackupLogPath, output, 0644)
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("failed to write pg_basebackup log to file %s", pgBaseBackupLogPath), err)
		}

		var storageFactory s3storage.S3Storage
		switch req.GetS3Storage().GetType() {
		case S3StorageType_Minio:
			storageFactory, err = s3storage.NewMinioClient(req.GetS3Storage().GetEndpoint(), req.GetS3Storage().GetAccessKey(), req.GetS3Storage().GetSecretKey(), req.GetS3Storage().GetSsl())
			if err != nil {
				return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create s3 client", err)
			}
		default:
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "unsupported s3 storage type", nil)
		}

		errGrp := new(errgroup.Group)

		if err = filepath.Walk(backupDir, func(filePath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			key := filepath.Join(req.GetBackupFile(), strings.TrimPrefix(filePath, backupDir))

			errGrp.Go(func() error {

				if err := storageFactory.UploadFileToS3(ctx, req.GetS3Storage().GetBucket(), key, filePath); err != nil {
					return err
				}

				return nil
			})

			return nil
		}); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to walk backup directory", err)
		}

		if err := errGrp.Wait(); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to upload to s3", err)
		}

		// remove local temporary backup directory
		if _, err := os.Stat(backupDir); err == nil {
			_ = os.RemoveAll(backupDirPrefix)
			s.logger.Infof("successfully removed local temporary backup directory")
		}

	} else {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "storage type not supported: only 's3' is currently supported", nil)
	}

	return common.LogAndReturnSuccess(s.logger, newPostgresqlResponse, "physical backup and upload to s3 success")
}

func (s *service) LogicalBackup(ctx context.Context, req *LogicalBackupRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "postgresql logical backup", map[string]interface{}{
		"database":            req.GetDatabase(),
		"table":               req.GetTable(),
		"logical_backup_mode": req.GetLogicalBackupMode(),
		"username":            req.GetUsername(),
		"password":            req.GetPassword(),
		"backup_file":         req.GetBackupFile(),
		"storage_type":        req.GetStorageType(),
		"bucket":              req.GetS3Storage().GetBucket(),
		"endpoint":            req.GetS3Storage().GetEndpoint(),
		"access_key":          req.GetS3Storage().GetAccessKey(),
		"secret_key":          req.GetS3Storage().GetSecretKey(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "service status check failed", err)
	}

	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "decrypt password failed", err)
	}

	_, err = exec.LookPath("pg_dumpall")
	if err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "pg_dumpall command is not installed or not in PATH", nil)
	}

	var cmd *exec.Cmd

	switch req.GetLogicalBackupMode() {
	case LogicalBackupMode_Full:
		cmd = exec.CommandContext(ctx,
			"pg_dumpall",
			"-U",
			req.GetUsername(),
			"-h",
			"127.0.0.1",
		)
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))
	default:
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("logical backup mode '%s' is not supported", req.GetLogicalBackupMode()), nil)
	}

	if req.GetS3Storage() != nil {
		var storageFactory s3storage.S3Storage
		switch req.GetS3Storage().GetType() {
		case S3StorageType_Minio:
			storageFactory, err = s3storage.NewMinioClient(req.GetS3Storage().GetEndpoint(), req.GetS3Storage().GetAccessKey(), req.GetS3Storage().GetSecretKey(), req.GetS3Storage().GetSsl())
			if err != nil {
				return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create s3 client", err)
			}
		default:
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "unsupported s3 storage type", nil)
		}

		// execute pgdumpall command
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			s.logger.Errorf("failed to get stdout pipe: %v", err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			s.logger.Errorf("failed to get stderr pipe: %v", err)
		}

		s.logger.Info("starting pg_dumpall command...")
		if err := cmd.Start(); err != nil {
			s.logger.Errorf("failed to start pg_dumpall: %v", err)
		}

		stdoutBytes, err := io.ReadAll(stdout)
		if err != nil {
			s.logger.Errorf("failed to read pg_dumpall stdout: %v", err)
		}

		stderrBytes, err := io.ReadAll(stderr)
		if err != nil {
			s.logger.Errorf("failed to read pg_dumpall stderr: %v", err)
		}

		errCh := make(chan error)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func(ctx context.Context, cmd *exec.Cmd, errCh chan error) {
			s.logger.Info("uploading to S3...")

			defer wg.Done()

			err := storageFactory.UploadContentToS3(ctx, req.GetS3Storage().GetBucket(), req.GetBackupFile(), stdoutBytes)

			errCh <- err

		}(ctx, cmd, errCh)

		if err := <-errCh; err != nil {
			if err := cmd.Cancel(); err != nil {
				s.logger.Errorf("failed to cancel command: %v", err)
			}
			if waitErr := cmd.Wait(); waitErr != nil {
				s.logger.Errorf("command wait failed: %v", waitErr)
			}
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to upload to s3", err)
		}

		s.logger.Info("waiting for pg_dumpall to exit...")
		if err := cmd.Wait(); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("failed to execute pg_dumpall: %v %s", err, string(stderrBytes)), nil)
		}

		wg.Wait()
	}

	return common.LogAndReturnSuccess(s.logger, newPostgresqlResponse, "logical backup and upload to s3 success")
}

func (s *service) removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		if err := d.Close(); err != nil {
			s.logger.Errorf("failed to close directory: %v", err)
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

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "postgresql restore", map[string]interface{}{
		"storage_type": req.GetStorageType(),
		"backup_file":  req.GetBackupFile(),
		"secret_key":   req.GetS3Storage().GetSecretKey(),
		"access_key":   req.GetS3Storage().GetAccessKey(),
		"bucket":       req.GetS3Storage().GetBucket(),
		"endpoint":     req.GetS3Storage().GetEndpoint(),
	})

	// 1. Check if service is stopped
	if _, err := s.slm.CheckServiceStopped(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "service status check failed", err)
	}

	if req.GetS3Storage() != nil {
		// 2. Get environment variables
		dataDirValue, err := getEnvVarOrError(dataDirKey)
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to get DATA_DIR environment variable", err)
		}

		var storageFactory s3storage.S3Storage
		switch req.GetS3Storage().GetType() {
		case S3StorageType_Minio:
			storageFactory, err = s3storage.NewMinioClient(req.GetS3Storage().GetEndpoint(), req.GetS3Storage().GetAccessKey(), req.GetS3Storage().GetSecretKey(), req.GetS3Storage().GetSsl())
			if err != nil {
				return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create s3 client", err)
			}
		default:
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "unsupported s3 storage type", nil)
		}

		// 4. Clear data directory
		if err := s.removeContents(dataDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("failed to clear data directory '%s'", dataDirValue), err)
		}

		// 5. Extract backup files from S3
		if err = s.extractFileFromS3(ctx, storageFactory, req.GetS3Storage().GetBucket(), req.GetBackupFile(), "base.tar", dataDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to unarchive base.tar", err)
		}

		if err = s.extractFileFromS3(ctx, storageFactory, req.GetS3Storage().GetBucket(), req.GetBackupFile(), "pg_wal.tar", filepath.Join(dataDirValue, "pg_wal")); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to unarchive pg_wal.tar", err)
		}

		// 6. Create standby signal file
		if _, err := os.Stat(filepath.Join(dataDirValue, standbyFile)); err != nil {
			s.logger.Info("standby.signal file not found, attempting to create it")

			if err := os.WriteFile(filepath.Join(dataDirValue, standbyFile), []byte{}, 0644); err != nil {
				return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create standby.signal file", err)
			}
		}

		// 7. Set file ownership
		if err = s.recursiveChown(dataDirValue, 1001, 1001); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to set file ownership", err)
		}

	} else {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "storage type not supported: only 's3' is currently supported", nil)
	}

	return common.LogAndReturnSuccess(s.logger, newPostgresqlResponse, "restore from s3 succeeded")
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

func (s *service) extractFileFromS3(ctx context.Context, storageFactory s3storage.S3Storage, bucket, key, filename, targetDir string) error {
	fileKey := filepath.Join(key, filename)

	output, err := storageFactory.DownloadContentFromS3(ctx, bucket, fileKey)

	if err != nil {
		return fmt.Errorf("download %s failed: %v", fileKey, err)
	}
	defer func() {
		if err := output.Close(); err != nil {
			s.logger.Errorf("failed to close response body: %v", err)
		}
	}()

	tr := tar.NewReader(output)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// calculate target file path
		targetPath := filepath.Join(targetDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir: // directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg: // common file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer func() {
				if err := outFile.Close(); err != nil {
					s.logger.Errorf("failed to close output file: %v", err)
				}
			}()

			// copy file content
			if _, err := io.Copy(outFile, tr); err != nil {
				return err
			}

			// set file mode
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}

	return nil
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}

//func init() {
//	app.RegistryGrpcApp(svr)
//}
