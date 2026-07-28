// Package engine contains the reusable websocket stress-test engine: config,
// connection workers, live stats, and run orchestration. Both the CLI
// (websocket_test) and the GUI (websocket_test_gui) build on this package.
package engine

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HeaderList holds raw "Key: Value" header strings and implements
// flag.Value so the CLI can bind it directly with fs.Var.
type HeaderList []string

func (h *HeaderList) String() string { return strings.Join(*h, ",") }

func (h *HeaderList) Set(v string) error {
	*h = append(*h, v)
	return nil
}

func (h HeaderList) ToHTTPHeader() (http.Header, error) {
	return ParseHeaders(h)
}

// ParseHeaders turns "Key: Value" lines into an http.Header.
func ParseHeaders(lines []string) (http.Header, error) {
	out := http.Header{}
	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header %q, expected \"Key: Value\"", raw)
		}
		out.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	return out, nil
}

// Config describes a single stress-test run.
type Config struct {
	URL            string
	Connections    int
	Ramp           int
	Duration       time.Duration
	Rate           float64
	Payload        string
	PayloadSize    int
	Binary         bool
	Echo           bool
	Headers        HeaderList
	Subprotocol    string
	Insecure       bool
	ConnectTimeout time.Duration
	WriteTimeout   time.Duration
	Interval       time.Duration
}

// DefaultConfig returns the same defaults the CLI flags use, so other
// front-ends (like the GUI) don't have to duplicate them.
func DefaultConfig() Config {
	return Config{
		Connections:    10,
		Ramp:           0,
		Duration:       30 * time.Second,
		Rate:           1,
		PayloadSize:    128,
		ConnectTimeout: 10 * time.Second,
		WriteTimeout:   5 * time.Second,
		Interval:       2 * time.Second,
	}
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	if c.Connections <= 0 {
		return errors.New("connections must be > 0")
	}
	if c.Rate < 0 {
		return errors.New("rate must be >= 0")
	}
	if c.PayloadSize <= 0 {
		return errors.New("payload-size must be > 0")
	}
	if _, err := c.Headers.ToHTTPHeader(); err != nil {
		return err
	}
	return nil
}
