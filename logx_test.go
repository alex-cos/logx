package logx_test

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/alex-cos/logx"
)

func TestConsole(t *testing.T) {
	t.Parallel()

	logger := logx.NewConsoleLogger("Info", false, true)

	logger.Info("Test")
}

func TestFile(t *testing.T) {
	t.Parallel()

	tempdir := t.TempDir()
	logpath := filepath.Join(tempdir, "log")

	file, closeFile := logx.NewFileRotate(logpath, true)
	defer closeFile()

	logger := logx.New([]io.Writer{file}, "Debug", true, true).With("service", "my_service")

	logger.Info("Test")

	time.Sleep(100 * time.Millisecond)
}

func TestLoki(t *testing.T) {
	t.Parallel()

	loki, stop := logx.NewLokiClient("localhost", 3100,
		logx.WithPeriod(time.Second),
		logx.WithLabels(map[string]string{
			"app":          "my_app",
			"service_name": "my_service",
		}),
	)
	defer stop()
	logger := logx.New([]io.Writer{loki}, "Debug", true, true).With("service", "my_service")

	logger.Debug("This a debug message")
	time.Sleep(10 * time.Millisecond)
	logger.Info("This is a test")
	time.Sleep(10 * time.Millisecond)
	logger.Warn("This is a warning")
	time.Sleep(10 * time.Millisecond)
	logger.Error("This is an error")

	time.Sleep(5 * time.Second)
}
