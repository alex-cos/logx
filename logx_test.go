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
