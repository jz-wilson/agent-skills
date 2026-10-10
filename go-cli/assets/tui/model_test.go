package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
)

// Update is a pure function: drive it with messages, assert on the model.
func TestUpdateMovesAndPicks(t *testing.T) {
	var m tea.Model = newModel([]string{"a", "b", "c"})
	for _, k := range []tea.KeyPressMsg{{Code: tea.KeyDown}, {Code: tea.KeyDown}, {Code: tea.KeyUp}} {
		m, _ = m.Update(k)
	}
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := m.(model).chosen; got != "b" {
		t.Fatalf("chosen = %q, want b", got)
	}
	if cmd == nil {
		t.Fatal("enter should return tea.Quit")
	}
}

// teatest runs the real program loop against a fake terminal.
func TestProgramPicks(t *testing.T) {
	tm := teatest.NewTestModel(t, newModel([]string{"a", "b"}), teatest.WithInitialTermSize(40, 10))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return bytes.Contains(b, []byte("> a")) })
	tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second))
	if got := final.(model).chosen; got != "b" {
		t.Fatalf("chosen = %q, want b", got)
	}
}

// The non-TTY path must work without a terminal: a bytes.Buffer is never one.
func TestNonInteractive(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--index", "1", "x", "y"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if out.String() != "y\n" {
		t.Fatalf("stdout = %q", out.String())
	}
	out.Reset()
	errb.Reset()
	if code := run(context.Background(), []string{"x", "y"}, strings.NewReader(""), &out, &errb); code != 1 || !strings.Contains(errb.String(), "--index") {
		t.Fatalf("want exit 1 naming --index, got %d: %s", code, errb.String())
	}
}
