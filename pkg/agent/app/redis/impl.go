package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// service service instance
	svr = &service{}
)

type service struct {
	redisOps RedisOperationServer
	UnimplementedRedisOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

// Common helper methods

// newRedisResponse creates a new Sentinel Response with the given message
func newRedisResponse(message string) *Response {
	return &Response{Message: message}
}

func (s *service) Config() error {
	s.redisOps = app.GetGrpcApp(appName).(RedisOperationServer)
	s.logger = zap.L().Named("[REDIS]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterRedisOperationServer(server, svr)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "redis set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"password": req.GetPassword(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "service status check failed", err)
	}

	// 2. Create connection
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: req.GetPassword(),
		DB:       0,
	})

	defer func() {
		if err := client.Close(); err != nil {
			s.logger.Errorf("failed to close connection: %v", err)
		}
	}()

	// 3. Execute set variable
	err := client.Do(ctx, "CONFIG", "SET", req.GetKey(), req.GetValue()).Err()
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, fmt.Sprintf("failed to SET %s=%s", req.GetKey(), req.GetValue()), err)
	}

	return common.LogAndReturnSuccess(s.logger, newRedisResponse, fmt.Sprintf("set variable %s=%s successfully", req.GetKey(), req.GetValue()))
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
