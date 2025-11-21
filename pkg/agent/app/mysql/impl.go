package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app/s3storage"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
)

const (
	dataDirKey     = "DATA_DIR"
	relayLogDirKey = "RELAY_LOG_DIR"
	binLogDirKey   = "BIN_LOG_DIR"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	mysqlOps MysqlOperationServer
	UnimplementedMysqlOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

// Common helper methods

// newMysqlResponse creates a new MySQL Response with the given message
func newMysqlResponse(message string) *Response {
	return &Response{Message: message}
}

// newMysqlDB creates a MySQL database connection
func (s *service) newMysqlDB(ctx context.Context, username, password, socketFile string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@unix(%s)/?timeout=5s&multiStatements=true&interpolateParams=true",
		username, password, socketFile)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	if err = db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping database: %v, and failed to close db: %v", err, closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
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
	s.mysqlOps = app.GetGrpcApp(appName).(MysqlOperationServer)
	s.logger = zap.L().Named("[MYSQL]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterMysqlOperationServer(server, svr)
}

func (s *service) Clone(ctx context.Context, req *CloneRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "mysql clone", map[string]interface{}{
		"username":              req.GetUsername(),
		"password":              req.GetPassword(),
		"source_host":           req.GetSourceHost(),
		"source_port":           req.GetSourcePort(),
		"socket_file":           req.GetSocketFile(),
		"source_clone_password": req.GetSourceClonePassword(),
		"source_clone_username": req.GetSourceCloneUser(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "service status check failed", err)
	}

	// 2. Create database connection
	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "decrypt password failed", err)
	}
	db, err := s.newMysqlDB(ctx, req.GetUsername(), password, req.GetSocketFile())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "database connection failed", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			s.logger.Errorf("failed to close database connection: %v", err)
		}
	}()

	// cloneStatus: Not Started, In Progress, Completed and Failed.
	var clonePluginStatus, cloneStatus, cloneErrMsg string

	// 3. Check clone plugin status
	if err := db.QueryRowContext(ctx, checkCloneAvaliableSql).Scan(&clonePluginStatus); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to get clone plugin status", err)
	}
	if clonePluginStatus != "ACTIVE" {
		return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("clone plugin is not active, current status: %s", clonePluginStatus), nil)
	}

	// 4. Set valid donor list
	sourceAddr := net.JoinHostPort(req.GetSourceHost(), strconv.Itoa(int(req.GetSourcePort())))
	if _, err = db.ExecContext(ctx, SetValidDonorListSql, sourceAddr); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to set clone_valid_donor_list=%s", sourceAddr), err)
	}

	execSql := fmt.Sprintf(ExecCloneSql, req.GetSourceCloneUser(), req.GetSourceHost(), req.GetSourcePort(), req.GetSourceClonePassword())
	if _, err = db.ExecContext(ctx, execSql); err != nil {
		s.logger.Warnf("failed to execute clone instance: %v", err)
	}

	//Start the timer and produce a Timer object
	timeoutCh := time.After(time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

LOOP:
	for {
		select {
		case <-ticker.C:
			// close the old connection, use new connection to query
			newDb, err := s.newMysqlDB(ctx, req.GetUsername(), password, req.GetSocketFile())
			if err != nil {
				s.logger.Warnf("failed to create new database connection: %v", err)
				continue LOOP
			}
			defer func() {
				if err := newDb.Close(); err != nil {
					s.logger.Errorf("failed to close database connection: %v", err)
				}
			}()

			if err := newDb.QueryRowContext(ctx, getCloneStatusSql).Scan(&cloneStatus, &cloneErrMsg); err != nil {
				s.logger.Warnf("query clone status failed: %v", err)
				if closeErr := newDb.Close(); closeErr != nil {
					s.logger.Errorf("failed to close database connection: %v", closeErr)
				}
				continue LOOP
			}

			s.logger.Infof("current clone status: %s", cloneStatus)
			switch cloneStatus {
			case "Completed":
				break LOOP
			case "Failed":
				return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to clone", err)
			}
		case <-timeoutCh:
			return common.LogAndReturnError(s.logger, newMysqlResponse, "clone process timed out", err)
		}
	}

	return common.LogAndReturnSuccess(s.logger, newMysqlResponse, "mysql clone completed successfully")
}

