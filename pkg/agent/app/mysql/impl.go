package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"

	"github.com/upmio/unit-operator/pkg/agent/vars"

	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
)

const (
	relayLogDirEnvKey = "RELAY_LOG_DIR"
	binLogDirEnvKey   = "BIN_LOG_DIR"
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

	socketFile  string
	confFile    string
	dataDir     string
	archMode    string
	relayLogDir string
	binLogDir   string
}

func (s *service) Config() error {
	s.mysqlOps = app.GetGrpcApp(appName).(MysqlOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)

	dataMount, err := util.IsEnvVarSet(vars.DataMountEnvKey)
	if err != nil {
		return err
	}

	confDir, err := util.IsEnvVarSet(vars.ConfigDirEnvKey)
	if err != nil {
		return err
	}

	dataDir, err := util.IsEnvVarSet(vars.DataDirEnvKey)
	if err != nil {
		return err
	}

	archMode, err := util.IsEnvVarSet(vars.ArchModeEnvKey)
	if err != nil {
		return err
	}

	relayLogDir, err := util.IsEnvVarSet(relayLogDirEnvKey)
	if err != nil {
		return err
	}

	binLogDir, err := util.IsEnvVarSet(binLogDirEnvKey)
	if err != nil {
		return err
	}

	s.socketFile = filepath.Join(dataMount, "mysqld.sock")
	s.confFile = filepath.Join(confDir, "mysql.cnf")
	s.dataDir = dataDir
	s.archMode = archMode
	s.relayLogDir = relayLogDir
	s.binLogDir = binLogDir

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterMysqlOperationServer(server, svr)
}

func (s *service) Clone(ctx context.Context, req *CloneRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mysql clone", map[string]interface{}{
		"username":          req.GetUsername(),
		"source_host":       req.GetSourceHost(),
		"source_port":       req.GetSourcePort(),
		"source_clone_user": req.GetSourceCloneUser(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create mysql connection
	db, err := s.newDBConn(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeDBConn(db)

	// Check clone plugin status
	var clonePluginStatus string

	if err := db.QueryRowContext(ctx, checkCloneAvaliableSql).Scan(&clonePluginStatus); err != nil {
		s.logger.Errorw("failed to check clone plugin status", zap.Error(err))
		return nil, err
	} else if clonePluginStatus != "ACTIVE" {
		s.logger.Errorw("clone plugin is not active", zap.String("plugin", clonePluginStatus))
		return nil, err
	}

	// Set valid donor list
	addr := net.JoinHostPort(req.GetSourceHost(), strconv.FormatInt(req.GetSourcePort(), 10))
	if _, err = db.ExecContext(ctx, SetValidDonorListSql, addr); err != nil {
		s.logger.Errorw("failed to set valid donor list", zap.Error(err))
		return nil, err
	}

	password, err := util.DecryptPlainTextPassword(req.GetSourceCloneUser())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetSourceCloneUser()))
		return nil, err
	}

	// The clone command itself may return a warning, but it will not fail immediately here
	execSQL := fmt.Sprintf(ExecCloneSql, req.GetSourceCloneUser(), req.GetSourceHost(), req.GetSourcePort(), password)
	if _, err = db.ExecContext(ctx, execSQL); err != nil {
		s.logger.Warnw(
			"clone command returned error, clone may still be running",
			zap.Error(err),
		)
	}

	if err := s.waitForCloneComplete(ctx, req.GetUsername(), time.Minute); err != nil {
		s.logger.Errorw("failed to wait for clone complete", zap.Error(err))
		return nil, err
	}

	s.logger.Info("clone successfully")
	return nil, nil
}

