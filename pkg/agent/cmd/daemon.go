package cmd

import (
	"context"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/daemon"
	"github.com/upmio/unit-operator/pkg/agent/app/milvus"
	"github.com/upmio/unit-operator/pkg/agent/app/mysql"
	"github.com/upmio/unit-operator/pkg/agent/app/postgresql"
	"github.com/upmio/unit-operator/pkg/agent/app/proxysql"
	"github.com/upmio/unit-operator/pkg/agent/app/redis"
	"github.com/upmio/unit-operator/pkg/agent/app/sentinel"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/protocol"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"github.com/upmio/unit-operator/pkg/agent/version"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Adding a new service requires import
	_ "github.com/upmio/unit-operator/pkg/agent/app/config"
	_ "github.com/upmio/unit-operator/pkg/agent/app/service"
)

var (
	configPath string
	file       *os.File
	httpEnable bool
)

// RootCmd represents the base command when called without any subcommands
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run as a daemon process",
	RunE: func(cmd *cobra.Command, args []string) error {

		defer func() {
			if err := file.Close(); err != nil {
				fmt.Printf("failed to close file: %v\n", err)
			}
		}()

		// initialize global variables
		if err := conf.LoadConfigFromToml(configPath); err != nil {
			return err
		}

		// initialize the global logging configurationClick to apply
		if err := loadGlobalLogger(); err != nil {
			return err
		}

		defer func() {
			if err := zap.L().Sync(); err != nil {
				fmt.Printf("failed to sync logger: %v\n", err)
			}
		}()

		unitType := os.Getenv("UNIT_TYPE")
		if unitType == "" {
			return fmt.Errorf("UNIT_TYPE must be set")
		}

		logger := zap.L().Named("[INIT]").Sugar()
		if err := util.ValidateAndSetAESKey(); err != nil {
			logger.Error(err)
			return err
		}

		// initialize the global app
		if err := app.InitAllApp(); err != nil {
			return err
		}

		// Make sure global variable config has been initialized
		_ = conf.GetConf()

		// start service
		//ch := make(chan os.Signal, 1)
		//signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)

		ctx, stop := signal.NotifyContext(context.Background(),
			syscall.SIGTERM,
			syscall.SIGINT,
			syscall.SIGHUP,
			syscall.SIGQUIT,
		)
		defer stop()

		// init service
		svr, err := newService()
		if err != nil {
			return err
		}

		// wait signal
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go svr.waitSign(ctx, wg)

		switch unitType {
		case "redis":
			redis.RegistryGrpcApp()
			archMode, err := util.GetEnvVarOrError(vars.ArchModeEnvKey)
			if err != nil {
				return err
			}

			if archMode == "cluster" {
				logger.Info("start redis cluster backup config daemon")

				configDir, err := util.GetEnvVarOrError(vars.ConfigDirEnvKey)
				if err != nil {
					return err
				}

				namespace, err := util.GetEnvVarOrError("NAMESPACE")
				if err != nil {
					return err
				}

				podName, err := util.GetEnvVarOrError("POD_NAME")
				if err != nil {
					return err
				}

				wg.Add(1)
				go daemon.StartRedisClusterNodesConfBackup(ctx, wg, namespace, podName, configDir)
			}

		case "redis-sentinel":
			sentinel.RegistryGrpcApp()
		case "mysql":
			mysql.RegistryGrpcApp()
		case "postgresql":
			postgresql.RegistryGrpcApp()
		case "proxysql":
			proxysql.RegistryGrpcApp()
		case "milvus":
			milvus.RegistryGrpcApp()
		}

		// start service
		if err := svr.start(); err != nil {
			if !strings.Contains(err.Error(), "http: Server closed") {
				return err
			}
		}
		wg.Wait()
		return nil
	},
}

func init() {
	daemonCmd.PersistentFlags().StringVarP(&configPath, "file", "f", "/etc/unit-agent/config.toml", "Specify the config file path")
	daemonCmd.PersistentFlags().BoolVarP(&httpEnable, "http-enable", "", false, "Specify whether http service need to start")
}

type service struct {
	grpc   *protocol.GrpcService
	logger *zap.SugaredLogger
}

func newService() (*service, error) {
	svr := &service{
		grpc:   protocol.NewGrpcService(),
		logger: zap.L().Named("[INIT]").Sugar(),
	}

	return svr, nil
}

func (s *service) start() error {
	s.logger.Info(fmt.Sprintf("loaded grpc apps %s", app.LoadedGrpcApp()))
	go s.grpc.Start()

	return nil
}

//nolint:staticcheck // S1000: keeping select structure for future extensibility
func (s *service) waitSign(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			zap.L().Named("[GRPC SERVICE]").Sugar().Infof("start graceful shutdown")

			s.grpc.Stop()
			wg.Done()
			return
		}
	}
}

func loadGlobalLogger() error {

	//logger, _ := zap.NewProduction()
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "timestamp"
	cfg.MessageKey = "message"
	cfg.NameKey = "module"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder

	if _, err := os.Stat(conf.GetConf().PathDir); os.IsNotExist(err) {
		err := os.Mkdir(conf.GetConf().PathDir, 0755)
		if err != nil {
			return fmt.Errorf("create %s directory fail, error: %v ", conf.GetConf().PathDir, err)
		}
	}

	logJsonfile := filepath.Join(conf.GetConf().PathDir, version.ServiceName+"-json.log")
	fileJson, err := os.Create(logJsonfile)
	if err != nil {
		return fmt.Errorf("create log json file fail, error: %v ", err)
	}

	logfile := filepath.Join(conf.GetConf().PathDir, version.ServiceName+".log")
	file, err := os.Create(logfile)
	if err != nil {
		return fmt.Errorf("create log file fail, error: %v ", err)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(fileJson), conf.GetConf().GetLogLevel()),
		zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), zapcore.AddSync(file), conf.GetConf().GetLogLevel()),
		zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), zapcore.Lock(os.Stdout), conf.GetConf().GetLogLevel()),
	)
	//logger := zap.New(core, zap.AddCaller())
	logger := zap.New(core)

	zap.ReplaceGlobals(logger)

	return nil
}