func (s *service) PhysicalBackup(ctx context.Context, req *PhysicalBackupRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "mysql physical backup", map[string]interface{}{
		"username":     req.GetUsername(),
		"password":     req.GetPassword(),
		"tool":         req.GetPhysicalBackupTool(),
		"socket_file":  req.GetSocketFile(),
		"parallel":     req.GetParallel(),
		"backup_file":  req.GetBackupFile(),
		"conf_file":    req.GetConfFile(),
		"storage_type": req.GetStorageType(),
		"bucket":       req.GetS3Storage().GetBucket(),
		"endpoint":     req.GetS3Storage().GetEndpoint(),
		"access_key":   req.GetS3Storage().GetAccessKey(),
		"secret_key":   req.GetS3Storage().GetSecretKey(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "service status check failed", err)
	}

	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "decrypt password failed", err)
	}

	var cmd *exec.Cmd

	switch req.GetPhysicalBackupTool() {
	case PhysicalBackupTool_Xtrabackup:
		if req.GetS3Storage() != nil {
			_, err := exec.LookPath("xtrabackup")
			if err != nil {
				return common.LogAndReturnError(s.logger, newMysqlResponse, "xtrabackup command is not installed or not in PATH", err)
			}

			cmd = exec.CommandContext(ctx,
				"xtrabackup",
				fmt.Sprintf("--defaults-file=%s", req.GetConfFile()),
				fmt.Sprintf("--socket=%s", req.GetSocketFile()),
				fmt.Sprintf("--user=%s", req.GetUsername()),
				fmt.Sprintf("--password=%s", password),
				fmt.Sprintf("--extra-lsndir=%s", "/tmp/s3_tmp_dir"),
				fmt.Sprintf("--target-dir=%s", "/tmp/s3_tmp_dir"),
				"--backup",
				"--stream=xbstream",
			)

			_, err = exec.LookPath("xbcloud")
			if err != nil {
				return common.LogAndReturnError(s.logger, newMysqlResponse, "xbcloud command is not installed or not in PATH", err)
			}

			var xbcloudCmd *exec.Cmd

			if req.GetS3Storage().GetSsl() {
				xbcloudCmd = exec.CommandContext(ctx,
					"xbcloud",
					"put",
					"--storage=s3",
					"--s3-ssl",
					"--s3-verify-ssl=0",
					fmt.Sprintf("--s3-endpoint=%s", req.GetS3Storage().GetEndpoint()),
					fmt.Sprintf("--s3-bucket=%s", req.GetS3Storage().GetBucket()),
					fmt.Sprintf("--s3-access-key=%s", req.GetS3Storage().GetAccessKey()),
					fmt.Sprintf("--parallel=%d", req.GetParallel()),
					fmt.Sprintf("--backup-dir=%s", req.GetBackupFile()),
					req.GetBackupFile(),
				)
			} else {
				xbcloudCmd = exec.CommandContext(ctx,
					"xbcloud",
					"put",
					"--storage=s3",
					fmt.Sprintf("--s3-endpoint=%s", req.GetS3Storage().GetEndpoint()),
					fmt.Sprintf("--s3-bucket=%s", req.GetS3Storage().GetBucket()),
					fmt.Sprintf("--s3-access-key=%s", req.GetS3Storage().GetAccessKey()),
					fmt.Sprintf("--s3-secret-key=%s", req.GetS3Storage().GetSecretKey()),
					fmt.Sprintf("--parallel=%d", req.GetParallel()),
					req.GetBackupFile(),
				)
			}

			// Use command executor for piped commands
			executor := common.NewCommandExecutor(ctx, s.logger)
			if err := executor.ExecutePipedCommands(cmd, xbcloudCmd, "backup"); err != nil {
				return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to execute backup commands", err)
			}

			successMsg := "physical backup completed and successfully uploaded to S3"
			s.logger.Info(successMsg)
			return &Response{Message: successMsg}, nil
		} else {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "storage type not supported: only 's3' is currently supported", errors.New("not supported"))
		}
	default:
		return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("physical backup tool %s", req.GetPhysicalBackupTool()), errors.New("not supported"))
	}
}

