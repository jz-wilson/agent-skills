---
name: go-cli
description: Write Go scripts, command-line tools, and terminal UIs the current way (Go 1.27, Kong, log/slog, testscript, golangci-lint v2, goreleaser v2, Bubble Tea/Lip Gloss/Bubbles/huh v2 under charm.land), including a semantic color scheme with emoji (NO_COLOR- and pipe-aware), spinners, progress bars, tables, trees, config files and env vars, shell completion, subprocesses, HTTP retries, and bounded concurrency. Use when the user wants to write, scaffold, extend, polish, fix, lint, test, or release a Go script, CLI, or TUI — "write a Go script that…", "make a Go CLI", "add a subcommand", "add a spinner / progress bar", "make the output prettier", "add shell completion", "read a config file", "my Go CLI prints ANSI codes when piped", "build a TUI in Go", "set up golangci-lint / goreleaser" — or names kong, bubbletea, lipgloss, bubbles, huh, testscript, teatest, or go-task in a Go context. Also use it for any Bubble Tea, Lip Gloss, Bubbles or huh v1 → v2 port, upgrade or migration ("port this bubbletea v1 model to v2"), even before the code is shared: v2 moved to charm.land and changed View, key messages and styles, and most examples online are still v1.
---

# Go CLI

Three tiers. Pick the lowest one that fits, and step up only on a named trigger.

| Tier | When | Stack | Checks | Reference |
|---|---|---|---|---|
| **Script** | One job, one person, run with `go run` | stdlib only: `flag`, `log/slog`, `os` | `gofmt`, `go vet` | [references/script.md](references/script.md) |
| **Tool** | A second subcommand, **or** any third-party dependency, **or** someone else installs it | Kong, `internal/` as needed | golangci-lint v2, testscript, go-task at 4+ tasks, goreleaser when shipping binaries | [references/tool.md](references/tool.md) |
| **TUI** | Long-lived, stateful screens: live lists, viewports, panes | Tool tier + Bubble Tea v2, Lip Gloss v2, Bubbles v2 | Tool checks + pure `Update` tests + teatest v2 | [references/tui.md](references/tui.md) |

A prompt or a spinner is not a TUI.

- **Color scheme and emoji (🔵 info blue, ✅ ok green, 🟡 warn yellow, ❌ error red), spinners, progress bars, tables, trees, boxes, themed logs:** [references/ux.md](references/ux.md). Use them in any Tool-tier command that takes more than a moment or prints more than a line. It shows exactly how the output should look, on a terminal and piped.
- **Prompts in a Tool-tier CLI:** `huh` standalone (tui.md, "Prompts without a TUI").
- **Running other programs, HTTP calls, parallel work, delete/overwrite with `--dry-run`/`--yes`:** [references/recipes.md](references/recipes.md).
- **Config file + env vars (flag > env > file > default) and shell completion:** [references/config.md](references/config.md).

Full compiling source for every example is in `assets/`. Copy from there.

| Asset | What it is |
|---|---|
| `script` | stdlib-only one-file script |
| `tool` | `greeter`: Kong subcommands, testscript, lint config, Taskfile, goreleaser |
| `tui` | `pick`: a picker that draws on stderr; huh confirm |
| `dashboard` | `portview`: full-screen table + detail pane + filter + live refresh (Linux `/proc`) |
| `ux` | theme, status lines, spinner, progress bar, table, tree, box |
| `recipes` | subprocess, HTTP, bounded concurrency, `--dry-run`/`--yes` |
| `config` | `greeter` again: flag > env > file > default; shell completion |
| `report` | `modfresh`: report-style CLI. Parallel lookups behind a spinner `Tally`, then a table/TSV/`--json`, with `--fail-outdated` for CI |

**Start a new project from an asset** (Linux/WSL, into the Linux filesystem):

```bash
~/.claude/skills/go-cli/scripts/new.sh report ~/src/depcheck github.com/jz-wilson/depcheck
```

