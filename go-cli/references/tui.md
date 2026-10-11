# TUI tier

The Tool tier plus Bubble Tea v2 (Elm architecture: `Init`/`Update`/`View`), Lip Gloss v2 for styling, and Bubbles v2 for stock components (list, table, viewport, textinput, spinner, help/key).

## Gate: do you need a TUI?

| Need | Use | Not |
|---|---|---|
| Ask a few questions | `huh` form, standalone (below) | Bubble Tea |
| Colored or styled one-off output | `lipgloss.Println(style.Render(...))` | Bubble Tea |
| Spinner, progress bar, status lines, tables, trees, boxes | [ux.md](ux.md): huh spinner, `bubbles/progress.ViewAs`, Lip Gloss | Bubble Tea |
| Live, stateful screens: navigable lists, scrolling viewports, panes, dashboards | **Bubble Tea** | |
| A quick form or dashboard where Bubble Tea's verbosity hurts | tview (widget-based) is a reasonable alternative | |

## Architecture

- **The core returns data; the TUI is one way to get input or show output.** The same result must be reachable with flags and no terminal (`--index`, `--yes`, `--json`).
- **Where to draw depends on the kind of TUI:**
  - **Picker** (produces a result, like `pick` below or fzf): draw on **stderr** (`tea.WithOutput(stderr)`) and print the result to stdout after exit, so `x=$(tool)` works. Gate on stdin and stderr being terminals.
  - **Dashboard or viewer** (no result, like `assets/dashboard`): draw on **stdout** (the default) and gate on stdin **and stdout** being terminals. Then `tool | cat` takes the plain path (TSV/`--json`) instead of drawing a TUI onto a terminal while stdout goes into a pipe.
- **Start the TUI only when interactive:** the gate above passes, `--no-tui` is not set, and `CI` is unset. Otherwise take the flag or plain path, or fail with an error naming the flag.
- **The CLI surface is Tool tier:** Kong (root struct for a single command), exit codes, a dotted module path, and testscript for `--help`, `--no-tui`, `--json` and bad flags all apply here too.
- `tea.WithContext(ctx)` so Ctrl-C (signal) and the program share cancellation.
- `NO_COLOR` and the color profile: Bubble Tea v2 detects the profile (via `colorprofile`) and downsamples Lip Gloss output. You don't need to handle it.

## v1 → v2 gotchas

Most examples online (and in training data) are v1. These lines mark a v1 snippet; translate before using it. Source: the bubbletea and lipgloss `UPGRADE_GUIDE_V2.md`, read 2026-10-09.

| v1 | v2 |
|---|---|
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` (same for lipgloss, bubbles, huh) |
| `View() string` | `View() tea.View`, built with `tea.NewView(s)` |
| `tea.WithAltScreen()`, `tea.EnterAltScreen` | `v.AltScreen = true` on the returned View (also `MouseMode`, `WindowTitle`, `Cursor`) |
| `case tea.KeyMsg:` | `case tea.KeyPressMsg:` (`KeyMsg` is now an interface that also matches releases) |
| `msg.Type == tea.KeyEnter`, `msg.Runes` | `msg.Code == tea.KeyEnter`, `msg.Text` (string), `msg.Mod` |
| `" "` for space | `"space"` |
| `tea.KeyCtrlC` | `msg.String() == "ctrl+c"` |
| `tea.MouseMsg` struct | `MouseClickMsg`, `MouseReleaseMsg`, `MouseWheelMsg`, `MouseMotionMsg` |
| `p.Start()` | `p.Run()` |
| `tea.Sequentially` | `tea.Sequence` |
| `lipgloss.Color` is a type, `AdaptiveColor` | `lipgloss.Color(...)` returns `color.Color`; use `LightDark(isDark)` (`compat.AdaptiveColor` as a stopgap) |
| `lipgloss.NewRenderer` | gone; styles are plain values, and downsampling happens at print time |
| bubbles `m.Width = 40` | `m.SetWidth(40)` / `m.Width()` |

Upgrade bubbletea, lipgloss and bubbles together. Mixed majors don't compile.

## Verified example

`pick` chooses one of its arguments. Built, tested and linted with Go 1.27.2, bubbletea v2.1.0, lipgloss v2.0.6, huh v2.0.3, Kong v1.16.1 and x/term v0.47.0 on 2026-10-09.

```bash
go get charm.land/bubbletea/v2@latest charm.land/lipgloss/v2@latest charm.land/huh/v2@latest \
  github.com/alecthomas/kong@latest golang.org/x/term@latest \
  github.com/charmbracelet/x/exp/teatest/v2@latest
