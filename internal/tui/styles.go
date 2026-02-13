package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Fire theme colors
	colorFire      = lipgloss.Color("#FF6B35")
	colorWarm      = lipgloss.Color("#FFB347")
	colorEmber     = lipgloss.Color("#FF4500")
	colorDim       = lipgloss.Color("#666666")
	colorText      = lipgloss.Color("#EEEEEE")
	colorSuccess   = lipgloss.Color("#73D216")
	colorError     = lipgloss.Color("#EF2929")
	colorHighlight = lipgloss.Color("#FFA07A")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorFire).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorFire).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorWarm).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	cursorStyle = lipgloss.NewStyle().
			Foreground(colorEmber).
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
