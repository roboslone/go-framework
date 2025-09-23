package framework

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Log(level zapcore.Level, msg string, fields ...zap.Field)
}
