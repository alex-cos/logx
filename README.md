# logx — Extensible logger with file rotation and Loki support

`logx` is a lightweight, extensible Go logging module that supports structured logging
to multiple destinations:

- local rotating files (`FileRotate`), and/or
- a **Grafana Loki** instance (`LokiClient`).

It’s built for simplicity, safety, and performance — ideal for production systems.

---

## Installation

```bash
go get https://github.com/alex-cos/logx.git
```

---

## Features

- Multiple outputs — file, Loki, console, or custom writers
- Flexible configuration — log levels, JSON output, colored console logs
- Buffered, non-blocking Loki client with automatic batching & retries
- Automatic file rotation using lumberjack
- Thread-safe and efficient for concurrent applications
- Easily testable (implements io.Writer)

---

## LokiClient options

| Option                          | Description                                  | Default            |
| :------------------------------ | :------------------------------------------- | :----------------- |
| WithLabels(map[string]string)   | Add static Loki labels (service, env, etc.)  | {}                 |
| WithBatchSize(int)              | Max number of entries before sending a batch | 100                |
| WithBufferSize(int)             | Size of the internal log buffer              | 1000               |
| WithPeriod(time.Duration)       | Interval between automatic batch flushes     | 15s                |
| WithWriteTimeout(time.Duration) | Timeout for writing to the buffer            | 100ms              |
| WithSendTimeout(time.Duration)  | Timeout for HTTP send operations             | 5s                 |
| WithHTTPClient(*http.Client)    | Custom HTTP client (TLS, proxy, auth, etc.)  | http.DefaultClient |

---

## Quick Example

Write logs to a local rotating file

```go
package main

import (
  "io"
  "path/filepath"

  "github.com/yourusername/logx"
)

func main() {
  logpath := filepath.Join("logs", "app")

  // Create a rotating file writer
  file := logx.NewFileRotate(logpath, true)
  defer file.Close()

  // Create the logger
  logger := logx.New([]io.Writer{file}, "Info", true, true).
    With("service", "demo")

  logger.Info("Application started")
  logger.Warn("Disk almost full")
  logger.Error("Critical error", "code", 500)
}
```

---

## Send logs to Loki

```go
package main

import (
  "io"

  "github.com/yourusername/logx"
)

func main() {
  // Create a Loki client
  loki, closeLoki := logx.NewLokiClient("localhost", 3100,
    logx.WithLabels(map[string]string{
      "service": "my_service",
      "env":     "dev",
    }),
  )
  defer closeLoki()

  // Combine file + Loki writers
  file := logx.NewFileRotate("logs/app", true)
  defer file.Close()

  logger := logx.New([]io.Writer{file, loki}, "Debug", true, true)

  logger.Info("Hello from logx!")
  logger.Error("Something went wrong", "code", 42)
}
```

---

## Security & Production Notes

- Use a custom http.Client (WithHTTPClient) when sending logs to Grafana Cloud or TLS endpoints.
- Set realistic timeouts for slow networks or proxies.
- The Loki client is non-blocking — logs may be dropped if the buffer is full.
- Errors and retries are reported to stderr.
-Consider exposing Prometheus metrics (sent, dropped, failed) for monitoring.
