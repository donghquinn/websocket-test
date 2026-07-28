package engine

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Run is a single in-flight (or finished) stress-test run.
type Run struct {
	Cfg    *Config
	Stats  *Stats
	cancel context.CancelFunc
	doneCh chan struct{}
}

// StartRun validates cfg, then launches all connection workers in the
// background and returns immediately with a handle to observe/stop the run.
// parent controls external cancellation (e.g. an OS signal in the CLI, or
// context.Background() plus an explicit Stop() call from a GUI); Cfg.Duration
// additionally bounds the run's lifetime when non-zero.
func StartRun(parent context.Context, cfg *Config) (*Run, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	headers, err := cfg.Headers.ToHTTPHeader()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parent)
	if cfg.Duration > 0 {
		timer := time.AfterFunc(cfg.Duration, cancel)
		go func() {
			<-ctx.Done()
			timer.Stop()
		}()
	}

	run := &Run{
		Cfg:    cfg,
		Stats:  NewStats(),
		cancel: cancel,
		doneCh: make(chan struct{}),
	}

	go func() {
		defer close(run.doneCh)
		var wg sync.WaitGroup
		wg.Add(cfg.Connections)
		launchWorkers(ctx, cfg, run.Stats, headers, &wg)
		wg.Wait()
	}()

	return run, nil
}

// Stop ends the run immediately, as if its duration had elapsed.
func (r *Run) Stop() { r.cancel() }

// Done is closed once every worker has returned.
func (r *Run) Done() <-chan struct{} { return r.doneCh }

// Finished reports whether every worker has already returned.
func (r *Run) Finished() bool {
	select {
	case <-r.doneCh:
		return true
	default:
		return false
	}
}

func launchWorkers(ctx context.Context, cfg *Config, st *Stats, headers http.Header, wg *sync.WaitGroup) {
	if cfg.Ramp <= 0 {
		for i := 0; i < cfg.Connections; i++ {
			go func() {
				defer wg.Done()
				RunWorker(ctx, cfg, st, headers)
			}()
		}
		return
	}

	interval := time.Second / time.Duration(cfg.Ramp)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	launched := 0
	for launched < cfg.Connections {
		select {
		case <-ctx.Done():
			// stop ramping; account for connections that will never launch
			remaining := cfg.Connections - launched
			for range remaining {
				wg.Done()
			}
			return
		case <-ticker.C:
			go func() {
				defer wg.Done()
				RunWorker(ctx, cfg, st, headers)
			}()
			launched++
		}
	}
}