It copies the asset, rewrites the module path and imports, sets `go 1.27`, adds `.gitattributes`, runs tidy, gofmt, vet, tests and lint, then makes the first commit. It also **renames the tool**: the asset's single `kong.Name` becomes the module path's last element everywhere, along with error prefixes, the testscript command, `.gitignore`, the env prefix (`MODFRESH_` → `DEPCHECK_`) and Go identifiers (`runModfresh` → `runDepcheck`). Recipes, with several binaries, keep their names. It refuses a dotless module path or a non-empty destination. Verified 2026-10-09: all 8 assets scaffold, rename with 0 leftover references and pass. It was used for real to build `certcheck` from `report`.

What remains is replacing the domain code. For `report`, that's `proxy.go` (the lookups), the `Row` fields and `cli.go`'s flags. `client.go` (HTTP with retries), `ui.go`, `theme.go`, `exit.go` and `tty.go` are generic; keep them, or `rm client.go` if the tool doesn't use HTTP.

**Long references start with a Contents block** that gives line numbers. Read just the section you need (`Read` with `offset`/`limit`) instead of the whole file.

**User preference:** any wait longer than a moment shows a *ticking* spinner, so it's obvious work is happening. For a known count, use `ui.NewTally`: spinner + bar + `n/total`, with the spinner ticking even while no item has finished. For an unknown length, use `ui.Spin`. A bare progress bar that freezes between items doesn't count (ux.md rule 8).

Every reference has a **How it looks** section with real captured output. Match that look; don't invent a different style per project.

**Existing projects keep their stack.** If the repo already uses Cobra, urfave/cli, or Bubble Tea v1, follow it. Do not migrate unless asked.

## Versions

Checked 2026-10-09 against the GitHub API, proxy.golang.org and go.dev. Versions decay. Before pinning, re-check with `go list -m -versions <module>`, or `go get <module>@latest` and read go.mod.

| Module | Version | Note |
|---|---|---|
| Go | 1.27.2 | `go.mod`: `go 1.27`; add a `toolchain` line only when needed |
| `github.com/alecthomas/kong` | v1.16.1 | Arg parsing for the Tool and TUI tiers |
| `charm.land/bubbletea/v2` | v2.1.0 | **Not** `github.com/charmbracelet/bubbletea` (that is v1) |
| `charm.land/lipgloss/v2` | v2.0.6 | |
| `charm.land/bubbles/v2` | v2.2.1 | |
| `charm.land/huh/v2` | v2.0.3 | |
| `charm.land/log/v2` | v2.0.1 | Pretty `slog` handler for human-facing logs (ux.md) |
| `charm.land/glamour/v2` | v2.0.1 | Markdown rendering, only when needed |
| `go.abhg.dev/komplete` | v0.1.0 | Kong shell completion: bash, zsh, fish; no PowerShell (config.md) |
| `golang.org/x/mod` | v0.41.0 | `modfile`, `module.EscapePath`, `semver` for go.mod tooling (report asset) |
| `golang.org/x/sync` | v0.24.0 | `errgroup` for bounded concurrency (recipes.md) |
| `github.com/charmbracelet/colorprofile` | v0.4.3 | Per-stream color/emoji decision (ux.md) |
| `golang.org/x/term` | v0.47.0 | TTY detection; replaces `mattn/go-isatty` |
| `github.com/rogpeppe/go-internal` | v1.16.0 | `testscript` |
| `github.com/charmbracelet/x/exp/teatest/v2` | pseudo-version | Experimental, untagged: pin the commit `go get` picks |
| golangci-lint | v2.14.0 | Binary install, **not** `go tool` (its docs advise against it) |
| goreleaser | v2.18.3 | |
| go-task | v3.54.0 | |

## Rules (all tiers)

