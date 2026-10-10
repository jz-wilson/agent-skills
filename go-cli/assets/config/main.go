// Command greeter shows layered configuration (flags, env, config file, defaults) and shell completion.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"go.abhg.dev/komplete"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var cli CLI
	parser, err := kong.New(&cli,
		kong.Name("greeter"),
		kong.Description("A minimal configuration and completion example."),
		kong.Writers(stdout, stderr),
		kong.UsageOnError(),
		// Later paths win over earlier ones; a missing file is skipped, a malformed one is an error.
		kong.Configuration(kong.JSON, defaultConfigPaths()...),
	)
	if err != nil {
		fmt.Fprintf(stderr, "greeter: %v\n", err)
		return 1
	}
	// No-op unless the shell is asking for completions (COMP_LINE is set); then it prints and exits.
	komplete.Run(parser)

	kctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "greeter: %v\n", err)
		return 2
	}
	kctx.BindTo(ctx, (*context.Context)(nil))
	if err := kctx.Run(&Globals{Stdout: stdout}); err != nil {
		fmt.Fprintf(stderr, "greeter: %v\n", err)
		return 1
	}
	return 0
}
