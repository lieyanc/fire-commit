package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Enter):
			if len(m.messages) > 0 {
				m.phase = PhaseSelect
				return m, nil
			}
		case key.Matches(msg, keys.Quit):
			m.cancel()
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) viewLoading() string {
	var b strings.Builder
	contentWidth := m.contentWidth()

	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("%s Generating commit messages (%d/%d finished, %d ready)\n",
		m.spinner.View(), m.finished, m.total, m.completed))

	for i := 0; i < m.total; i++ {
		preview := compactPreview(m.partial[i])
		switch {
		case m.slotFailed[i]:
			b.WriteString("\n")
			b.WriteString(renderWrappedLine("  ✗ ", "  "+errorStyle.Render("✗")+" ", "request failed", dimStyle, contentWidth))
		case m.slotDone[i]:
			b.WriteString("\n")
			if preview == "" {
				preview = "(empty response)"
			}
			b.WriteString(renderWrappedLine("  ✓ ", "  "+successStyle.Render("✓")+" ", preview, dimStyle, contentWidth))
		case preview != "":
			b.WriteString("\n")
			b.WriteString(renderWrappedLine("  ~ ", "  "+selectedStyle.Render("~")+" ", preview, dimStyle, contentWidth))
		default:
			b.WriteString("\n")
			b.WriteString(renderWrappedLine("    ", "    ", "...", dimStyle, contentWidth))
		}
	}

	if m.stat != "" && m.finished == 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(wrapText(m.stat, contentWidth)))
	}

	if len(m.messages) > 0 {
		b.WriteString(helpStyle.Render("\n\n  enter select ready messages • q quit"))
	} else {
		b.WriteString(helpStyle.Render("\n\n  q quit"))
	}

	return m.renderBox(b.String())
}

func compactPreview(s string) string {
	if s == "" {
		return ""
	}
	oneLine := strings.ReplaceAll(s, "\n", " ")
	return strings.Join(strings.Fields(oneLine), " ")
}
