package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
)

// UI writes human-facing feedback to stderr using the theme in theme.go.
// Colors are downsampled by lipgloss.Fprint* (stripped for pipes, NO_COLOR,
// TERM=dumb); emoji only on a terminal; animations only on a terminal outside CI.
type UI struct {
	w           io.Writer
	interactive bool
	emoji       bool
}

func newUI(stderr io.Writer) *UI {
	return &UI{w: stderr, interactive: isTerminal(stderr), emoji: useEmoji(stderr)}
}

var faint = lipgloss.NewStyle().Faint(true)

func (u *UI) line(l Level, format string, a ...any) {
	lipgloss.Fprintln(u.w, prefix(l, u.emoji)+" "+fmt.Sprintf(format, a...))
}

func (u *UI) Step(format string, a ...any)  { u.line(LevelStep, format, a...) }
func (u *UI) Info(format string, a ...any)  { u.line(LevelInfo, format, a...) }
func (u *UI) OK(format string, a ...any)    { u.line(LevelSuccess, format, a...) }
func (u *UI) Warn(format string, a ...any)  { u.line(LevelWarn, format, a...) }
func (u *UI) Error(format string, a ...any) { u.line(LevelError, format, a...) }

// Spin runs action behind a spinner. Without a terminal it prints the title as a
// plain step line and runs the action, so logs and CI stay clean. (huh's own
// accessible mode still emits ANSI codes and no newline as of huh v2.0.3.)
func (u *UI) Spin(ctx context.Context, title string, action func(context.Context) error) error {
	if !u.interactive {
		u.Step("%s", title)
		return action(ctx)
	}
	return spinner.New().
		Title(title).
		Context(ctx).
		ActionWithErr(action).
		WithOutput(u.w).
		Run()
}

// Tally is one live progress line: spinner, title, bar, n/total. The spinner
// ticks on its own goroutine, so the line keeps moving while work is in flight
// even when no item has finished yet. Inc is safe from many goroutines (use it
// as the per-item callback of an errgroup fan-out). Without a terminal it draws
// nothing. Always call Done, which stops the spinner and erases the line.
type Tally struct {
	mu                 sync.Mutex
	u                  *UI
	title              string
	done, total, frame int
	bar                progress.Model
	stop               chan struct{}
	wg                 sync.WaitGroup
}

var spinStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

func (u *UI) NewTally(title string, total int) *Tally {
	t := &Tally{u: u, title: title, total: total, bar: progress.New(progress.WithDefaultBlend(), progress.WithWidth(30))}
	if u.interactive {
		t.stop = make(chan struct{})
		t.wg.Add(1)
		go t.spin()
	}
	return t
}

func (t *Tally) spin() {
	defer t.wg.Done()
	tick := time.NewTicker(spinner.Dots.FPS)
	defer tick.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-tick.C:
			t.mu.Lock()
			t.frame++
			t.draw()
			t.mu.Unlock()
		}
	}
}

// Inc marks one item finished.
func (t *Tally) Inc() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.done++
	if t.u.interactive {
		t.draw()
	}
}

// draw redraws the line in place; call with t.mu held.
func (t *Tally) draw() {
	frame := spinner.Dots.Frames[t.frame%len(spinner.Dots.Frames)] // frames carry a trailing space
	lipgloss.Fprintf(t.u.w, "\r%s%s %s %d/%d", spinStyle.Render(frame), t.title,
		t.bar.ViewAs(float64(t.done)/float64(t.total)), t.done, t.total)
}

// Done stops the spinner and erases the line; print the outcome line after it.
func (t *Tally) Done() {
	if t.stop != nil {
		close(t.stop)
		t.wg.Wait()
		t.stop = nil
	}
	if t.u.interactive {
		fmt.Fprint(t.u.w, "\r\033[K")
	}
}

// Progress runs step for each of total items sequentially behind a Tally, then
// prints the ok line. Without a terminal only the ok line appears.
func (u *UI) Progress(ctx context.Context, title string, total int, step func(i int)) {
	t := u.NewTally(title, total)
	defer t.Done()
	for i := range total {
		if ctx.Err() != nil {
			return
		}
		step(i)
		t.Inc()
	}
	t.Done()
	u.OK("%s %d/%d", title, total, total)
}

// Box frames a short note, e.g. next steps at the end of a run.
func (u *UI) Box(title, body string) {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
	heading := lipgloss.NewStyle().Bold(true).Render(title)
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		lines[i] = faint.Render("$ ") + l
	}
	lipgloss.Fprintln(u.w, box.Render(heading+"\n"+strings.Join(lines, "\n")))
}
