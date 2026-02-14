package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Brand color palette — based on primary #018EEE
	colorPrimary   = lipgloss.Color("#018EEE") // Brand blue
	colorAccent    = lipgloss.Color("#52B0FF") // Light blue — selected, spinner
	colorDeep      = lipgloss.Color("#005FBB") // Deep blue — cursor, emphasis
	colorHighlight = lipgloss.Color("#7EC8FF") // Pale blue — highlights
	colorDim       = lipgloss.Color("#666666") // Gray — secondary text
	colorText      = lipgloss.Color("#EEEEEE") // Light gray — body text
	colorSuccess   = lipgloss.Color("#2ECC71") // Green — success
	colorError     = lipgloss.Color("#EF2929") // Red — error

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorDeep).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			MarginTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	highlightStyle = lipgloss.NewStyle().
			Foreground(colorHighlight)
)
