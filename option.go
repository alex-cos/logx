package logx

import (
	"maps"
	"net/http"
	"time"
)

type LokiOption func(*LokiClient)

func WithHTTPS(b bool) LokiOption {
	return func(c *LokiClient) {
		c.useHTTPS = b
	}
}

func WithBasicAuth(username, password string) LokiOption {
	return func(c *LokiClient) {
		c.username = username
		c.password = password
	}
}

func WithBearerToken(token string) LokiOption {
	return func(c *LokiClient) {
		c.bearer = token
	}
}

func WithLabels(labels map[string]string) LokiOption {
	return func(c *LokiClient) {
		maps.Copy(c.labels, labels)
	}
}

func WithHttpClient(httpClient *http.Client) LokiOption {
	return func(c *LokiClient) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithBatchSize(size int) LokiOption {
	return func(c *LokiClient) {
		if size > 0 && size < 1000 {
			c.batchSize = size
		}
	}
}

func WithPeriod(d time.Duration) LokiOption {
	return func(c *LokiClient) {
		if d > 0 {
			c.period = d
		}
	}
}

func WithWriteTimeout(d time.Duration) LokiOption {
	return func(c *LokiClient) {
		if d > 0 {
			c.writeTimeout = d
		}
	}
}

func WithSendTimeout(d time.Duration) LokiOption {
	return func(c *LokiClient) {
		if d > 0 {
			c.sendTimeout = d
		}
	}
}

func WithBufferSize(size int) LokiOption {
	return func(c *LokiClient) {
		if size > 0 {
			c.buffer = make(chan []any, size)
		}
	}
}
