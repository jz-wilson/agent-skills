// Command portview shows listening TCP sockets and the processes that own them.
// On a terminal it runs a live TUI; piped, or with --no-tui or --json, it prints once.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

// version is set by goreleaser via -ldflags "-X main.version=...".
var version = ""

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	parser, cli, err := newParser(stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "portview: %v\n", err)
		return 1
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "portview: %v\n", err)
		return 2
	}
	g := &Globals{Log: newLogger(stderr, cli.Verbose), Stdin: stdin, Stdout: stdout}
	kctx.BindTo(ctx, (*context.Context)(nil))
	if err := kctx.Run(g); err != nil {
		fmt.Fprintf(stderr, "portview: %v\n", err)
		return 1
	}
	return 0
}

func buildVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 12 {
			return "devel+" + s.Value[:12]
		}
	}
	return "devel"
}
