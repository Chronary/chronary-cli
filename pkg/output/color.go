package output

import "github.com/charmbracelet/lipgloss"

var (
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// ColorStatus returns a colored string for resource statuses.
func ColorStatus(status string, noColor bool) string {
	if noColor {
		return status
	}
	switch status {
	case "active", "ok":
		return green.Render(status)
	case "paused":
		return yellow.Render(status)
	case "decommissioned", "error":
		return red.Render(status)
	default:
		return dim.Render(status)
	}
}
