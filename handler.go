// nolint: hugeparam, gocritic
package logx

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
)

type handler struct {
	slog.Handler
	basePath string
	caller   bool
	stack    bool
}

func (h *handler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.Handler.Enabled(ctx, l)
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	if h.caller {
		caller := h.getCaller(5)
		if caller != "" {
			r.AddAttrs(slog.String(CallerKey, caller))
		}
	}
	if h.stack && r.Level == slog.LevelError {
		stack := h.getStack(5)
		if stack != "" {
			r.AddAttrs(slog.String(StackKey, stack))
		}
	}

	return h.Handler.Handle(ctx, r)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{
		Handler:  h.Handler.WithAttrs(attrs),
		basePath: h.basePath,
		caller:   h.caller,
		stack:    h.stack,
	}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{
		Handler:  h.Handler.WithGroup(name),
		basePath: h.basePath,
		caller:   h.caller,
		stack:    h.stack,
	}
}

func (h *handler) getCaller(skip int) string {
	pc := make([]uintptr, 1)
	n := runtime.Callers(skip, pc)
	if n > 0 {
		frames := runtime.CallersFrames(pc)
		frame, _ := frames.Next()
		file := filepath.ToSlash(frame.File)
		if h.basePath != "" {
			file = strings.TrimPrefix(file, h.basePath)
			file = strings.TrimPrefix(file, "/")
		} else {
			file = filepath.Base(file)
		}
		return fmt.Sprintf("%s:%d", file, frame.Line)
	}

	return ""
}

func (h *handler) getStack(skip int) string {
	pcs := make([]uintptr, 10)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	stack := []string{}
	for {
		frame, more := frames.Next()
		file := filepath.ToSlash(frame.File)
		stack = append(stack, fmt.Sprintf("%s:%d", file, frame.Line))
		if !more {
			break
		}
	}
	if len(stack) > 0 {
		return "[" + strings.Join(stack, ",") + "]"
	}

	return ""
}
