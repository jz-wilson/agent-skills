package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jz-wilson/portview/internal/scan"
)

const (
	detailLines = 4 // content lines inside the detail box
	// Rows the table does not own: title, detail box (content + 2 border), status, help.
	chromeLines = 1 + detailLines + 2 + 1 + 1
)

var (
	accent     = lipgloss.Color("212")
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	faintStyle = lipgloss.NewStyle().Faint(true)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
)

// scannedMsg carries one scan result; refreshMsg asks for the next one.
type scannedMsg struct {
	at    time.Time
	socks []scan.Socket
	err   error
}

type refreshMsg struct{}

type keyMap struct {
	Up, Down, Filter, Accept, Clear, Quit key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Accept: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
		Clear:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// helpKeys adapts a binding list to help.KeyMap, so each mode shows its own keys.
type helpKeys []key.Binding

func (h helpKeys) ShortHelp() []key.Binding  { return h }
func (h helpKeys) FullHelp() [][]key.Binding { return [][]key.Binding{h} }

type model struct {
	root     string
	interval time.Duration

	socks   []scan.Socket // latest scan
	visible []scan.Socket // socks after the filter; row i of the table is visible[i]
	err     error
	at      time.Time

	table     table.Model
	detail    viewport.Model
	filter    textinput.Model
	filtering bool
	help      help.Model
	keys      keyMap

	width, height int
}

func newModel(root string, interval time.Duration) model {
	t := table.New(
		table.WithColumns(columns(80)),
		table.WithFocused(true),
		table.WithStyles(table.Styles{
			Header:   titleStyle.Padding(0, 1),
			Cell:     lipgloss.NewStyle().Padding(0, 1),
			Selected: lipgloss.NewStyle().Bold(true).Foreground(accent),
		}),
	)
	f := textinput.New()
	f.Prompt = "/ "
	f.Placeholder = "port, address, pid, process"
	f.SetVirtualCursor(true)
	d := viewport.New()
	d.SoftWrap = true
	return model{root: root, interval: interval, table: t, detail: d, filter: f, help: help.New(), keys: newKeyMap()}
}

func scanCmd(root string) tea.Cmd {
	return func() tea.Msg {
		socks, err := scan.Listening(root)
		return scannedMsg{at: time.Now(), socks: socks, err: err}
	}
}

func (m model) Init() tea.Cmd { return scanCmd(m.root) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case scannedMsg:
		m.err, m.at = msg.err, msg.at
		if msg.err == nil {
			m.socks = msg.socks
			m.rebuild()
		}
		// Schedule the next scan only after this one lands, so scans never pile up.
		return m, tea.Tick(m.interval, func(time.Time) tea.Msg { return refreshMsg{} })

	case refreshMsg:
		return m, scanCmd(m.root)

	case tea.KeyPressMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Filter):
			m.filtering = true
			m.table.Blur()
			return m, m.filter.Focus()
		case key.Matches(msg, m.keys.Clear):
			m.filter.Reset()
			m.rebuild()
			return m, nil
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		m.syncDetail()
		return m, cmd
	}
	return m, nil
}

func (m model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		return m, tea.Quit
	case key.Matches(msg, m.keys.Accept), key.Matches(msg, m.keys.Clear):
		if key.Matches(msg, m.keys.Clear) {
			m.filter.Reset()
		}
		m.filtering = false
		m.filter.Blur()
		m.table.Focus()
		m.rebuild()
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.rebuild() // live filtering
	return m, cmd
}

// rebuild reapplies the filter to the latest scan and keeps the cursor on the same socket.
func (m *model) rebuild() {
	var selected *scan.Socket
	if s, ok := m.selected(); ok {
		selected = &s
	}
	m.visible = m.visible[:0:0]
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	rows := make([]table.Row, 0, len(m.socks))
	for _, s := range m.socks {
		row := table.Row{strconv.Itoa(s.Port), s.Addr, pidText(s), processText(s)}
		if q != "" && !strings.Contains(strings.ToLower(strings.Join(row, "\x00")+"\x00"+s.Cmdline), q) {
			continue
		}
		m.visible = append(m.visible, s)
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
	cursor := min(m.table.Cursor(), max(len(rows)-1, 0))
	for i, s := range m.visible {
		if selected != nil && s.Proto == selected.Proto && s.Addr == selected.Addr && s.Port == selected.Port {
			cursor = i
		}
	}
	m.table.SetCursor(cursor)
	m.syncDetail()
}

func (m model) selected() (scan.Socket, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.visible) {
		return scan.Socket{}, false
	}
	return m.visible[i], true
}

func (m *model) syncDetail() {
	m.detail.SetContent(detailText(m.selected()))
	m.detail.GotoTop()
}

func detailText(s scan.Socket, ok bool) string {
	if !ok {
		return "No listening sockets match."
	}
	cmd := s.Cmdline
	switch {
	case s.PID == 0:
		cmd = "(owner not inspectable: run as the same user or root)"
	case cmd == "":
		cmd = "(empty: kernel thread or hidden cmdline)"
	}
	return fmt.Sprintf("Port     %d/%s\nAddress  %s\nProcess  %s (pid %s)\nCommand  %s",
		s.Port, s.Proto, s.Addr, processText(s), pidText(s), cmd)
}

// columns sizes the table to the terminal; ADDRESS takes whatever the fixed columns leave.
func columns(width int) []table.Column {
	const port, pid, proc, padding = 6, 7, 20, 2 * 4
	addr := max(width-port-pid-proc-padding, 12)
	return []table.Column{
		{Title: "PORT", Width: port},
		{Title: "ADDRESS", Width: addr},
		{Title: "PID", Width: pid},
		{Title: "PROCESS", Width: proc},
	}
}

func (m *model) layout() {
	m.table.SetColumns(columns(m.width))
	m.table.SetWidth(m.width)
	m.table.SetHeight(max(m.height-chromeLines, 3))
	m.detail.SetWidth(max(m.width-4, 10)) // box border (2) + padding (2)
	m.detail.SetHeight(detailLines)
	m.help.SetWidth(m.width)
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}
	title := titleStyle.Render("portview") + faintStyle.Render(fmt.Sprintf("  listening TCP · refresh %s", m.interval))
	box := boxStyle.Width(m.width).Render(m.detail.View())

	var status string
	switch {
	case m.err != nil:
		status = prefix(LevelError, true) + " " + m.err.Error()
	case m.filtering:
		status = m.filter.View()
	case m.filter.Value() != "":
		status = faintStyle.Render(fmt.Sprintf("filter %q: %d of %d", m.filter.Value(), len(m.visible), len(m.socks)))
	default:
		status = faintStyle.Render(fmt.Sprintf("%d sockets · updated %s", len(m.socks), m.at.Format("15:04:05")))
	}

	keys := helpKeys{m.keys.Up, m.keys.Down, m.keys.Filter, m.keys.Clear, m.keys.Quit}
	if m.filtering {
		keys = helpKeys{m.keys.Accept, m.keys.Clear}
	}
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, title, m.table.View(), box, status, m.help.View(keys)))
	v.AltScreen = true
	return v
}
