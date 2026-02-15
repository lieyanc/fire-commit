package tui

import (
	"context"
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
			if len(m.messages) == 0 {
				return m, nil
			}
			m.confirmCursor = confirmCommitOnly
			m.phase = PhaseConfirm
			return m, nil
		case key.Matches(msg, keys.Edit):
			if len(m.messages) == 0 {
				return m, nil
			}
			m.editArea.SetValue(m.messages[m.cursor])
			m.editing = true
			m.phase = PhaseEdit
			return m, m.editArea.Focus()
		case key.Matches(msg, keys.Regen):
			m.resetForRegeneration()
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
	contentWidth := m.contentWidth()

	b.WriteString(titleStyle.Render("🔥 fire-commit"))
	b.WriteString("\n\n")
	b.WriteString("Select a commit message:\n\n")

	for i, msg := range m.messages {
		if i == m.cursor {
			b.WriteString(renderWrappedLine("  > ", cursorStyle.Render("  > "), msg, selectedStyle, contentWidth))
		} else {
			b.WriteString(renderWrappedLine("    ", "    ", msg, normalStyle, contentWidth))
		}
		b.WriteString("\n")
	}

	if pending := m.pendingCount(); pending > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("Generating %d more suggestion(s) in background...", pending)))
	}

	b.WriteString(helpStyle.Render("\n  ↑/↓/j/k select • enter confirm • e edit • r regen • q quit"))

	return m.renderBox(b.String())
}

func (m *Model) resetForRegeneration() {
	n := m.cfg.Generation.NumSuggestions
	if n <= 0 {
		n = 3
	}

	if m.cancel != nil {
		m.cancel()
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.generationID++

	m.messages = make([]string, 0, n)
	m.partial = make([]string, n)
	m.slotDone = make([]bool, n)
	m.slotFailed = make([]bool, n)
	m.completed = 0
	m.finished = 0
	m.failed = 0
	m.total = n
	m.resultCh = nil
	m.cursor = 0
	m.editing = false
	m.editArea.Blur()

	m.confirmCursor = confirmCommitOnly
	m.versionTag = ""
	m.editingTag = false
	m.tagInput.SetValue("")
	defaultHints := buildTagHints("")
	m.tagInput.Placeholder = defaultHints.base
	m.tagHintBase = defaultHints.base
	m.tagHintMinor = defaultHints.minor
	m.tagHintPatch = defaultHints.patch
	m.tagInput.Blur()
	m.wantPush = false

	m.committed = false
	m.pushed = false
	m.commitErr = nil
	m.pushErr = nil
	m.tagged = false
	m.tagErr = nil
	m.tagPushed = false
	m.tagPushErr = nil

	m.spinner = newSpinner()
}
