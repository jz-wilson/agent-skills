package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

type CLI struct {
	Verbose bool     `short:"v" help:"Verbose logging to stderr."`
	Clean   CleanCmd `cmd:"" help:"Delete the *.tmp files in a directory."`
}

type CleanCmd struct {
	Dir    string `arg:"" help:"Directory to clean."`
	DryRun bool   `name:"dry-run" short:"n" help:"Print what would be removed; change nothing."`
	Yes    bool   `short:"y" help:"Do not ask for confirmation."`
}

func (c *CleanCmd) Run(env *Env) error {
	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		return err
	}
	var targets []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tmp") {
			targets = append(targets, filepath.Join(c.Dir, e.Name()))
		}
	}
	if len(targets) == 0 {
		env.UI.Info("Nothing to remove in %s", c.Dir)
		return nil
	}

	if c.DryRun { // checked first: --dry-run --yes must still change nothing
		for _, t := range targets {
			fmt.Fprintln(env.Stdout, "would remove", t) // the plan is data: stdout, plain
		}
		env.UI.Info("Dry run: %d files would be removed, nothing changed", len(targets))
		return nil
	}
	if !c.Yes {
		if !interactive(env.Stdin, env.Stderr) {
			// Missing required input is a usage error (2), same as a missing flag.
			return &exitError{code: 2, err: errors.New("refusing to delete without confirmation: pass --yes (or --dry-run to preview)")}
		}
		ok, err := confirm(env.Stderr, fmt.Sprintf("Remove %d files from %s?", len(targets), c.Dir))
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("aborted, nothing removed") // exit 1: the requested work was not done
		}
	}

	for _, t := range targets {
		if err := os.Remove(t); err != nil {
			return err
		}
		fmt.Fprintln(env.Stdout, "removed", t)
	}
	env.UI.OK("Removed %d files", len(targets))
	return nil
}

// interactive: a person can answer, i.e. both stdin and stderr are terminals.
func interactive(stdin io.Reader, stderr io.Writer) bool {
	f, ok := stdin.(*os.File)
	return ok && term.IsTerminal(int(f.Fd())) && isTerminal(stderr)
}

// confirm asks on stderr, keeping stdout free for data. Ctrl-C returns huh.ErrUserAborted.
func confirm(tty io.Writer, title string) (bool, error) {
	var ok bool
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&ok),
	)).WithOutput(tty).WithAccessible(os.Getenv("ACCESSIBLE") != "").Run()
	return ok, err
}
