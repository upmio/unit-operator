package proxysql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"os"
	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
)

const (
	portKey = "ADMIN_PORT"
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

// Common helper methods

// newProxysqlResponse creates a new ProxySQL Response with the given message
func newProxysqlResponse(message string) *Response {
	return &Response{Message: message}
}

// createProxySQLDB creates a ProxySQL database connection
func (s *service) createProxySQLDB(ctx context.Context, username, password string) (*sql.DB, error) {
	port := os.Getenv(portKey)
	if port == "" {
		return nil, fmt.Errorf("environment variable %s is not set", portKey)
	}

	addr := net.JoinHostPort("127.0.0.1", port)
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/?timeout=5s&multiStatements=true&interpolateParams=true",
		username, password, addr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection %s: %v", addr, err)
	}

	if err = db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping %s: %v, and failed to close db: %v", addr, err, closeErr)
		}
		return nil, fmt.Errorf("failed to ping %s: %v", addr, err)
	}

	return db, nil
}

func (s *service) Config() error {
	s.proxysqlOps = app.GetGrpcApp(appName).(ProxysqlOperationServer)
	s.logger = zap.L().Named("[PROXYSQL]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterProxysqlOperationServer(server, svr)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*Response, error) {
	s.logger.With(
		"key", req.GetKey(),
		"value", req.GetValue(),
		"section", req.GetSection(),
		"username", req.GetUsername(),
		"password", req.GetPassword(),
	).Info("receive proxysql set variable request")

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newProxysqlResponse, "service status check failed", err)
	}

	// 2. Create database connection
	db, err := s.createProxySQLDB(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newProxysqlResponse, "database connection failed", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			s.logger.Errorf("failed to close database connection: %v", err)
		}
	}()

	// 3. Set variable
	execSql := fmt.Sprintf(setVariableSql, req.GetSection(), req.GetKey(), req.GetValue())
	if _, err = db.ExecContext(ctx, execSql); err != nil {
		return common.LogAndReturnError(s.logger, newProxysqlResponse, fmt.Sprintf("failed to SET %s=%s", req.GetKey(), req.GetValue()), err)
	}

	// 4. Load variables to runtime based on section
	switch req.GetSection() {
	case "admin":
		if _, err = db.ExecContext(ctx, loadAdminVariableSql); err != nil {
			return common.LogAndReturnError(s.logger, newProxysqlResponse, "failed to execute LOAD ADMIN VARIABLES TO RUNTIME", err)
		}
	case "mysql":
		if _, err = db.ExecContext(ctx, loadMysqlVariableSql); err != nil {
			return common.LogAndReturnError(s.logger, newProxysqlResponse, "failed to execute LOAD MYSQL VARIABLES TO RUNTIME", err)
		}
	}

	return common.LogAndReturnSuccess(s.logger, newProxysqlResponse, fmt.Sprintf("set variable %s=%s successfully", req.GetKey(), req.GetValue()))
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}

//func init() {
//	app.RegistryGrpcApp(svr)
//}
