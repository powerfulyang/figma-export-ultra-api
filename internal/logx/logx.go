// Package logx provides structured logging functionality
package logx

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger to provide a consistent interface
type Logger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	prefix string
}

var globalLogger *Logger

func init() {
	// 初始化默认全局 logger
	var err error
	globalLogger, err = New("")
	if err != nil {
		panic(err)
	}
}

// IsLocalDev checks if the environment is local development
func IsLocalDev(appEnv string) bool {
	return appEnv == "local" || appEnv == "dev" || appEnv == "development"
}

// New creates a new logger with the specified prefix
func New(prefix string) (*Logger, error) {
	config := getLoggerConfig()

	appEnv := os.Getenv("APP_ENV")
	if IsLocalDev(appEnv) {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	zapLogger, err := config.Build(
		zap.AddCallerSkip(1), // 跳过封装层
	)
	if err != nil {
		return nil, err
	}

	return &Logger{
		zap:    zapLogger,
		sugar:  zapLogger.Sugar(),
		prefix: prefix,
	}, nil
}

// customTimeEncoder 自定义时间编码器
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// getLoggerConfig returns the zap configuration
func getLoggerConfig() zap.Config {
	config := zap.NewProductionConfig()

	// Customize the configuration
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	config.Development = false
	config.DisableCaller = false
	config.DisableStacktrace = false
	config.Sampling = nil // Disable sampling for now

	// Customize the encoder
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Use console encoder for better readability
	config.Encoding = "console"

	return config
}

// Init configures the global logger with legacy interface
func Init(level, format string) {
	lvl := parseLevel(level)
	config := getLoggerConfig()

	// 根据格式设置编码
	switch strings.ToLower(format) {
	case "json":
		config.Encoding = "json"
		config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder // JSON 格式使用小写
	default:
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Console 格式使用彩色
	}

	config.Level = zap.NewAtomicLevelAt(lvl)

	zapLogger, err := config.Build(
		zap.AddCallerSkip(1), // 跳过封装层
	)
	if err != nil {
		panic(err)
	}

	globalLogger = &Logger{
		zap:    zapLogger,
		sugar:  zapLogger.Sugar(),
		prefix: "",
	}
}

// L returns the global sugar logger instance that supports slog-style key-value logging
func L() *zap.SugaredLogger {
	if globalLogger == nil {
		return nil
	}
	return globalLogger.sugar
}

// GetLogger returns the underlying zap logger for advanced usage
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		return nil
	}
	return globalLogger.zap
}

// Global returns the global logger instance
func Global() *Logger {
	return globalLogger
}

func parseLevel(s string) zapcore.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Logger methods for structured logging

// SetLevel sets the minimum log level for the logger
func (l *Logger) SetLevel(level zapcore.Level) {
	if l.zap != nil {
		l.zap = l.zap.WithOptions(zap.IncreaseLevel(level))
		l.sugar = l.zap.Sugar()
	}
}

// Close closes the logger and flushes any buffered log entries
func (l *Logger) Close() error {
	if l.zap != nil {
		return l.zap.Sync()
	}
	return nil
}

// Sugar returns the sugar logger for key-value style logging
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

// Zap returns the underlying zap logger for structured logging
func (l *Logger) Zap() *zap.Logger {
	return l.zap
}

// Debug logs a debug message with structured fields
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	if l.zap == nil {
		return
	}
	l.zap.Debug(msg, fields...)
}

// Info logs an info message with structured fields
func (l *Logger) Info(msg string, fields ...zap.Field) {
	if l.zap == nil {
		return
	}
	l.zap.Info(msg, fields...)
}

// Warn logs a warning message with structured fields
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	if l.zap == nil {
		return
	}
	l.zap.Warn(msg, fields...)
}

// Error logs an error message with structured fields
func (l *Logger) Error(msg string, fields ...zap.Field) {
	if l.zap == nil {
		return
	}
	l.zap.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	if l.zap == nil {
		os.Exit(1)
		return
	}
	l.zap.Fatal(msg, fields...)
}
