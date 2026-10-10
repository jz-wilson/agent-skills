// Command tour shows every polish component once: status lines, spinner,
// progress bar, table, tree, box, and a pretty slog handler.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"charm.land/lipgloss/v2/tree"
	"golang.org/x/term"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, stdout, stderr io.Writer) int {
	ui := newUI(stderr)
	log := newLogger(stderr, os.Getenv("VERBOSE") != "")

	ui.Step("Resolving config")
	ui.Info("Using profile %q", "default")
	log.Info("loaded config", "path", "~/.config/tour.toml", "profiles", 3)
	log.Debug("cache", "dir", "~/.cache/tour", "hit", true)

	var releases []release
	err := ui.Spin(ctx, "Fetching releases…", func(ctx context.Context) error {
		var err error
		releases, err = fetchReleases(ctx)
		return err
	})
	if err != nil {
		ui.Error("fetch releases: %v", err)
		return 1
	}
	ui.OK("Fetched %d releases", len(releases))

	ui.Progress(ctx, "Downloading", 20, func(int) { time.Sleep(40 * time.Millisecond) })
	ui.Warn("2 assets have no checksum")
	ui.Error("mirror eu-west unreachable, skipped")

	printReleases(stdout, releases)
	fmt.Fprintln(stderr)
	lipgloss.Fprintln(stderr, layoutTree())
	fmt.Fprintln(stderr)
	ui.Box("Next steps", "tour install v2.1.0\ntour verify --all")
	return 0
}

type release struct{ Tag, Date, Assets string }

func fetchReleases(ctx context.Context) ([]release, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(1500 * time.Millisecond):
	}
	return []release{
		{"v2.1.0", "2026-10-08", "12"},
		{"v2.0.6", "2026-08-11", "12"},
		{"v2.0.0", "2026-02-24", "10"},
	}, nil
}

// printReleases is data, so it goes to stdout: a styled table for a human, TSV for a pipe.
func printReleases(w io.Writer, rs []release) {
	if !isTerminal(w) {
		for _, r := range rs {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.Tag, r.Date, r.Assets)
		}
		return
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1)
	cell := lipgloss.NewStyle().Padding(0, 1)
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("TAG", "DATE", "ASSETS").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			return cell
		})
	for _, r := range rs {
		t.Row(r.Tag, r.Date, r.Assets)
	}
	lipgloss.Fprintln(w, t)
}

func layoutTree() *tree.Tree {
	return tree.Root(".").
		Child("main.go", "cli.go", tree.Root("internal").Child("fetch", "verify"), "testdata").
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240")))
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd())) && os.Getenv("CI") == ""
}
