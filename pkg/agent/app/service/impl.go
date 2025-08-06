package service

import (
	"context"
	"fmt"
	"github.com/abrander/go-supervisord"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// service service instance
	svr = &service{}
)

type service struct {
	service ServiceLifecycleServer
	UnimplementedServiceLifecycleServer
	logger *zap.SugaredLogger
	client *supervisord.Client
}

// Common helper methods

// newServiceResponse creates a new Service Response with the given message
func newServiceResponse(message string) *ServiceResponse {
	return &ServiceResponse{Message: message}
}

// getProcessInfo gets the process information for unit_app
func (s *service) getProcessInfo() (*supervisord.ProcessInfo, error) {
	pi, err := s.client.GetProcessInfo("unit_app")
	if err != nil {
		return nil, fmt.Errorf("failed to get process information: %v", err)
	}
	return pi, nil
}


// convertProcessState converts supervisord state to ProcessState
func convertProcessState(state supervisord.ProcessState) ProcessState {
	switch state {
	case supervisord.StateStopped:
		return ProcessState_StateStopped
	case supervisord.StateStarting:
		return ProcessState_StateStarting
	case supervisord.StateRunning:
		return ProcessState_StateRunning
	case supervisord.StateBackoff:
		return ProcessState_StateBackoff
	case supervisord.StateStopping:
		return ProcessState_StateStopping
	case supervisord.StateExited:
		return ProcessState_StateExited
	case supervisord.StateFatal:
		return ProcessState_StateFatal
	case supervisord.StateUnknown:
		return ProcessState_StateUnknown
	default:
		return ProcessState_StateUnknown
	}
}

func (s *service) Config() error {
	client, err := conf.GetConf().Supervisor.GetSupervisorClient()
	if err != nil {
		return err
	}

	s.client = client
	s.service = app.GetGrpcApp(appName).(ServiceLifecycleServer)
	s.logger = zap.L().Named("[SLM]").Sugar()

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterServiceLifecycleServer(server, svr)
}

func (s *service) StartService(_ context.Context, _ *ServiceRequest) (*ServiceResponse, error) {
	// 1. Get process information
	pi, err := s.getProcessInfo()
	if err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to get process information", err)
	}

	// 2. Check if service is already running
	if pi.State == supervisord.StateRunning {
		return common.LogAndReturnSuccess(s.logger, newServiceResponse, "service is already running, no start needed")
	}

	// 3. Start the service
	if err = s.client.StartProcess("unit_app", true); err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to start service", err)
	}

	return common.LogAndReturnSuccess(s.logger, newServiceResponse, "start service process successfully")
}

func (s *service) StopService(_ context.Context, _ *ServiceRequest) (*ServiceResponse, error) {
	// 1. Get process information
	pi, err := s.getProcessInfo()
	if err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to get process information", err)
	}

	// 2. Check if service is already stopped
	if pi.State == supervisord.StateStopped {
		return common.LogAndReturnSuccess(s.logger, newServiceResponse, "service is already stopped, no stop needed")
	}

	// 3. Stop the service
	if err = s.client.StopProcess("unit_app", true); err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to stop service", err)
	}

	return common.LogAndReturnSuccess(s.logger, newServiceResponse, "stop service process successfully")
}

func (s *service) GetServiceStatus(_ context.Context, _ *ServiceRequest) (*ServiceStatusResponse, error) {
	// 1. Get process information
	pi, err := s.getProcessInfo()
	if err != nil {
		s.logger.Error("failed to get process information: %v", err)
		return nil, fmt.Errorf("failed to get process information: %v", err)
	}

	// 2. Convert and return status
	status := convertProcessState(pi.State)
	return &ServiceStatusResponse{
		ServiceStatus: status,
	}, nil
}

func (s *service) RestartService(_ context.Context, _ *ServiceRequest) (*ServiceResponse, error) {
	// 1. Get process information
	pi, err := s.getProcessInfo()
	if err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to get process information", err)
	}

	// 2. Stop service if not already stopped
	if pi.State != supervisord.StateStopped {
		if err = s.client.StopProcess("unit_app", true); err != nil {
			return common.LogAndReturnError(s.logger, newServiceResponse, "failed to stop service", err)
		}
		s.logger.Info("stop service process successfully")
	} else {
		s.logger.Info("service is already stopped, no stop needed")
	}

	// 3. Start the service
	if err = s.client.StartProcess("unit_app", true); err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to start service", err)
	}

	return common.LogAndReturnSuccess(s.logger, newServiceResponse, "restart service process successfully")
}

func (s *service) CheckServiceStatus(ctx context.Context, _ *ServiceRequest) (*ServiceResponse, error) {
	statusResp, err := s.GetServiceStatus(ctx, &ServiceRequest{})
	if err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to get service status", err)
	}
	if statusResp.ServiceStatus != ProcessState_StateRunning {
		errMsg := fmt.Sprintf("service is not running, current status: %s", statusResp.ServiceStatus.String())
		return common.LogAndReturnError(s.logger, newServiceResponse, errMsg, nil)
	}
	return common.LogAndReturnSuccess(s.logger, newServiceResponse, "service is running")
}

func (s *service) CheckServiceStopped(ctx context.Context, _ *ServiceRequest) (*ServiceResponse, error) {
	statusResp, err := s.GetServiceStatus(ctx, &ServiceRequest{})
	if err != nil {
		return common.LogAndReturnError(s.logger, newServiceResponse, "failed to get service status", err)
	}
	if statusResp.ServiceStatus == ProcessState_StateStarting ||
		statusResp.ServiceStatus == ProcessState_StateUnknown ||
		statusResp.ServiceStatus == ProcessState_StateRunning {
		errMsg := fmt.Sprintf("service is not stopped, current status: %s", statusResp.ServiceStatus.String())
		return common.LogAndReturnError(s.logger, newServiceResponse, errMsg, nil)
	}
	return common.LogAndReturnSuccess(s.logger, newServiceResponse, "service is stopped")
}

func init() {
	app.RegistryGrpcApp(svr)
}
