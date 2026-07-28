package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/donghquinn/websocket-test/engine"
)

type CLIConfig struct {
	engine.Config
	Output   string
	NoReport bool
	Quiet    bool
}

func defaultReportName() string {
	return fmt.Sprintf("wsstress-report-%s.json", time.Now().Format("20060102-150405"))
}

func parseConfig(args []string) (*CLIConfig, error) {
	fs := flag.NewFlagSet("wsstress", flag.ContinueOnError)
	def := engine.DefaultConfig()
	cfg := &CLIConfig{}

	fs.StringVar(&cfg.URL, "url", "", "target websocket URL, e.g. ws://localhost:8080/ws (required)")
	fs.IntVar(&cfg.Connections, "connections", def.Connections, "total number of concurrent connections to open")
	fs.IntVar(&cfg.Ramp, "ramp", def.Ramp, "connections to open per second while ramping up (0 = open all at once)")
	fs.DurationVar(&cfg.Duration, "duration", def.Duration, "total test duration; 0 = run until Ctrl+C")
	fs.Float64Var(&cfg.Rate, "rate", def.Rate, "messages sent per second, per connection (0 = hold connections open without sending)")
	fs.StringVar(&cfg.Payload, "payload", "", "literal message payload to send; overrides -payload-size")
	fs.IntVar(&cfg.PayloadSize, "payload-size", def.PayloadSize, "size in bytes of a generated random payload, when -payload is not set")
	fs.BoolVar(&cfg.Binary, "binary", false, "send generated payloads as binary frames instead of text (ignored when -echo is set)")
	fs.BoolVar(&cfg.Echo, "echo", false, "wrap payloads in a JSON envelope and measure round-trip latency; requires a server that echoes messages back verbatim")
	fs.Var(&cfg.Headers, "header", "extra HTTP header for the handshake, \"Key: Value\" (repeatable)")
	fs.StringVar(&cfg.Subprotocol, "subprotocol", "", "websocket subprotocol to request")
	fs.BoolVar(&cfg.Insecure, "insecure", false, "skip TLS certificate verification for wss:// targets")
	fs.DurationVar(&cfg.ConnectTimeout, "connect-timeout", def.ConnectTimeout, "handshake timeout per connection")
	fs.DurationVar(&cfg.WriteTimeout, "write-timeout", def.WriteTimeout, "write deadline per message")
	fs.DurationVar(&cfg.Interval, "interval", def.Interval, "interval between live stats reports")
	fs.StringVar(&cfg.Output, "output", defaultReportName(), "path to write the final JSON report")
	fs.BoolVar(&cfg.NoReport, "no-report", false, "don't write a JSON report file")
	fs.BoolVar(&cfg.Quiet, "quiet", false, "suppress live interval reports, print only the final summary")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if err := cfg.Config.Validate(); err != nil {
		return nil, fmt.Errorf("-%s", err)
	}

	return cfg, nil
}
