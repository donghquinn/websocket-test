package main

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Stats aggregates counters from every worker goroutine. Counter fields are
// updated with atomics; latencies need a mutex since percentiles require a
// sorted view across all samples.
type Stats struct {
	connectAttempts int64
	connectSuccess  int64
	connectFail     int64
	activeConns     int64
	disconnects     int64
	msgsSent        int64
	bytesSent       int64
	msgsRecv        int64
	bytesRecv       int64
	errors          int64

	start time.Time

	mu            sync.Mutex
	latencies     []time.Duration
	lastReportIdx int
}

func NewStats() *Stats {
	return &Stats{start: time.Now()}
}

func (s *Stats) ConnectAttempt()               { atomic.AddInt64(&s.connectAttempts, 1) }
func (s *Stats) ConnectSuccess()               { atomic.AddInt64(&s.connectSuccess, 1) }
func (s *Stats) ConnectFail()                  { atomic.AddInt64(&s.connectFail, 1) }
func (s *Stats) ConnActive(delta int64)        { atomic.AddInt64(&s.activeConns, delta) }
func (s *Stats) Disconnect()                   { atomic.AddInt64(&s.disconnects, 1) }
func (s *Stats) MsgSent(bytes int)             { atomic.AddInt64(&s.msgsSent, 1); atomic.AddInt64(&s.bytesSent, int64(bytes)) }
func (s *Stats) MsgRecv(bytes int)             { atomic.AddInt64(&s.msgsRecv, 1); atomic.AddInt64(&s.bytesRecv, int64(bytes)) }
func (s *Stats) Error()                        { atomic.AddInt64(&s.errors, 1) }

func (s *Stats) Latency(d time.Duration) {
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.mu.Unlock()
}

// Snapshot describes stats over a window: either "since last report" (live
// interval reports) or "since test start" (final summary).
type Snapshot struct {
	Elapsed         time.Duration
	ActiveConns     int64
	ConnectAttempts int64
	ConnectSuccess  int64
	ConnectFail     int64
	Disconnects     int64
	MsgsSent        int64
	BytesSent       int64
	MsgsRecv        int64
	BytesRecv       int64
	Errors          int64

	WindowMsgsSent int64
	WindowMsgsRecv int64
	WindowSeconds  float64

	LatencyCount int
	LatencyMin   time.Duration
	LatencyAvg   time.Duration
	LatencyP50   time.Duration
	LatencyP90   time.Duration
	LatencyP99   time.Duration
	LatencyMax   time.Duration
}

// window snapshot also advances the "last report" cursor, so repeated calls
// report only newly-observed latency samples each time.
func (s *Stats) window(prevSent, prevRecv int64, since time.Time) Snapshot {
	sn := Snapshot{
		Elapsed:         time.Since(s.start),
		ActiveConns:     atomic.LoadInt64(&s.activeConns),
		ConnectAttempts: atomic.LoadInt64(&s.connectAttempts),
		ConnectSuccess:  atomic.LoadInt64(&s.connectSuccess),
		ConnectFail:     atomic.LoadInt64(&s.connectFail),
		Disconnects:     atomic.LoadInt64(&s.disconnects),
		MsgsSent:        atomic.LoadInt64(&s.msgsSent),
		BytesSent:       atomic.LoadInt64(&s.bytesSent),
		MsgsRecv:        atomic.LoadInt64(&s.msgsRecv),
		BytesRecv:       atomic.LoadInt64(&s.bytesRecv),
		Errors:          atomic.LoadInt64(&s.errors),
	}
	sn.WindowMsgsSent = sn.MsgsSent - prevSent
	sn.WindowMsgsRecv = sn.MsgsRecv - prevRecv
	sn.WindowSeconds = time.Since(since).Seconds()

	s.mu.Lock()
	newSamples := append([]time.Duration(nil), s.latencies[s.lastReportIdx:]...)
	s.lastReportIdx = len(s.latencies)
	s.mu.Unlock()

	fillLatencyStats(&sn, newSamples)
	return sn
}

// Final computes a summary over every latency sample collected during the
// whole run, regardless of what's already been reported in windows.
func (s *Stats) Final() Snapshot {
	sn := Snapshot{
		Elapsed:         time.Since(s.start),
		ActiveConns:     atomic.LoadInt64(&s.activeConns),
		ConnectAttempts: atomic.LoadInt64(&s.connectAttempts),
		ConnectSuccess:  atomic.LoadInt64(&s.connectSuccess),
		ConnectFail:     atomic.LoadInt64(&s.connectFail),
		Disconnects:     atomic.LoadInt64(&s.disconnects),
		MsgsSent:        atomic.LoadInt64(&s.msgsSent),
		BytesSent:       atomic.LoadInt64(&s.bytesSent),
		MsgsRecv:        atomic.LoadInt64(&s.msgsRecv),
		BytesRecv:       atomic.LoadInt64(&s.bytesRecv),
		Errors:          atomic.LoadInt64(&s.errors),
	}

	s.mu.Lock()
	all := append([]time.Duration(nil), s.latencies...)
	s.mu.Unlock()

	fillLatencyStats(&sn, all)
	return sn
}

func fillLatencyStats(sn *Snapshot, samples []time.Duration) {
	sn.LatencyCount = len(samples)
	if len(samples) == 0 {
		return
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	var total time.Duration
	for _, d := range samples {
		total += d
	}
	sn.LatencyMin = samples[0]
	sn.LatencyMax = samples[len(samples)-1]
	sn.LatencyAvg = total / time.Duration(len(samples))
	sn.LatencyP50 = percentile(samples, 50)
	sn.LatencyP90 = percentile(samples, 90)
	sn.LatencyP99 = percentile(samples, 99)
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p / 100 * float64(len(sorted)-1))
	return sorted[idx]
}