func (s *service) LogicalBackup(ctx context.Context, req *LogicalBackupRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "mysql logical backup", map[string]interface{}{
		"username":            req.GetUsername(),
		"password":            req.GetPassword(),
		"table":               req.GetTable(),
		"database":            req.GetDatabase(),
		"logical_backup_mode": req.GetLogicalBackupMode(),
		"backup_file":         req.GetBackupFile(),
		"conf_file":           req.GetConfFile(),
		"storage_type":        req.GetStorageType(),
		"socket_file":         req.GetSocketFile(),
		"bucket":              req.GetS3Storage().GetBucket(),
		"endpoint":            req.GetS3Storage().GetEndpoint(),
		"access_key":          req.GetS3Storage().GetAccessKey(),
		"secret_key":          req.GetS3Storage().GetSecretKey(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "service status check failed", err)
	}

	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "decrypt password failed", err)
	}

	_, err = exec.LookPath("mysqldump")
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "mysqldump command is not installed or not in PATH", nil)
	}

	var cmd *exec.Cmd

	switch req.GetLogicalBackupMode() {
	case LogicalBackupMode_Full:
		cmd = exec.CommandContext(ctx,
			"mysqldump",
			fmt.Sprintf("--defaults-file=%s", req.GetConfFile()),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--socket=%s", req.GetSocketFile()),
			"--single-transaction",
			"--set-gtid-purged=OFF",
			"--all-databases",
		)

	case LogicalBackupMode_Database:
		cmd = exec.CommandContext(ctx,
			"mysqldump",
			fmt.Sprintf("--defaults-file=%s", req.GetConfFile()),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--socket=%s", req.GetSocketFile()),
			"--single-transaction",
			"--set-gtid-purged=OFF",
			fmt.Sprintf("--databases %s", req.GetDatabase()),
		)
	case LogicalBackupMode_Table:
		cmd = exec.CommandContext(ctx,
			"mysqldump",
			fmt.Sprintf("--defaults-file=%s", req.GetConfFile()),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--socket=%s", req.GetSocketFile()),
			"--single-transaction",
			"--set-gtid-purged=OFF",
			req.GetDatabase(),
			req.GetTable(),
		)
	default:
		return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("logical backup mode '%s' is not supported", req.GetLogicalBackupMode()), nil)
	}

	if req.GetS3Storage() != nil {
		var storageFactory s3storage.S3Storage
		switch req.GetS3Storage().GetType() {
		case S3StorageType_Minio:
			storageFactory, err = s3storage.NewMinioClient(req.GetS3Storage().GetEndpoint(), req.GetS3Storage().GetAccessKey(), req.GetS3Storage().GetSecretKey(), req.GetS3Storage().GetSsl())
			if err != nil {
				return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to create s3 client", err)
			}
		default:
			return common.LogAndReturnError(s.logger, newMysqlResponse, "unsupported s3 storage type", nil)
		}

		// execute mysqldump command
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			s.logger.Errorf("failed to get stdout pipe: %v", err)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			s.logger.Errorf("failed to get stderr pipe: %v", err)
		}

		s.logger.Info("starting mysqldump command...")
		if err := cmd.Start(); err != nil {
			s.logger.Errorf("failed to start mysqldump: %v", err)
		}

		stdoutBytes, err := io.ReadAll(stdout)
		if err != nil {
			s.logger.Errorf("failed to read mysqldump stdout: %v", err)
		}

		stderrBytes, err := io.ReadAll(stderr)
		if err != nil {
			s.logger.Errorf("failed to read mysqldump stderr: %v", err)
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
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to upload to s3", err)
		}

		s.logger.Info("waiting for mysqldump to exit...")
		if err := cmd.Wait(); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to execute mysqldump: %v %s", err, string(stderrBytes)), nil)
		}

		wg.Wait()
	}

	return common.LogAndReturnSuccess(s.logger, newMysqlResponse, "logical backup and upload to s3 success")
}

