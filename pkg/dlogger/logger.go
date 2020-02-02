// Package dlogger exposes a simple zap logger, with log levels
package dlogger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// LogLevelInfo sets the log level to info
	LogLevelInfo = "info"

	// LogLevelDebug sets the log level to debug
	LogLevelDebug = "debug"

	// LogLevelNone sets logger to no logging
	LogLevelNone = "none"
)

// GetLogger returns a zap logger with the specified level
func GetLogger(logLevel string) (*zap.Logger, error) {
	if logLevel == LogLevelNone {
		return zap.NewNop(), nil
	}
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

// MustGetLogger returns a zap logger with the specified level or panics
func MustGetLogger(logLevel string) *zap.Logger {
	l, err := GetLogger(logLevel)
	if err != nil {
		panic(err)
	}
	return l
}
