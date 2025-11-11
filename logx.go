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
)

func New(writers []io.Writer, level string, json, utc bool) *slog.Logger {
	slevel := parseLevel(level)
	root := findModuleRoot()
	w := io.MultiWriter(writers...)

	handlerOptions := &slog.HandlerOptions{
		AddSource:   true,
		Level:       slevel,
		ReplaceAttr: computeReplaceAttr(root, utc),
	}

	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(w, handlerOptions)
	} else {
		handler = slog.NewTextHandler(w, handlerOptions)
	}

	return slog.New(handler)
}

func NewFileLogger(
	logpath string,
	level string,
	json bool,
	utc bool,
	verbose bool,
) (*slog.Logger, Close) {
	w := []io.Writer{}

	file, closeFile := NewFileRotate(logpath, utc)
	w = append(w, file)
	if verbose {
		w = append(w, os.Stdout)
	}

	return New(w, level, json, utc), closeFile
}

func NewConsoleLogger(level string, json, utc bool) *slog.Logger {
	return New([]io.Writer{os.Stdout}, level, json, utc)
}

func NewFileRotate(logpath string, utc bool) (io.Writer, Close) {
	dir := filepath.Dir(logpath)
	filename := filepath.Base(logpath)
	ext := filepath.Ext(filename)
	basename := strings.TrimSuffix(filename, ext)
	fileconfig := filerotate.Config{
		DidClose: func(path string, didRotate bool) {
			// By default do noting
		},
		PathIfShouldRotate: func(creationTime time.Time, now time.Time) string {
			if creationTime.YearDay() == now.YearDay() {
				return ""
			}
			d := now
			if utc {
				d = now.UTC()
			}
			name := fmt.Sprintf("%s_%s%s", basename, d.Format(FileDateTimeFormat), ext)
			return filepath.Join(dir, name)
		},
	}
	file, err := filerotate.New(&fileconfig)
	if err != nil {
		panic(err)
	}

	return file, func() {
		file.Close()
	}
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
		return slog.LevelDebug
	}
}

func computeReplaceAttr(root string, utc bool) func(groups []string, a slog.Attr) slog.Attr {
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
			if v, ok := a.Value.Any().(*slog.Source); ok {
				file := v.File
				if root != "" {
					if rel, err := filepath.Rel(root, file); err == nil {
						file = rel
					}
				} else {
					file = filepath.Base(file)
				}
				file = filepath.ToSlash(file)
				return slog.Attr{
					Key:   "caller",
					Value: slog.StringValue(fmt.Sprintf("%s:%d", file, v.Line)),
				}
			}
		}

		return a
	}
}

func findModuleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}
