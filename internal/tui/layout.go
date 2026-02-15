package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	fallbackContentWidth = 72
	minContentWidth      = 32
	minInputWidth        = 20
)

func (m *Model) resizeInputs() {
	contentWidth := m.contentWidth()

	editWidth := contentWidth - 4
	if editWidth < minInputWidth {
		editWidth = minInputWidth
	}
	m.editArea.SetWidth(editWidth)

	editHeight := 5
	if m.height > 0 {
		// Keep the editor compact while still using extra rows on larger terminals.
		editHeight = m.height / 3
		if editHeight < 4 {
			editHeight = 4
		}
		if editHeight > 10 {
			editHeight = 10
		}
	}
	m.editArea.SetHeight(editHeight)

	tagWidth := contentWidth / 2
	if tagWidth > 30 {
		tagWidth = 30
	}
	if tagWidth < 12 {
		tagWidth = 12
	}
	m.tagInput.Width = tagWidth
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return fallbackContentWidth
	}
	width := m.width - boxStyle.GetHorizontalFrameSize() - 2
	if width < minContentWidth {
		return minContentWidth
	}
	return width
}

func (m Model) renderBox(content string) string {
	style := boxStyle
	if m.width > 0 {
		maxWidth := m.width - 2
		if maxWidth > 0 {
			style = style.MaxWidth(maxWidth)
		}
	}
	return style.Render(content)
}

func wrapText(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}

	var b strings.Builder
	lineWidth := 0

	for _, r := range text {
		switch r {
		case '\n':
			b.WriteRune(r)
			lineWidth = 0
			continue
		case '\t':
			r = ' '
		}

		rw := lipgloss.Width(string(r))
		if rw <= 0 {
			continue
		}

		if lineWidth+rw > width {
			b.WriteByte('\n')
			lineWidth = 0
			if r == ' ' {
				continue
			}
		}

		b.WriteRune(r)
		lineWidth += rw
	}

	return strings.TrimRight(b.String(), " ")
}

func renderWrappedLine(prefix, prefixView, text string, style lipgloss.Style, width int) string {
	textWidth := width - lipgloss.Width(prefix)
	if textWidth < 8 {
		textWidth = 8
	}

	wrapped := wrapText(text, textWidth)
	lines := strings.Split(wrapped, "\n")
	indent := strings.Repeat(" ", lipgloss.Width(prefix))

	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == 0 {
			b.WriteString(prefixView)
		} else {
			b.WriteString(indent)
		}
		b.WriteString(style.Render(line))
	}
	return b.String()
}
