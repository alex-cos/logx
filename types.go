package logx

import "path/filepath"

type Close func()

type Option func(*config)

type config struct {
	basePath string
	source   bool
	json     bool
	utc      bool
	caller   bool
	stack    bool
}

func WithBasePath(path string) Option {
	return func(c *config) {
		c.basePath = filepath.ToSlash(path)
	}
}

func WithSource() Option {
	return func(c *config) {
		c.source = true
	}
}

func WithJSON() Option {
	return func(c *config) {
		c.json = true
	}
}

func WithUTC() Option {
	return func(c *config) {
		c.utc = true
	}
}

func WithCaller() Option {
	return func(c *config) {
		c.caller = true
	}
}

func WithStack() Option {
	return func(c *config) {
		c.stack = true
	}
}

func parseConfig(opts ...Option) *config {
	cfg := &config{
		basePath: "",
		source:   false,
		json:     false,
		utc:      false,
		caller:   false,
		stack:    false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
