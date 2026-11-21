// Command ps-top-next is the rewrite of ps-top: it collects performance_schema
// metrics on one interval and prints table views on another.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/sjmudd/ps-top/internal/app"
)

func main() {
	cfg := app.Defaults
	flag.StringVar(&cfg.DSN, "dsn", "ps_top_monitor:pstoppw@tcp(127.0.0.1:3307)/", "DSN")
	flag.DurationVar(&cfg.CollectInterval, "collect-interval", cfg.CollectInterval, "collect interval")
	flag.DurationVar(&cfg.PresentInterval, "present-interval", cfg.PresentInterval, "presentation interval")
	flag.DurationVar(&cfg.Window, "window", cfg.Window, "window; a multiple of the collect interval")
	run := flag.Duration("for", 0, "run time; 0 runs until interrupted")
	flag.Parse()

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
