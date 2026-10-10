package main

import (
	"encoding/json"
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

func printJSON(w io.Writer, path string, rows []Row, outdated, failed int) error {
	if rows == nil {
		rows = []Row{} // [] not null
	}
	return json.NewEncoder(w).Encode(struct {
		File     string `json:"file"`
		Outdated int    `json:"outdated"`
		Failed   int    `json:"failed"`
		Modules  []Row  `json:"modules"`
	}{path, outdated, failed, rows})
}

var statusColor = map[string]lipgloss.Style{
	statusCurrent: lipgloss.NewStyle().Foreground(lipgloss.Green),
	statusPatch:   lipgloss.NewStyle().Foreground(lipgloss.Yellow),
	statusMinor:   lipgloss.NewStyle().Foreground(lipgloss.Yellow),
	statusMajor:   lipgloss.NewStyle().Foreground(lipgloss.Red),
	statusError:   lipgloss.NewStyle().Foreground(lipgloss.Red),
}

// printRows is data, so it goes to stdout: a styled table for a human, TSV for a pipe.
func printRows(w io.Writer, rows []Row) {
	if len(rows) == 0 {
		return
	}
	if !isTerminal(w) {
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Module, r.Current, orDash(r.Latest), status(r))
		}
		return
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1)
	cell := lipgloss.NewStyle().Padding(0, 1)
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("MODULE", "CURRENT", "LATEST", "STATUS").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			if col == 3 {
				return statusColor[rows[row].Status].Padding(0, 1)
			}
			return cell
		})
	for _, r := range rows {
		t.Row(r.Module, r.Current, orDash(r.Latest), truncate(status(r), maxCell))
	}
	lipgloss.Fprintln(w, t)
}

// status carries the failure reason so a pipe or a glance shows why, not just "error".
func status(r Row) string {
	if r.Error != "" {
		return r.Status + ": " + r.Error
	}
	return r.Status
}

// maxCell caps free-text cells in the terminal table: one long error would
// otherwise widen the table past the terminal and wrap every row. TSV and
// JSON keep the full text.
const maxCell = 30

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
