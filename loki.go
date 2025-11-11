package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type lokiRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]any           `json:"values"`
}

type LokiClient struct {
	host         string
	port         int
	baseURL      string
	httpClient   *http.Client
	labels       map[string]string
	batchSize    int
	writeTimeout time.Duration
	sendTimeout  time.Duration
	period       time.Duration
	buffer       chan []any
	wg           sync.WaitGroup
	once         sync.Once
}

// -----------------------------------------------------------------------------
// Options
// -----------------------------------------------------------------------------

type Option func(*LokiClient)

func WithLabels(labels map[string]string) Option {
	return func(c *LokiClient) {
		for k, v := range labels {
			c.labels[k] = v
		}
	}
}

func WithHttpClient(httpClient *http.Client) Option {
	return func(c *LokiClient) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithBatchSize(size int) Option {
	return func(c *LokiClient) {
		if size > 0 {
			c.batchSize = size
		}
	}
}

func WithPeriod(d time.Duration) Option {
	return func(c *LokiClient) {
		if d > 0 {
			c.period = d
		}
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(c *LokiClient) {
		if d > 0 {
			c.writeTimeout = d
		}
	}
}

func WithSendTimeout(d time.Duration) Option {
	return func(c *LokiClient) {
		if d > 0 {
			c.sendTimeout = d
		}
	}
}

func WithBufferSize(size int) Option {
	return func(c *LokiClient) {
		if size > 0 {
			c.buffer = make(chan []any, size)
		}
	}
}

// -----------------------------------------------------------------------------
// Constructor
// -----------------------------------------------------------------------------

func NewLokiClient(host string, port int, opts ...Option) (*LokiClient, Close) {
	c := &LokiClient{
		host:         host,
		port:         port,
		baseURL:      "loki/api/v1",
		httpClient:   http.DefaultClient,
		labels:       make(map[string]string),
		batchSize:    100,
		writeTimeout: 100 * time.Millisecond,
		sendTimeout:  5 * time.Second,
		period:       15 * time.Second,
		buffer:       make(chan []any, 1000),
		wg:           sync.WaitGroup{},
		once:         sync.Once{},
	}

	for _, o := range opts {
		o(c)
	}

	c.wg.Add(1)
	go c.run()

	return c, c.stop
}

// -----------------------------------------------------------------------------
// Public
// -----------------------------------------------------------------------------

func (c *LokiClient) Write(input []byte) (int, error) {
	var values map[string]any

	defer func() {
		recover() // nolint: errcheck
	}()

	err := json.Unmarshal(input, &values)
	if err != nil {
		return 0, err
	}
	datetime, ok := values["time"]
	if !ok {
		return 0, errors.New("missing time parameter")
	}
	datetimeStr, ok := datetime.(string)
	if !ok {
		return 0, errors.New("wrong time format")
	}
	d, err := time.Parse(DateTimeFormatMilli, datetimeStr)
	if err != nil {
		return 0, err
	}
	msg, ok := values["msg"]
	if !ok {
		return 0, errors.New("missing msg parameter")
	}

	delete(values, "time")
	delete(values, "msg")
	delete(values, "service")

	for k, v := range values {
		if _, ok := v.(string); !ok {
			values[k] = fmt.Sprintf("%v", v)
		}
	}

	msgStr, ok := msg.(string)
	if !ok {
		return 0, errors.New("wrong msg format")
	}

	select {
	case c.buffer <- []any{
		strconv.FormatInt(d.UnixNano(), 10),
		msgStr,
		values,
	}:
	case <-time.After(c.writeTimeout):
		fmt.Fprintf(os.Stderr, "[LokiClient] buffer is full, dropping log\n")
	}

	return len(input), nil
}

// ----------------------------------------------------------------------------
// Unexported functions
// ----------------------------------------------------------------------------

func (c *LokiClient) stop() {
	c.once.Do(func() {
		time.Sleep(500 * time.Millisecond)
		close(c.buffer)
	})
	c.wg.Wait()
}

func (c *LokiClient) run() {
	defer c.wg.Done()

	waitCheck := time.NewTicker(c.period)
	batch := [][]any{}
	for {
		select {
		case e, ok := <-c.buffer:
			if !ok {
				c.sendBatch(batch)
				return
			}
			batch = append(batch, e)
			if len(batch) >= c.batchSize {
				c.sendBatch(batch)
				batch = batch[:0]
			}

		case <-waitCheck.C:
			c.sendBatch(batch)
			batch = batch[:0]
		}
	}
}

func (c *LokiClient) sendBatch(batch [][]any) {
	var err error

	if len(batch) > 0 {
		for i := range 3 {
			err = c.send(context.Background(), batch)
			if err == nil {
				return
			}
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LokiClient] failed to send batch: %v\n", err)
	}
}

func (c *LokiClient) send(ctx context.Context, batch [][]any) error {
	ctx, cancel := context.WithTimeout(ctx, c.sendTimeout)
	defer cancel()

	buf, err := json.Marshal(&lokiRequest{
		Streams: []lokiStream{{
			Stream: c.labels,
			Values: batch,
		}},
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://%s:%d/%s/push", c.host, c.port, c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoLokiClient")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		io.ReadAll(io.LimitReader(resp.Body, 2048)) // nolint: errcheck
		return fmt.Errorf("server returned status %s (%d)", resp.Status, resp.StatusCode)
	}

	return nil
}
