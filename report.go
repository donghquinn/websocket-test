package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/donghquinn/websocket-test/engine"
)

func printLiveLine(sn engine.Snapshot) {
	sentRate := 0.0
	recvRate := 0.0
	if sn.WindowSeconds > 0 {
		sentRate = float64(sn.WindowMsgsSent) / sn.WindowSeconds
		recvRate = float64(sn.WindowMsgsRecv) / sn.WindowSeconds
	}

	line := fmt.Sprintf("[%6s] active=%d connected=%d/%d failed=%d sent=%.0f/s recv=%.0f/s errors=%d",
		sn.Elapsed.Round(time.Second), sn.ActiveConns, sn.ConnectSuccess, sn.ConnectAttempts, sn.ConnectFail,
		sentRate, recvRate, sn.Errors)

	if sn.LatencyCount > 0 {
		line += fmt.Sprintf(" | latency p50=%s p90=%s p99=%s max=%s",
			sn.LatencyP50.Round(time.Microsecond), sn.LatencyP90.Round(time.Microsecond),
			sn.LatencyP99.Round(time.Microsecond), sn.LatencyMax.Round(time.Microsecond))
	}
	fmt.Println(line)
}

func printFinalReport(sn engine.Snapshot) {
	fmt.Println()
	fmt.Println("==================== Summary ====================")
	fmt.Printf("Duration:            %s\n", sn.Elapsed.Round(time.Millisecond))
	fmt.Printf("Connections:         %d attempted, %d succeeded, %d failed\n", sn.ConnectAttempts, sn.ConnectSuccess, sn.ConnectFail)
	fmt.Printf("Disconnects:         %d\n", sn.Disconnects)
	fmt.Printf("Messages sent:       %d (%s)\n", sn.MsgsSent, humanBytes(sn.BytesSent))
	fmt.Printf("Messages received:   %d (%s)\n", sn.MsgsRecv, humanBytes(sn.BytesRecv))
	fmt.Printf("Errors:              %d\n", sn.Errors)
	if sn.Elapsed.Seconds() > 0 {
		fmt.Printf("Avg send throughput: %.1f msg/s\n", float64(sn.MsgsSent)/sn.Elapsed.Seconds())
		fmt.Printf("Avg recv throughput: %.1f msg/s\n", float64(sn.MsgsRecv)/sn.Elapsed.Seconds())
	}
	if sn.LatencyCount > 0 {
		fmt.Println("Round-trip latency (echo mode):")
		fmt.Printf("  samples=%d min=%s avg=%s p50=%s p90=%s p99=%s max=%s\n",
			sn.LatencyCount, sn.LatencyMin.Round(time.Microsecond), sn.LatencyAvg.Round(time.Microsecond),
			sn.LatencyP50.Round(time.Microsecond), sn.LatencyP90.Round(time.Microsecond),
			sn.LatencyP99.Round(time.Microsecond), sn.LatencyMax.Round(time.Microsecond))
	}
	fmt.Println("===================================================")
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

type jsonReport struct {
	DurationSeconds    float64 `json:"duration_seconds"`
	ConnectAttempts    int64   `json:"connect_attempts"`
	ConnectSuccess     int64   `json:"connect_success"`
	ConnectFail        int64   `json:"connect_fail"`
	Disconnects        int64   `json:"disconnects"`
	MsgsSent           int64   `json:"messages_sent"`
	BytesSent          int64   `json:"bytes_sent"`
	MsgsRecv           int64   `json:"messages_received"`
	BytesRecv          int64   `json:"bytes_received"`
	Errors             int64   `json:"errors"`
	LatencySamples     int     `json:"latency_samples,omitempty"`
	LatencyMinMs       float64 `json:"latency_min_ms,omitempty"`
	LatencyAvgMs       float64 `json:"latency_avg_ms,omitempty"`
	LatencyP50Ms       float64 `json:"latency_p50_ms,omitempty"`
	LatencyP90Ms       float64 `json:"latency_p90_ms,omitempty"`
	LatencyP99Ms       float64 `json:"latency_p99_ms,omitempty"`
	LatencyMaxMs       float64 `json:"latency_max_ms,omitempty"`
}

func writeJSONReport(path string, sn engine.Snapshot) error {
	r := jsonReport{
		DurationSeconds: sn.Elapsed.Seconds(),
		ConnectAttempts: sn.ConnectAttempts,
		ConnectSuccess:  sn.ConnectSuccess,
		ConnectFail:     sn.ConnectFail,
		Disconnects:     sn.Disconnects,
		MsgsSent:        sn.MsgsSent,
		BytesSent:       sn.BytesSent,
		MsgsRecv:        sn.MsgsRecv,
		BytesRecv:       sn.BytesRecv,
		Errors:          sn.Errors,
		LatencySamples:  sn.LatencyCount,
	}
	if sn.LatencyCount > 0 {
		r.LatencyMinMs = float64(sn.LatencyMin.Microseconds()) / 1000
		r.LatencyAvgMs = float64(sn.LatencyAvg.Microseconds()) / 1000
		r.LatencyP50Ms = float64(sn.LatencyP50.Microseconds()) / 1000
		r.LatencyP90Ms = float64(sn.LatencyP90.Microseconds()) / 1000
		r.LatencyP99Ms = float64(sn.LatencyP99.Microseconds()) / 1000
		r.LatencyMaxMs = float64(sn.LatencyMax.Microseconds()) / 1000
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