func (s *service) removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		if err := d.Close(); err != nil {
			s.logger.Errorf("failed to close database connection: %v", err)
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

func (s *service) GtidPurge(ctx context.Context, req *GtidPurgeRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "mysql gtid purge", map[string]interface{}{
		"username":    req.GetUsername(),
		"password":    req.GetPassword(),
		"arch_mode":   req.GetArchMode(),
		"socket_file": req.GetSocketFile(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "service status check failed", err)
	}

	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "decrypt password failed", err)
	}

	// 2. Get data directory environment variable
	dataDirValue, err := getEnvVarOrError(dataDirKey)
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to get DATA_DIR environment variable", err)
	}

	// 3. Generate gtid_purge.sql based on xtrabackup_binlog_info
	gtid := make([]string, 0)
	xbBinlogInfo := filepath.Join(dataDirValue, "xtrabackup_binlog_info")
	if !util.IsFileExist(xbBinlogInfo) {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "xtrabackup_binlog_info file not found", nil)
	}

	if content, err := os.ReadFile(xbBinlogInfo); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to read xtrabackup_binlog_info file", err)
	} else {
		line := bytes.Split(content, []byte("\n"))

		for index, field := range line {
			if index == 0 {
				fieldBytes := bytes.Fields(field)
				if len(fieldBytes) == 3 {
					lastElement := len(fieldBytes) - 1
					gtid = append(gtid, string(fieldBytes[lastElement]))
				}
			} else {
				if _, fieldBytes, found := bytes.Cut(field, []byte(",")); found {
					gtid = append(gtid, string(fieldBytes))
				}
			}
		}
	}

	if len(gtid) == 0 {
		return common.LogAndReturnSuccess(s.logger, newMysqlResponse, "no need to gtid purged")
	}

	gtidStr := strings.Join(gtid, ",")
	s.logger.Info(gtidStr)

	// 4. Create database connection
	db, err := s.newMysqlDB(ctx, req.GetUsername(), password, req.GetSocketFile())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "database connection failed", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			s.logger.Errorf("failed to close database connection: %v", err)
		}
	}()

	// 5. Handle replication mode reset operations
	if req.ArchMode == ArchMode_Replication {
		if _, err := db.Exec("STOP REPLICA;"); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to execute stop replica", err)
		}

		if _, err := db.Exec("RESET REPLICA;"); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to execute reset replica", err)
		}

		if _, err := db.Exec("RESET MASTER;"); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to execute reset master", err)
		}
	}

	// 6. Set GTID_PURGED
	if _, err := db.Exec("SET @@GLOBAL.GTID_PURGED=?;", gtidStr); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to set gtid_purged=%s", gtidStr), err)
	}

	return common.LogAndReturnSuccess(s.logger, newMysqlResponse, "gtid purge executed successfully")
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "mysql restore", map[string]interface{}{
		"parallel":     req.GetParallel(),
		"storage_type": req.GetStorageType(),
		"backup_file":  req.GetBackupFile(),
		"secret_key":   req.GetS3Storage().GetSecretKey(),
		"access_key":   req.GetS3Storage().GetAccessKey(),
		"bucket":       req.GetS3Storage().GetBucket(),
		"endpoint":     req.GetS3Storage().GetEndpoint(),
	})

	// 1. Check if service is stopped
	if _, err := s.slm.CheckServiceStopped(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "service status check failed", err)
	}

	s.logger.Infof("S3 object '%s' found", req.GetBackupFile())

	if req.GetS3Storage() != nil {
		// 4. Get environment variables
		dataDirValue, err := getEnvVarOrError(dataDirKey)
		if err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to get DATA_DIR environment variable", err)
		}

		relayLogDirValue, err := getEnvVarOrError(relayLogDirKey)
		if err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to get RELAY_LOG_DIR environment variable", err)
		}

		binLogDirValue, err := getEnvVarOrError(binLogDirKey)
		if err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to get BIN_LOG_DIR environment variable", err)
		}

		// 5. Clean directories
		if err := s.removeContents(dataDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to clear data directory '%s'", dataDirValue), err)
		}

		if err := s.removeContents(relayLogDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to clear relay log directory '%s'", relayLogDirValue), err)
		}

		if err := s.removeContents(binLogDirValue); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to clear bin log directory '%s'", binLogDirValue), err)
		}

		var xbcloudCmd *exec.Cmd
		if req.GetS3Storage().GetSsl() {
			xbcloudCmd = exec.CommandContext(ctx,
				"xbcloud",
				"get",
				"--storage=s3",
				"--s3-ssl",
				"--s3-verify-ssl=0",
				fmt.Sprintf("--s3-endpoint=%s", req.GetS3Storage().GetEndpoint()),
				fmt.Sprintf("--s3-bucket=%s", req.GetS3Storage().GetBucket()),
				fmt.Sprintf("--s3-access-key=%s", req.GetS3Storage().GetAccessKey()),
				fmt.Sprintf("--s3-secret-key=%s", req.GetS3Storage().GetSecretKey()),
				fmt.Sprintf("--parallel=%d", req.GetParallel()),
				req.GetBackupFile(),
			)
		} else {
			xbcloudCmd = exec.CommandContext(ctx,
				"xbcloud",
				"get",
				"--storage=s3",
				fmt.Sprintf("--s3-endpoint=%s", req.GetS3Storage().GetEndpoint()),
				fmt.Sprintf("--s3-bucket=%s", req.GetS3Storage().GetBucket()),
				fmt.Sprintf("--s3-access-key=%s", req.GetS3Storage().GetAccessKey()),
				fmt.Sprintf("--s3-secret-key=%s", req.GetS3Storage().GetSecretKey()),
				fmt.Sprintf("--parallel=%d", req.GetParallel()),
				req.GetBackupFile(),
			)
		}

		xbstreamCmd := exec.CommandContext(ctx,
			"xbstream",
			"-x",
			"-C",
			dataDirValue,
			fmt.Sprintf("--parallel=%d", req.GetParallel()),
		)

		// Use command executor for piped commands
		executor := common.NewCommandExecutor(ctx, s.logger)
		if err := executor.ExecutePipedCommands(xbcloudCmd, xbstreamCmd, "restore"); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "failed to execute restore commands", err)
		}

		xtrabackupCmd := exec.CommandContext(ctx,
			"xtrabackup",
			"--prepare",
			"--apply-log-only",
			fmt.Sprintf("--target-dir=%s", dataDirValue),
		)

		// Use command executor for single command
		if err := executor.ExecuteCommand(xtrabackupCmd, "restore"); err != nil {
			return common.LogAndReturnError(s.logger, newMysqlResponse, "xtrabackup command execution failed", err)
		}
	} else {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "storage type not supported: only 's3' is currently supported", nil)
	}

	return common.LogAndReturnSuccess(s.logger, newMysqlResponse, "restore from s3 succeeded")
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "mysql set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"username": req.GetUsername(),
		"password": req.GetPassword(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "service status check failed", err)
	}

	password, err := common.GetPlainTextPassword(req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "decrypt password failed", err)
	}

	// 2. Create database connection
	db, err := s.newMysqlDB(ctx, req.GetUsername(), password, req.GetSocketFile())
	if err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, "database connection failed", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			s.logger.Errorf("failed to close database connection: %v", err)
		}
	}()

	// 3. Execute set variable SQL
	execSQL := fmt.Sprintf(setVariableSql, req.GetKey(), req.GetValue())
	if _, err = db.ExecContext(ctx, execSQL); err != nil {
		return common.LogAndReturnError(s.logger, newMysqlResponse, fmt.Sprintf("failed to SET %s=%s", req.GetKey(), req.GetValue()), err)
	}

	return common.LogAndReturnSuccess(s.logger, newMysqlResponse, fmt.Sprintf("set variable %s=%s successfully", req.GetKey(), req.GetValue()))
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}

//func init() {
//	app.RegistryGrpcApp(svr)
//}
