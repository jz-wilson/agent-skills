package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"time"

	"golang.org/x/mod/modfile"
)

type CLI struct {
	Verbose      bool          `short:"v" help:"Verbose logging to stderr."`
	JSON         bool          `help:"Print machine-readable JSON to stdout."`
	FailOutdated bool          `help:"Exit 1 when any dependency is outdated (for CI)."`
	Jobs         int           `short:"j" default:"8" help:"Proxy lookups in flight at once."`
	Proxy        string        `default:"https://proxy.golang.org" env:"MODFRESH_PROXY" help:"Module proxy base URL."`
	Timeout      time.Duration `default:"10s" help:"Per-attempt timeout."`
	Path         string        `arg:"" optional:"" default:"go.mod" help:"go.mod file to check."`
}

// Validate runs at parse time (a failure is a usage error, exit 2).
func (c *CLI) Validate() error {
	if c.Jobs < 1 {
		return errors.New("--jobs must be at least 1")
	}
	return nil
}

func (c *CLI) Run(ctx context.Context, env *Env) error {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return err // *PathError already names the operation and path
	}
	f, err := modfile.Parse(c.Path, data, nil)
	if err != nil {
		return err
	}
	var mods []Module
	for _, r := range f.Require {
		if !r.Indirect {
			mods = append(mods, Module{Path: r.Mod.Path, Version: r.Mod.Version})
		}
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Path < mods[j].Path })
	if len(mods) == 0 {
		env.UI.Info("%s has no direct dependencies", c.Path)
		return c.report(env, nil)
	}

	client := newClient("modfresh/"+buildVersion(), c.Timeout, 3, env.Log)
	env.UI.Step("Checking %d direct dependencies in %s", len(mods), c.Path)
	t := env.UI.NewTally("Querying "+c.Proxy, len(mods))
	rows, err := checkAll(ctx, client, c.Proxy, mods, c.Jobs, t.Inc)
	t.Done()
	if err != nil {
		return err
	}
	return c.report(env, rows)
}

func (c *CLI) report(env *Env, rows []Row) error {
	outdated, failed := summarize(rows)
	if c.JSON {
		if err := printJSON(env.Stdout, c.Path, rows, outdated, failed); err != nil {
			return err
		}
	} else {
		printRows(env.Stdout, rows)
	}
	switch {
	case failed > 0:
		env.UI.Error("%d of %d lookups failed", failed, len(rows))
	case outdated > 0:
		env.UI.Warn("%d of %d dependencies are outdated", outdated, len(rows))
	default:
		env.UI.OK("All %d dependencies are up to date", len(rows))
	}
	if failed > 0 {
		return fmt.Errorf("%d lookups failed", failed)
	}
	if outdated > 0 && c.FailOutdated {
		return fmt.Errorf("%d dependencies are outdated", outdated)
	}
	return nil
}

// buildVersion is the module version stamped by `go install`, else "devel".
func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}
