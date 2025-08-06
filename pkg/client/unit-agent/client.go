package unit_agent

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
	"log"
	"net"

	"github.com/upmio/unit-operator/pkg/agent/app/config"
	"github.com/upmio/unit-operator/pkg/agent/app/service"
)

func SyncConfig(agentHostType, unitsetHeadlessSvc, host, port, namespace, templateConfigmapName, valueConfigmapName, mainContainerName string, extendConfigmaps []string) (string, error) {

	addr := fmtUnitAgentDomainAddr(agentHostType, unitsetHeadlessSvc, host, namespace, port)
	//klog.Infof("[SyncConfig] addr: [%s]", addr)

	//conn, err := grpc.Dial(addr, grpc.WithInsecure())
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Errorf("[SyncConfig] make connection err:[%s]", err.Error())
		return "", fmt.Errorf("sync config make connection error:[%s]", err.Error())
	}
	defer conn.Close()

	client := config.NewSyncConfigServiceClient(conn)

	req := config.SyncConfigRequest{
		TemplateConfigmapName: templateConfigmapName,
		ValueConfigmapName:    valueConfigmapName,
		Namespace:             namespace,
		Key:                   mainContainerName,
		ExtendValueConfigmaps: extendConfigmaps,
	}

	resp, err := client.SyncConfig(context.Background(), &req)
	if err != nil {
		return resp.GetMessage(), err
	}

	return "", nil
}

func ServiceLifecycleManagement(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {

	addr := fmtUnitAgentDomainAddr(agentHostType, unitsetHeadlessSvc, host, namespace, port)
	//klog.Infof("[ServiceLifecycleManagement] addr: [%s]", addr)

	//conn, err := grpc.Dial(addr, grpc.WithInsecure())
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		klog.Errorf("[ServiceLifecycleManagement] make connection err:[%s]", err.Error())
		return "", fmt.Errorf("service lifecycle management make connection error:[%s]", err.Error())
	}
	defer conn.Close()

	req := service.ServiceRequest{}

	client := service.NewServiceLifecycleClient(conn)

	switch actionType {
	case "start":
		resp, err := client.StartService(context.Background(), &req)
		if err != nil {
			return resp.GetMessage(), err
		}
	case "stop":
		resp, err := client.StopService(context.Background(), &req)
		if err != nil {
			return resp.GetMessage(), err
		}
	case "restart":
		resp, err := client.RestartService(context.Background(), &req)
		if err != nil {
			return resp.GetMessage(), err
		}
	default:
		return "", fmt.Errorf("[%s] server not support", actionType)
	}

	return fmt.Sprintf("[%s] server ok", actionType), nil
}

func GetServiceProcessState(agentHostType, unitsetHeadlessSvc, host, namespace, port string) (string, error) {

	addr := fmtUnitAgentDomainAddr(agentHostType, unitsetHeadlessSvc, host, namespace, port)

	//conn, err := grpc.Dial(addr, grpc.WithInsecure())
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	req := service.ServiceRequest{}

	client := service.NewServiceLifecycleClient(conn)

	resp, err := client.GetServiceStatus(context.Background(), &req)
	if err != nil {
		return "unknown", err
	}

	return parserProcessState(int32(resp.GetServiceStatus())), nil
}

func fmtUnitAgentDomainAddr(agentHostType, unitsetHeadlessSvc, host, namespace, port string) string {
	switch agentHostType {
	case "domain":
		// xpfbhzrx-kafka-yo7-1.xpfbhzrx-kafka-yo7-headless.test2024.svc
		fullDomainName := fmt.Sprintf("%s.%s.%s.svc", host, unitsetHeadlessSvc, namespace)
		return net.JoinHostPort(fullDomainName, port)
	case "ip":
		return net.JoinHostPort(host, port)
	}

	return ""
}

func parserProcessState(processState int32) string {
	out := ""
	//const (
	//	ProcessState_StateStopped  ProcessState = 0
	//	ProcessState_StateStarting ProcessState = 1
	//	ProcessState_StateRunning  ProcessState = 2
	//	ProcessState_StateBackoff  ProcessState = 3
	//	ProcessState_StateStopping ProcessState = 4
	//	ProcessState_StateExited   ProcessState = 5
	//	ProcessState_StateFatal    ProcessState = 6
	//	ProcessState_StateUnknown  ProcessState = 7
	//)
	switch processState {
	case 0:
		out = "stopped"
	case 1:
		out = "starting"
	case 2:
		out = "running"
	case 3:
		out = "backoff"
	case 4:
		out = "stopping"
	case 5:
		out = "exited"
	case 6:
		out = "fatal"
	case 7:
		out = "unknown"
	default:
		out = "unknown"
	}

	return out
}
