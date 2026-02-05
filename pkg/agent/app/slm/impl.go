package slm

import (
	"context"
	"fmt"

	"github.com/abrander/go-supervisord"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// service instance
	svr = &service{}
)

type supervisorClient interface {
	GetProcessInfo(string) (*supervisord.ProcessInfo, error)
	StartProcess(string, bool) error
	StopProcess(string, bool) error
}

type service struct {
	service ServiceLifecycleServer
	UnimplementedServiceLifecycleServer
	logger *zap.SugaredLogger
	client supervisorClient
}

const (
	processName = "unit_app"
)

func (s *service) Config() error {
	s.service = app.GetGrpcApp(appName).(ServiceLifecycleServer)
	s.logger = zap.L().Named(appName).Sugar()

	client, err := conf.GetConf().GetSupervisorClient()
	if err != nil {
		return err
	}
	s.client = client

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterServiceLifecycleServer(server, svr)
}

func (s *service) StartProcess(_ context.Context, _ *common.Empty) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "start process", map[string]interface{}{})

	// Get process information
	pi, err := s.getProcessInfo()
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}

	// Check if process is already running
	if pi.State == supervisord.StateRunning {
		s.logger.Info("process is already running, no start needed")
		return nil, nil
	}

	// Start process
	if err = s.client.StartProcess(processName, true); err != nil {
		s.logger.Errorw("failed to start process", zap.Error(err))
		return nil, err
	}

	s.logger.Info("start process successfully")

	return nil, nil
}

func (s *service) StopProcess(_ context.Context, _ *common.Empty) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "stop process", map[string]interface{}{})

	// Get process information
	pi, err := s.getProcessInfo()
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}

	// Check if process is already stopped
	if pi.State == supervisord.StateStopped {
		s.logger.Info("process is already stopped, no stop needed")
		return nil, nil
	}

	// Stop process
	if err = s.client.StopProcess(processName, true); err != nil {
		s.logger.Errorw("failed to stop process", zap.Error(err))
		return nil, err
	}

	s.logger.Info("stop process successfully")

	return nil, nil
}

func (s *service) RestartProcess(ctx context.Context, empty *common.Empty) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "restart process", map[string]interface{}{})

	// Stop the process
	if _, err := s.StopProcess(ctx, empty); err != nil {
		return nil, err
	}

	// Start the service
	if _, err := s.StartProcess(ctx, empty); err != nil {
		return nil, err
	}

	s.logger.Info("restart process successfully")

	return nil, nil
}

func (s *service) CheckProcessStarted(_ context.Context, _ *common.Empty) (*common.Empty, error) {
	pi, err := s.getProcessInfo()
	if err != nil {
		return nil, err
	}

	if pi.State != supervisord.StateRunning {
		return nil, fmt.Errorf("process is not started, current status is %d", pi.State)
	}

	return nil, nil
}

func (s *service) CheckProcessStopped(_ context.Context, _ *common.Empty) (*common.Empty, error) {
	pi, err := s.getProcessInfo()
	if err != nil {
		return nil, err
	}

	if pi.State == supervisord.StateStarting || pi.State == supervisord.StateUnknown || pi.State == supervisord.StateRunning {
		return nil, fmt.Errorf("process is not stopped, current status is %d", pi.State)
	}

	return nil, nil
}

// getProcessInfo gets the process information for unit_app
func (s *service) getProcessInfo() (*supervisord.ProcessInfo, error) {
	pi, err := s.client.GetProcessInfo(processName)
	if err != nil {
		return nil, fmt.Errorf("failed to get process information: %v", err)
	}
	return pi, nil
}

func init() {
	app.RegistryGrpcApp(svr)
}