```

`main.go`: the flag path, the non-TTY gate, and the TUI on stderr.

→ Read [`assets/tui/main.go`](../assets/tui/main.go) (93 lines; builds, tests and lints clean).

How it looks: `pick alpha beta gamma` after one ↓. This is the frame `View()` renders, drawn on stderr; the cursor line is pink and bold, the help line faint.

```
  alpha
> beta
  gamma
↑/↓ move • enter pick • q quit
```

On enter, `View` returns an empty view (the `done` field), so the frame is erased rather than left on screen, and only `beta` is printed to stdout. Verified in a pty: stdout was exactly `beta`, so `x=$(pick alpha beta gamma)` captures it. Piped or in CI, it never draws:

```
$ pick a b | cat
pick: not a terminal: pass --index N     # exit 1
$ pick --index 1 a b | cat
b
```

`model.go`:

→ Read [`assets/tui/model.go`](../assets/tui/model.go) (65 lines; builds, tests and lints clean).

## Dashboard pattern: table + detail pane + filter + live refresh

`assets/dashboard` is `portview`, a verified, live-refreshing view of listening TCP sockets, read from `/proc` (Linux). It was built from this skill by a sub-agent on 2026-10-09 and re-verified: vet, `test -race` and lint at 0 issues, plus a real tmux session. Copy its `model.go` shape for any "list of things + details + filter" TUI. The scanner (`internal/scan`) is a pure function tested against a fake `/proc` tree; its `fd` entries are symlinks, so run it from Linux/WSL.

How it looks (`tmux new-session -d -x 100 -y 24 portview`, keys `/`, `8799`, Enter, then `tmux capture-pane -p`; colors stripped):

```
portview  listening TCP · refresh 2s
 PORT    ADDRESS                                                      PID      PROCESS
 8799    0.0.0.0                                                      532474   python3
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ Port     8799/tcp                                                                                │
│ Address  0.0.0.0                                                                                 │
│ Process  python3 (pid 532474)                                                                    │
│ Command  python3 -m http.server 8799                                                             │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
filter "8799": 1 of 21
↑/k up • ↓/j down • / filter • esc clear • q quit
```

### Bubbles v2 component APIs (verified with `go doc`, bubbles v2.2.1)

| Component | Construct | Size and content | Notes |
|---|---|---|---|
| `table` | `table.New(table.WithColumns(cols), table.WithFocused(true), table.WithStyles(table.Styles{Header, Cell, Selected}))` | `SetColumns`, `SetRows([]table.Row)`, `SetWidth`, `SetHeight` (**includes the header row**), `Cursor()`/`SetCursor` | `m.table, cmd = m.table.Update(msg)` returns `(table.Model, tea.Cmd)`, not `tea.Model`. It handles keys only while focused (`Focus()`/`Blur()`) |
| `viewport` | `viewport.New()` (no args) | `SetWidth`, `SetHeight`, `SetContent`, `GotoTop()` | `SoftWrap = true` field for wrapping |
| `textinput` | `textinput.New()`; set `Prompt`, `Placeholder` | `Value()`, `Reset()` | `Focus()` **returns a `tea.Cmd`**; return it. `SetVirtualCursor(true)` draws the cursor inside the view |
| `help` + `key` | `help.New()`; `key.NewBinding(key.WithKeys("up","k"), key.WithHelp("↑/k","up"))` | `help.SetWidth`, `help.View(km)` | `km` must implement `ShortHelp() []key.Binding` and `FullHelp() [][]key.Binding`. A `type helpKeys []key.Binding` adapter lets each mode show its own keys. Match with `key.Matches(msg, binding)` |

### Patterns `model.go` gets right

- **Size first:**
  - Store `tea.WindowSizeMsg` and call one `layout()` that sizes every component.
  - `View()` returns `tea.NewView("")` until the first size arrives.
  - Set `v.AltScreen = true` on the returned view for full-screen apps.
- **Budget the height explicitly:** `const chromeLines = title + detail box content + 2 border + status + help`. Then `table.SetHeight(height - chromeLines)`. Add a test that `lipgloss.Height(m.View().Content) <= height`; an off-by-one scrolls the whole frame.
- **Refresh by chaining, not a free-running ticker:** `Init` returns the first scan `Cmd`. Handling its result message returns `tea.Tick(interval, …refreshMsg)`, and `refreshMsg` starts the next scan. Scans never overlap, and a slow scan just stretches the interval.
- **Modes:** in filter mode, route every key to the textinput first (typing `q` must not quit; tested). Only `ctrl+c`, `enter` and `esc` are intercepted. Blur the table while filtering and focus it again after.
- **Keep the selection across refreshes:** remember the selected item's identity before rebuilding rows, then restore the cursor to it.
- **Work goes in `tea.Cmd`s**, never in `Update`: `scanCmd(root)` returns a func that does the I/O and returns a message.

→ Read [`assets/dashboard/model.go`](../assets/dashboard/model.go) (276 lines; builds, tests and lints clean).

## Prompts without a TUI (huh)

`huh` runs standalone, without a Bubble Tea program of your own. Use it in Tool-tier CLIs for confirmations and small forms. Gate it exactly like the TUI (`interactive(...)`), and keep a flag for every value it asks for.

→ Read [`assets/tui/confirm.go`](../assets/tui/confirm.go) (16 lines; builds, tests and lints clean).

`WithAccessible(true)` swaps the TUI for plain line prompts that work with screen readers. Turn it on from an env var or flag, never by default.

## Tests

1. **Pure `Update` tests first.** A model is a value: send `tea.KeyPressMsg{Code: tea.KeyDown}` and assert on the returned model. They're fast, need no terminal, and cover most logic. To **type text** into a textinput, send `tea.KeyPressMsg{Code: r, Text: string(r)}`; `Code` alone moves nothing into the input. In teatest, `tm.Type("8799")` does this for you.
2. **teatest v2 for the real loop:** a handful of tests, not one per key. It lives in `x/exp`, is untagged and experimental, so `go get` pins a pseudo-version: keep it pinned and update it on purpose. `tm.FinalOutput` + `teatest.RequireEqualOutput` gives golden files (`go test -update` rewrites them). Set `teatest.WithInitialTermSize` so output is deterministic. `WaitFor` must match text that is actually on screen, not text in a pane for an unselected row. Its 1 s default is tight under `-race`, so pass `teatest.WithDuration(5*time.Second)` as `WaitFor`'s last argument. Use `FinalModel` to assert on state.
4. **Look at the real thing before calling it done.** Run it in tmux and capture the screen:
   ```bash
   tmux new-session -d -s t -x 100 -y 24 ./tool; sleep 1
   tmux send-keys -t t / ; tmux send-keys -t t -l 'query'; tmux send-keys -t t Enter; sleep 1
   tmux capture-pane -t t -p        # the rendered frame, as text
   tmux send-keys -t t q            # and check it quits
   ```
   tmux renders Bubble Tea v2 correctly, including the alt screen and kitty keyboard sequences. `script` + pyte needs preprocessing (cut at `ESC[?1049l`, strip `ESC[>…u`), so prefer tmux for full-screen apps.
3. **The non-TTY path:** call `run` with `bytes.Buffer`s. A buffer is never a terminal, so this exercises the fallback for free.

→ Read [`assets/tui/model_test.go`](../assets/tui/model_test.go) (55 lines; builds, tests and lints clean).
