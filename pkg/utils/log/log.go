package log

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLoggerFromFlagsAndEnv(logDir, logLevel, logFileMaxSize string) logr.Logger {

	// Apply precedence: flag > env > default
	if logDir == "" {
		logDir = os.Getenv("LOG_PATH")
		if logDir == "" {
			logDir = "/tmp"
		}
	}

	if logFileMaxSize == "" {
		logFileMaxSize = os.Getenv("LOG_MAX_SIZE")
		if logFileMaxSize == "" {
			logFileMaxSize = "10"
		}
	}

	if logLevel == "" {
		logLevel = os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}
	}

	return InitLogger(logDir, logLevel, logFileMaxSize)
}

// InitLogger initializes a logr.Logger with zap + lumberjack.
// It writes logs to both stdout (for `kubectl logs`) and a rotating file.
//
// Parameters:
//
//	logpath  - directory for log file, default is /tmp
//	loglevel - log verbosity ("debug", "info", "error")
//	maxSize  - maximum log file size in MB before rotation
//
// Returns:
//
//	logr.Logger compatible logger (zapr wrapper of zap)
func InitLogger(logpath, loglevel, maxSize string) logr.Logger {

	// Lumberjack handles log file rotation
	hook := &lumberjack.Logger{
		Filename:   "/tmp/unit-operator.log",
		MaxSize:    10,   // default 10 MB per file
		MaxBackups: 30,   // keep last 30 backups
		MaxAge:     7,    // keep logs for 7 days
		Compress:   true, // compress rotated files
	}

	if logpath != "" {
		hook.Filename = filepath.Join(logpath, "unit-operator.log")
	}
	if maxSize != "" {
		if size, err := strconv.Atoi(maxSize); err == nil {
			hook.MaxSize = size
		}
	}

	// Set log level
	var level zapcore.Level
	switch loglevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}
	atomicLevel := zap.NewAtomicLevelAt(level)

	// Write logs to both stdout (visible with kubectl logs)
	// and file (with rotation by lumberjack)
	writer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout), // required for kubectl logs
		zapcore.AddSync(hook),      // rotating file log
	)

	// Encoder configuration for JSON structured logs
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "linenum",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // e.g. "info", "error"
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // human readable ISO8601
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder, // include full file path
		EncodeName:     zapcore.FullNameEncoder,
	}

	// Build zap core with JSON encoder and multi-output writer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		atomicLevel,
	)

	// Construct zap logger with options:
	// - AddCaller: include file/line info
	// - Development: enable stacktrace in dev mode
	// - Fields: add static fields (e.g. app name)
	logger := zap.New(core,
		zap.AddCaller(),
		zap.Development(),
		zap.Fields(zap.String("app", "unit-operator")),
	)

	// Wrap zap logger with zapr (logr interface)
	return zapr.NewLogger(logger)
}

//func initLogger() {
//	// 日志级别
//	level := zapcore.DebugLevel
//
//	// 编码器配置（可以换成 zap.NewProductionEncoderConfig()）
//	encCfg := zap.NewDevelopmentEncoderConfig()
//	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder // 时间格式
//	encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
//
//	core := zapcore.NewCore(
//		zapcore.NewConsoleEncoder(encCfg),
//		zapcore.Lock(os.Stdout),
//		level,
//	)
//
//	// ⚡ 关键点：加上 AddCaller 和 AddCallerSkip(1)，这样就能看到文件和行号
//	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
//
//	// 转成 controller-runtime 可用的 logr.Logger
//	ctrl.SetLogger(ctrzap.NewRaw(zap.UseDevMode(true), zap.WrapCore(func(c zapcore.Core) zapcore.Core {
//		return core
//	})))
//}
