// Package observability provides structured logging and telemetry setup.
package observability

import (
	"log/slog"
	"os"
	"strings"

	tlog "go.temporal.io/sdk/log"
)

// InitLogger configures the global slog logger with JSON output at the given level.
func InitLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)
	return logger
}

// TemporalSlogAdapter adapts slog.Logger to Temporal's log.Logger interface.
type TemporalSlogAdapter struct {
	logger *slog.Logger
}

// NewTemporalSlogAdapter creates a Temporal log adapter from a slog.Logger.
func NewTemporalSlogAdapter(logger *slog.Logger) *TemporalSlogAdapter {
	return &TemporalSlogAdapter{logger: logger}
}

func (a *TemporalSlogAdapter) Debug(msg string, keyvals ...any) {
	a.logger.Debug(msg, keyvals...)
}

func (a *TemporalSlogAdapter) Info(msg string, keyvals ...any) {
	a.logger.Info(msg, keyvals...)
}

func (a *TemporalSlogAdapter) Warn(msg string, keyvals ...any) {
	a.logger.Warn(msg, keyvals...)
}

func (a *TemporalSlogAdapter) Error(msg string, keyvals ...any) {
	a.logger.Error(msg, keyvals...)
}

// Compile-time check.
var _ tlog.Logger = (*TemporalSlogAdapter)(nil)
