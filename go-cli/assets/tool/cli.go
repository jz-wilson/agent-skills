package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/alecthomas/kong"
)

// CLI is the whole command-line surface. Kong builds parsing and --help from it.
type CLI struct {
	Verbose bool `short:"v" help:"Verbose logging to stderr."`

	Greet   GreetCmd   `cmd:"" help:"Greet someone."`
	Version VersionCmd `cmd:"" help:"Print the version."`
}

// Globals is bound into every command's Run method.
type Globals struct {
	Log    *slog.Logger
	Stdout io.Writer
}

type GreetCmd struct {
	Name  string `arg:"" help:"Who to greet."`
	Shout bool   `help:"Uppercase the greeting."`
	JSON  bool   `name:"json" help:"Print JSON instead of text."`
}

func (c *GreetCmd) Run(ctx context.Context, g *Globals) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	msg := "hello, " + c.Name
	if c.Shout {
		msg = strings.ToUpper(msg)
	}
	g.Log.Debug("greeting", "name", c.Name, "shout", c.Shout)
	if c.JSON {
		return json.NewEncoder(g.Stdout).Encode(map[string]string{"greeting": msg})
	}
	_, err := fmt.Fprintln(g.Stdout, msg)
	return err
}

type VersionCmd struct{}

func (VersionCmd) Run(g *Globals) error {
	_, err := fmt.Fprintln(g.Stdout, buildVersion())
	return err
}

func newParser(stdout, stderr io.Writer) (*kong.Kong, *CLI, error) {
	var cli CLI
	parser, err := kong.New(&cli,
		kong.Name("greeter"),
		kong.Description("A minimal Kong example."),
		kong.Writers(stdout, stderr),
		kong.UsageOnError(),
	)
	return parser, &cli, err
}

func newLogger(w io.Writer, verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}
