package main

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"portview": main})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(e *testscript.Env) error {
			abs, err := filepath.Abs(filepath.Join("internal", "scan", "testdata", "proc"))
			if err != nil {
				return err
			}
			e.Setenv("FAKEPROC", abs)
			return nil
		},
	})
}
