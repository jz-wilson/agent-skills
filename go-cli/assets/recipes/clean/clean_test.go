package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range []string{"a.tmp", "b.tmp", "keep.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runClean(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer // buffers are never terminals: this is the non-TTY path
	code = run(t.Context(), append([]string{"clean"}, args...), strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func count(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestDryRunChangesNothing(t *testing.T) {
	dir := setup(t)
	code, stdout, _ := runClean(t, "--dry-run", "--yes", dir)
	if code != 0 || strings.Count(stdout, "would remove") != 2 || count(t, dir) != 3 {
		t.Fatalf("code=%d stdout=%q files=%d", code, stdout, count(t, dir))
	}
}

func TestNoTTYWithoutYesIsUsageError(t *testing.T) {
	dir := setup(t)
	code, _, stderr := runClean(t, dir)
	if code != 2 || !strings.Contains(stderr, "--yes") || count(t, dir) != 3 {
		t.Fatalf("code=%d stderr=%q files=%d", code, stderr, count(t, dir))
	}
}

func TestYesRemovesOnlyTmp(t *testing.T) {
	dir := setup(t)
	code, stdout, _ := runClean(t, "--yes", dir)
	if code != 0 || strings.Count(stdout, "removed") != 2 || count(t, dir) != 1 {
		t.Fatalf("code=%d stdout=%q files=%d", code, stdout, count(t, dir))
	}
}
