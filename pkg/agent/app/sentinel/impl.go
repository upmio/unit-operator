package sentinel

import (
	"context"
	"fmt"
	composev1alpha1 "github.com/upmio/compose-operator/api/v1alpha1"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/event"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

var (
	// service service instance
	svr = &service{}
)

type service struct {
	sentinelOps   SentinelOperationServer
	composeClient client.Client
	recorder      event.IEventRecorder
	UnimplementedSentinelOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

// Common helper methods

// newSentinelResponse creates a new Sentinel Response with the given message
func newSentinelResponse(message string) *Response {
	return &Response{Message: message}
}

func (s *service) Config() error {
	s.sentinelOps = app.GetGrpcApp(appName).(SentinelOperationServer)
	c, err := conf.GetConf().GetComposeClient()
	if err != nil {
		return err
	}

	if s.recorder, err = event.NewIEventRecorder(); err != nil {
		return err
	}

	s.composeClient = c
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)

	s.logger = zap.L().Named("[SENTINEL]").Sugar()
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterSentinelOperationServer(server, svr)
}

func (s *service) UpdateRedisReplication(ctx context.Context, req *UpdateRedisReplicationRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "redis sentinel update redis replication", map[string]interface{}{
		"namespace":              req.GetNamespace(),
		"master_host":            req.GetMasterHost(),
		"master_port":            req.GetMasterPort(),
		"unit_name":              req.GetSelfUnitName(),
		"redis_replication_name": req.GetRedisReplicationName(),
	})

	instance := &composev1alpha1.RedisReplication{}

	// 1. Fetch redis replication resource
	if err := s.composeClient.Get(context.Background(), types.NamespacedName{
		Namespace: req.GetNamespace(),
		Name:      req.GetRedisReplicationName(),
	}, instance); err != nil {
		return common.LogAndReturnErrorWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover",
			fmt.Sprintf("failed to fetch redis replication[%s] in namespace[%s]", req.GetRedisReplicationName(), req.GetNamespace()), err)
	}

	// 2. Check if master host is already set correctly
	if req.GetMasterHost() == instance.Spec.Source.AnnounceHost && req.GetMasterPort() == int64(instance.Spec.Source.AnnouncePort) {
		successMsg := fmt.Sprintf("the source node of redis replication[%s] in namespace[%s] is already %s:%d, no update needed",
			req.GetRedisReplicationName(), req.GetNamespace(), req.GetMasterHost(), req.GetMasterPort())
		return common.LogAndReturnSuccessWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover", successMsg)
	}

	// 3. Find the master host in replica set and swap
	found := false
	for index, node := range instance.Spec.Replica {
		if req.GetMasterHost() == node.AnnounceHost && req.GetMasterPort() == int64(node.AnnouncePort) {
			msg := fmt.Sprintf("found node host %s:%d in replica, will update", req.GetMasterHost(), req.GetMasterPort())
			s.logger.Info(msg)
			if err := s.recorder.SendNormalEventToUnit(req.GetSelfUnitName(), req.GetNamespace(), "Failover", msg); err != nil {
				s.logger.Errorf("failed to send normal event: %v", err)
			}

			found = true
			instance.Spec.Source, instance.Spec.Replica[index] = node, instance.Spec.Source
		}
	}

	if !found {
		return common.LogAndReturnErrorWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover",
			fmt.Sprintf("cannot find host %s:%d in redis replication", req.GetMasterHost(), req.GetMasterPort()), nil)
	}

	// 4. Update the redis replication resource
	if err := s.composeClient.Update(ctx, instance); err != nil {
		return common.LogAndReturnErrorWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover",
			"failed to update redis replication", err)
	}

	return common.LogAndReturnSuccessWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover",
		"update redis replication successfully")
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "redis sentinel set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"password": req.GetPassword(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newSentinelResponse, "service status check failed", err)
	}

	// 2. Create connection
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:26379",
		Password: req.GetPassword(),
	})

	defer func() {
		if err := client.Close(); err != nil {
			s.logger.Errorf("failed to close connection: %v", err)
		}
	}()

	// 3. Execute set variable

	masterName := "mymaster"
	if err := client.Do(ctx, "SENTINEL", "SET", masterName, req.GetKey(), req.GetValue()).Err(); err != nil {
		return common.LogAndReturnError(s.logger, newSentinelResponse, fmt.Sprintf("failed to SET %s=%s", req.GetKey(), req.GetValue()), err)
	}

	return common.LogAndReturnSuccess(s.logger, newSentinelResponse, fmt.Sprintf("set variable %s=%s successfully", req.GetKey(), req.GetValue()))
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}

//func init() {
//	app.RegistryGrpcApp(svr)
//}