1. **`main` is three lines, `run` does the work.** It receives `ctx`, args and the streams, returns an exit code, and never touches `os.Stdout`, `os.Args` or `os.Exit` itself. That is what makes it testable.

   ```go
   func main() {
       ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
       code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
       stop()
       os.Exit(code)
   }
   ```
   Call `stop()` before `os.Exit`, because `os.Exit` skips defers.
2. **Ctrl-C cancels a context.** Thread `ctx` through everything that blocks or loops, and check `ctx.Err()` in long loops.
3. **stdout is data. stderr is everything else**: logs, progress, prompts, the TUI itself. Then `x=$(tool …)` and `tool | jq` work.
4. **Log with `log/slog`** using `slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level})`. `-v` sets `slog.LevelDebug`. No `log.Fatal` or `panic` for expected errors.
5. **Exit codes:** 0 for success, 1 for a runtime error, 2 for a usage error. That covers bad flags, missing args and unknown commands. With `flag.ContinueOnError`, `-h` returns `flag.ErrHelp`; map it to 0. **Report-style tools** (checkers, linters, scanners) separate *could not check* from *found something*:
   - An item that couldn't be checked (a network error, a lookup failure) exits 1, because the tool didn't do its job.
   - A finding (outdated, expiring) exits 0 and is shown in the output. A `--fail-*` flag (`--fail-outdated`, `--fail-within N`) turns findings into exit 1 for CI or cron.
6. **Errors:** wrap with context, `fmt.Errorf("load config %s: %w", path, err)`; test with `errors.Is`/`errors.As`; print once, at the top of `run`, as `tool: <err>`. Don't re-add context the error already carries: `*os.PathError` already names the op and path, so `return err`.
7. **Color and emoji only for a human.** Style with Lip Gloss and print with `lipgloss.Fprint*`. The stream's color profile (`colorprofile.Detect`) then strips color for pipes, files and `TERM=dumb`, honors `NO_COLOR` (no color, emoji kept) and `CLICOLOR_FORCE`. Emoji and animations appear only on a terminal. Use the semantic theme in ux.md. Don't hand-roll ANSI codes or `if NO_COLOR` branches.
8. **`--json` whenever output is structured**, writing only `json.NewEncoder(stdout).Encode(v)` to stdout: no color, no progress.
9. **No package-level mutable state.** The exceptions are `var version` (set by ldflags) and immutable styles.
10. **Never prompt without a terminal.** Every prompted value must also be settable by a flag. When stdin is not a TTY, fail with an error that names the flag.
11. **Dotted module path from the start:** `go mod init github.com/<user>/<tool>` (or `example.com/<tool>`), never a bare `go mod init tool`. goimports and gofumpt treat a dotless path as stdlib, which merges your `internal/` imports into the stdlib group and fails lint. `go mod init` writes the patch version (`go 1.27.2`); `go mod edit -go=1.27` keeps the minimum loose for a tool others build.

## Done means verified

Before calling Go code done, run these and paste the results:

```bash
gofmt -l .            # empty output (Tool tier and up: golangci-lint fmt --diff ./...)
go vet ./...
go test ./...         # Tool/TUI tiers
golangci-lint run ./...   # Tool/TUI tiers
```

Then run the program once for real, the way a user would: a happy path, a usage error (check the exit code is 2), and the piped or non-TTY path (`tool … | cat`).

Before release, also cross-compile: `GOOS=windows go build -o /dev/null . && GOOS=darwin go build -o /dev/null .`. All the examples pass this.

## Windows

Facts checked 2026-10-09 against `go doc os/signal` and colorprofile v0.4.3's source; the code was cross-compiled and run.

