package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Yellow
	errorColor     = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	bgColor        = lipgloss.Color("#1F2937") // Dark gray

	// Base styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	activeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 1)

	// Status styles
	statusActiveStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	statusPendingStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	statusMutedStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Tab styles
	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(mutedColor)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(primaryColor).
			Bold(true).
			Underline(true)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// List item styles
	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(secondaryColor).
				SetString("▸ ")

	// Progress bar
	progressFullStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Header/Footer
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Width(80)

	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 1)
)

// RenderProgressBar renders a progress bar
func RenderProgressBar(percent float64, width int) string {
	filled := int(percent * float64(width))
	empty := width - filled

	bar := progressFullStyle.Render(repeat("█", filled)) +
		progressEmptyStyle.Render(repeat("░", empty))

	return bar
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// StatusIcon returns an icon for a status
func StatusIcon(status string) string {
	switch status {
	case "active", "running", "enabled", "valid":
		return statusActiveStyle.Render("●")
	case "pending", "waiting", "modified":
		return statusPendingStyle.Render("○")
	case "error", "failed", "invalid":
		return statusErrorStyle.Render("✗")
	case "completed", "done":
		return statusActiveStyle.Render("✓")
	default:
		return statusMutedStyle.Render("○")
	}
}

// FormatDuration formats a duration string
func FormatDuration(seconds int64) string {
	if seconds < 60 {
		return "<1m"
	}
	if seconds < 3600 {
		return lipgloss.NewStyle().Render(repeat(" ", 0)) + 
			statusMutedStyle.Render(string(rune(seconds/60+'0'))+"m")
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if hours < 24 {
		return statusMutedStyle.Render(string(rune(hours+'0'))+"h"+string(rune(mins+'0'))+"m")
	}
	days := hours / 24
	return statusMutedStyle.Render(string(rune(days+'0'))+"d")
}
