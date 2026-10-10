package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
	"golang.org/x/sync/errgroup"
)

type CLI struct {
	Verbose bool    `short:"v" help:"Verbose logging to stderr."`
	Sums    SumsCmd `cmd:"" help:"SHA-256 of files, hashed concurrently."`
}

type SumsCmd struct {
	Jobs  int      `short:"j" default:"4" help:"Files hashed at once."`
	Files []string `arg:"" help:"Files to hash."`
}

// Validate runs at parse time (a failure is a usage error, exit 2). SetLimit(0)
// would block forever, so reject it here.
func (c *SumsCmd) Validate() error {
	if c.Jobs < 1 {
		return errors.New("--jobs must be at least 1")
	}
	return nil
}

func (c *SumsCmd) Run(ctx context.Context, env *Env) error {
	t := newTally(env.UI, "Hashing", len(c.Files))
	sums, err := hashAll(ctx, c.Files, c.Jobs, t.inc)
	t.clear()
	if err != nil {
		return err
	}
	env.UI.OK("Hashed %d files", len(sums))
	for i, f := range c.Files { // input order, whatever order they finished in
		fmt.Fprintf(env.Stdout, "%s  %s\n", sums[i], f)
	}
	return nil
}

// hashAll hashes paths with at most jobs running at once. The first error cancels
// the rest. done is called from the worker goroutines, so it must be safe for that.
func hashAll(ctx context.Context, paths []string, jobs int, done func()) ([]string, error) {
	sums := make([]string, len(paths)) // worker i writes only sums[i]: no lock, no race
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(jobs) // g.Go blocks while jobs are running
	for i, p := range paths {
		if gctx.Err() != nil {
			break // a worker failed or Ctrl-C: stop scheduling
		}
		g.Go(func() error {
			s, err := hashFile(gctx, p)
			if err != nil {
				return err
			}
			sums[i] = s
			done()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return sums, ctx.Err() // the parent: gctx is always cancelled once Wait returns
}

func hashFile(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // read-only: a close error loses no data
	h := sha256.New()
	buf := make([]byte, 1<<20)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := f.Read(buf)
		h.Write(buf[:n])
		if errors.Is(err, io.EOF) {
			return hex.EncodeToString(h.Sum(nil)), nil
		}
		if err != nil {
			return "", err
		}
	}
}

// tally counts finished items and redraws one progress line. Safe for concurrent
// use; only a terminal sees the bar, the caller prints the final ok line.
type tally struct {
	mu          sync.Mutex
	ui          *UI
	title       string
	done, total int
	bar         progress.Model
}

func newTally(ui *UI, title string, total int) *tally {
	return &tally{ui: ui, title: title, total: total, bar: progress.New(progress.WithDefaultBlend(), progress.WithWidth(30))}
}

func (t *tally) inc() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.done++
	if t.ui.interactive {
		lipgloss.Fprint(t.ui.w, "\r"+t.title+" "+t.bar.ViewAs(float64(t.done)/float64(t.total)))
	}
}

func (t *tally) clear() {
	if t.ui.interactive {
		fmt.Fprint(t.ui.w, "\r\033[K")
	}
}
