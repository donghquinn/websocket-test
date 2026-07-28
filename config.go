package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type headerList []string

func (h *headerList) String() string { return strings.Join(*h, ",") }

func (h *headerList) Set(v string) error {
	*h = append(*h, v)
	return nil
}

func (h headerList) toHTTPHeader() (http.Header, error) {
	out := http.Header{}
	for _, raw := range h {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid -header %q, expected \"Key: Value\"", raw)
		}
		out.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	return out, nil
}

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
	Headers        headerList
	Subprotocol    string
	Insecure       bool
	ConnectTimeout time.Duration
	WriteTimeout   time.Duration
	Interval       time.Duration
	Output         string
	Quiet          bool
}

func parseConfig(args []string) (*Config, error) {
	fs := flag.NewFlagSet("wsstress", flag.ContinueOnError)
	cfg := &Config{}

	fs.StringVar(&cfg.URL, "url", "", "target websocket URL, e.g. ws://localhost:8080/ws (required)")
	fs.IntVar(&cfg.Connections, "connections", 10, "total number of concurrent connections to open")
	fs.IntVar(&cfg.Ramp, "ramp", 0, "connections to open per second while ramping up (0 = open all at once)")
	fs.DurationVar(&cfg.Duration, "duration", 30*time.Second, "total test duration; 0 = run until Ctrl+C")
	fs.Float64Var(&cfg.Rate, "rate", 1, "messages sent per second, per connection (0 = hold connections open without sending)")
	fs.StringVar(&cfg.Payload, "payload", "", "literal message payload to send; overrides -payload-size")
	fs.IntVar(&cfg.PayloadSize, "payload-size", 128, "size in bytes of a generated random payload, when -payload is not set")
	fs.BoolVar(&cfg.Binary, "binary", false, "send generated payloads as binary frames instead of text (ignored when -echo is set)")
	fs.BoolVar(&cfg.Echo, "echo", false, "wrap payloads in a JSON envelope and measure round-trip latency; requires a server that echoes messages back verbatim")
	fs.Var(&cfg.Headers, "header", "extra HTTP header for the handshake, \"Key: Value\" (repeatable)")
	fs.StringVar(&cfg.Subprotocol, "subprotocol", "", "websocket subprotocol to request")
	fs.BoolVar(&cfg.Insecure, "insecure", false, "skip TLS certificate verification for wss:// targets")
	fs.DurationVar(&cfg.ConnectTimeout, "connect-timeout", 10*time.Second, "handshake timeout per connection")
	fs.DurationVar(&cfg.WriteTimeout, "write-timeout", 5*time.Second, "write deadline per message")
	fs.DurationVar(&cfg.Interval, "interval", 2*time.Second, "interval between live stats reports")
	fs.StringVar(&cfg.Output, "output", "", "path to write a final JSON report (optional)")
	fs.BoolVar(&cfg.Quiet, "quiet", false, "suppress live interval reports, print only the final summary")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.URL == "" {
		return nil, errors.New("-url is required")
	}
	if cfg.Connections <= 0 {
		return nil, errors.New("-connections must be > 0")
	}
	if cfg.Rate < 0 {
		return nil, errors.New("-rate must be >= 0")
	}
	if cfg.PayloadSize <= 0 {
		return nil, errors.New("-payload-size must be > 0")
	}
	if _, err := cfg.Headers.toHTTPHeader(); err != nil {
		return nil, err
	}

	return cfg, nil
}
