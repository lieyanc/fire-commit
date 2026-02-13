package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/llm"
)

// Phase represents the current phase of the TUI.
type Phase int

const (
	PhaseLoading Phase = iota
	PhaseStreaming
	PhaseSelect
	PhaseEdit
	PhaseConfirm
	PhaseCommitting
	PhaseDone
)

// Model is the top-level bubbletea model.
type Model struct {
	phase Phase
	cfg   *config.Config
	diff  string
	stat  string

	// Loading/Streaming
	spinner    spinner.Model
	streamBuf  strings.Builder
	streamCh   <-chan llm.StreamChunk
	streamDone bool

	// Select
	messages []string
	cursor   int

	// Edit
	editArea textarea.Model
	editing  bool

	// Confirm
	confirmCursor int // 0=commit only, 1=commit+push, 2=cancel

	// Result
	committed bool
	pushed    bool
	commitErr error
	pushErr   error

	// Window
	width  int
	height int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Whether user chose to push
	wantPush bool
}

// streamChunkMsg wraps a chunk received from the LLM stream.
type streamChunkMsg struct {
	chunk llm.StreamChunk
}

// commitDoneMsg signals the commit operation completed.
type commitDoneMsg struct{ err error }

// pushDoneMsg signals the push operation completed.
type pushDoneMsg struct{ err error }

// NewModel creates a new TUI model.
func NewModel(cfg *config.Config, diff, stat string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle

	ta := textarea.New()
	ta.Placeholder = "Edit commit message..."
	ta.Focus()
	ta.CharLimit = 500
	ta.SetWidth(60)
	ta.SetHeight(5)

	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		phase:         PhaseLoading,
		cfg:           cfg,
		diff:          diff,
		stat:          stat,
		spinner:       s,
		editArea:      ta,
		confirmCursor: 0,
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startGeneration())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, keys.CtrlC) {
			m.cancel()
			return m, tea.Quit
		}
	}

	switch m.phase {
	case PhaseLoading, PhaseStreaming:
		return m.updateLoading(msg)
	case PhaseSelect:
		return m.updateSelect(msg)
	case PhaseEdit:
		return m.updateEdit(msg)
	case PhaseConfirm:
		return m.updateConfirm(msg)
	case PhaseCommitting:
		return m.updateCommitting(msg)
	case PhaseDone:
		return m.updateDone(msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.phase {
	case PhaseLoading, PhaseStreaming:
		return m.viewLoading()
	case PhaseSelect:
		return m.viewSelect()
	case PhaseEdit:
		return m.viewEdit()
	case PhaseConfirm:
		return m.viewConfirm()
	case PhaseCommitting:
		return m.viewCommitting()
	case PhaseDone:
		return m.viewDone()
	}
	return ""
}

func (m Model) startGeneration() tea.Cmd {
	return func() tea.Msg {
		provider, err := llm.NewProvider(m.cfg)
		if err != nil {
			return streamChunkMsg{chunk: llm.StreamChunk{Err: err}}
		}

		opts := llm.GenerateOptions{
			NumSuggestions: m.cfg.Generation.NumSuggestions,
			Language:       m.cfg.Generation.Language,
		}

		ch, err := provider.GenerateCommitMessages(m.ctx, m.diff, opts)
		if err != nil {
			return streamChunkMsg{chunk: llm.StreamChunk{Err: err}}
		}

		// Read first chunk to kick things off
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{chunk: llm.StreamChunk{Done: true}}
		}

		// Store the channel in a closure and return the first chunk
		// We need to use a command to continue reading
		return startStreamMsg{ch: ch, first: chunk}
	}
}

type startStreamMsg struct {
	ch    <-chan llm.StreamChunk
	first llm.StreamChunk
}

func waitForChunk(ch <-chan llm.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{chunk: llm.StreamChunk{Done: true}}
		}
		return streamChunkMsg{chunk: chunk}
	}
}

// Run starts the TUI program.
func Run(cfg *config.Config, diff, stat string) error {
	m := NewModel(cfg, diff, stat)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
