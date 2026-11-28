// Package app wires the program: it probes the server, builds the metrics,
// starts the collector, and presents views on its own interval.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // driver for sql.Open("mysql", ...)

	"github.com/sjmudd/ps-top/v2/internal/collector"
	"github.com/sjmudd/ps-top/v2/internal/logging"
	"github.com/sjmudd/ps-top/v2/internal/mapper"
	"github.com/sjmudd/ps-top/v2/internal/metric"
	"github.com/sjmudd/ps-top/v2/internal/metrics"
	"github.com/sjmudd/ps-top/v2/internal/presenter"
	"github.com/sjmudd/ps-top/v2/internal/probe"
	"github.com/sjmudd/ps-top/v2/internal/registry"
	"github.com/sjmudd/ps-top/v2/internal/setupinstruments"
	"github.com/sjmudd/ps-top/v2/internal/snapshot"
	"github.com/sjmudd/ps-top/v2/internal/tui"
)

// Config holds the settings the user can change.
type Config struct {
	DSN             string
	CollectInterval time.Duration
	PresentInterval time.Duration
	Window          time.Duration // difference over the CollectIntervals that fit in this, at least 1
	Width           int           // lines are cut at Width runes; 0: not cut
	Views           []ViewChoice  // views to show, in order; none: every usable view in its default style
	Count           int           // output blocks to write, the first included; 0: until ctx is done
	Limit           int           // at most Limit rows per view; 0: no limit
	TotalsOnly      bool          // each view's title, headings and totals line only
	Anonymise       bool          // show schema, table and user names anonymised, e.g. schema1.table2
	DatabaseFilter  []string      // show only these schemas; none: all
	// Munge renames file I/O's "schema.table" names, e.g. by ~/.pstoprc's
	// [munge] rules (internal/munge). nil: no renaming.
	Munge func(string) string
	// TUI shows the views full screen, one at a time (internal/tui), rather
	// than printing them all as text.
	TUI         bool
	Program     string        // the program's name, as run, for the TUI's top line
	Version     string        // ps-top's version, for the TUI's top line
	MessageTime time.Duration // how long the TUI shows a message
}

// ParseDatabaseFilter parses a list of schema names as the command line
// gives it, as v1 does: names separated by ",", each trimmed of
// spaces. Empty names and names containing a space are left out, e.g.
// " shop, crm,,a b" gives shop and crm.
func ParseDatabaseFilter(s string) []string {
	var names []string
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if name != "" && !strings.Contains(name, " ") {
			names = append(names, name)
		}
	}
	return names
}

// ViewChoice is a view to show, by name, and its style: "current",
// "difference", or "" for the view's default.
type ViewChoice struct {
	Name, Style string
}

// ParseViews parses a list of views as the command line gives it: names
// separated by ",", each with an optional ":style", e.g.
// "table_io_latency:current,file_io_latency". Names and styles are checked
// later, against the views.
func ParseViews(s string) ([]ViewChoice, error) {
	if s == "" {
		return nil, nil
	}
	var out []ViewChoice
	for _, part := range strings.Split(s, ",") {
		name, style, _ := strings.Cut(part, ":")
		if name == "" {
			return nil, fmt.Errorf("app: view list %q has an empty view name", s)
		}
		out = append(out, ViewChoice{Name: name, Style: style})
	}
	return out, nil
}

// Defaults are the settings used when the user changes none.
var Defaults = Config{
	CollectInterval: time.Second,
	PresentInterval: 2 * time.Second,
	Window:          10 * time.Second,
}

// restoreTimeout limits putting back the instrument settings at exit.
const restoreTimeout = 10 * time.Second

// fileMapVariables are the server settings file name mapping needs.
var fileMapVariables = []string{
	"datadir", "log_bin_basename",
	"relay_log_basename", "relay_log", "pid_file",
	"master_info_file", "relay_log_info_file", // MariaDB only
}

