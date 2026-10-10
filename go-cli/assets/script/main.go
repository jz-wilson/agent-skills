// Command countlines counts non-empty lines in the files given as arguments.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("countlines", flag.ContinueOnError)
	fs.SetOutput(stderr)
	verbose := fs.Bool("v", false, "verbose logging")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: countlines [-v] FILE...")
		return 2
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level}))

	for _, path := range fs.Args() {
		if err := ctx.Err(); err != nil {
			fmt.Fprintln(stderr, "countlines: interrupted")
			return 1
		}
		log.Debug("counting", "path", path)
		n, err := countLines(path)
		if err != nil {
			fmt.Fprintf(stderr, "countlines: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%d\t%s\n", n, path)
	}
	return 0
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err // *os.PathError already names the op and path
	}
	defer f.Close() //nolint:errcheck // read-only: a close error loses no data

	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			n++
		}
	}
	if err := sc.Err(); err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	return n, nil
}
