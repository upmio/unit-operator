package cmd

import (
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/mysql"
	"github.com/upmio/unit-operator/pkg/agent/app/postgresql"
	"github.com/upmio/unit-operator/pkg/agent/app/proxysql"
	"github.com/upmio/unit-operator/pkg/agent/app/sentinel"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/protocol"
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
		defer file.Close()

		// initialize global variables
		if err := conf.LoadConfigFromToml(configPath); err != nil {
			return err
		}

		// initialize the global logging configurationClick to apply
		if err := loadGlobalLogger(); err != nil {
			return err
		}

		defer zap.L().Sync()

		unitType := os.Getenv("UNIT_TYPE")
		zap.L().Named("[INIT]").Sugar().Infof("get env UNIT_TYPE=%s", unitType)

		if err := util.ValidateAndSetAESKey(); err != nil {
			zap.L().Named("[INIT]").Sugar().Error(err)
			return err
		}

		switch unitType {
		case "redis-sentinel":
			sentinel.RegistryGrpcApp()
			zap.L().Named("[INIT]").Sugar().Infof("registry sentinel grpc app")
		case "mysql":
			mysql.RegistryGrpcApp()
			zap.L().Named("[INIT]").Sugar().Infof("registry mysql grpc app")
		case "postgresql":
			postgresql.RegistryGrpcApp()
			zap.L().Named("[INIT]").Sugar().Infof("registry postgresql grpc app")
		case "proxysql":
			proxysql.RegistryGrpcApp()
			zap.L().Named("[INIT]").Sugar().Infof("registry proxysql grpc app")
		}

		// initialize the global app
		if err := app.InitAllApp(); err != nil {
			return err
		}

		// Make sure global variable config has been initialized
		_ = conf.GetConf()

		// start service
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)

		// init service
		svr, err := newService()
		if err != nil {
			return err
		}

		// wait signal
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go svr.waitSign(ch, wg)

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
	http   *protocol.HTTPService
	grpc   *protocol.GrpcService
	logger *zap.SugaredLogger
}

func newService() (*service, error) {
	http := protocol.NewHTTPService()
	grpc := protocol.NewGrpcService()
	svr := &service{
		http:   http,
		grpc:   grpc,
		logger: zap.L().Named("[INIT]").Sugar(),
	}

	return svr, nil
}

func (s *service) start() error {
	s.logger.Info(fmt.Sprintf("loaded http apps: %s", app.LoadedHttpApp()))
	s.logger.Info(fmt.Sprintf("loaded grpc apps %s", app.LoadedGrpcApp()))
	go s.grpc.Start()

	if httpEnable {
		return s.http.Start()
	} else {
		return nil
	}
}

func (s *service) waitSign(sign chan os.Signal, wg *sync.WaitGroup) {
	for {
		select {
		case sg := <-sign:
			switch v := sg.(type) {
			default:
				if httpEnable {
					zap.L().Named("[HTTP SERVICE]").Sugar().Infof("receive signal '%v', start graceful shutdown", v.String())
					if err := s.http.Stop(); err != nil {
						zap.L().Named("[HTTP SERVICE]").Sugar().Errorf("http graceful shutdown err: %s, force exit", err)
					} else {
						zap.L().Named("[HTTP SERVICE]").Info("http service stop complete")
					}
				}
				s.grpc.Stop()
				wg.Done()
				return
			}
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

	if _, err := os.Stat(conf.GetConf().Log.PathDir); os.IsNotExist(err) {
		err := os.Mkdir(conf.GetConf().Log.PathDir, 0755)
		if err != nil {
			return fmt.Errorf("Create %s directory fail, error: %v ", conf.GetConf().Log.PathDir, err)
		}
	}

	logJsonfile := filepath.Join(conf.GetConf().Log.PathDir, version.ServiceName+"-json.log")
	fileJson, err := os.Create(logJsonfile)
	if err != nil {
		return fmt.Errorf("Create log json file fail, error: %v ", err)
	}

	logfile := filepath.Join(conf.GetConf().Log.PathDir, version.ServiceName+".log")
	file, err := os.Create(logfile)
	if err != nil {
		return fmt.Errorf("Create log file fail, error: %v ", err)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(fileJson), conf.GetConf().Log.GetLogLevel()),
		zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), zapcore.AddSync(file), conf.GetConf().Log.GetLogLevel()),
		zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), zapcore.Lock(os.Stdout), conf.GetConf().Log.GetLogLevel()),
	)
	//logger := zap.New(core, zap.AddCaller())
	logger := zap.New(core)

	zap.ReplaceGlobals(logger)

	//zap.L().Named("[INIT]").Info(conf.Banner)
	zap.L().Named("[INIT]").Info(fmt.Sprintf("log level: %s", conf.GetConf().Level))

	return nil
}
