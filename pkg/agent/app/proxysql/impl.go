package proxysql

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	proxysqlOps ProxysqlOperationServer
	UnimplementedProxysqlOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

func (s *service) Config() error {
	s.proxysqlOps = app.GetGrpcApp(appName).(ProxysqlOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterProxysqlOperationServer(server, svr)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "proxysql set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"section":  req.GetSection(),
		"username": req.GetUsername(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create proxysql connection
	db, err := s.newDBConn(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeDBConn(db)

	// Execute set variable
	execSQL := fmt.Sprintf("SET %s-%s = %s", req.GetSection(), req.GetKey(), req.GetValue())
	if _, err = db.ExecContext(ctx, execSQL); err != nil {
		s.logger.Errorw("failed to set variable", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))
		return nil, err
	}

	// Load variables to runtime based on section
	switch req.GetSection() {
	case "admin":
		if _, err = db.ExecContext(ctx, `LOAD ADMIN VARIABLES TO RUNTIME`); err != nil {
			s.logger.Errorw("failed to load admin section variable to runtime", zap.Error(err))
			return nil, err
		}
	case "mysql":
		if _, err = db.ExecContext(ctx, `LOAD MYSQL VARIABLES TO RUNTIME`); err != nil {
			s.logger.Errorw("failed to load mysql section variable to runtime variable", zap.Error(err))
			return nil, err
		}
	}

	s.logger.Info("set variable successfully")
	return nil, nil
}

// newDBConn creates a ProxySQL database connection
func (s *service) newDBConn(ctx context.Context, username string) (*sql.DB, error) {
	password, err := util.DecryptPlainTextPassword(username)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", username))
		return nil, err
	}

	addr := net.JoinHostPort("127.0.0.1", "6032")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/?timeout=5s&multiStatements=true&interpolateParams=true",
		username, password, addr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.PingContext(ctx); err != nil {
		s.logger.Errorw("failed to ping proxysql", zap.Error(err))

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
		s.logger.Errorw("failed to close proxysql connection", zap.Error(err))
	}
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
