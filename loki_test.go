package logx_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/alex-cos/logx"
)

func TestLokiBasicAuth(t *testing.T) {
	t.Parallel()

	loki, stop := logx.NewLokiClient("localhost", 3100,
		logx.WithHttpClient(http.DefaultClient),
		logx.WithHTTPS(false),
		logx.WithBasicAuth("johnDoe", "12345"),
		logx.WithBearerToken(""),
		logx.WithBufferSize(1000),
		logx.WithBatchSize(120),
		logx.WithPeriod(time.Second),
		logx.WithWriteTimeout(100*time.Millisecond),
		logx.WithSendTimeout(3*time.Second),
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

	time.Sleep(2 * time.Second)
}

func TestLokiToken(t *testing.T) {
	t.Parallel()

	loki, stop := logx.NewLokiClient("localhost", 3100,
		logx.WithHttpClient(http.DefaultClient),
		logx.WithHTTPS(false),
		logx.WithBasicAuth("", ""),
		logx.WithBearerToken("myToken"),
		logx.WithBufferSize(1000),
		logx.WithBatchSize(120),
		logx.WithPeriod(time.Second),
		logx.WithWriteTimeout(100*time.Millisecond),
		logx.WithSendTimeout(3*time.Second),
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

	time.Sleep(2 * time.Second)
}
