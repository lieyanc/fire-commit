package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lieyanc/fire-commit/internal/git"
)

const (
	confirmCommitAndPush = iota
	confirmCommitOnly
	confirmCancel

	confirmOptionCount = 3
)

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editingTag {
		return m.updateTagInput(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Escape):
			m.phase = PhaseSelect
			return m, nil
		case key.Matches(msg, keys.Tab), key.Matches(msg, keys.Down):
			m.confirmCursor = (m.confirmCursor + 1) % confirmOptionCount
		case key.Matches(msg, keys.Up):
			m.confirmCursor = (m.confirmCursor + confirmOptionCount - 1) % confirmOptionCount
		case key.Matches(msg, keys.Push):
			if m.confirmCursor == confirmCommitAndPush {
				m.confirmCursor = confirmCommitOnly
			} else {
				m.confirmCursor = confirmCommitAndPush
			}
		case key.Matches(msg, keys.Version):
			if m.versionTag != "" {
				// Clear existing tag
				m.versionTag = ""
			} else {
				// Enter tag editing mode
				m.editingTag = true
				m.tagInput.SetValue("")
				m.tagInput.Placeholder = git.LatestTag()
				if m.tagInput.Placeholder == "" {
					m.tagInput.Placeholder = "v1.0.0"
				}
				m.tagInput.Focus()
				return m, m.tagInput.Cursor.BlinkCmd()
			}
		case key.Matches(msg, keys.Enter):
			switch m.confirmCursor {
			case confirmCommitAndPush:
				m.wantPush = true
				m.phase = PhaseCommitting
				return m, tea.Batch(m.spinner.Tick, m.doCommit())
			case confirmCommitOnly:
				m.wantPush = false
				m.phase = PhaseCommitting
				return m, tea.Batch(m.spinner.Tick, m.doCommit())
			case confirmCancel:
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

func (m Model) updateTagInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			val := strings.TrimSpace(m.tagInput.Value())
			if val != "" {
				m.versionTag = val
			}
			m.editingTag = false
			m.tagInput.Blur()
			return m, nil
		case tea.KeyEscape:
			m.editingTag = false
			m.tagInput.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.tagInput, cmd = m.tagInput.Update(msg)
	return m, cmd
}

func (m Model) viewConfirm() string {
	var b strings.Builder
	contentWidth := m.contentWidth()

	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")
	b.WriteString("Commit message:\n\n")
	b.WriteString(renderWrappedLine("  ", "  ", m.messages[m.cursor], highlightStyle, contentWidth))
	b.WriteString("\n\n")

	// Version tag line
	if m.editingTag {
		b.WriteString("Version tag: ")
		b.WriteString(m.tagInput.View())
		b.WriteString("\n\n")
	} else if m.versionTag != "" {
		b.WriteString("Version tag: ")
		b.WriteString(selectedStyle.Render(m.versionTag))
		b.WriteString(dimStyle.Render("  (v to clear)"))
		b.WriteString("\n\n")
	} else {
		b.WriteString("Version tag: ")
		b.WriteString(dimStyle.Render("(none)"))
		b.WriteString("\n\n")
	}

	options := []string{"Commit & Push", "Commit only", "Cancel"}
	for i, opt := range options {
		if i == m.confirmCursor {
			b.WriteString(renderWrappedLine("  > ", cursorStyle.Render("  > "), opt, selectedStyle, contentWidth))
		} else {
			b.WriteString(renderWrappedLine("    ", "    ", opt, normalStyle, contentWidth))
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("\n  ↑/↓/tab select • enter confirm • p toggle push • v version • esc back • q quit"))

	return m.renderBox(b.String())
}

func (m Model) doCommit() tea.Cmd {
	msg := m.messages[m.cursor]
	return func() tea.Msg {
		err := git.Commit(msg)
		return commitDoneMsg{err: err}
	}
}

func (m Model) doTag() tea.Cmd {
	tag := m.versionTag
	return func() tea.Msg {
		err := git.Tag(tag)
		return tagDoneMsg{err: err}
	}
}

func (m Model) doGitPush() tea.Cmd {
	return func() tea.Msg {
		err := git.Push()
		return pushDoneMsg{err: err}
	}
}

func (m Model) doPushTag() tea.Cmd {
	tag := m.versionTag
	return func() tea.Msg {
		err := git.PushTag(tag)
		return tagPushDoneMsg{err: err}
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
		if m.versionTag != "" {
			return m, tea.Batch(m.spinner.Tick, m.doTag())
		}
		if m.wantPush {
			return m, tea.Batch(m.spinner.Tick, m.doGitPush())
		}
		m.phase = PhaseDone
		return m, nil

	case tagDoneMsg:
		if msg.err != nil {
			m.tagErr = msg.err
			// Tag failed, but still proceed to push commit if wanted
			if m.wantPush {
				return m, tea.Batch(m.spinner.Tick, m.doGitPush())
			}
			m.phase = PhaseDone
			return m, nil
		}
		m.tagged = true
		if m.wantPush {
			return m, tea.Batch(m.spinner.Tick, m.doGitPush())
		}
		m.phase = PhaseDone
		return m, nil

	case pushDoneMsg:
		m.pushErr = msg.err
		m.pushed = msg.err == nil
		// If tag was created successfully, also push the tag
		if m.tagged {
			return m, tea.Batch(m.spinner.Tick, m.doPushTag())
		}
		m.phase = PhaseDone
		return m, nil

	case tagPushDoneMsg:
		m.tagPushErr = msg.err
		m.tagPushed = msg.err == nil
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
	} else {
		b.WriteString(m.spinner.View() + " Committing...")
		return m.renderBox(b.String())
	}

	if m.versionTag != "" {
		if m.tagged {
			b.WriteString(successStyle.Render("✓ Tagged: " + m.versionTag))
			b.WriteString("\n")
		} else if m.tagErr != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Tag failed: %s", m.tagErr)))
			b.WriteString("\n")
		} else {
			b.WriteString(m.spinner.View() + " Creating tag " + m.versionTag + "...")
			return m.renderBox(b.String())
		}
	}

	if m.wantPush {
		if m.pushed {
			b.WriteString(successStyle.Render("✓ Pushed"))
			b.WriteString("\n")
		} else if m.pushErr != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Push failed: %s", m.pushErr)))
			b.WriteString("\n")
		} else {
			b.WriteString(m.spinner.View() + " Pushing...")
			return m.renderBox(b.String())
		}

		if m.tagged {
			if m.tagPushed {
				b.WriteString(successStyle.Render("✓ Tag pushed"))
				b.WriteString("\n")
			} else if m.tagPushErr != nil {
				b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Tag push failed: %s", m.tagPushErr)))
				b.WriteString("\n")
			} else {
				b.WriteString(m.spinner.View() + " Pushing tag " + m.versionTag + "...")
				return m.renderBox(b.String())
			}
		}
	}

	return m.renderBox(b.String())
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
	contentWidth := m.contentWidth()

	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")

	if m.commitErr != nil {
		b.WriteString(errorStyle.Render(wrapText(fmt.Sprintf("✗ Error: %s", m.commitErr), contentWidth)))
		b.WriteString(helpStyle.Render("\n\n  Press any key to exit"))
		return m.renderBox(b.String())
	}

	if m.committed {
		b.WriteString(successStyle.Render("✓ Committed:"))
		b.WriteString("\n")
		b.WriteString(renderWrappedLine("  ", "  ", m.messages[m.cursor], highlightStyle, contentWidth))
		b.WriteString("\n")
	}

	if m.versionTag != "" {
		if m.tagged {
			b.WriteString(successStyle.Render("✓ Tagged: " + m.versionTag))
		} else if m.tagErr != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("✗ Tag failed: %s", m.tagErr)))
		}
		b.WriteString("\n")
	}

	if m.wantPush {
		if m.pushErr != nil {
			b.WriteString(errorStyle.Render(wrapText(fmt.Sprintf("✗ Push failed: %s", m.pushErr), contentWidth)))
		} else if m.pushed {
			branch, _ := git.CurrentBranch()
			b.WriteString(successStyle.Render(fmt.Sprintf("✓ Pushed to origin/%s", branch)))
		}
		b.WriteString("\n")

		if m.tagged {
			if m.tagPushErr != nil {
				b.WriteString(errorStyle.Render(wrapText(fmt.Sprintf("✗ Tag push failed: %s", m.tagPushErr), contentWidth)))
			} else if m.tagPushed {
				b.WriteString(successStyle.Render("✓ Tag pushed: " + m.versionTag))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render("\n  Press any key to exit"))

	return m.renderBox(b.String())
}
