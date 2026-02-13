package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lieyanc/fire-commit/internal/git"
)

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Escape):
			m.phase = PhaseSelect
			return m, nil
		case key.Matches(msg, keys.Tab), key.Matches(msg, keys.Down):
			m.confirmCursor = (m.confirmCursor + 1) % 3
		case key.Matches(msg, keys.Up):
			m.confirmCursor = (m.confirmCursor + 2) % 3
		case key.Matches(msg, keys.Enter):
			switch m.confirmCursor {
			case 0: // Commit only
				m.wantPush = false
				m.phase = PhaseCommitting
				return m, tea.Batch(m.spinner.Tick, m.doCommit())
			case 1: // Commit & Push
				m.wantPush = true
				m.phase = PhaseCommitting
				return m, tea.Batch(m.spinner.Tick, m.doCommit())
			case 2: // Cancel
				m.phase = PhaseSelect
				return m, nil
			}
		case key.Matches(msg, keys.Quit):
			m.cancel()
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) viewConfirm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")
	b.WriteString("Commit message:\n\n")
	b.WriteString(highlightStyle.Render("  " + m.messages[m.cursor]))
	b.WriteString("\n\n")

	options := []string{"Commit only", "Commit & Push", "Cancel"}
	for i, opt := range options {
		if i == m.confirmCursor {
			b.WriteString(cursorStyle.Render("  > "))
			b.WriteString(selectedStyle.Render(opt))
		} else {
			b.WriteString("    ")
			b.WriteString(normalStyle.Render(opt))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  ↑/↓/tab select • enter confirm • esc back"))

	return boxStyle.Render(b.String())
}

func (m Model) doCommit() tea.Cmd {
	msg := m.messages[m.cursor]
	return func() tea.Msg {
		err := git.Commit(msg)
		return commitDoneMsg{err: err}
	}
}

func (m Model) doGitPush() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return pushDoneMsg{err: err}
	}
}

func (m Model) updateCommitting(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case commitDoneMsg:
		if msg.err != nil {
			m.commitErr = msg.err
			m.phase = PhaseDone
			return m, nil
		}
		m.committed = true
		if m.wantPush {
			return m, tea.Batch(m.spinner.Tick, m.doGitPush())
		}
		m.phase = PhaseDone
		return m, nil

	case pushDoneMsg:
		m.pushErr = msg.err
		m.pushed = msg.err == nil
		m.phase = PhaseDone
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) viewCommitting() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")

	if m.committed {
		b.WriteString(successStyle.Render("✓ Committed"))
		b.WriteString("\n")
		b.WriteString(m.spinner.View() + " Pushing...")
	} else {
		b.WriteString(m.spinner.View() + " Committing...")
	}

	return boxStyle.Render(b.String())
}

func (m Model) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		_ = msg
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewDone() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")

	if m.commitErr != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.commitErr)))
		b.WriteString(helpStyle.Render("\n\n  Press any key to exit"))
		return boxStyle.Render(b.String())
	}

	if m.committed {
		b.WriteString(successStyle.Render("✓ Committed: "))
		b.WriteString(highlightStyle.Render(m.messages[m.cursor]))
		b.WriteString("\n")
	}

	if m.wantPush {
		if m.pushErr != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Push failed: %s", m.pushErr)))
		} else if m.pushed {
			branch, _ := git.CurrentBranch()
			b.WriteString(successStyle.Render(fmt.Sprintf("✓ Pushed to origin/%s", branch)))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  Press any key to exit"))

	return boxStyle.Render(b.String())
}
