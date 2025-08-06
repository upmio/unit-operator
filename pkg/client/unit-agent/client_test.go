package unit_agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/upmio/unit-operator/pkg/agent/app/config"
	"github.com/upmio/unit-operator/pkg/agent/app/service"
	"google.golang.org/grpc"
)

func TestSyncConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := config.NewMockSyncConfigServiceClient(ctrl)
	conn := grpc.ClientConn{}
	defer conn.Close()

	client := config.NewSyncConfigServiceClient(&conn)
	assert.NotNil(t, client)

	mockClient.EXPECT().SyncConfig(gomock.Any(), gomock.Any()).Return(&config.SyncConfigResponse{Message: "success"}, nil)

	addr := fmtUnitAgentDomainAddr("domain", "svc", "host", "namespace", "port")
	conn1, err := grpc.Dial(addr, grpc.WithInsecure())
	assert.NoError(t, err)
	defer conn1.Close()

	req := config.SyncConfigRequest{
		Namespace:             "namespace",
		TemplateConfigmapName: "configmapName",
		ExtendValueConfigmaps: []string{"extendConfigmap"},
	}

	resp, err := mockClient.SyncConfig(context.Background(), &req)
	if err != nil {
		assert.NoError(t, err)
	}

	assert.Equal(t, "success", resp.GetMessage())
}

func TestServiceLifecycleManagement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service.NewMockServiceLifecycleClient(ctrl)
	conn := grpc.ClientConn{}
	defer conn.Close()

	client := service.NewServiceLifecycleClient(&conn)
	assert.NotNil(t, client)

	// Test start service
	mockClient.EXPECT().StartService(gomock.Any(), gomock.Any()).Return(&service.ServiceResponse{Message: "started"}, nil)
	resp, err := client.StartService(context.Background(), &service.ServiceRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "started", resp.GetMessage())

	// Test stop service
	mockClient.EXPECT().StopService(gomock.Any(), gomock.Any()).Return(&service.ServiceResponse{Message: "stopped"}, nil)
	resp, err = client.StopService(context.Background(), &service.ServiceRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "stopped", resp.GetMessage())

	// Test restart service
	mockClient.EXPECT().RestartService(gomock.Any(), gomock.Any()).Return(&service.ServiceResponse{Message: "restarted"}, nil)
	resp, err = client.RestartService(context.Background(), &service.ServiceRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "restarted", resp.GetMessage())
}

func TestGetServiceProcessState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := service.NewMockServiceLifecycleClient(ctrl)
	conn := grpc.ClientConn{}
	defer conn.Close()

	client := service.NewServiceLifecycleClient(&conn)
	assert.NotNil(t, client)

	mockClient.EXPECT().GetServiceStatus(gomock.Any(), gomock.Any()).Return(&service.ServiceStatusResponse{
		ServiceStatus: 2,
	}, nil)
	resp, err := mockClient.GetServiceStatus(context.Background(), &service.ServiceRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.GetServiceStatus())

	state := parserProcessState(int32(resp.GetServiceStatus()))
	assert.Equal(t, "running", state)
}

func TestFmtUnitAgentDomainAddr(t *testing.T) {
	tests := []struct {
		agentHostType      string
		unitsetHeadlessSvc string
		host               string
		namespace          string
		port               string
		expected           string
	}{
		{"domain", "svc", "host", "namespace", "port", "host.svc.namespace.svc:port"},
		{"ip", "", "192.168.1.1", "", "port", "192.168.1.1:port"},
	}

	for _, tt := range tests {
		t.Run(tt.agentHostType, func(t *testing.T) {
			addr := fmtUnitAgentDomainAddr(tt.agentHostType, tt.unitsetHeadlessSvc, tt.host, tt.namespace, tt.port)
			assert.Equal(t, tt.expected, addr)
		})
	}
}

func TestParserProcessState(t *testing.T) {
	tests := []struct {
		processState int32
		expected     string
	}{
		{0, "stopped"},
		{1, "starting"},
		{2, "running"},
		{3, "backoff"},
		{4, "stopping"},
		{5, "exited"},
		{6, "fatal"},
		{7, "unknown"},
		{8, "unknown"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("processState_%d", tt.processState), func(t *testing.T) {
			state := parserProcessState(tt.processState)
			assert.Equal(t, tt.expected, state)
		})
	}
}
