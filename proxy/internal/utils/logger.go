package utils

import (
	"os"

	"github.com/veriteknik/registry-proxy/internal/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger is the global structured logger instance
	Logger *zap.Logger
)

func init() {
	Logger = NewLogger()
}

// NewLogger creates a new zap logger based on the environment.
// In production, uses JSON encoding. In development, uses console encoding.
func NewLogger() *zap.Logger {
	env := os.Getenv("ENVIRONMENT")
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

	// Set log level from environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(logLevel)); err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Hooks(func(entry zapcore.Entry) error {
			// Increment error log counter for error-level logs
			if entry.Level == zapcore.ErrorLevel {
				metrics.RecordErrorLog("error")
			}
			return nil
		}),
	)
	if err != nil {
		// Fallback to no-op logger if creation fails
		return zap.NewNop()
	}

	return logger
}

// Sync flushes any buffered log entries. Should be called before program exit.
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
