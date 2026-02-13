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

	case startStreamMsg:
		m.phase = PhaseStreaming
		m.streamCh = msg.ch
		// Process first chunk
		if msg.first.Err != nil {
			m.commitErr = msg.first.Err
			m.phase = PhaseDone
			return m, nil
		}
		if msg.first.Done {
			m.messages = parseMessages(m.streamBuf.String())
			m.phase = PhaseSelect
			return m, nil
		}
		m.streamBuf.WriteString(msg.first.Content)
		return m, waitForChunk(m.streamCh)

	case streamChunkMsg:
		if msg.chunk.Err != nil {
			m.commitErr = msg.chunk.Err
			m.phase = PhaseDone
			return m, nil
		}
		if msg.chunk.Done {
			m.messages = parseMessages(m.streamBuf.String())
			if len(m.messages) == 0 {
				m.commitErr = fmt.Errorf("LLM returned no commit messages")
				m.phase = PhaseDone
				return m, nil
			}
			m.phase = PhaseSelect
			return m, nil
		}
		m.phase = PhaseStreaming
		m.streamBuf.WriteString(msg.chunk.Content)
		return m, waitForChunk(m.streamCh)
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) viewLoading() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")

	if m.phase == PhaseStreaming {
		b.WriteString(m.spinner.View() + " Generating commit messages...\n\n")
		// Show streaming content
		content := m.streamBuf.String()
		if content != "" {
			b.WriteString(dimStyle.Render(content))
		}
	} else {
		b.WriteString(m.spinner.View() + " Analyzing changes...\n")
		if m.stat != "" {
			b.WriteString(dimStyle.Render(m.stat))
		}
	}

	return boxStyle.Render(b.String())
}

// parseMessages splits the LLM response into individual commit messages.
func parseMessages(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	var messages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip common prefixes like "1. ", "- ", "* "
		line = stripListPrefix(line)
		if line != "" {
			messages = append(messages, line)
		}
	}
	return messages
}

func stripListPrefix(s string) string {
	// Strip numbered list: "1. ", "2) ", etc.
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if (c == '.' || c == ')') && i > 0 {
			return strings.TrimSpace(s[i+1:])
		}
		break
	}
	// Strip bullet: "- ", "* "
	if len(s) > 2 && (s[0] == '-' || s[0] == '*') && s[1] == ' ' {
		return strings.TrimSpace(s[2:])
	}
	return s
}
