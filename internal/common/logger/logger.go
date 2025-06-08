package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the global logger instance.
var logger *zap.Logger

// Init initializes the global logger with the specified configuration.
func Init(level string, production bool) error {
	cfg := zap.NewDevelopmentConfig()
	if production {
		cfg = zap.NewProductionConfig()
	}

	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return errors.Wrap(err, "invalid log level")
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := cfg.Build()
	if err != nil {
		return errors.Wrap(err, "failed to build logger")
	}

	logger = l
	return nil
}

// Get returns the global logger instance.
func Get() *zap.Logger {
	if logger == nil {
		// Fallback to a default logger if not initialized
		cfg := zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		l, _ := cfg.Build()
		logger = l
	}
	return logger
}

// Info logs a message at Info level.
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a message at Warn level.
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs a message at Error level.
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Sync flushes any buffered log entries.
func Sync() error {
	return Get().Sync()
}