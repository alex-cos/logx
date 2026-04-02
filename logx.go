package logx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kjk/common/filerotate"
)

const (
	DateTimeFormatMilli = "2006-01-02T15:04:05.000Z07:00"
	DateTimeFormatMicro = "2006-01-02T15:04:05.000000Z07:00"
	DateTimeFormatNano  = "2006-01-02T15:04:05.000000000Z07:00"
	FileDateTimeFormat  = "2006-01-02"
	SError              = "error"
	CallerKey           = "caller"
	StackKey            = "stack"
)

func New(writers []io.Writer, level string, opts ...Option) *slog.Logger {
	cfg := parseConfig(opts...)

	slevel := parseLevel(level)
	w := io.MultiWriter(writers...)

	handlerOptions := &slog.HandlerOptions{
		AddSource:   cfg.source,
		Level:       slevel,
		ReplaceAttr: computeReplaceAttr(cfg.source, cfg.utc),
	}

	wrapped := &handler{
		Handler:  nil,
		basePath: "",
		caller:   cfg.caller,
		stack:    cfg.stack,
	}
	if cfg.json {
		wrapped.Handler = slog.NewJSONHandler(w, handlerOptions)
	} else {
		wrapped.Handler = slog.NewTextHandler(w, handlerOptions)
	}

	return slog.New(wrapped)
}

func NewFileLogger(
	logpath string,
	level string,
	opts ...Option,
) (*slog.Logger, Close, error) {
	w := []io.Writer{}

	file, err := os.OpenFile(logpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}
	w = append(w, file)

	return New(w, level, opts...), func() {
		file.Close()
	}, nil
}

func NewConsoleLogger(level string, opts ...Option) *slog.Logger {
	return New([]io.Writer{os.Stdout}, level, opts...)
}

func NewFileRotate(logpath string, opts ...Option) (io.Writer, Close, error) {
	dir := filepath.Dir(logpath)
	filename := filepath.Base(logpath)
	ext := filepath.Ext(filename)
	basename := strings.TrimSuffix(filename, ext)
	cfg := parseConfig(opts...)
	fileconfig := filerotate.Config{
		DidClose: func(path string, didRotate bool) {
			// By default do noting
		},
		PathIfShouldRotate: func(creationTime time.Time, now time.Time) string {
			if creationTime.YearDay() == now.YearDay() {
				return ""
			}
			d := now
			if cfg.utc {
				d = now.UTC()
			}
			name := fmt.Sprintf("%s_%s%s", basename, d.Format(FileDateTimeFormat), ext)
			return filepath.Join(dir, name)
		},
	}
	file, err := filerotate.New(&fileconfig)
	if err != nil {
		return nil, nil, err
	}

	return file, func() {
		file.Close()
	}, nil
}

func Error(err error) slog.Attr {
	if err == nil {
		return slog.Attr{} // nolint: exhaustruct
	}
	return slog.Any(SError, err)
}

// ----------------------------------------------------------------------------
// Unexported functions
// ----------------------------------------------------------------------------

func parseLevel(s string) slog.Level {
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

func computeReplaceAttr(source, utc bool) func(groups []string, a slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.TimeKey:
			t := a.Value.Time()
			if utc {
				t = t.UTC()
			}
			return slog.Attr{
				Key:   slog.TimeKey,
				Value: slog.StringValue(t.Format(DateTimeFormatMilli)),
			}
		case slog.LevelKey:
			return slog.Attr{
				Key:   slog.LevelKey,
				Value: slog.StringValue(strings.ToLower(a.Value.String())),
			}
		case slog.SourceKey:
			if source {
				return a
			} else {
				return slog.Attr{
					Key:   "",
					Value: slog.AnyValue(nil),
				}
			}
		case CallerKey:
			return a
		default:
			return a
		}
	}
}
