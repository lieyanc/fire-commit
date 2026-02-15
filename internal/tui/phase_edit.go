package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Escape):
			m.editing = false
			m.editArea.Blur()
			m.phase = PhaseSelect
			return m, nil
		case key.Matches(msg, keys.CtrlC):
			m.cancel()
			return m, tea.Quit
		case key.Matches(msg, keys.Save):
			value := strings.TrimSpace(m.editArea.Value())
			if value != "" {
				m.messages[m.cursor] = value
			}
			m.editing = false
			m.editArea.Blur()
			m.confirmCursor = confirmCommitOnly
			m.phase = PhaseConfirm
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editArea, cmd = m.editArea.Update(msg)
	return m, cmd
}

func (m Model) viewEdit() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")
	b.WriteString("Edit commit message:\n\n")
	b.WriteString(m.editArea.View())
	b.WriteString(helpStyle.Render("\n\n  ctrl+s save • esc cancel"))

	return m.renderBox(b.String())
}
