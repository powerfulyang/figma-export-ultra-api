// Package logx provides structured logging functionality
package logx

import (
	"log/slog"
	"os"
	"strings"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// Init configures the global logger.
func Init(level, format string) {
	lvl := parseLevel(level)
	switch strings.ToLower(format) {
	case "json":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	default:
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	}
}

// L returns the global logger instance
func L() *slog.Logger { return logger }

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
