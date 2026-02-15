package tui

import (
	"testing"

	"github.com/lieyanc/fire-commit/internal/config"
)

func newGenerationTestModel() Model {
	cfg := config.DefaultConfig()
	cfg.Generation.NumSuggestions = 3
	return NewModel(cfg, "diff", "stat")
}

func TestUpdateSelectsEarlyWhenFirstMessageReady(t *testing.T) {
	t.Parallel()

	m := newGenerationTestModel()

	next, _ := m.Update(messageReadyMsg{
		generationID: m.generationID,
		index:        0,
		content:      "feat(api): add endpoint",
		done:         true,
	})
	got := next.(Model)

	if got.phase != PhaseSelect {
		t.Fatalf("phase got %v want %v", got.phase, PhaseSelect)
	}
	if len(got.messages) != 1 {
		t.Fatalf("messages len got %d want 1", len(got.messages))
	}
	if got.completed != 1 || got.finished != 1 {
		t.Fatalf("counters got completed=%d finished=%d want 1,1", got.completed, got.finished)
	}
	if got.pendingCount() != 2 {
		t.Fatalf("pending got %d want 2", got.pendingCount())
	}
}

func TestUpdateAppendsStreamDeltasPerSlot(t *testing.T) {
	t.Parallel()

	m := newGenerationTestModel()

	next, _ := m.Update(messageReadyMsg{
		generationID: m.generationID,
		index:        1,
		delta:        "feat(api): ",
	})
	got := next.(Model)
	if got.partial[1] != "feat(api): " {
		t.Fatalf("first delta got %q", got.partial[1])
	}

	next, _ = got.Update(messageReadyMsg{
		generationID: got.generationID,
		index:        1,
		delta:        "add endpoint",
	})
	got = next.(Model)
	if got.partial[1] != "feat(api): add endpoint" {
		t.Fatalf("second delta got %q", got.partial[1])
	}

	next, _ = got.Update(messageReadyMsg{
		generationID: got.generationID,
		index:        1,
		content:      "feat(api): add endpoint",
		done:         true,
	})
	got = next.(Model)
	if got.partial[1] != "feat(api): add endpoint" {
		t.Fatalf("final content got %q", got.partial[1])
	}
	if len(got.messages) != 1 {
		t.Fatalf("messages len got %d want 1", len(got.messages))
	}
}

func TestUpdateIgnoresStaleGenerationEvents(t *testing.T) {
	t.Parallel()

	m := newGenerationTestModel()
	m.generationID = 2

	next, _ := m.Update(messageReadyMsg{
		generationID: 1,
		index:        0,
		content:      "feat: stale",
		done:         true,
	})
	got := next.(Model)

	if len(got.messages) != 0 || got.completed != 0 || got.finished != 0 {
		t.Fatalf("stale event mutated state")
	}
	if got.phase != PhaseLoading {
		t.Fatalf("phase got %v want %v", got.phase, PhaseLoading)
	}
}
