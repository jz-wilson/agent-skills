package main

import (
	"context"
	"runtime/debug"
	"time"
)

type CLI struct {
	Verbose bool   `short:"v" help:"Verbose logging to stderr."`
	Get     GetCmd `cmd:"" help:"GET a URL and print the body."`
}

type GetCmd struct {
	URL     string        `arg:"" help:"URL to fetch."`
	Timeout time.Duration `default:"10s" help:"Per-attempt timeout."`
	Tries   int           `default:"3" help:"Attempts for 5xx, 429 and network errors."`
}

func (c *GetCmd) Run(ctx context.Context, env *Env) error {
	client := newClient("fetch/"+buildVersion(), c.Timeout, c.Tries, env.Log)
	body, err := client.Get(ctx, c.URL)
	if err != nil {
		return err
	}
	_, err = env.Stdout.Write(body)
	return err
}

// buildVersion is the module version stamped by `go install`, else "devel".
func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}
