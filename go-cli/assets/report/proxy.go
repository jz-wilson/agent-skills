package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

const (
	statusCurrent = "up to date"
	statusPatch   = "patch behind"
	statusMinor   = "minor behind"
	statusMajor   = "major behind"
	statusError   = "error"
)

type Module struct{ Path, Version string }

// Row is one dependency's result, in both the table and the JSON output.
type Row struct {
	Module  string `json:"module"`
	Current string `json:"current"`
	Latest  string `json:"latest,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// latestURL builds the proxy's @latest URL. The path is case-escaped per the
// proxy protocol (uppercase letter -> "!" + lowercase).
func latestURL(proxy, path string) (string, error) {
	esc, err := module.EscapePath(path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(proxy, "/") + "/" + esc + "/@latest", nil
}

func fetchLatest(ctx context.Context, c *Client, proxy, path string) (string, error) {
	url, err := latestURL(proxy, path)
	if err != nil {
		return "", err
	}
	body, err := c.Get(ctx, url)
	if err != nil {
		return "", err
	}
	var info struct{ Version string }
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("decode %s: %w", url, err)
	}
	if !semver.IsValid(info.Version) {
		return "", fmt.Errorf("proxy returned invalid version %q for %s", info.Version, path)
	}
	return info.Version, nil
}

// classify compares versions. The proxy's @latest only covers the module's own
// major-version path, so a newer /vN path is never reported (by design).
func classify(current, latest string) string {
	switch {
	case semver.Compare(current, latest) >= 0:
		return statusCurrent
	case semver.Major(current) != semver.Major(latest):
		return statusMajor
	case semver.MajorMinor(current) != semver.MajorMinor(latest):
		return statusMinor
	default:
		return statusPatch
	}
}

// checkAll looks up every module with at most jobs requests in flight. A failed
// lookup becomes an error row so one bad module does not hide the rest; only
// cancellation aborts. done is called from worker goroutines.
func checkAll(ctx context.Context, c *Client, proxy string, mods []Module, jobs int, done func()) ([]Row, error) {
	rows := make([]Row, len(mods)) // worker i writes only rows[i]: no lock, no race
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(jobs)
	for i, m := range mods {
		if gctx.Err() != nil {
			break
		}
		g.Go(func() error {
			defer done()
			rows[i] = Row{Module: m.Path, Current: m.Version}
			latest, err := fetchLatest(gctx, c, proxy, m.Path)
			if err != nil {
				rows[i].Status, rows[i].Error = statusError, err.Error()
				return nil
			}
			rows[i].Latest, rows[i].Status = latest, classify(m.Version, latest)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return rows, ctx.Err()
}

func summarize(rows []Row) (outdated, failed int) {
	for _, r := range rows {
		switch r.Status {
		case statusCurrent:
		case statusError:
			failed++
		default:
			outdated++
		}
	}
	return outdated, failed
}