func (s *service) waitForCloneComplete(ctx context.Context, username string, timeout time.Duration) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			db, err := s.newDBConn(ctx, username)
			if err != nil {
				s.logger.Warnw(
					"clone in progress: failed to establish database connection, will retry",
					zap.Error(err),
				)
				continue
			}

			var status, errMsg string
			err = db.QueryRowContext(ctx, getCloneStatusSql).Scan(&status, &errMsg)
			s.closeDBConn(db)

			if err != nil {
				s.logger.Warnw(
					"clone in progress: failed to query clone status, will retry",
					zap.Error(err),
				)
				continue
			}

			s.logger.Infow("clone status updated", zap.String("status", status))

			switch status {
			case "Completed":
				return nil
			case "Failed":
				return fmt.Errorf("clone failed: %s", errMsg)
			}

		case <-timeoutCh:
			return fmt.Errorf("timeout waiting for clone to complete")
		}
	}
}

func (s *service) PhysicalBackup(ctx context.Context, req *PhysicalBackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mysql physical backup", map[string]interface{}{
		"username":    req.GetUsername(),
		"backup_tool": req.GetTool().String(),
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

	var cmd1, cmd2 *exec.Cmd

	switch req.GetTool() {
	case Tool_Xtrabackup:
		cmd1 = exec.CommandContext(ctx,
			"xtrabackup",
			fmt.Sprintf("--defaults-file=%s", s.confFile),
			fmt.Sprintf("--socket=%s", s.socketFile),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--extra-lsndir=%s", "/tmp/s3_tmp_dir"),
			fmt.Sprintf("--target-dir=%s", "/tmp/s3_tmp_dir"),
			"--backup",
			"--stream=xbstream",
		)

		if req.GetObjectStorage().GetSsl() {
			cmd2 = exec.CommandContext(ctx,
				"xbcloud",
				"put",
				"--storage=s3",
				"--s3-ssl",
				"--s3-verify-ssl=0",
				"--parallel=10",
				fmt.Sprintf("--s3-endpoint=https://%s", req.GetObjectStorage().GetEndpoint()),
				fmt.Sprintf("--s3-bucket=%s", req.GetObjectStorage().GetBucket()),
				fmt.Sprintf("--s3-access-key=%s", req.GetObjectStorage().GetAccessKey()),
				fmt.Sprintf("--s3-secret-key=%s", req.GetObjectStorage().GetSecretKey()),
				req.GetBackupFile(),
			)
		} else {
			cmd2 = exec.CommandContext(ctx,
				"xbcloud",
				"put",
				"--storage=s3",
				"--parallel=10",
				fmt.Sprintf("--s3-endpoint=http://%s", req.GetObjectStorage().GetEndpoint()),
				fmt.Sprintf("--s3-bucket=%s", req.GetObjectStorage().GetBucket()),
				fmt.Sprintf("--s3-access-key=%s", req.GetObjectStorage().GetAccessKey()),
				fmt.Sprintf("--s3-secret-key=%s", req.GetObjectStorage().GetSecretKey()),
				req.GetBackupFile(),
			)
		}
	default:
		err = fmt.Errorf("unsupported tool: %s", req.GetTool().String())
		s.logger.Errorw("failed to physical backup mysql", zap.Error(err))
		return nil, err
	}

	// Use command executor for piped commands
	executor := common.NewCommandExecutor(s.logger)
	if err := executor.ExecutePipedCommands(cmd1, cmd2, "backup"); err != nil {
		s.logger.Errorw("failed to physical backup mysql", zap.Error(err))
		return nil, err
	}

	s.logger.Info("physical backup mysql successfully")

	return nil, nil
}

