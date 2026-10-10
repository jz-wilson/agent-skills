package scan

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestListeningFakeProc(t *testing.T) {
	got, err := Listening(filepath.Join("testdata", "proc"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Socket{
		{Proto: "tcp6", Addr: "::", Port: 22, PID: 200, Process: "sshd", Cmdline: "sshd: /usr/sbin/sshd -D [listener]", Inode: 1005},
		{Proto: "tcp", Addr: "127.0.0.53", Port: 53, Inode: 1004},
		{Proto: "tcp6", Addr: "::1", Port: 631, Inode: 1006},
		{Proto: "tcp", Addr: "127.0.0.1", Port: 5432, PID: 150, Process: "postgres", Cmdline: "/usr/lib/postgresql/16/bin/postgres -D /var/lib/postgresql/16/main", Inode: 1002},
		{Proto: "tcp", Addr: "0.0.0.0", Port: 8765, PID: 100, Process: "python3", Cmdline: "python3 -m http.server 8765", Inode: 1001},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sockets, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("socket %d:\n got  %+v\n want %+v", i, got[i], want[i])
		}
	}
}

func TestListeningMissingProc(t *testing.T) {
	_, err := Listening(filepath.Join("testdata", "nope"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want ErrNotExist", err)
	}
}

func TestParseAddr(t *testing.T) {
	for _, tc := range []struct {
		in   string
		addr string
		port int
	}{
		{"0100007F:1F90", "127.0.0.1", 8080},
		{"00000000000000000000000001000000:0050", "::1", 80},
		{"0000000000000000FFFF00000100007F:0050", "::ffff:127.0.0.1", 80},
	} {
		addr, port, err := parseAddr(tc.in)
		if err != nil || addr != tc.addr || port != tc.port {
			t.Errorf("parseAddr(%q) = %q, %d, %v; want %q, %d", tc.in, addr, port, err, tc.addr, tc.port)
		}
	}
	for _, bad := range []string{"", "nocolon", "0100007F:ZZ", "0100:0050"} {
		if _, _, err := parseAddr(bad); err == nil {
			t.Errorf("parseAddr(%q) should fail", bad)
		}
	}
}
