package log

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	zap0 "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	"path/filepath"
	"strconv"
	"time"
)

func InitLogger(logpath, loglevel, maxSize string) logr.Logger {
	// split log
	hook := lumberjack.Logger{
		Filename:   "/tmp/unit-operator.log", // log file path, default: "/tmp/unit-operator.log"
		MaxSize:    10,                       // 10M for each log file, default 10M
		MaxBackups: 30,                       // keep 30 backups, unlimited by default
		MaxAge:     7,                        // reserved for 7 days, unlimited by default
		Compress:   true,                     // compress or not, no compression by default
	}

	if logpath != "" {
		logFileName := filepath.Join(
			logpath,
			fmt.Sprintf("%s-%s", "unit-operator", time.Now().Format("2006-01-02-15:04:05")),
		)
		hook.Filename = logFileName
	}

	if maxSize != "" {
		size, _ := strconv.Atoi(maxSize)
		hook.MaxSize = size
	}

	write := zapcore.AddSync(&hook)
	// setting the log level
	// debug: info debug warn
	// info: warn info
	// warn: warn
	// debug -> info -> warn -> error
	var level zapcore.Level
	switch loglevel {
	case "debug":
		level = zap0.DebugLevel
	case "info":
		level = zap0.InfoLevel
	case "error":
		level = zap0.ErrorLevel
	default:
		level = zap0.InfoLevel
	}
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "linenum",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // lowercase character encoders
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC time format
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.FullCallerEncoder,      // full caller encoder
		EncodeName:     zapcore.FullNameEncoder,
	}
	// set log level
	atomicLevel := zap0.NewAtomicLevel()
	atomicLevel.SetLevel(level)
	core := zapcore.NewCore(
		// zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewJSONEncoder(encoderConfig),
		// zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&write)), // print to console and file
		write,
		level,
	)
	// open development mode, stack trace
	caller := zap0.AddCaller()
	// open file and line number
	development := zap0.Development()
	// set initialization fields, e.g. add a server name
	filed := zap0.Fields(zap0.String("unit-operator", "unit-operator"))
	// constructor Log
	//logger := zap0.New(core, caller, development, filed)
	//logger.Info("DefaultLogger init success")

	return zapr.NewLogger(zap0.New(core, caller, development, filed))
}
