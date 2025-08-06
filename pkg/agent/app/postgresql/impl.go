package postgresql

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// createS3Client creates an S3 client
// createS3ClientWithCustomResolver creates an S3 client with custom resolver (for restore operations)
func (s *service) createS3ClientWithCustomResolver(s3Config *S3Storage) (*s3.Client, error) {
	if s3Config == nil {
		return nil, fmt.Errorf("S3 storage configuration is required")
	}
	
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               s3Config.GetEndpoint(),
			HostnameImmutable: true,
		}, nil
	})
	
	creds := credentials.NewStaticCredentialsProvider(
		s3Config.GetAccessKey(), 
		s3Config.GetSecretKey(), 
		"",
	)
	
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(creds),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}
	
	return s3.NewFromConfig(cfg), nil
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

	var cmd *exec.Cmd

	if req.GetS3Storage() != nil {
		path, err := exec.LookPath("pg_basebackup")
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

		// PGPASSWORD="${repl_password}" pg_basebackup -D "${backup_dir}/${BACKUP_FILE}" -U "${REPL_USER}" -h "${SOURCE_HOST}" -P -Ft -v -X stream
		cmd = exec.CommandContext(ctx,
			path,
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
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", req.GetPassword()))

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

		// 5. Create S3 client
		awsS3Client, err := s.createS3ClientWithCustomResolver(req.GetS3Storage())
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create S3 client", err)
		}
		uploader := manager.NewUploader(awsS3Client)

		errGrp := new(errgroup.Group)

		if err = filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			key := filepath.Join(req.GetBackupFile(), strings.TrimPrefix(path, backupDir))

			errGrp.Go(func() error {

				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				if _, err = uploader.Upload(ctx, &s3.PutObjectInput{
					Bucket: aws.String(req.GetS3Storage().GetBucket()),
					Body:   file,
					Key:    aws.String(key),
				}); err != nil {
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
		"database":             req.GetDatabase(),
		"table":                req.GetTable(),
		"logical_backup_mode":  req.GetLogicalBackupMode(),
		"username":             req.GetUsername(),
		"password":             req.GetPassword(),
		"backup_file":          req.GetBackupFile(),
		"storage_type":         req.GetStorageType(),
		"bucket":               req.GetS3Storage().GetBucket(),
		"endpoint":             req.GetS3Storage().GetEndpoint(),
		"access_key":           req.GetS3Storage().GetAccessKey(),
		"secret_key":           req.GetS3Storage().GetSecretKey(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "service status check failed", err)
	}

	path, err := exec.LookPath("pg_dumpall")
	if err != nil {
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, "pg_dumpall command is not installed or not in PATH", nil)
	}

	var cmd *exec.Cmd

	switch req.GetLogicalBackupMode() {
	case LogicalBackupMode_Full:
		cmd = exec.CommandContext(ctx,
			path,
			"-U",
			req.GetUsername(),
			"-h",
			"127.0.0.1",
		)
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("PGPASSWORD=%s", req.GetPassword()))
	default:
		return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("logical backup mode '%s' is not supported", req.GetLogicalBackupMode()), nil)
	}

	if req.GetS3Storage() != nil {
		// 2. Create S3 client
		awsS3Client, err := s.createS3ClientWithCustomResolver(req.GetS3Storage())
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create S3 client", err)
		}
		uploader := manager.NewUploader(awsS3Client)

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

			result, err := uploader.Upload(ctx, &s3.PutObjectInput{
				Bucket: aws.String(req.GetS3Storage().GetBucket()),
				Key:    aws.String(req.GetBackupFile()),
				Body:   bytes.NewReader(stdoutBytes),
			})

			errCh <- err

			if err == nil {
				s.logger.Infof("upload successful: %s", result.Location)
			}
		}(ctx, cmd, errCh)

		if err := <-errCh; err != nil {
			cmd.Cancel()
			cmd.Wait()
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
	defer d.Close()
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

		// 3. Create S3 client with custom resolver
		awsS3Client, err := s.createS3ClientWithCustomResolver(req.GetS3Storage())
		if err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to create S3 client", err)
		}

		// 4. Clear data directory
		if err := s.removeContents(dataDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, fmt.Sprintf("failed to clear data directory '%s'", dataDirValue), err)
		}

		// 5. Extract backup files from S3
		if err = s.extractFileFromS3(awsS3Client, req.GetS3Storage().GetBucket(), req.GetBackupFile(), "base.tar", dataDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newPostgresqlResponse, "failed to unarchive base.tar", err)
		}

		if err = s.extractFileFromS3(awsS3Client, req.GetS3Storage().GetBucket(), req.GetBackupFile(), "pg_wal.tar", filepath.Join(dataDirValue, "pg_wal")); err != nil {
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
			return fmt.Errorf("access path %s failed, %v", path, err)
		}

		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("chown %s to 1001 failed: %v", path, err)
		}

		return nil
	})
}

func (s *service) extractFileFromS3(client *s3.Client, bucket, key, filename, targetDir string) error {
	fileKey := filepath.Join(key, filename)

	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return fmt.Errorf("download %s failed: %v", fileKey, err)
	}
	defer output.Body.Close()

	tr := tar.NewReader(output.Body)
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
			defer outFile.Close()

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
