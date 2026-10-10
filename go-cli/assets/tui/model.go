package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	helpStyle   = lipgloss.NewStyle().Faint(true)
)

type model struct {
	items  []string
	cursor int
	chosen string
	done   bool // set on quit so the last frame is cleared, not left on screen
}

func newModel(items []string) model { return model{items: items} }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "space":
			m.chosen = m.items[m.cursor]
			m.done = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	var b strings.Builder
	for i, item := range m.items {
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("> " + item))
		} else {
			b.WriteString("  " + item)
		}
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("↑/↓ move • enter pick • q quit"))
	return tea.NewView(b.String())
}