func (s *service) LogicalBackup(ctx context.Context, req *LogicalBackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mysql logical backup", map[string]interface{}{
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
			"mysqldump",
			fmt.Sprintf("--defaults-file=%s", s.confFile),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--socket=%s", s.socketFile),
			"--single-transaction",
			"--set-gtid-purged=OFF",
			"--all-databases",
		)
	case LogicalBackupMode_Database:
		cmd = exec.CommandContext(ctx,
			"mysqldump",
			fmt.Sprintf("--defaults-file=%s", s.confFile),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--socket=%s", s.socketFile),
			"--single-transaction",
			"--set-gtid-purged=OFF",
			fmt.Sprintf("--databases %s", req.GetDatabase()),
		)
	case LogicalBackupMode_Table:
		cmd = exec.CommandContext(ctx,
			"mysqldump",
			fmt.Sprintf("--defaults-file=%s", s.confFile),
			fmt.Sprintf("--user=%s", req.GetUsername()),
			fmt.Sprintf("--password=%s", password),
			fmt.Sprintf("--socket=%s", s.socketFile),
			"--single-transaction",
			"--set-gtid-purged=OFF",
			req.GetDatabase(),
			req.GetTable(),
		)
	default:
		err = fmt.Errorf("unsupported logical backup mode %s", req.GetLogicalBackupMode().String())
		s.logger.Errorw("failed to logical backup mysql", zap.Error(err))
		return nil, err
	}

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

	s.logger.Info("logical backup mysql successfully")
	return nil, nil
}

