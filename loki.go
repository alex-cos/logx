package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	baseURL = "loki/api/v1"
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
	useHTTPS     bool
	username     string
	password     string
	bearer       string
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

func WithHTTPS(b bool) Option {
	return func(c *LokiClient) {
		c.useHTTPS = b
	}
}

func WithBasicAuth(username, password string) Option {
	return func(c *LokiClient) {
		c.username = username
		c.password = password
	}
}

func WithBearerToken(token string) Option {
	return func(c *LokiClient) {
		c.bearer = token
	}
}

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
		if size > 0 && size < 1000 {
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
		useHTTPS:     false,
		username:     "",
		password:     "",
		bearer:       "",
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

	if len(batch) == 0 {
		return
	}
	for i := range 3 {
		err = c.send(context.Background(), batch)
		if err == nil {
			return
		}
		sleep := time.Second * time.Duration(i+1)
		sleep += time.Duration(rand.Intn(400)) * time.Millisecond // nolint: gosec
		time.Sleep(sleep)
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
	scheme := "http"
	if c.useHTTPS {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d/%s/push", scheme, c.host, c.port, baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoLokiClient")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		// empty response buffer
		io.ReadAll(io.LimitReader(resp.Body, 2048)) // nolint:errcheck
		return fmt.Errorf("server returned status %s (%d)", resp.Status, resp.StatusCode)
	}

	return nil
}
