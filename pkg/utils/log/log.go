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

//func InitLogger(logpath, loglevel, maxSize string) logr.Logger {
//	// split log
//	hook := lumberjack.Logger{
//		Filename:   "/tmp/unit-operator.log", // log file path, default: "/tmp/unit-operator.log"
//		MaxSize:    10,                       // 10M for each log file, default 10M
//		MaxBackups: 30,                       // keep 30 backups, unlimited by default
//		MaxAge:     7,                        // reserved for 7 days, unlimited by default
//		Compress:   true,                     // compress or not, no compression by default
//	}
//
//	if logpath != "" {
//		logFileName := filepath.Join(
//			logpath,
//			fmt.Sprintf("%s-%s", "unit-operator", time.Now().Format("2006-01-02-15:04:05")),
//		)
//		hook.Filename = logFileName
//	}
//
//	if maxSize != "" {
//		size, _ := strconv.Atoi(maxSize)
//		hook.MaxSize = size
//	}
//
//	write := zapcore.AddSync(&hook)
//	// setting the log level
//	// debug: info debug warn
//	// info: warn info
//	// warn: warn
//	// debug -> info -> warn -> error
//	var level zapcore.Level
//	switch loglevel {
//	case "debug":
//		level = zap0.DebugLevel
//	case "info":
//		level = zap0.InfoLevel
//	case "error":
//		level = zap0.ErrorLevel
//	default:
//		level = zap0.InfoLevel
//	}
//	encoderConfig := zapcore.EncoderConfig{
//		TimeKey:        "time",
//		LevelKey:       "level",
//		NameKey:        "logger",
//		CallerKey:      "linenum",
//		MessageKey:     "msg",
//		StacktraceKey:  "stacktrace",
//		LineEnding:     zapcore.DefaultLineEnding,
//		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // lowercase character encoders
//		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC time format
//		EncodeDuration: zapcore.SecondsDurationEncoder, //
//		EncodeCaller:   zapcore.FullCallerEncoder,      // full caller encoder
//		EncodeName:     zapcore.FullNameEncoder,
//	}
//	// set log level
//	atomicLevel := zap0.NewAtomicLevel()
//	atomicLevel.SetLevel(level)
//	core := zapcore.NewCore(
//		// zapcore.NewConsoleEncoder(encoderConfig),
//		zapcore.NewJSONEncoder(encoderConfig),
//		// zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&write)), // print to console and file
//		write,
//		level,
//	)
//	// open development mode, stack trace
//	caller := zap0.AddCaller()
//	// open file and line number
//	development := zap0.Development()
//	// set initialization fields, e.g. add a server name
//	filed := zap0.Fields(zap0.String("unit-operator", "unit-operator"))
//	// constructor Log
//	//logger := zap0.New(core, caller, development, filed)
//	//logger.Info("DefaultLogger init success")
//
//	return zapr.NewLogger(zap0.New(core, caller, development, filed))
//}

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
