package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.messages)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			m.phase = PhaseConfirm
			return m, nil
		case key.Matches(msg, keys.Edit):
			m.editArea.SetValue(m.messages[m.cursor])
			m.editing = true
			m.phase = PhaseEdit
			return m, m.editArea.Focus()
		case key.Matches(msg, keys.Regen):
			m.streamBuf.Reset()
			m.streamDone = false
			m.messages = nil
			m.cursor = 0
			m.phase = PhaseLoading
			return m, tea.Batch(m.spinner.Tick, m.startGeneration())
		case key.Matches(msg, keys.Quit):
			m.cancel()
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewSelect() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")
	b.WriteString("Select a commit message:\n\n")

	for i, msg := range m.messages {
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("  > "))
			b.WriteString(selectedStyle.Render(msg))
		} else {
			b.WriteString("    ")
			b.WriteString(normalStyle.Render(msg))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render(fmt.Sprintf("\n  ↑/↓ select • enter confirm • e edit • r regen • q quit")))

	return boxStyle.Render(b.String())
}
