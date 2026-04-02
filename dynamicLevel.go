package logx

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
)

// DynamicLevel allows changing slog level at runtime.

type DynamicLevel struct {
	level atomic.Int32
}

func (d *DynamicLevel) Enabled(_ context.Context, l slog.Level) bool {
	return l >= slog.Level(d.level.Load())
}

func (d *DynamicLevel) SetLevel(l slog.Level) {
	d.level.Store(int32(l)) // nolint: gosec
}

func (d *DynamicLevel) Level() slog.Level {
	return slog.Level(d.level.Load())
}

func ParseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
