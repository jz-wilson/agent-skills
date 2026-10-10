package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/alecthomas/kong"
	"golang.org/x/term"

	"github.com/jz-wilson/portview/internal/scan"
)

// CLI is the whole command-line surface. With one command, the flags and Run live on the root.
type CLI struct {
	NoTUI    bool          `name:"no-tui" help:"Print a TSV table once (port, address, pid, process) instead of starting the TUI."`
	JSON     bool          `name:"json" help:"Print JSON once instead of starting the TUI."`
	Interval time.Duration `help:"TUI refresh interval." default:"2s"`
	Verbose  bool          `short:"v" help:"Verbose logging to stderr."`
	Root     string        `help:"Proc filesystem root, for tests." default:"/proc" hidden:""`
	Version  kong.VersionFlag
}

// Globals is bound into Run.
type Globals struct {
	Log    *slog.Logger
	Stdin  io.Reader
	Stdout io.Writer
}

func (c *CLI) Run(ctx context.Context, g *Globals) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("reads /proc, so it only runs on Linux (this is %s)", runtime.GOOS)
	}
	if !c.JSON && !c.NoTUI && interactive(g.Stdin, g.Stdout) {
		p := tea.NewProgram(newModel(c.Root, c.Interval),
			tea.WithContext(ctx), tea.WithInput(g.Stdin), tea.WithOutput(g.Stdout))
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		return nil
	}
	socks, err := scan.Listening(c.Root)
	if err != nil {
		return err
	}
	g.Log.Debug("scanned", "listening", len(socks))
	if c.JSON {
		if socks == nil {
			socks = []scan.Socket{} // encode [] rather than null
		}
		return json.NewEncoder(g.Stdout).Encode(socks)
	}
	return writeTSV(g.Stdout, socks)
}

// writeTSV prints one row per socket; unknown owners show "-". No header, so it pipes into cut/sort.
func writeTSV(w io.Writer, socks []scan.Socket) error {
	for _, s := range socks {
		if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Port, s.Addr, pidText(s), processText(s)); err != nil {
			return err
		}
	}
	return nil
}

func pidText(s scan.Socket) string {
	if s.PID == 0 {
		return "-"
	}
	return strconv.Itoa(s.PID)
}

func processText(s scan.Socket) string {
	if s.Process == "" {
		return "-"
	}
	return s.Process
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

func newParser(stdout, stderr io.Writer) (*kong.Kong, *CLI, error) {
	var cli CLI
	parser, err := kong.New(&cli,
		kong.Name("portview"),
		kong.Description("Show listening TCP sockets and their processes (Linux)."),
		kong.Vars{"version": buildVersion()},
		kong.Writers(stdout, stderr),
		kong.UsageOnError(),
	)
	return parser, &cli, err
}
