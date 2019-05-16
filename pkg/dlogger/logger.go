package dlogger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GetLogger(logLevel string) (*zap.Logger, error) {
	zapConfig := zap.NewProductionConfig()
	var lvl zapcore.Level
	err := lvl.UnmarshalText([]byte(logLevel))
	if err != nil {
		return nil, err
	}
	zapConfig.Level = zap.NewAtomicLevelAt(lvl)
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}
	return logger, nil
}
