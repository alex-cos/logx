package logx_test

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/alex-cos/logx"
)

func TestConsole(t *testing.T) {
	t.Parallel()

	logger := logx.NewConsoleLogger(
		"Info",
		logx.WithJSON(),
		logx.WithUTC(),
	)

	logger.Debug("Debug")
	logger.Info("Info")
	logger.Warn("Warn")
	logger.Error("Error", logx.Error(errors.New("my error")))
}

func TestFile(t *testing.T) {
	t.Parallel()

	tempdir := t.TempDir()
	logpath := filepath.Join(tempdir, "log")

	logger, closeFile, err := logx.NewFileLogger(logpath, "Warn", logx.WithUTC())
	if err != nil {
		t.Fatalf("NewFileRotate failed: %v", err)
	}
	defer closeFile()

	logger.With("source", "my_source")

	logger.Debug("Debug")
	logger.Info("Info")
	logger.Warn("Warn")
	logger.Error("Error", logx.Error(errors.New("my error")))

	time.Sleep(100 * time.Millisecond)
}

func TestFileRotate(t *testing.T) {
	t.Parallel()

	tempdir := t.TempDir()
	logpath := filepath.Join(tempdir, "log")

	file, closeFile, err := logx.NewFileRotate(logpath, logx.WithUTC())
	if err != nil {
		t.Fatalf("NewFileRotate failed: %v", err)
	}
	defer closeFile()

	logger := logx.New(
		[]io.Writer{file},
		"Debug",
		logx.WithJSON(),
		logx.WithUTC(),
	).With("service", "my_service")

	logger.Debug("Debug")
	logger.Info("Info")
	logger.Warn("Warn")
	logger.Error("Error", logx.Error(errors.New("my error")))

	time.Sleep(100 * time.Millisecond)
}
