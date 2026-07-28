package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/donghquinn/websocket-test/engine"
)

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "wsstress:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Printf("wsstress: %d connections -> %s (ramp=%d/s rate=%.2f msg/s/conn echo=%v)\n",
		cfg.Connections, cfg.URL, cfg.Ramp, cfg.Rate, cfg.Echo)

	run, err := engine.StartRun(ctx, &cfg.Config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wsstress:", err)
		os.Exit(1)
	}

	stopReporter := make(chan struct{})
	reporterDone := make(chan struct{})
	go runReporter(cfg, run.Stats, stopReporter, reporterDone)

	<-run.Done()
	close(stopReporter)
	<-reporterDone

	final := run.Stats.Final()
	printFinalReport(final)

	if !cfg.NoReport && cfg.Output != "" {
		if err := writeJSONReport(cfg.Output, final); err != nil {
			fmt.Fprintln(os.Stderr, "wsstress: failed to write report:", err)
			os.Exit(1)
		}
		fmt.Println("Report written to", cfg.Output)
	}
}

func runReporter(cfg *CLIConfig, st *engine.Stats, stop <-chan struct{}, done chan<- struct{}) {
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
			sn := st.Window(prevSent, prevRecv, since)
			printLiveLine(sn)
			prevSent, prevRecv = sn.MsgsSent, sn.MsgsRecv
			since = time.Now()
		}
	}
}
