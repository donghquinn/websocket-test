# wsstress

A generic WebSocket stress-testing CLI. Opens many concurrent connections
against any WebSocket endpoint, optionally ramping up gradually, and drives
per-connection message load while reporting live and final stats (throughput,
error counts, and round-trip latency percentiles when using `-echo`).

The connection/stats engine lives in `engine/` as an importable package.
[`../websocket_test_gui`](../websocket_test_gui) is a sibling web-dashboard
front-end that reuses it via a local `go.mod` replace directive, if you'd
rather drive tests from a browser than flags.

## Build

```
go build -o wsstress .
```

## Usage

```
./wsstress -url ws://localhost:8080/ws -connections 200 -ramp 50 -duration 30s -rate 5
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-url` | (required) | Target WebSocket URL (`ws://` or `wss://`) |
| `-connections` | `10` | Total concurrent connections to open |
| `-ramp` | `0` | Connections opened per second (`0` = open all at once) |
| `-duration` | `30s` | Total test duration (`0` = run until Ctrl+C) |
| `-rate` | `1` | Messages sent per second, per connection (`0` = just hold connections open) |
| `-payload` | | Literal message payload; overrides `-payload-size` |
| `-payload-size` | `128` | Size in bytes of a generated random payload |
| `-binary` | `false` | Send generated payloads as binary frames (ignored with `-echo`) |
| `-echo` | `false` | Wrap payloads in a JSON envelope and measure round-trip latency; requires the target to echo messages back verbatim |
| `-header` | | Extra handshake header, `"Key: Value"` (repeatable) |
| `-subprotocol` | | WebSocket subprotocol to request |
| `-insecure` | `false` | Skip TLS certificate verification for `wss://` |
| `-connect-timeout` | `10s` | Handshake timeout per connection |
| `-write-timeout` | `5s` | Write deadline per message |
| `-interval` | `2s` | Interval between live stats lines |
| `-output` | `wsstress-report-<timestamp>.json` | Path to write the final JSON report |
| `-no-report` | `false` | Don't write a JSON report file |
| `-quiet` | `false` | Suppress live stats, print only the final summary |

Every run writes a JSON report by default (`wsstress-report-YYYYMMDD-HHMMSS.json`
in the current directory). Pass `-output <path>` to control the location, or
`-no-report` to skip writing one.

### Measuring latency

By default the tool only measures throughput and connection health. To get
round-trip latency percentiles, pass `-echo` against a server that echoes
messages back unchanged (e.g. a simple echo server) — the client stamps each
message with a send timestamp and computes latency when it sees it come back.

### Examples

Connection-scale test: ramp up to 5,000 idle connections and hold them:

```
./wsstress -url ws://localhost:8080/ws -connections 5000 -ramp 200 -duration 60s -rate 0
```

Throughput/latency test against an echo endpoint:

```
./wsstress -url ws://localhost:8080/echo -connections 100 -rate 20 -echo -duration 30s -output report.json
```

Authenticated endpoint:

```
./wsstress -url wss://api.example.com/ws -header "Authorization: Bearer <token>" -connections 50
```
