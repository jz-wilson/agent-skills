package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type CLI struct {
	Verbose bool    `short:"v" help:"Verbose logging to stderr."`
	Exec    ExecCmd `cmd:"" help:"Run a program (no shell) and propagate its exit code."`
}

type ExecCmd struct {
	Capture bool     `help:"Capture the child's stdout and print it on our stdout (default: stream everything to stderr)."`
	Argv    []string `arg:"" help:"Program and arguments, after --."`
}

func (c *ExecCmd) Run(ctx context.Context, env *Env) error {
	name := c.Argv[0]
	path, err := exec.LookPath(name)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%s is not installed (not found on PATH)", name)
		}
		return err
	}

	// Argv goes straight to the program: no shell, so nothing is re-parsed or injected.
	cmd := exec.CommandContext(ctx, path, c.Argv[1:]...) //nolint:gosec // running the user's program is the job; there is no shell
	cmd.Stdin = env.Stdin
	cmd.Cancel = func() error { return interrupt(cmd.Process) } // ask politely first
	cmd.WaitDelay = 5 * time.Second                             // then Kill, and close its pipes

	env.UI.Step("Running %s", strings.Join(c.Argv, " "))
	if c.Capture {
		out, err := cmd.Output() // Stderr is nil, so it lands in ExitError.Stderr
		if err != nil {
			return childError(ctx, name, err)
		}
		_, err = env.Stdout.Write(out)
		return err
	}
	// The child's output is not our data: stream it to stderr. Same writer for
	// both, so exec serializes the copies.
	cmd.Stdout, cmd.Stderr = env.Stderr, env.Stderr
	if err := cmd.Run(); err != nil {
		return childError(ctx, name, err)
	}
	return nil
}

// interrupt sends SIGINT; Windows cannot signal a process, so it kills.
func interrupt(p *os.Process) error {
	if runtime.GOOS == "windows" {
		return p.Kill()
	}
	return p.Signal(os.Interrupt)
}

// childError passes the child's exit status on as our own exit code.
func childError(ctx context.Context, name string, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("%s: interrupted", name)
	}
	ee, ok := errors.AsType[*exec.ExitError](err)
	if !ok {
		return fmt.Errorf("run %s: %w", name, err)
	}
	msg := fmt.Errorf("%s: %w", name, err) // "sh: exit status 3"
	if s := strings.TrimSpace(string(ee.Stderr)); s != "" {
		msg = fmt.Errorf("%w: %s", msg, s)
	}
	return &exitError{code: max(ee.ExitCode(), 1), err: msg} // -1 means killed by a signal
}
