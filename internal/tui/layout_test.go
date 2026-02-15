package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lieyanc/fire-commit/internal/config"
)

func TestWrapTextHonorsWidth(t *testing.T) {
	t.Parallel()

	got := wrapText("feat(api): add endpoint with robust validation rules", 12)
	lines := strings.Split(got, "\n")

	for _, line := range lines {
		if lipgloss.Width(line) > 12 {
			t.Fatalf("line %q exceeds width 12", line)
		}
	}
}

func TestContentWidthFallbackAndMin(t *testing.T) {
	t.Parallel()

	m := Model{}
	if got := m.contentWidth(); got != fallbackContentWidth {
		t.Fatalf("fallback width got %d want %d", got, fallbackContentWidth)
	}

	m.width = 8
	if got := m.contentWidth(); got != minContentWidth {
		t.Fatalf("min width got %d want %d", got, minContentWidth)
	}
}

func TestResetForRegenerationClearsTransientState(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Generation.NumSuggestions = 4

	m := NewModel(cfg, "diff", "stat")
	m.cursor = 3
	m.completed = 2
	m.total = 1
	m.confirmCursor = confirmCancel
	m.versionTag = "v9.9.9"
	m.editingTag = true
	m.tagInput.SetValue("v9.9.9")
	m.wantPush = true
	m.committed = true
	m.pushed = true
	m.commitErr = errors.New("commit failed")
	m.pushErr = errors.New("push failed")
	m.tagged = true
	m.tagErr = errors.New("tag failed")
	m.tagPushed = true
	m.tagPushErr = errors.New("tag push failed")

	m.resetForRegeneration()

	if len(m.messages) != 4 {
		t.Fatalf("messages len got %d want 4", len(m.messages))
	}
	if m.completed != 0 || m.total != 4 {
		t.Fatalf("generation counters not reset: completed=%d total=%d", m.completed, m.total)
	}
	if m.cursor != 0 {
		t.Fatalf("cursor got %d want 0", m.cursor)
	}
	if m.confirmCursor != confirmCommitOnly {
		t.Fatalf("confirm cursor got %d want %d", m.confirmCursor, confirmCommitOnly)
	}
	if m.versionTag != "" || m.tagInput.Value() != "" || m.editingTag {
		t.Fatalf("tag state not cleared")
	}
	if m.wantPush || m.committed || m.pushed || m.tagged || m.tagPushed {
		t.Fatalf("result flags not reset")
	}
	if m.commitErr != nil || m.pushErr != nil || m.tagErr != nil || m.tagPushErr != nil {
		t.Fatalf("result errors not reset")
	}
}

func TestUpdateConfirmPushToggle(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	m := NewModel(cfg, "diff", "stat")
	m.phase = PhaseConfirm
	m.confirmCursor = confirmCommitOnly

	next, _ := m.updateConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got := next.(Model)
	if got.confirmCursor != confirmCommitAndPush {
		t.Fatalf("first toggle got %d want %d", got.confirmCursor, confirmCommitAndPush)
	}

	next, _ = got.updateConfirm(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got = next.(Model)
	if got.confirmCursor != confirmCommitOnly {
		t.Fatalf("second toggle got %d want %d", got.confirmCursor, confirmCommitOnly)
	}
}
