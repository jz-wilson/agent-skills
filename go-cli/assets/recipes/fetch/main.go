package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// Env is bound into every command's Run method.
type Env struct {
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	UI             *UI
	Log            *slog.Logger
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("fetch"), kong.Writers(stdout, stderr), kong.UsageOnError())
	if err != nil {
		fmt.Fprintf(stderr, "fetch: %v\n", err)
		return 1
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "fetch: %v\n", err)
		return 2
	}
	kctx.BindTo(ctx, (*context.Context)(nil))
	env := &Env{Stdin: stdin, Stdout: stdout, Stderr: stderr, UI: newUI(stderr), Log: newLogger(stderr, cli.Verbose)}
	if err := kctx.Run(env); err != nil {
		fmt.Fprintf(stderr, "fetch: %v\n", err)
		if ee, ok := errors.AsType[*exitError](err); ok {
			return ee.code
		}
		return 1
	}
	return 0
}
