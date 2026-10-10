// Command pick lets the user choose one of its arguments and prints the choice to stdout.
// The TUI draws on stderr, so `choice=$(pick a b c)` works like fzf.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	"golang.org/x/term"
)

type CLI struct {
	Items   []string `arg:"" help:"Items to choose from."`
	Index   int      `help:"Pick this index without the TUI (non-interactive)." default:"-1"`
	NoTUI   bool     `name:"no-tui" help:"Never start the TUI."`
	Confirm bool     `help:"Ask for confirmation before printing."`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("pick"), kong.Writers(stdout, stderr), kong.UsageOnError())
	if err != nil {
		fmt.Fprintf(stderr, "pick: %v\n", err)
		return 1
	}
	if _, err := parser.Parse(args); err != nil {
		fmt.Fprintf(stderr, "pick: %v\n", err)
		return 2
	}

	choice, err := choose(ctx, cli, stdin, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "pick: %v\n", err)
		return 1
	}
	if choice == "" {
		return 1 // user cancelled
	}
	if cli.Confirm && interactive(stdin, stderr) {
		ok, err := confirm(fmt.Sprintf("Use %q?", choice))
		if err != nil || !ok {
			return 1
		}
	}
	fmt.Fprintln(stdout, choice)
	return 0
}

// choose returns the picked item; the TUI is only one way to get it.
func choose(ctx context.Context, cli CLI, stdin io.Reader, tty io.Writer) (string, error) {
	if cli.Index >= 0 {
		if cli.Index >= len(cli.Items) {
			return "", fmt.Errorf("--index %d out of range (have %d items)", cli.Index, len(cli.Items))
		}
		return cli.Items[cli.Index], nil
	}
	if cli.NoTUI || !interactive(stdin, tty) {
		return "", errors.New("not a terminal: pass --index N")
	}
	p := tea.NewProgram(newModel(cli.Items), tea.WithContext(ctx), tea.WithInput(stdin), tea.WithOutput(tty))
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("tui: %w", err)
	}
	return final.(model).chosen, nil
}

// interactive reports whether both ends are terminals and nothing asked for plain mode.
func interactive(in io.Reader, out io.Writer) bool {
	if os.Getenv("CI") != "" {
		return false
	}
	return isTerminal(in) && isTerminal(out)
}

func isTerminal(v any) bool {
	f, ok := v.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
