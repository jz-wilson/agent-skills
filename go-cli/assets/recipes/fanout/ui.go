package main

import (
	"context"
	"fmt"
	"io"
	"strings"

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

// Progress redraws one bar in place with \r; without a terminal it prints only the result.
func (u *UI) Progress(ctx context.Context, title string, total int, step func(i int)) {
	bar := progress.New(progress.WithDefaultBlend(), progress.WithWidth(30))
	for i := range total {
		if ctx.Err() != nil {
			return
		}
		step(i)
		if u.interactive {
			lipgloss.Fprint(u.w, "\r"+title+" "+bar.ViewAs(float64(i+1)/float64(total)))
		}
	}
	if u.interactive {
		fmt.Fprint(u.w, "\r\033[K") // clear the bar line
	}
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