// allViews returns every view, in the order presented, with file I/O
// mapping file names with fileMap. Names and styles do not depend on fileMap,
// so a nil fileMap serves to check a view list before connecting.
func allViews(fileMap func(string) string) []presenter.Renderer {
	return []presenter.Renderer{
		presenter.TableIOLatency,
		presenter.TableIOOps,
		presenter.FileIOLatency(fileMap),
		presenter.TableLockLatency,
		presenter.UserLatency,
		presenter.MutexLatency,
		presenter.StagesLatency,
		presenter.MemoryUsage,
		presenter.PsTopStages,
		presenter.PsTopRenders,
		presenter.PsTopProcess,
		presenter.PsTop,
	}
}

// forTUI returns the views of vs the TUI shows: ps-top's own metrics as one
// screen (presenter.PsTop), not as the printer's three views. With tui
// false, the views the printer shows: the three, not PsTop.
func forTUI(vs []presenter.Renderer, tui bool) []presenter.Renderer {
	three := []string{presenter.PsTopStages.Name, presenter.PsTopRenders.Name, presenter.PsTopProcess.Name}
	var out []presenter.Renderer
	for _, v := range vs {
		n := v.ViewName()
		if tui && slices.Contains(three, n) || !tui && n == presenter.PsTop.Name {
			continue
		}
		out = append(out, v)
	}
	return out
}

// views returns every view, with file I/O mapping file names with the
// server's settings, so they are made after the probe, and renaming table
// names with munge (nil: none).
func views(srv *probe.Server, munge func(string) string) ([]presenter.Renderer, error) {
	m, err := mapper.NewFileName(mapper.FileNameSettings{
		DataDir:          srv.Variables["datadir"],
		BinlogBase:       srv.Variables["log_bin_basename"],
		RelayLogBase:     srv.Variables["relay_log_basename"],
		RelayLog:         srv.Variables["relay_log"],
		PidFile:          srv.Variables["pid_file"],
		MasterInfoFile:   srv.Variables["master_info_file"],
		RelayLogInfoFile: srv.Variables["relay_log_info_file"],
		Munge:            munge,
	})
	if err != nil {
		return nil, err
	}
	return allViews(m.Map), nil
}

// choose returns the views to show, in order: those of choices, each in its
// style, or every view of available in its default style if choices is
// empty. A name not in all, a style its metric does not accept, a view and
// style given twice, or a view not in available (its metric is skipped) is
// an error. Passing all as available checks choices without a server.
func choose(all, available []presenter.Renderer, choices []ViewChoice) ([]presenter.Renderer, error) {
	if len(choices) == 0 {
		return available, nil
	}
	var names []string
	for _, v := range all {
		names = append(names, v.ViewName())
	}
	var out []presenter.Renderer
	seen := map[ViewChoice]bool{}
	for _, c := range choices {
		i := slices.Index(names, c.Name)
		if i < 0 {
			return nil, fmt.Errorf("app: no view %q, views are %s", c.Name, strings.Join(names, ", "))
		}
		j := slices.IndexFunc(available, func(a presenter.Renderer) bool { return a.ViewName() == c.Name })
		if j < 0 {
			return nil, fmt.Errorf("app: view %s cannot be shown: its metric %s is skipped", c.Name, all[i].Metric())
		}
		v := available[j]
		if c.Style != "" {
			t, err := metric.ParseType(c.Style)
			if err != nil {
				return nil, fmt.Errorf("app: view %s: %w", c.Name, err)
			}
			if v, err = v.WithStyle(t); err != nil {
				return nil, err
			}
		}
		key := ViewChoice{Name: c.Name, Style: v.Style()}
		if seen[key] {
			return nil, fmt.Errorf("app: view %s:%s given twice", key.Name, key.Style)
		}
		seen[key] = true
		out = append(out, v)
	}
	return out, nil
}

// firstViews returns the views of the first output block, as vmstat's
// first line: each view whose metric accepts current values shows values up
// to now, the others as they are.
func firstViews(vs []presenter.Renderer) []presenter.Renderer {
	out := make([]presenter.Renderer, len(vs))
	for i, v := range vs {
		out[i] = v
		if v.Accepts(metric.CurrentValue) {
			if cv, err := v.WithStyle(metric.CurrentValue); err == nil {
				out[i] = cv
			}
		}
	}
	return out
}

// Run connects, probes, starts the collector and writes every view to out
// each cfg.PresentInterval until ctx is done (returns nil) or something fails.
func Run(ctx context.Context, cfg Config, out io.Writer) error {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	return run(ctx, cfg, db, out)
}

