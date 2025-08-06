package sentinel

import (
	"context"
	"fmt"
	composev1alpha1 "github.com/upmio/compose-operator/api/v1alpha1"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/event"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
)

var (
	// service service instance
	svr = &service{}
)

type service struct {
	sentinelOps    SentinelOperationServer
	gauntletClient client.Client
	recorder       event.IEventRecorder
	UnimplementedSentinelOperationServer
	logger *zap.SugaredLogger
}

// Common helper methods

// newSentinelResponse creates a new Sentinel Response with the given message
func newSentinelResponse(message string) *Response {
	return &Response{Message: message}
}


func (s *service) Config() error {
	s.sentinelOps = app.GetGrpcApp(appName).(SentinelOperationServer)
	c, err := conf.GetConf().Kube.GetGauntletClient()
	if err != nil {
		return err
	}

	if s.recorder, err = event.NewIEventRecorder(); err != nil {
		return err
	}

	s.gauntletClient = c
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
	s.logger.With(
		"namespace", req.GetNamespace(),
		"master_host", req.GetMasterHost(),
		"unit_name", req.GetSelfUnitName(),
		"redis_replication_name", req.GetRedisReplicationName(),
	).Info("receive update redis replication request")

	instance := &composev1alpha1.RedisReplication{}

	// 1. Fetch redis replication resource
	if err := s.gauntletClient.Get(context.Background(), types.NamespacedName{
		Namespace: req.GetNamespace(),
		Name:      req.GetRedisReplicationName(),
	}, instance); err != nil {
		return common.LogAndReturnErrorWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover", 
			fmt.Sprintf("failed to fetch redis replication[%s] in namespace[%s]", req.GetRedisReplicationName(), req.GetNamespace()), err)
	}

	// 2. Check if master host is already set correctly
	if req.GetMasterHost() == instance.Spec.Source.Host {
		successMsg := fmt.Sprintf("the source node host of redis replication[%s] in namespace[%s] is already %s, no update needed", 
			req.GetRedisReplicationName(), req.GetNamespace(), req.GetMasterHost())
		return common.LogAndReturnSuccessWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover", successMsg)
	}

	// 3. Find the master host in replica set and swap
	found := false
	for index, node := range instance.Spec.Replica {
		if req.GetMasterHost() == node.Host {
			msg := fmt.Sprintf("found node host %s in replica set, will update", req.GetMasterHost())
			s.logger.Info(msg)
			s.recorder.SendNormalEventToUnit(req.GetSelfUnitName(), req.GetNamespace(), "Failover", msg)

			found = true
			instance.Spec.Source, instance.Spec.Replica[index] = node, instance.Spec.Source
		}
	}

	if !found {
		return common.LogAndReturnErrorWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover", 
			fmt.Sprintf("cannot find host %s in redis replication", req.GetMasterHost()), nil)
	}

	// 4. Update the redis replication resource
	if err := s.gauntletClient.Update(ctx, instance); err != nil {
		return common.LogAndReturnErrorWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover", 
			"failed to update redis replication", err)
	}

	return common.LogAndReturnSuccessWithEvent(s.logger, s.recorder, newSentinelResponse, req.GetSelfUnitName(), req.GetNamespace(), "Failover", 
		"update redis replication successfully")
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}

//func init() {
//	app.RegistryGrpcApp(svr)
//}
