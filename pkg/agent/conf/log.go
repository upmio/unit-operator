package conf

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (l *Log) GetLogLevel() zapcore.Level {
	switch strings.ToLower(l.Level) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

const (
	Banner = `
███████╗██╗   ██╗███╗   ██╗████████╗██████╗  ██████╗ ██████╗ ██╗   ██╗ ██████╗██╗      ██████╗ ██╗   ██╗██████╗ 
██╔════╝╚██╗ ██╔╝████╗  ██║╚══██╔══╝██╔══██╗██╔═══██╗██╔══██╗╚██╗ ██╔╝██╔════╝██║     ██╔═══██╗██║   ██║██╔══██╗
███████╗ ╚████╔╝ ██╔██╗ ██║   ██║   ██████╔╝██║   ██║██████╔╝ ╚████╔╝ ██║     ██║     ██║   ██║██║   ██║██║  ██║
╚════██║  ╚██╔╝  ██║╚██╗██║   ██║   ██╔══██╗██║   ██║██╔═══╝   ╚██╔╝  ██║     ██║     ██║   ██║██║   ██║██║  ██║
███████║   ██║   ██║ ╚████║   ██║   ██║  ██║╚██████╔╝██║        ██║   ╚██████╗███████╗╚██████╔╝╚██████╔╝██████╔╝
╚══════╝   ╚═╝   ╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═╝        ╚═╝    ╚═════╝╚══════╝ ╚═════╝  ╚═════╝ ╚═════╝`
)