// run is Run on an open db.
func run(ctx context.Context, cfg Config, db *sql.DB, out io.Writer) (err error) {
	reg := registry.New(time.Now())
	if cfg.CollectInterval <= 0 || cfg.PresentInterval <= 0 {
		return fmt.Errorf("app: intervals must be more than 0: collect %v, present %v", cfg.CollectInterval, cfg.PresentInterval)
	}
	if cfg.Window < cfg.CollectInterval {
		return fmt.Errorf("app: window %v must be at least the collect interval %v", cfg.Window, cfg.CollectInterval)
	}
	// The window spans n collect intervals, those that fit in it, rounded
	// down: from the snapshot n before the latest to the latest, e.g. a 10s
	// window at a 3s collect interval is 3, 9s. The collector sizes each ring
	// from n.
	n := int(cfg.Window / cfg.CollectInterval)
	if cfg.Count < 0 || cfg.Limit < 0 || cfg.Width < 0 {
		return fmt.Errorf("app: count %d, limit %d and width %d must not be negative", cfg.Count, cfg.Limit, cfg.Width)
	}
	// Check the view list before connecting: names and styles do not depend
	// on the server.
	if _, err := choose(allViews(nil), allViews(nil), cfg.Views); err != nil {
		return err
	}

	req := metrics.Request()
	req.Variables = append(req.Variables, fileMapVariables...)
	req.Variables = append(req.Variables, serverVariables...)
	req.Status = append(req.Status, serverStatus...)
	req.Updates = append(req.Updates, probe.Update{Table: setupinstruments.Table, Column: "enabled"})
	srv, err := probe.Run(db, req)
	probed := time.Now()
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
	vs, err := views(srv, cfg.Munge)
	if err != nil {
		return err
	}
	cms, skipped, err := usable(srv, reg, cfg.DatabaseFilter)
	if err != nil {
		return err
	}
	own, err := registryMetrics(reg)
	if err != nil {
		return err
	}
	cms = append(cms, own...) // last: each collection includes its own stages
	if !cfg.TUI {
		if _, err := fmt.Fprintf(out, "%v %s\n", srv.Flavour, srv.Version); err != nil {
			return err
		}
		for _, s := range skipped {
			if _, err := fmt.Fprintf(out, "skipped: %s\n", s); err != nil {
				return err
			}
		}
	}
	vs = forTUI(vs, cfg.TUI)
	vs, err = choose(vs, shown(vs, cms), cfg.Views)
	if err != nil {
		return err
	}
	names := make([]string, len(vs))
	for i, v := range vs {
		names[i] = v.ViewName() + ":" + v.Style()
	}
	slog.Info("chose views to show", "views", strings.Join(names, ","))
	first := firstViews(vs)
	// One registry entry per view and style, shared by the first block and
	// the later ones where they show the same style.
	entries := map[ViewChoice]*registry.Presenter{}
	withStats := func(v presenter.Renderer) (presenter.Renderer, error) {
		key := ViewChoice{Name: v.ViewName(), Style: v.Style()}
		if entries[key] == nil {
			st, err := reg.Presenter(key.Name, key.Style)
			if err != nil {
				return nil, err
			}
			entries[key] = st
		}
		return v.WithStats(entries[key]), nil
	}
	for i := range vs {
		if vs[i], err = withStats(vs[i]); err != nil {
			return err
		}
		if first[i], err = withStats(first[i]); err != nil {
			return err
		}
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
		cancel() // stops the output: the collector has stopped
		errc <- err
	}()
	// The first output waits for the first collection, at most 3 collect
	// intervals: later requests are answered from what is held.
	if err := c.WaitForData(ctx); err != nil {
		cancel()
		if cerr := <-errc; cerr != nil {
			return cerr // the collector failed before any data
		}
		if errors.Is(err, context.Canceled) {
			return nil // stopped before any data
		}
		return err
	}

	opts := presenter.Options{N: n, Width: cfg.Width, Limit: cfg.Limit, TotalsOnly: cfg.TotalsOnly, DatabaseFilter: cfg.DatabaseFilter}
	if cfg.Anonymise {
		opts.Anonymiser = presenter.NewAnonymiser()
	}
	if cfg.TUI {
		info, err := serverInfo(srv, probed)
		if err != nil {
			return err
		}
		var message string
		if len(skipped) > 0 {
			message = "skipped: " + strings.Join(skipped, "; ")
		}
		anon := opts.Anonymiser // the a key turns anonymising on: one Anonymiser for the run
		if anon == nil {
			anon = presenter.NewAnonymiser()
		}
		terr := tui.Run(ctx, tui.Config{
			Program: cfg.Program, Version: cfg.Version, Server: info,
			Views: vs, First: first, Source: c, Control: c, Options: opts, Anonymiser: anon,
			CollectInterval: cfg.CollectInterval, Interval: cfg.PresentInterval,
			MessageTime: cfg.MessageTime, Message: message,
		})
		cancel()
		if cerr := <-errc; cerr != nil {
			return cerr // the collector failed: the TUI stopped because of it
		}
		return terr
	}
	start := time.Now()
	// block writes one output block of views. The first is written at once:
	// the first collection is done.
	block := func(views []presenter.Renderer) error {
		var b strings.Builder
		fmt.Fprintf(&b, "--- %.0fs ---\n", time.Since(start).Seconds()) // whole seconds: intervals are
		for _, v := range views {
			lines, err := v.Render(ctx, c, opts)
			if err != nil {
				if ctx.Err() != nil {
					break
				}
				return err
			}
			b.WriteString(strings.Join(lines, "\n") + "\n\n")
		}
		_, err := io.WriteString(out, b.String())
		return err
	}
	if err := block(first); err != nil {
		return err
	}
	blocks := 1
	tick := time.NewTicker(cfg.PresentInterval)
	defer tick.Stop()
	for cfg.Count == 0 || blocks < cfg.Count {
		select {
		case err := <-errc:
			return err
		case <-tick.C:
			if err := block(vs); err != nil {
				return err
			}
			blocks++
		}
	}
	return nil
}

