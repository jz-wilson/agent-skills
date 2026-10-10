package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// The test binary doubles as the child process, so the tests need no external tools.
func TestMain(m *testing.M) {
	switch os.Getenv("RECIPE_HELPER") {
	case "":
		os.Exit(m.Run())
	case "exit3":
		fmt.Fprintln(os.Stderr, "boom")
		os.Exit(3)
	case "hello":
		fmt.Println("hello")
	case "sleep":
		time.Sleep(time.Minute)
	}
}

func runExec(t *testing.T, ctx context.Context, helper string, extra ...string) (int, string, string) {
	t.Helper()
	t.Setenv("RECIPE_HELPER", helper)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := append(append([]string{"exec"}, extra...), "--", exe)
	var stdout, stderr bytes.Buffer
	code := run(ctx, args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestExitCodePropagates(t *testing.T) {
	code, _, stderr := runExec(t, t.Context(), "exit3")
	if code != 3 || !strings.Contains(stderr, "boom") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestCaptureKeepsStdoutClean(t *testing.T) {
	code, stdout, _ := runExec(t, t.Context(), "hello", "--capture")
	if code != 0 || stdout != "hello\n" {
		t.Fatalf("code=%d stdout=%q", code, stdout)
	}
}

func TestStreamGoesToStderr(t *testing.T) {
	_, stdout, stderr := runExec(t, t.Context(), "hello")
	if stdout != "" || !strings.Contains(stderr, "hello\n") {
		t.Fatalf("stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestNotInstalled(t *testing.T) {
	var stderr bytes.Buffer
	code := run(t.Context(), []string{"exec", "--", "no-such-tool-xyz"}, strings.NewReader(""), &bytes.Buffer{}, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "no-such-tool-xyz is not installed") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestCancelKillsChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	code, _, stderr := runExec(t, ctx, "sleep")
	if code != 1 || !strings.Contains(stderr, "interrupted") || time.Since(start) > 4*time.Second {
		t.Fatalf("code=%d after %v stderr=%q", code, time.Since(start), stderr)
	}
}