- **Signals:** `os.Interrupt` (Ctrl-C, Ctrl-Break) is the only signal Windows sends you. `syscall.SIGTERM` compiles, and arrives on console close, logoff or shutdown. Keep `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` as-is; it's correct on all three OSes. Never use `syscall.SIGKILL`, `SIGHUP` or `SIGUSR1` logic without a build tag.
- **Color:** colorprofile treats Windows 10 build 14931+ as TrueColor and pre-10586 as no-color (unless ANSICON or ConEmu is present). Windows Terminal renders the ux.md theme. Legacy conhost is UNVERIFIED: if it shows raw `←[31m`, that's the console, not your code.
- **Emoji:** Windows Terminal draws them; legacy conhost shows boxes. A user stuck on conhost gets the plain mode (`label: message`) with `TERM=dumb`.
- **Paths:** use `filepath` (not `path`) for OS paths, `os.UserConfigDir()`/`os.UserHomeDir()` (not `$HOME`), and `os.DevNull` (not `/dev/null`). Never build paths by string concatenation with `/`.
- **Executables:** `exec.LookPath("git")` finds `git.exe` via PATHEXT. Never invoke `sh -c`, because it doesn't exist on a stock Windows machine. goreleaser adds `.exe` itself, and `go install` puts binaries in `%USERPROFILE%\go\bin`.
- **Line endings:** with `core.autocrlf=true`, `.txtar` golden files and shell scripts can be checked out as CRLF and break testscript and shebangs. Ship a `.gitattributes` with `* text=auto eol=lf`.
- **This workstation:** Windows Application Control may block freshly built unsigned binaries (`go run`, `go test`) with "An Application Control policy has blocked this file". Build and test in WSL instead. A binary cross-compiled there (`GOOS=windows go build`) did run on Windows. Don't try to bypass the policy.

## Linux and WSL

Measured 2026-10-09 on WSL 2 (Ubuntu 24.04, zsh, Go 1.27.2). Everything here applies to plain Linux too, except the `/mnt/c` and `.exe` items.

- **Install Go without sudo:** `curl -fsSL "https://go.dev/dl/$(curl -fsS 'https://go.dev/VERSION?m=text' | head -1).linux-amd64.tar.gz" | tar -C ~/.local -xz`, then add `export PATH=$HOME/.local/go/bin:$HOME/go/bin:$PATH` to `~/.profile` **and** `~/.zshrc`/`~/.bashrc`. A non-login shell (`wsl.exe -- cmd`, editor terminals) may not read `.profile`. `go install` puts binaries in `$(go env GOPATH)/bin` = `~/go/bin`.
- **Work in the Linux filesystem (`~/src/...`), not `/mnt/c`:**
  - Builds there are faster: a cold `go build` of the ux example took 7.8 s on `/mnt/c` vs 5.3 s on ext4.
  - Files on `/mnt/c` show as `-rwxrwxrwx`, which gives git spurious mode changes.
  - A checkout shared with Windows git (`core.autocrlf=true`) hands WSL CRLF files that break shebangs and `.txtar` tests.
  - Use one clone per side, and LF via `.gitattributes`.
- **Static binaries for sharing:** with cgo on (gcc present, `CGO_ENABLED=1` by default), `go build` produces a **dynamically linked** ELF whenever `net` or `os/user` is imported; the fetch recipe is one example. It may not run on another distro or Alpine. Build release binaries with `CGO_ENABLED=0` (statically linked, verified). goreleaser's config in tool.md already does.
- **`-race` needs cgo:** `CGO_ENABLED=0 go test -race` fails with "-race requires cgo". Keep gcc (`sudo apt install build-essential`) for local race testing.
- **Signals:** SIGINT and SIGTERM (systemd stop, `kill`, `docker stop`) both arrive via `signal.NotifyContext`. **Don't catch SIGPIPE.** When stdout's reader exits (`tool | head -1`), Go ends the program quietly with signal 13 (shell status 141), which is the correct Unix behavior. Verified with the script example.
- **Color:** in WSL launched from Windows, `TERM=xterm-256color` and `COLORTERM` is usually unset, so colorprofile picks ANSI256 and true-color gradients are downsampled. That's fine. `export COLORTERM=truecolor` if the terminal supports it (Windows Terminal does). The theme's basic ANSI colors are unaffected.
- **Windows from WSL:**
  - `GOOS=windows go build -o tool.exe .` works with no extra setup.
  - The `.exe` runs straight from WSL (`/mnt/c/.../tool.exe`) through interop. Put it under `/mnt/c` first so Windows paths resolve.
  - This is also the workaround when Windows Application Control blocks local Go builds.
