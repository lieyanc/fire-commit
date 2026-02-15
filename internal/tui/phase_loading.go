package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case startResultsMsg:
		m.resultCh = msg.ch
		return m, waitForMessage(m.resultCh)

	case messageReadyMsg:
		if msg.err != nil {
			// If this is the very first result and it's a provider error,
			// treat it as fatal (e.g. bad API key).
			if m.completed == 0 && m.resultCh == nil {
				m.commitErr = msg.err
				m.phase = PhaseDone
				return m, nil
			}
			// Otherwise, shrink total — this slot failed, skip it.
			m.total--
			if m.total <= 0 {
				m.commitErr = fmt.Errorf("all LLM requests failed: %w", msg.err)
				m.phase = PhaseDone
				return m, nil
			}
			return m, waitForMessage(m.resultCh)
		}

		m.messages[msg.index] = msg.content
		m.completed++

		if m.completed >= m.total {
			m.messages = compactMessages(m.messages)
			if len(m.messages) == 0 {
				m.commitErr = fmt.Errorf("LLM returned no commit messages")
				m.phase = PhaseDone
				return m, nil
			}
			m.phase = PhaseSelect
			return m, nil
		}
		return m, waitForMessage(m.resultCh)

	case allDoneMsg:
		m.messages = compactMessages(m.messages)
		if len(m.messages) == 0 {
			m.commitErr = fmt.Errorf("LLM returned no commit messages")
			m.phase = PhaseDone
			return m, nil
		}
		m.phase = PhaseSelect
		return m, nil
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

	b.WriteString(fmt.Sprintf("%s Generating commit messages (%d/%d)...\n",
		m.spinner.View(), m.completed, m.total))

	// Show completed messages
	for i, msg := range m.messages {
		if msg != "" {
			b.WriteString("\n")
			b.WriteString(renderWrappedLine("  ✓ ", "  "+successStyle.Render("✓")+" ", msg, dimStyle, contentWidth))
		} else if i < m.total {
			b.WriteString("\n")
			b.WriteString(renderWrappedLine("    ", "    ", "...", dimStyle, contentWidth))
		}
	}

	if m.stat != "" && m.completed == 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(wrapText(m.stat, contentWidth)))
	}

	return m.renderBox(b.String())
}

// compactMessages removes empty strings from the messages slice.
func compactMessages(msgs []string) []string {
	var result []string
	for _, m := range msgs {
		if m != "" {
			result = append(result, m)
		}
	}
	return result
}
