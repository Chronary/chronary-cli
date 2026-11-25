package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	xansi "github.com/charmbracelet/x/ansi"
)

// TableDef defines a table's columns and how to extract cell values.
type TableDef struct {
	Headers []string
	Rows    [][]string
}

// RenderTable prints a Lip Gloss table to stdout.
func RenderTable(def TableDef, noColor bool) {
	if len(def.Rows) == 0 {
		fmt.Println("No results.")
		return
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	if noColor {
		headerStyle = headerStyle.UnsetForeground()
		cellStyle = cellStyle.UnsetForeground()
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		Headers(def.Headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	for _, row := range def.Rows {
		t.Row(row...)
	}

	fmt.Println(t)
}

// Truncate shortens a string to maxLen terminal columns, adding "..." if
// needed. It measures display width, not bytes, so non-ASCII text is never cut
// mid-character and wide (CJK) text that fits is left alone (#2722).
func Truncate(s string, maxLen int) string {
	return xansi.Truncate(strings.TrimSpace(s), maxLen, "...")
}
