package main

import (
	"image/color"
	"io"
	"log/slog"
	"os"

	"charm.land/lipgloss/v2"
	charmlog "charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
)

// Level is the meaning of a line; the theme decides how it looks.
type Level int

const (
	LevelDebug Level = iota
	LevelStep
	LevelInfo
	LevelSuccess
	LevelWarn
	LevelError
)

type levelStyle struct {
	emoji string
	label string
	color color.Color
}

// theme is the one place the color scheme and emoji live. Basic ANSI colors
// (not hex) so the user's terminal palette decides the exact shade. Emoji are
// ones that are double-width by default; avoid ℹ️/⚠️ (U+FE0F variants), which
// some terminals draw one column wide and overlap the next character.
var theme = map[Level]levelStyle{
	LevelDebug:   {"🐛", "debug", lipgloss.BrightBlack},
	LevelStep:    {"🔹", "step", lipgloss.Cyan},
	LevelInfo:    {"🔵", "info", lipgloss.Blue},
	LevelSuccess: {"✅", "ok", lipgloss.Green},
	LevelWarn:    {"🟡", "warn", lipgloss.Yellow},
	LevelError:   {"❌", "error", lipgloss.Red},
}

// Output modes, decided once per stream:
//
//	terminal              emoji + colored label
//	terminal + NO_COLOR   emoji + bold label, no color (NO_COLOR is only about color)
//	pipe, file, TERM=dumb plain "label: message", no emoji, no ANSI (grep-friendly)
//
// CLICOLOR_FORCE=1 forces the terminal look into a pipe.
func useEmoji(w io.Writer) bool {
	return colorprofile.Detect(w, os.Environ()) != colorprofile.NoTTY
}

// prefix renders "💡 info:" (or "info:" without emoji). lipgloss.Fprint* strips
// or downsamples the color to fit the stream, so callers never branch on color.
func prefix(l Level, emoji bool) string {
	s := theme[l]
	label := lipgloss.NewStyle().Bold(true).Foreground(s.color).Render(s.label + ":")
	if emoji {
		return s.emoji + " " + label
	}
	return label
}

// newLogger returns a slog logger whose level labels use the same theme.
func newLogger(w io.Writer, verbose bool) *slog.Logger {
	level := charmlog.InfoLevel
	if verbose {
		level = charmlog.DebugLevel
	}
	l := charmlog.NewWithOptions(w, charmlog.Options{Level: level})
	styles := charmlog.DefaultStyles()
	emoji := useEmoji(w)
	for cl, tl := range map[charmlog.Level]Level{
		charmlog.DebugLevel: LevelDebug,
		charmlog.InfoLevel:  LevelInfo,
		charmlog.WarnLevel:  LevelWarn,
		charmlog.ErrorLevel: LevelError,
	} {
		styles.Levels[cl] = lipgloss.NewStyle().SetString(prefix(tl, emoji))
	}
	l.SetStyles(styles)
	return slog.New(l)
}
