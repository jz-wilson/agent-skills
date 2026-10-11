# Script tier

One `main.go`, standard library only, run with `go run .` (or `go run main.go`). A `go.mod` is still worth having (`go mod init example.com/name`) so `go vet` and editors behave, but its `require` block stays empty.

## Shape

- `flag.NewFlagSet(name, flag.ContinueOnError)` inside `run`, never the global `flag.Parse()`. That keeps `run` testable and lets you choose the exit codes.
- `fs.SetOutput(stderr)` so usage text goes to stderr.
- On `flag.ErrHelp`, exit 0. On any other parse error, exit 2; the FlagSet has already printed the reason and usage.
- Positional args come from `fs.Args()`. Print one usage line and exit 2 when they're missing.
- **Filter-style scripts** (no file arguments means read stdin, like `wc` or `grep`): add `stdin io.Reader` to `run`'s parameters and have `main` pass `os.Stdin`; tests pass a `strings.Reader`. Treat a lone `-` as stdin too. Missing args are then not a usage error.
- Log with `slog` to stderr; `-v` turns on Debug.

## Step up to the Tool tier when

- you want a second subcommand (`tool add`, `tool list`),
- you need any `go get`: a third-party dependency,
- someone else will install it (`go install`, a release, a teammate's machine).

Don't step up for more flags, more files in `package main`, or tests. Scripts can have a `main_test.go` that calls `run` with `bytes.Buffer`s.

## Verified example

Built and vetted with Go 1.27.2 on 2026-10-09.

→ Read [`assets/script/main.go`](../assets/script/main.go) (80 lines; builds, tests and lints clean).

## How it looks

Real output (WSL, Go 1.27.2). stdout and stderr are interleaved here; in a pipe only the `N<TAB>path` lines reach stdout.

```
$ countlines
usage: countlines [-v] FILE...                      # exit 2

$ countlines -v main.go go.mod
time=2026-10-09T13:12:21.752-04:00 level=DEBUG msg=counting path=main.go
73	main.go
time=2026-10-09T13:12:21.753-04:00 level=DEBUG msg=counting path=go.mod
2	go.mod

$ countlines nope.txt
countlines: open nope.txt: no such file or directory   # exit 1
```

A Script tier stays plain like this: no color, no spinner. Polish is for the Tool tier ([ux.md](ux.md)).

Things this example demonstrates on purpose:

- `run` returns an int and `main` only wires up signals and the real streams.
- `return 0, err` for `os.Open`: `*os.PathError` already reads `open x.txt: no such file…`, so wrapping it again would print the path twice.
- `ctx.Err()` checked in the loop, so Ctrl-C stops between files.
- Data (`count<TAB>path`) on stdout; usage, errors and debug logs on stderr.
