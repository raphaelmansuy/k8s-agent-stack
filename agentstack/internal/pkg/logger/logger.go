// Package logger provides structured logging with zap.
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new logger instance configured for the current environment.
func New() (*zap.Logger, error) {
	env := os.Getenv("AGENTSTACK_ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	var config zap.Config
	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Parse log level from environment
	if level := os.Getenv("AGENTSTACK_LOG_LEVEL"); level != "" {
		var lvl zapcore.Level
		if err := lvl.UnmarshalText([]byte(level)); err == nil {
			config.Level = zap.NewAtomicLevelAt(lvl)
		}
	}

	return config.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCaller(),
	)
}

// NewNop creates a no-op logger for testing.
func NewNop() *zap.Logger {
	return zap.NewNop()
}