// usable chooses each metric's query for srv and returns the metrics it can
// collect, each with its entry in reg, reading only the rows of
// filteredSchemas (nil: no filter) where its query can be limited to them
// (see metrics.Metric.Collect). A metric with no usable query, e.g. its
// table is missing, is left out and described in skipped. It fails only if
// no metric is usable.
func usable(srv *probe.Server, reg *registry.Registry, filteredSchemas []string) (cms []collector.Metric, skipped []string, err error) {
	for _, m := range metrics.All() {
		q, err := srv.Choose(m.Queries)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", m.Def.Name, err))
			slog.Warn("skipped metric: no usable query", logging.KeyMetric, m.Def.Name, logging.KeyErr, err)
			continue
		}
		slog.Info("chose query for metric", logging.KeyMetric, m.Def.Name, "candidate", q.Name)
		st, err := reg.Collector(m.Def.Name)
		if err != nil {
			return nil, nil, err
		}
		cms = append(cms, collector.Metric{
			Def:     m.Def,
			Collect: func(db *sql.DB) ([]snapshot.Entry, error) { return m.Collect(db, q.Query, filteredSchemas, st) },
			Stats:   st,
		})
	}
	if len(cms) == 0 {
		return nil, nil, fmt.Errorf("app: no metric is usable: %s", strings.Join(skipped, "; "))
	}
	return cms, skipped, nil
}

// registryMetrics returns the metrics registry as two metrics, ps-top's own
// stages and its process figures (see registry.StagesMetric), each with an
// entry of its own in reg. They read reg, not the database.
func registryMetrics(reg *registry.Registry) ([]collector.Metric, error) {
	var cms []collector.Metric
	for _, m := range []struct {
		def  metric.Definition
		read func(*registry.Collector) []snapshot.Entry
	}{
		{registry.StagesDefinition(), reg.Stages},
		{registry.ProcessDefinition(), reg.Process},
	} {
		st, err := reg.Collector(m.def.Name)
		if err != nil {
			return nil, err
		}
		cms = append(cms, collector.Metric{
			Def:     m.def,
			Collect: func(*sql.DB) ([]snapshot.Entry, error) { return m.read(st), nil },
			Stats:   st,
		})
	}
	return cms, nil
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
