package presenter

import (
	"context"
	"fmt"

	"github.com/sjmudd/ps-top/internal/metric"
	"github.com/sjmudd/ps-top/internal/registry"
)

// Combined is several views shown as one, one under the other, e.g. ps-top's
// own metrics on one TUI screen. All parts show the same style. Its renders
// are recorded once, in its own registry entry: the parts' renders are not
// recorded.
type Combined struct {
	Name  string // e.g. "ps_top"
	Title string // e.g. "ps-top internals"
	Parts []Renderer

	stats *registry.Presenter // set by WithStats
}

// ViewName returns the combined view's name.
func (c Combined) ViewName() string { return c.Name }

// Metric returns the first part's metric.
func (c Combined) Metric() string { return c.Parts[0].Metric() }

// Style returns the first part's style: all parts show the same.
func (c Combined) Style() string { return c.Parts[0].Style() }

// Accepts reports whether every part can be shown in style t.
func (c Combined) Accepts(t metric.MetricType) bool {
	for _, p := range c.Parts {
		if !p.Accepts(t) {
			return false
		}
	}
	return true
}

// WithStyle returns a copy of c with every part showing style t.
func (c Combined) WithStyle(t metric.MetricType) (Renderer, error) {
	if !c.Accepts(t) {
		return nil, fmt.Errorf("presenter: view %s cannot show style %s", c.Name, t.Name())
	}
	parts := make([]Renderer, len(c.Parts))
	for i, p := range c.Parts {
		var err error
		if parts[i], err = p.WithStyle(t); err != nil {
			return nil, err
		}
	}
	c.Parts = parts
	return c, nil
}

// WithStats returns a copy of c recording its renders in st.
func (c Combined) WithStats(st *registry.Presenter) Renderer {
	c.stats = st
	return c
}

// Render returns the combined view's lines (see Table), each cut at o.Width
// runes.
func (c Combined) Render(ctx context.Context, src Source, o Options) ([]string, error) {
	t, err := c.Table(ctx, src, o)
	if err != nil {
		return nil, err
	}
	return t.Lines(o.Width), nil
}

// Table returns each part's table as a part of one Table, titled c.Title.
// Its style is the first part's. It is waiting while every part is.
func (c Combined) Table(ctx context.Context, src Source, o Options) (Table, error) {
	st := c.stats
	if st == nil {
		st = new(registry.Presenter) // not in any registry
	}
	defer st.Render.Start().Stop()
	t := Table{Title: c.Title, Waiting: true}
	for _, p := range c.Parts {
		pt, err := p.Table(ctx, src, o)
		if err != nil {
			return Table{}, err
		}
		if !pt.Waiting {
			t.Waiting = false
			if t.Style == "" {
				t.Style = pt.Style
			}
		}
		t.Parts = append(t.Parts, pt)
	}
	return t, nil
}
