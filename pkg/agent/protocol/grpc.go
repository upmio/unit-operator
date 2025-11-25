package protocol

import (
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcService struct {
	l *zap.SugaredLogger
	s *grpc.Server
}

func NewGrpcService() *GrpcService {
	server := grpc.NewServer()
	reflection.Register(server)

	return &GrpcService{
		s: server,
		l: zap.L().Named("[GRPC SERVICE]").Sugar(),
	}
}

func (g *GrpcService) Start() {
	if err := app.LoadGrpcApp(g.s); err != nil {
		g.l.Error("load grpc app failed", zap.Error(err))
	}

	addr := conf.GetConf().GrpcAddr()
	lsr, err := net.Listen("tcp", addr)
	if err != nil {
		g.l.Errorf("listen grpc tcp conn error, %s", err)
		return
	}

	g.l.Infof("start grpc service successfully, listen address: [%s]", addr)

	if err := g.s.Serve(lsr); err != nil {
		if err == grpc.ErrServerStopped {
			g.l.Info("service is stopped")
		}

		g.l.Error("start grpc service error, %s", err.Error())
		return
	}
}

func (g *GrpcService) Stop() {
	g.s.GracefulStop()
	g.l.Info("service is stopped")
}