func (s *service) GtidPurge(ctx context.Context, req *GtidPurgeRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mysql gtid purge", map[string]interface{}{
		"username": req.GetUsername(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	gtid, err := s.generateGtidPurgeSql()
	if err != nil {
		s.logger.Errorw("failed to generate gtid string", zap.Error(err))
		return nil, err
	}

	if gtid == "" {
		s.logger.Infow("empty gtid string, no need to purge")
		return nil, nil
	}

	s.logger.Infow("generate gtid string", zap.String("gtid", gtid))

	// Create mysql connection
	db, err := s.newDBConn(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeDBConn(db)

	// Handle replication mode reset operations
	if s.archMode != "group_replication" {
		if _, err := db.Exec("STOP REPLICA;"); err != nil {
			s.logger.Errorw("failed to stop replication", zap.Error(err))
			return nil, err
		}

		if _, err := db.Exec("RESET REPLICA;"); err != nil {
			s.logger.Errorw("failed to reset replication", zap.Error(err))
			return nil, err
		}

		if _, err := db.Exec("RESET MASTER;"); err != nil {
			s.logger.Errorw("failed to reset master", zap.Error(err))
			return nil, err
		}
	}

	// Set GTID_PURGED
	if _, err := db.Exec("SET @@GLOBAL.GTID_PURGED=?;", gtid); err != nil {
		s.logger.Errorw("failed to set gtid purged", zap.Error(err))
		return nil, err
	}

	s.logger.Info("gtid purge executed successfully")

	return nil, nil
}

// generateGtidPurgeSql Generate gtid_purge.sql based on xtrabackup_binlog_info
func (s *service) generateGtidPurgeSql() (string, error) {
	gtid := make([]string, 0)

	binlogInfo := filepath.Join(s.dataDir, "xtrabackup_binlog_info")
	if !util.IsFileExist(binlogInfo) {
		return "", fmt.Errorf("xtrabackup_binlog_info file not found")
	}

	content, err := os.ReadFile(binlogInfo)
	if err != nil {
		return "", fmt.Errorf("failed to read xtrabackup_binlog_info file: %w", err)
	}

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

	return strings.Join(gtid, ","), nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mysql restore", map[string]interface{}{
		"backup_file": req.GetBackupFile(),
		"tool":        req.GetTool().String(),
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

	// Clean directories
	if err := s.removeContents(s.dataDir); err != nil {
		s.logger.Errorw("failed to remove contents", zap.Error(err), zap.String("dir", s.dataDir))
		return nil, err
	}

	if err := s.removeContents(s.relayLogDir); err != nil {
		s.logger.Errorw("failed to remove contents", zap.Error(err), zap.String("dir", s.relayLogDir))
		return nil, err
	}

	if err := s.removeContents(s.binLogDir); err != nil {
		s.logger.Errorw("failed to remove contents", zap.Error(err), zap.String("dir", s.binLogDir))
		return nil, err
	}

	var cmd1, cmd2 *exec.Cmd

	switch req.GetTool() {
	case Tool_Xtrabackup:
		cmd2 = exec.CommandContext(ctx,
			"xbstream",
			"-x",
			"--parallel=10",
			"-C",
			s.dataDir,
		)

		if req.GetObjectStorage().GetSsl() {
			cmd1 = exec.CommandContext(ctx,
				"xbcloud",
				"get",
				"--storage=s3",
				"--s3-ssl",
				"--s3-verify-ssl=0",
				"--parallel=10",
				fmt.Sprintf("--s3-endpoint=https://%s", req.GetObjectStorage().GetEndpoint()),
				fmt.Sprintf("--s3-bucket=%s", req.GetObjectStorage().GetBucket()),
				fmt.Sprintf("--s3-access-key=%s", req.GetObjectStorage().GetAccessKey()),
				fmt.Sprintf("--s3-secret-key=%s", req.GetObjectStorage().GetSecretKey()),
				req.GetBackupFile(),
			)
		} else {
			cmd1 = exec.CommandContext(ctx,
				"xbcloud",
				"get",
				"--storage=s3",
				"--parallel=10",
				fmt.Sprintf("--s3-endpoint=http://%s", req.GetObjectStorage().GetEndpoint()),
				fmt.Sprintf("--s3-bucket=%s", req.GetObjectStorage().GetBucket()),
				fmt.Sprintf("--s3-access-key=%s", req.GetObjectStorage().GetAccessKey()),
				fmt.Sprintf("--s3-secret-key=%s", req.GetObjectStorage().GetSecretKey()),
				req.GetBackupFile(),
			)
		}
	default:
		err := fmt.Errorf("unsupported tool: %s", req.GetTool().String())
		s.logger.Errorw("failed to restore mysql", zap.Error(err))
		return nil, err
	}

	// Use command executor for piped commands
	executor := common.NewCommandExecutor(s.logger)
	if err := executor.ExecutePipedCommands(cmd1, cmd2, "restore"); err != nil {
		s.logger.Errorw("failed to restore mysql", zap.Error(err))
		return nil, err
	}

	cmd := exec.CommandContext(ctx,
		"xtrabackup",
		"--prepare",
		"--apply-log-only",
		fmt.Sprintf("--target-dir=%s", s.dataDir),
	)

	// Use command executor for single command
	if err := executor.ExecuteCommand(cmd, "restore"); err != nil {
		s.logger.Errorw("failed to restore mysql", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore mysql successfully")
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

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mysql set variable", map[string]interface{}{
		"username": req.GetUsername(),
		"key":      req.GetKey(),
		"value":    req.GetValue(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create mysql connection
	db, err := s.newDBConn(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeDBConn(db)

	// Execute set variable SQL
	execSQL := fmt.Sprintf(setVariableSql, req.GetKey(), req.GetValue())
	if _, err = db.ExecContext(ctx, execSQL); err != nil {
		s.logger.Errorw("failed to set variable", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))
		return nil, err
	}

	s.logger.Info("set variable successfully")

	return nil, nil
}

// newDBConn creates a MySQL database connection
func (s *service) newDBConn(ctx context.Context, username string) (*sql.DB, error) {
	password, err := util.DecryptPlainTextPassword(username)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), "username", username)
		return nil, err
	}

	dsn := fmt.Sprintf("%s:%s@unix(%s)/?timeout=5s&multiStatements=true&interpolateParams=true",
		username, password, s.socketFile)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.PingContext(ctx); err != nil {
		s.logger.Errorw("failed to ping mysql", zap.Error(err))

		s.closeDBConn(db)
		return nil, err
	}

	return db, nil
}

func (s *service) closeDBConn(client *sql.DB) {
	if client == nil {
		return
	}

	if err := client.Close(); err != nil {
		s.logger.Errorw("failed to close mysql connection", zap.Error(err))
	}
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
