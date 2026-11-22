// Command ps-top is the rewrite of ps-top: it collects performance_schema
// metrics on one interval and prints table views on another.
//
// Until the TUI exists, only the printer style runs, chosen with --batch:
//
//	ps-top --batch                                         # all views, every --interval, until interrupted
//	ps-top --batch --interval=10s --count=6                # every 10 s, 6 outputs
//	ps-top --batch --view=table_io_latency:current,file_io_latency
//	ps-top --batch --view=user_latency --limit=10          # top 10 lines
//	ps-top --batch --totals                                # totals lines only
//
// The first output shows values up to now, as vmstat's first line does,
// the following ones each view in its style (by default the difference
// over --window, or current values).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/sjmudd/ps-top/internal/app"
)

func main() {
	cfg := app.Defaults
	flag.StringVar(&cfg.DSN, "dsn", "ps_top_monitor:pstoppw@tcp(127.0.0.1:3307)/", "DSN")
	flag.DurationVar(&cfg.CollectInterval, "collect-interval", cfg.CollectInterval, "collect interval")
	flag.DurationVar(&cfg.PresentInterval, "interval", cfg.PresentInterval, "time between outputs")
	flag.DurationVar(&cfg.Window, "window", cfg.Window, "difference over this long; a multiple of the collect interval")
	batch := flag.Bool("batch", false, "print the views, as text (the TUI is not built yet)")
	views := flag.String("view", "", "views to show, separated by \",\", each name[:current|difference]; default all")
	flag.IntVar(&cfg.Count, "count", 0, "outputs to print, the first included; 0 until interrupted")
	flag.IntVar(&cfg.Limit, "limit", 0, "at most this many lines per view; 0 no limit")
	flag.BoolVar(&cfg.TotalsOnly, "totals", false, "only each view's totals line, no detail lines")
	flag.IntVar(&cfg.Width, "width", 0, "cut lines at this many characters; 0 not cut")
	run := flag.Duration("for", 0, "run time; 0 runs until interrupted")
	flag.Parse()

	if !*batch {
		fmt.Fprintln(os.Stderr, "ps-top: the TUI is not built yet: use --batch")
		os.Exit(2)
	}
	var err error
	if cfg.Views, err = app.ParseViews(*views); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *run > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *run)
		defer cancel()
	}
	if err := app.Run(ctx, cfg, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
