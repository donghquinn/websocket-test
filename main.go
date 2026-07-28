package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "wsstress:", err)
		os.Exit(1)
	}

	headers, err := cfg.Headers.toHTTPHeader()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wsstress:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.Duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Duration)
		defer cancel()
	}

	st := NewStats()

	fmt.Printf("wsstress: %d connections -> %s (ramp=%d/s rate=%.2f msg/s/conn echo=%v)\n",
		cfg.Connections, cfg.URL, cfg.Ramp, cfg.Rate, cfg.Echo)

	var wg sync.WaitGroup
	wg.Add(cfg.Connections)
	go launchWorkers(ctx, cfg, st, headers, &wg)

	stopReporter := make(chan struct{})
	reporterDone := make(chan struct{})
	go runReporter(cfg, st, stopReporter, reporterDone)

	wg.Wait()
	close(stopReporter)
	<-reporterDone

	final := st.Final()
	printFinalReport(final)

	if cfg.Output != "" {
		if err := writeJSONReport(cfg.Output, final); err != nil {
			fmt.Fprintln(os.Stderr, "wsstress: failed to write report:", err)
			os.Exit(1)
		}
		fmt.Println("Report written to", cfg.Output)
	}
}

func launchWorkers(ctx context.Context, cfg *Config, st *Stats, headers http.Header, wg *sync.WaitGroup) {
	if cfg.Ramp <= 0 {
		for i := 0; i < cfg.Connections; i++ {
			go func() {
				defer wg.Done()
				runWorker(ctx, cfg, st, headers)
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
				runWorker(ctx, cfg, st, headers)
			}()
			launched++
		}
	}
}

func runReporter(cfg *Config, st *Stats, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	if cfg.Quiet {
		<-stop
		return
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var prevSent, prevRecv int64
	since := time.Now()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			sn := st.window(prevSent, prevRecv, since)
			printLiveLine(sn)
			prevSent, prevRecv = sn.MsgsSent, sn.MsgsRecv
			since = time.Now()
		}
	}
}
