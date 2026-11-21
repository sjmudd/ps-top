// Package app wires the program: it probes the server, builds the metrics,
// starts the collector, and presents views on its own interval.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // driver for sql.Open("mysql", ...)

	"github.com/sjmudd/ps-top/internal/collector"
	"github.com/sjmudd/ps-top/internal/filenamemap"
	"github.com/sjmudd/ps-top/internal/metrics"
	"github.com/sjmudd/ps-top/internal/presenter"
	"github.com/sjmudd/ps-top/internal/probe"
	"github.com/sjmudd/ps-top/internal/setupinstruments"
	"github.com/sjmudd/ps-top/internal/snapshot"
)

// Config holds the settings the user can change.
type Config struct {
	DSN             string
	CollectInterval time.Duration
	PresentInterval time.Duration
	Window          time.Duration // difference over this long; a multiple (1 or more) of CollectInterval
	Width           int           // output width in runes
}

// Defaults are the settings used when the user changes none.
var Defaults = Config{
	CollectInterval: time.Second,
	PresentInterval: 2 * time.Second,
	Window:          10 * time.Second,
	Width:           132, // test output: table lock names start at column 103
}

// restoreTimeout limits putting back the instrument settings at exit.
const restoreTimeout = 10 * time.Second

// fileMapVariables are the server settings file name mapping needs.
var fileMapVariables = []string{"datadir", "log_bin_basename", "relay_log_basename"}

// views returns the views in the order presented. File I/O maps file names
// with the server's settings, so views are made after the probe.
func views(srv *probe.Server) ([]presenter.View, error) {
	m, err := filenamemap.New(filenamemap.Settings{
		DataDir:      srv.Variables["datadir"],
		BinlogBase:   srv.Variables["log_bin_basename"],
		RelayLogBase: srv.Variables["relay_log_basename"],
	})
	if err != nil {
		return nil, err
	}
	return []presenter.View{presenter.TableIOLatency, presenter.TableIOOps, presenter.FileIOLatency(m.Map), presenter.MutexLatency, presenter.StagesLatency, presenter.TableLockLatency}, nil
}

// Run connects, probes, starts the collector and writes every view to out
// each cfg.PresentInterval until ctx is done (returns nil) or something fails.
func Run(ctx context.Context, cfg Config, out io.Writer) (err error) {
	if cfg.CollectInterval <= 0 || cfg.PresentInterval <= 0 {
		return fmt.Errorf("app: intervals must be more than 0: collect %v, present %v", cfg.CollectInterval, cfg.PresentInterval)
	}
	if cfg.Window < cfg.CollectInterval || cfg.Window%cfg.CollectInterval != 0 {
		return fmt.Errorf("app: window %v must be a multiple (1 or more) of the collect interval %v", cfg.Window, cfg.CollectInterval)
	}
	// The window spans n collect intervals: from the snapshot n before the
	// latest to the latest, so the ring holds n+1 snapshots.
	n := int(cfg.Window / cfg.CollectInterval)

	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)

	req := metrics.Request()
	req.Variables = append(req.Variables, fileMapVariables...)
	req.Updates = append(req.Updates, probe.Update{Table: setupinstruments.Table, Column: "enabled"})
	srv, err := probe.Run(db, req)
	if err != nil {
		return err
	}
	saved, err := setupinstruments.Enable(ctx, db, setupinstruments.Patterns, srv.Updatable[setupinstruments.Table])
	if err != nil {
		return err
	}
	// Runs before the connection is closed, after the collector is stopped.
	defer func() {
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restoreTimeout)
		defer cancel()
		if rerr := saved.Restore(rctx); rerr != nil && err == nil {
			err = rerr
		}
	}()
	vs, err := views(srv)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "%v %s\n", srv.Flavour, srv.Version); err != nil {
		return err
	}
	c, err := newCollector(db, srv, cfg.CollectInterval, n+1)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, 1)
	go func() {
		err := c.Run(ctx)
		cancel() // unblocks any pending request
		errc <- err
	}()

	start := time.Now()
	tick := time.NewTicker(cfg.PresentInterval)
	defer tick.Stop()
	for {
		select {
		case err := <-errc:
			return err
		case <-tick.C:
			var b strings.Builder
			fmt.Fprintf(&b, "--- %.1fs ---\n", time.Since(start).Seconds())
			for _, v := range vs {
				lines, err := presenter.Render(ctx, c, v, n, cfg.Width)
				if err != nil {
					if ctx.Err() != nil {
						break
					}
					return err
				}
				b.WriteString(strings.Join(lines, "\n") + "\n\n")
			}
			if _, err := io.WriteString(out, b.String()); err != nil {
				return err
			}
		}
	}
}

// newCollector chooses each metric's query for srv and returns a collector of
// all metrics that collects every interval and holds ringSize snapshots each.
func newCollector(db *sql.DB, srv *probe.Server, interval time.Duration, ringSize int) (*collector.Collector, error) {
	var cms []collector.Metric
	for _, m := range metrics.All() {
		q, err := srv.Choose(m.Queries)
		if err != nil {
			return nil, err
		}
		cms = append(cms, collector.Metric{
			Def:     m.Def,
			Collect: func(db *sql.DB) ([]snapshot.Entry, error) { return m.Collect(db, q.Query) },
		})
	}
	return collector.New(db, interval, ringSize, cms)
}
