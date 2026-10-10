package main

import (
	"io"
	"os"

	"golang.org/x/term"
)

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd())) && os.Getenv("CI") == ""
}
