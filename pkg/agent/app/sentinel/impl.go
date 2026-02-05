package sentinel

import (
	"context"
	"fmt"

	composev1alpha1 "github.com/upmio/compose-operator/api/v1alpha1"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// this import  needs to be done otherwise the mysql driver don't work
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	sentinelOps SentinelOperationServer
	UnimplementedSentinelOperationServer
	logger *zap.SugaredLogger

	composeClient client.Client
	recorder      *common.EventRecorder
	slm           slm.ServiceLifecycleServer
}

func (s *service) Config() error {
	s.sentinelOps = app.GetGrpcApp(appName).(SentinelOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)

	c, err := conf.GetConf().GetComposeClient()
	if err != nil {
		return err
	}
	s.composeClient = c

	if s.recorder, err = common.NewEventRecorder(); err != nil {
		return err
	}

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterSentinelOperationServer(server, svr)
}

func (s *service) UpdateRedisReplication(ctx context.Context, req *UpdateRedisReplicationRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "update redis replication", map[string]interface{}{
		"namespace":              req.GetNamespace(),
		"master_host":            req.GetSourceHost(),
		"master_port":            req.GetSourcePort(),
		"self_unit_name":         req.GetSelfUnitName(),
		"redis_replication_name": req.GetRedisReplicationName(),
	})

	eventMsg := "failed to send event"
	// Fetch redis replication instance
	instance := &composev1alpha1.RedisReplication{}
	if err := s.composeClient.Get(context.Background(), types.NamespacedName{
		Namespace: req.GetNamespace(),
		Name:      req.GetRedisReplicationName(),
	}, instance); err != nil {
		s.logger.Errorw("failed to fetch redis replication instance", zap.Error(err), zap.String("namespace", req.GetNamespace()), zap.String("name", req.GetRedisReplicationName()))

		if eventErr := s.recorder.SendWarningEventToUnit(req.GetSelfUnitName(), req.GetNamespace(), "Failover", err.Error()); eventErr != nil {
			s.logger.Errorw(eventMsg, zap.Error(eventErr))
		}

		return nil, err
	}

	if err := s.ensureRedisReplicationInstance(ctx, instance, req.GetSourceHost(), req.GetSourcePort()); err != nil {
		s.logger.Errorw("failed to ensure redis replication instance", zap.Error(err))

		if eventErr := s.recorder.SendWarningEventToUnit(req.GetSelfUnitName(), req.GetNamespace(), "Failover", err.Error()); eventErr != nil {
			s.logger.Errorw(eventMsg, zap.Error(eventErr))
		}
		return nil, err
	}

	if eventErr := s.recorder.SendNormalEventToUnit(req.GetSelfUnitName(), req.GetNamespace(), "Failover", "ensure redis replication instance successfully"); eventErr != nil {
		s.logger.Errorw(eventMsg, zap.Error(eventErr))
	}

	return nil, nil
}

func (s *service) ensureRedisReplicationInstance(ctx context.Context, instance *composev1alpha1.RedisReplication, sourceHost string, sourcePort int64) error {
	// Check if master host is already set correctly
	if sourceHost == instance.Spec.Source.Host && sourcePort == int64(instance.Spec.Source.Port) {
		s.logger.Infow("ths source node of redis replication is already set correctly", zap.String("namespace", instance.Namespace), zap.String("name", instance.Name))
		return nil
	}

	// Find the node in replica, then switchover change to source
	for index, node := range instance.Spec.Replica {
		if sourceHost == node.Host && sourcePort == int64(node.Port) {
			instance.Spec.Source, instance.Spec.Replica[index] = node, instance.Spec.Source

			if err := s.composeClient.Update(ctx, instance); err != nil {
				return err
			}

			s.logger.Infow("update redis replication successfully", zap.String("namespace", instance.Namespace), zap.String("name", instance.Name), zap.String("host", sourceHost), zap.Int64("port", sourcePort))

			return nil
		}
	}

	return fmt.Errorf("cannot find node %s:%d in replica list", sourceHost, sourcePort)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "redis-sentinel set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"username": req.GetUsername(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create connection
	rdb, err := s.newRedisClient(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeRedisClient(rdb)

	// Execute set variable
	masterName := "mymaster"
	if err := rdb.Do(ctx, "SENTINEL", "SET", masterName, req.GetKey(), req.GetValue()).Err(); err != nil {
		s.logger.Errorw("failed to set variable", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))
		return nil, err
	}

	s.logger.Info("set variable successfully")
	return nil, nil
}

// newRedisClient creates a Redis connection
func (s *service) newRedisClient(ctx context.Context, username string) (*redis.Client, error) {
	// Decryp plaintext password
	password, err := util.DecryptPlainTextPassword(username)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", username))
		return nil, err
	}

	// Create redis sentinel connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:26379",
		Password: password,
	})

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		s.logger.Errorw("failed to ping redis", zap.Error(err))

		s.closeRedisClient(rdb)
		return nil, err
	}

	return rdb, nil
}

func (s *service) closeRedisClient(client *redis.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		s.logger.Errorw("failed to close redis connection", zap.Error(err))
	}
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
