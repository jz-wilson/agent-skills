package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/jz-wilson/portview/internal/scan"
)

const fakeProc = "internal/scan/testdata/proc"

func fakeSockets(t *testing.T) []scan.Socket {
	t.Helper()
	socks, err := scan.Listening(fakeProc)
	if err != nil {
		t.Fatal(err)
	}
	return socks
}

// loaded returns a sized model that has already received a scan result.
func loaded(t *testing.T) model {
	t.Helper()
	var m tea.Model = newModel(fakeProc, time.Second)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m, _ = m.Update(scannedMsg{at: time.Unix(0, 0), socks: fakeSockets(t)})
	return m.(model)
}

func press(m model, k tea.KeyPressMsg) model {
	next, _ := m.Update(k)
	return next.(model)
}

func typed(m model, s string) model {
	for _, r := range s {
		m = press(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestScanResultFillsTable(t *testing.T) {
	m := loaded(t)
	if got := len(m.table.Rows()); got != 5 {
		t.Fatalf("rows = %d, want 5", got)
	}
	if sel, _ := m.selected(); sel.Port != 22 {
		t.Fatalf("first row port = %d, want 22 (sorted)", sel.Port)
	}
}

func TestNavigationMovesDetail(t *testing.T) {
	m := loaded(t)
	m = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = press(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	m = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = press(m, tea.KeyPressMsg{Code: tea.KeyUp})
	sel, _ := m.selected()
	if sel.Port != 631 {
		t.Fatalf("selected port = %d, want 631", sel.Port)
	}
	if got := m.detail.GetContent(); !strings.Contains(got, "not inspectable") {
		t.Fatalf("detail for unowned socket = %q", got)
	}
	m = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.detail.GetContent(); !strings.Contains(got, "postgres -D /var/lib/postgresql/16/main") {
		t.Fatalf("detail should show the command line, got %q", got)
	}
}

func TestFilter(t *testing.T) {
	m := loaded(t)
	m = press(m, tea.KeyPressMsg{Code: '/', Text: "/"})
	if !m.filtering {
		t.Fatal("/ should start filtering")
	}
	m = typed(m, "zq") // typing q must filter, not quit
	if !m.filtering || len(m.visible) != 0 {
		t.Fatalf("q while filtering: filtering=%v visible=%d", m.filtering, len(m.visible))
	}
	m = press(press(m, tea.KeyPressMsg{Code: tea.KeyBackspace}), tea.KeyPressMsg{Code: tea.KeyBackspace})
	m = typed(m, "post")
	if len(m.visible) != 1 || m.visible[0].Process != "postgres" {
		t.Fatalf("visible = %+v, want only postgres", m.visible)
	}
	m = press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.filtering || len(m.visible) != 1 {
		t.Fatalf("enter should keep the filter and leave filter mode: filtering=%v visible=%d", m.filtering, len(m.visible))
	}
	m = press(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if len(m.visible) != 5 || m.filter.Value() != "" {
		t.Fatalf("esc should clear the filter: visible=%d value=%q", len(m.visible), m.filter.Value())
	}
}

func TestFilterMatchesCommandLine(t *testing.T) {
	m := loaded(t)
	m = press(m, tea.KeyPressMsg{Code: '/', Text: "/"})
	m = typed(m, "http.server")
	if len(m.visible) != 1 || m.visible[0].Port != 8765 {
		t.Fatalf("visible = %+v, want port 8765", m.visible)
	}
}

func TestQuit(t *testing.T) {
	_, cmd := loaded(t).Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("q should return tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q returned %T, want tea.QuitMsg", cmd())
	}
}

func TestRefreshKeepsSelectionAndSchedulesNextScan(t *testing.T) {
	m := loaded(t)
	m = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	m = press(m, tea.KeyPressMsg{Code: tea.KeyDown}) // port 631
	// A new scan where a socket appears above the selection.
	extra := append([]scan.Socket{{Proto: "tcp", Addr: "0.0.0.0", Port: 1}}, m.socks...)
	next, cmd := m.Update(scannedMsg{at: time.Unix(1, 0), socks: extra})
	m = next.(model)
	if sel, _ := m.selected(); sel.Port != 631 {
		t.Fatalf("selection moved to port %d after refresh, want 631", sel.Port)
	}
	if cmd == nil {
		t.Fatal("a scan result should schedule the next refresh")
	}
	if _, cmd := m.Update(refreshMsg{}); cmd == nil {
		t.Fatal("refreshMsg should start a scan")
	}
}

func TestScanErrorKeepsOldRowsAndShowsError(t *testing.T) {
	m := loaded(t)
	next, _ := m.Update(scannedMsg{err: errors.New("boom")})
	m = next.(model)
	if len(m.visible) != 5 {
		t.Fatalf("rows dropped on error: %d", len(m.visible))
	}
	if v := m.View().Content; !strings.Contains(v, "boom") {
		t.Fatalf("view should show the error:\n%s", v)
	}
}

func TestViewFitsTerminal(t *testing.T) {
	v := loaded(t).View()
	if !v.AltScreen {
		t.Fatal("TUI should use the alt screen")
	}
	lines := strings.Split(v.Content, "\n")
	if len(lines) > 24 {
		t.Fatalf("view is %d lines in a 24 line terminal:\n%s", len(lines), v.Content)
	}
}

// teatest runs the real program loop, including the scan command and the refresh tick.
func TestProgramFilters(t *testing.T) {
	tm := teatest.NewTestModel(t, newModel(fakeProc, 50*time.Millisecond), teatest.WithInitialTermSize(100, 24))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return bytes.Contains(b, []byte("python3")) })
	tm.Send(tea.KeyPressMsg{Code: '/', Text: "/"})
	tm.Type("sshd")
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(model)
	if len(final.visible) != 1 || final.visible[0].Port != 22 {
		t.Fatalf("visible = %+v, want only sshd on 22", final.visible)
	}
}

// Piped output must be plain TSV with "-" for unknown owners: a buffer is never a terminal.
func TestNonInteractiveTSVAndJSON(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--root", fakeProc}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	want := "22\t::\t200\tsshd\n53\t127.0.0.53\t-\t-\n631\t::1\t-\t-\n5432\t127.0.0.1\t150\tpostgres\n8765\t0.0.0.0\t100\tpython3\n"
	if out.String() != want {
		t.Fatalf("tsv =\n%q\nwant\n%q", out.String(), want)
	}

	out.Reset()
	if code := run(context.Background(), []string{"--json", "--root", fakeProc}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.HasPrefix(out.String(), `[{"proto":"tcp6","addr":"::","port":22,"pid":200,`) {
		t.Fatalf("json = %s", out.String())
	}
}

func TestUsageErrorExitsTwo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(context.Background(), []string{"--nope"}, strings.NewReader(""), &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
}
