package main

import (
	"bytes"
	"testing"
)

// A buffer is never a terminal: the UI must fall back to plain "label: message".
func TestPlainWhenNotATerminal(t *testing.T) {
	var buf bytes.Buffer
	ui := newUI(&buf)
	ui.Warn("disk %d%% full", 91)
	ui.Error("boom")
	if got, want := buf.String(), "warn: disk 91% full\nerror: boom\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
