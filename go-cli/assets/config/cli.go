package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"go.abhg.dev/komplete"
)

// CLI is the whole command-line surface.
// Precedence, highest first: flag, environment variable, config file, default tag.
type CLI struct {
	Config string `short:"c" placeholder:"PATH" help:"Config file (JSON) read after the default one, so it wins over it."`

	Greet      GreetCmd         `cmd:"" help:"Greet someone."`
	Completion komplete.Command `cmd:"" help:"Print a shell completion script."`
}

// BeforeResolve runs after parsing, before Kong fills flags nobody set on the command line.
// Kong's own order is flag > config file > env var > default, because env vars are applied
// as defaults. Resolvers run in order and the last non-nil answer wins, so loading --config
// and then the env vars here makes it flag > env var > config file > default.
// Flag values are not copied into the struct yet, so --config is read from the parse context.
func (*CLI) BeforeResolve(k *kong.Kong, ctx *kong.Context) error {
	for _, p := range ctx.Path {
		if p.Flag == nil || p.Flag.Name != "config" {
			continue
		}
		r, err := k.LoadConfig(ctx.FlagValue(p.Flag).(string))
		if err != nil {
			return err
		}
		ctx.AddResolver(r)
	}
	ctx.AddResolver(kong.ResolverFunc(func(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (any, error) {
		for _, name := range flag.Envs {
			if v := os.Getenv(name); v != "" {
				return v, nil
			}
		}
		return nil, nil
	}))
	return nil
}

// Globals is bound into every command's Run method.
type Globals struct {
	Stdout io.Writer
}

type GreetCmd struct {
	Name     string `arg:"" help:"Who to greet."`
	Greeting string `env:"GREETER_GREETING" default:"hello" help:"Word to greet with."`
	Shout    bool   `env:"GREETER_SHOUT" help:"Uppercase the greeting."`
}

func (c *GreetCmd) Run(ctx context.Context, g *Globals) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	msg := c.Greeting + ", " + c.Name
	if c.Shout {
		msg = strings.ToUpper(msg)
	}
	_, err := fmt.Fprintln(g.Stdout, msg)
	return err
}

// defaultConfigPaths is <user config dir>/greeter/config.json: $XDG_CONFIG_HOME or ~/.config on
// Linux, %AppData% on Windows, ~/Library/Application Support on macOS.
// With no resolvable config dir (no $HOME) there is simply no config file.
func defaultConfigPaths() []string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(dir, "greeter", "config.json")}
}
