package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestHashAllConcurrently(t *testing.T) {
	dir := t.TempDir()
	var paths, want []string
	for i := range 40 {
		p := filepath.Join(dir, strconv.Itoa(i))
		data := bytes.Repeat([]byte{byte(i)}, 1000*i)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		paths, want = append(paths, p), append(want, hex.EncodeToString(sum[:]))
	}
	var done atomic.Int32
	got, err := hashAll(t.Context(), paths, 8, func() { done.Add(1) })
	if err != nil || done.Load() != 40 {
		t.Fatalf("err=%v done=%d", err, done.Load())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sum %d = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestHashAllStopsOnError(t *testing.T) {
	_, err := hashAll(t.Context(), []string{"missing-a", "missing-b"}, 1, func() {})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v", err)
	}
}

func TestZeroJobsIsAUsageError(t *testing.T) {
	var stderr bytes.Buffer
	code := run(t.Context(), []string{"sums", "-j", "0", "x"}, strings.NewReader(""), &bytes.Buffer{}, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "--jobs") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}
