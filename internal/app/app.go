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
	"github.com/sjmudd/ps-top/internal/registry"
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
func views(srv *probe.Server) ([]presenter.Renderer, error) {
	m, err := filenamemap.New(filenamemap.Settings{
		DataDir:      srv.Variables["datadir"],
		BinlogBase:   srv.Variables["log_bin_basename"],
		RelayLogBase: srv.Variables["relay_log_basename"],
	})
	if err != nil {
		return nil, err
	}
	return []presenter.Renderer{
		presenter.TableIOLatency,
		presenter.TableIOOps,
		presenter.FileIOLatency(m.Map),
		presenter.MutexLatency,
		presenter.StagesLatency,
		presenter.TableLockLatency,
		presenter.MemoryUsage,
	}, nil
}

// Run connects, probes, starts the collector and writes every view to out
// each cfg.PresentInterval until ctx is done (returns nil) or something fails.
func Run(ctx context.Context, cfg Config, out io.Writer) (err error) {
	reg := registry.New(time.Now())
	if cfg.CollectInterval <= 0 || cfg.PresentInterval <= 0 {
		return fmt.Errorf("app: intervals must be more than 0: collect %v, present %v", cfg.CollectInterval, cfg.PresentInterval)
	}
	if cfg.Window < cfg.CollectInterval || cfg.Window%cfg.CollectInterval != 0 {
		return fmt.Errorf("app: window %v must be a multiple (1 or more) of the collect interval %v", cfg.Window, cfg.CollectInterval)
	}
	// The window spans n collect intervals: from the snapshot n before the
	// latest to the latest. The collector sizes each ring from n.
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
	cms, skipped, err := usable(srv, reg)
	if err != nil {
		return err
	}
	for _, s := range skipped {
		if _, err := fmt.Fprintf(out, "skipped: %s\n", s); err != nil {
			return err
		}
	}
	vs = shown(vs, cms)
	for i, v := range vs {
		st, err := reg.Presenter(v.ViewName(), v.Style())
		if err != nil {
			return err
		}
		vs[i] = v.WithStats(st)
	}
	c, err := collector.New(db, cfg.CollectInterval, n, cms)
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
				lines, err := v.Render(ctx, c, n, cfg.Width)
				if err != nil {
					if ctx.Err() != nil {
						break
					}
					return err
				}
				b.WriteString(strings.Join(lines, "\n") + "\n\n")
			}
			// ps-top's own metrics, not cut to the output width.
			b.WriteString("--- metrics registry ---\n")
			if err := reg.Report(&b, time.Now()); err != nil {
				return err
			}
			if _, err := io.WriteString(out, b.String()); err != nil {
				return err
			}
		}
	}
}

// usable chooses each metric's query for srv and returns the metrics it can
// collect, each with its entry in reg. A metric with no usable query, e.g. its
// table is missing, is left out and described in skipped. It fails only if no
// metric is usable.
func usable(srv *probe.Server, reg *registry.Registry) (cms []collector.Metric, skipped []string, err error) {
	for _, m := range metrics.All() {
		q, err := srv.Choose(m.Queries)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", m.Def.Name, err))
			continue
		}
		st, err := reg.Collector(m.Def.Name)
		if err != nil {
			return nil, nil, err
		}
		cms = append(cms, collector.Metric{
			Def:     m.Def,
			Collect: func(db *sql.DB) ([]snapshot.Entry, error) { return m.Collect(db, q.Query, st) },
			Stats:   st,
		})
	}
	if len(cms) == 0 {
		return nil, nil, fmt.Errorf("app: no metric is usable: %s", strings.Join(skipped, "; "))
	}
	return cms, skipped, nil
}

// shown returns the views of vs whose metric is in cms, in order.
func shown(vs []presenter.Renderer, cms []collector.Metric) []presenter.Renderer {
	var out []presenter.Renderer
	for _, v := range vs {
		for _, m := range cms {
			if m.Def.Name == v.Metric() {
				out = append(out, v)
				break
			}
		}
	}
	return out
}
