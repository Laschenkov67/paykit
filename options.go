package paykit

import (
	"log/slog"
	"net/http"
	"time"
)

type Config struct {
	HTTPClient *http.Client
	BaseURL    string
	Timeout    time.Duration
	Logger     *slog.Logger
	UserAgent  string
}

type Option func(*Config)

func WithHTTPClient(c *http.Client) Option { return func(cfg *Config) { cfg.HTTPClient = c } }

func WithBaseURL(u string) Option { return func(cfg *Config) { cfg.BaseURL = u } }

func WithTimeout(d time.Duration) Option { return func(cfg *Config) { cfg.Timeout = d } }

func WithLogger(l *slog.Logger) Option { return func(cfg *Config) { cfg.Logger = l } }

func WithUserAgent(ua string) Option { return func(cfg *Config) { cfg.UserAgent = ua } }

func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		Timeout:   30 * time.Second,
		UserAgent: "paykit-go/1.0",
	}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.Timeout}
	}
	return cfg
}