- **Paths and config:**
  - `os.UserConfigDir()` resolves to `$XDG_CONFIG_HOME` or `~/.config` and `os.UserCacheDir()` to `~/.cache`; never hardcode `~/.toolname`.
  - Completion scripts go to `~/.local/share/bash-completion/completions/` or a dir in `$fpath` (config.md).
- **This skill from WSL:** `~/.claude/skills/go-cli` in WSL is a symlink to the Windows copy (`ln -s /mnt/c/Users/<you>/.claude/skills/go-cli ~/.claude/skills/go-cli`), so the documented paths work on both sides, and Claude Code inside WSL sees the skill too.
- **Editing WSL files from Windows-side tools:** use the `\\wsl.localhost\<distro>\home\<user>\src\...` path with Read and Edit. That keeps the project on ext4, where builds are fast, while Windows tools edit it in place.
- **Running a Windows-side tool from WSL** (or the reverse): `wsl.exe -d <distro> -- bash /mnt/c/.../script.sh`. From Git Bash, prefix it with `MSYS_NO_PATHCONV=1`, or Git Bash rewrites `/mnt/c/...` into a Windows path.

## Keeping the skill current

`assets/` holds every example as a real project. That's the source of truth: copy files from there instead of retyping them from the docs. The `<!-- asset: … -->` code blocks in `references/` are generated from them.

```bash
scripts/refresh.sh            # upgrade all deps in a temp copy, run gofmt/vet/test -race/lint, report
scripts/refresh.sh --write    # keep the upgrades and re-sync the reference code blocks
scripts/refresh.sh --sync     # after hand-editing an asset: re-sync the docs and Contents blocks only
```

**Monthly, automatically:** a systemd user timer in WSL (`go-cli-refresh.timer`, `OnCalendar=monthly`, `Persistent=true`, so a missed run happens at the next WSL start) runs `scripts/scheduled-refresh.sh`. That does a dry-run refresh and pushes to ntfy. The unit sets `Environment=NTFY_URL=https://<ntfy-host>/<topic>`, and the token comes from `NTFY_TOKEN` or the `NTFY_TOKEN=` line of `$NTFY_ENV` (default `~/.config/vault-nightly/ntfy.env`). With either missing it logs what it would have sent.

| Result | Push |
|---|---|
| An example fails | **high** priority: "go-cli skill: refresh FAILED" |
| Passes, with upgrades | low priority: "updates available" (then run `--write` and update the Versions table) |
| Passes, nothing new | silent |

The log is at `~/.local/state/go-cli-refresh/last.log`. Check the timer with `systemctl --user list-timers go-cli-refresh.timer`, or run it now with `systemctl --user start go-cli-refresh.service`. All three result paths were tested against a local ntfy stand-in, and a real systemd run succeeded silently, on 2026-10-09.

To check that the description still triggers, run `evals/trigger.json` through skill-creator's `scripts/run_eval.py`. Last run on 2026-10-09: the bare "port this bubbletea v1 model to v2" went from 1/3 to 5/5 after the description gained its "even before the code is shared" sentence, and all 7 negatives stayed quiet. "add a spinner while my Go tool downloads…" is flaky (2/6 to 5/6 across runs); a 1/3 on it is noise, not a regression. Two harness caveats:
1. On Windows the stock script fails (`select()` on pipes, WinError 10038).
2. The installed `go-cli` skill shadows the eval's temporary copy, so a stock run reports 0 triggers. Either uninstall the skill first or count `Skill(go-cli)` calls as hits.

Run it from Linux, macOS or WSL. Then update the Versions table and its date from what it prints. If it fails, fix the asset first. Never publish an example that fails.
