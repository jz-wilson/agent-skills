package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateConfigHome points os.UserConfigDir at a temp dir on Linux, Windows and macOS
// and returns the path of the default config file.
func isolateConfigHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	// Clear anything inherited from the developer's shell. Kong treats an empty
	// variable as set, so unset rather than set to "". t.Setenv restores it afterwards.
	for _, k := range []string{"GREETER_GREETING", "GREETER_SHOUT"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "greeter", "config.json")
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPrecedence(t *testing.T) {
	tests := []struct {
		name string
		file string // content of the default config file; empty means no file
		env  string // GREETER_GREETING; empty means unset
		args []string
		want string
	}{
		{"default", "", "", []string{"greet", "ann"}, "hello, ann\n"},
		{"file beats default", `{"greeting":"hola"}`, "", []string{"greet", "ann"}, "hola, ann\n"},
		{"env beats file", `{"greeting":"hola"}`, "bonjour", []string{"greet", "ann"}, "bonjour, ann\n"},
		{"flag beats env", `{"greeting":"hola"}`, "bonjour", []string{"greet", "--greeting=ciao", "ann"}, "ciao, ann\n"},
		{"file sets a bool", `{"shout":true}`, "", []string{"greet", "ann"}, "HELLO, ANN\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := isolateConfigHome(t)
			if tt.file != "" {
				write(t, path, tt.file)
			}
			if tt.env != "" {
				t.Setenv("GREETER_GREETING", tt.env)
			}
			var stdout, stderr bytes.Buffer
			if code := run(context.Background(), tt.args, &stdout, &stderr); code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigFlagOverridesDefaultFile(t *testing.T) {
	path := isolateConfigHome(t)
	write(t, path, `{"greeting":"hola"}`)
	other := filepath.Join(t.TempDir(), "other.json")
	write(t, other, `{"greeting":"ciao"}`)

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--config", other, "greet", "ann"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "ciao, ann\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}

	// An env var still beats the --config file.
	t.Setenv("GREETER_GREETING", "bonjour")
	stdout.Reset()
	code = run(context.Background(), []string{"--config", other, "greet", "ann"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "bonjour, ann\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestConfigErrors(t *testing.T) {
	tests := []struct {
		name    string
		file    string // default config file content; empty means none
		args    []string
		want    int
		wantErr string
	}{
		{"malformed default file", `{"greeting":`, []string{"greet", "ann"}, 1, "config.json: "},
		{"missing --config file", "", []string{"--config", "nope.json", "greet", "ann"}, 2, "nope.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := isolateConfigHome(t)
			if tt.file != "" {
				write(t, path, tt.file)
			}
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tt.args, &stdout, &stderr)
			if code != tt.want || !strings.Contains(stderr.String(), tt.wantErr) || stdout.Len() != 0 {
				t.Fatalf("exit %d (want %d), stdout %q, stderr %q (want %q)", code, tt.want, stdout.String(), stderr.String(), tt.wantErr)
			}
		})
	}
}
