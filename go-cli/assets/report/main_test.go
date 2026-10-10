package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){"modfresh": main})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{Dir: "testdata"})
}

const goMod = `module example.com/app

go 1.27

require (
	github.com/BurntSushi/toml v1.3.0
	github.com/old/pkg v1.0.0
	github.com/same/pkg/v2 v2.5.0
	github.com/indirect/only v0.1.0 // indirect
)
`

// fakeProxy serves /<escaped path>/@latest. The BurntSushi key is the escaped
// form, so the test fails if the client forgets to case-escape.
func fakeProxy(t *testing.T) string {
	t.Helper()
	latest := map[string]string{
		"/github.com/!burnt!sushi/toml/@latest": "v1.3.2",
		"/github.com/old/pkg/@latest":           "v1.4.0",
		"/github.com/same/pkg/v2/@latest":       "v2.5.0",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, ok := latest[r.URL.EscapedPath()]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"Version":"` + v + `","Time":"2026-01-01T00:00:00Z"}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func runModfresh(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "go.mod")
	if err := os.WriteFile(path, []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	args = append(args, "--proxy", fakeProxy(t), path)
	code = run(context.Background(), args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func TestOutdatedPipedTSV(t *testing.T) {
	code, out, errs := runModfresh(t)
	if code != 0 {
		t.Fatalf("exit %d, stderr:\n%s", code, errs)
	}
	want := "github.com/BurntSushi/toml\tv1.3.0\tv1.3.2\tpatch behind\n" +
		"github.com/old/pkg\tv1.0.0\tv1.4.0\tminor behind\n" +
		"github.com/same/pkg/v2\tv2.5.0\tv2.5.0\tup to date\n"
	if out != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
	}
	if !strings.Contains(errs, "warn: 2 of 3 dependencies are outdated") {
		t.Errorf("stderr missing summary:\n%s", errs)
	}
	if strings.Contains(errs, "\x1b") {
		t.Errorf("ANSI codes leaked into a non-terminal stream:\n%q", errs)
	}
}

func TestFailOutdated(t *testing.T) {
	if code, _, _ := runModfresh(t, "--fail-outdated"); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
}

func TestJSON(t *testing.T) {
	code, out, _ := runModfresh(t, "--json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Outdated int
		Modules  []Row
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, out)
	}
	if got.Outdated != 2 || len(got.Modules) != 3 {
		t.Errorf("got %+v", got)
	}
}

func TestLookupFailureExits1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	mod := "module x\n\ngo 1.27\n\nrequire github.com/missing/mod v1.0.0\n"
	if err := os.WriteFile(path, []byte(mod), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"--proxy", fakeProxy(t), path}, strings.NewReader(""), &out, &errb)
	// The TSV status column carries the reason, not just "error".
	if code != 1 || !strings.Contains(out.String(), "\terror: ") || !strings.Contains(out.String(), "404") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out.String(), errb.String())
	}
}

func TestTruncate(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"short", "short"},
		{"exactly-ten", "exactly-t…"},
		{"日本語のエラーメッセージです", "日本語のエラーメッ…"},
	} {
		if got := truncate(tc.in, 10); got != tc.want {
			t.Errorf("truncate(%q, 10) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClassify(t *testing.T) {
	for _, tc := range []struct{ cur, latest, want string }{
		{"v1.2.3", "v1.2.3", statusCurrent},
		{"v1.2.4", "v1.2.3", statusCurrent}, // ahead of the proxy
		{"v1.2.3", "v1.2.4", statusPatch},
		{"v1.2.3", "v1.3.0", statusMinor},
		{"v0.0.0-20200101000000-abcdef123456", "v0.1.0", statusMinor},
		{"v0.9.0", "v1.0.0", statusMajor},
	} {
		if got := classify(tc.cur, tc.latest); got != tc.want {
			t.Errorf("classify(%s, %s) = %q, want %q", tc.cur, tc.latest, got, tc.want)
		}
	}
}

func TestLatestURLEscapes(t *testing.T) {
	got, err := latestURL("https://p.example/", "github.com/BurntSushi/toml")
	if err != nil || got != "https://p.example/github.com/!burnt!sushi/toml/@latest" {
		t.Errorf("got %q, %v", got, err)
	}
}
