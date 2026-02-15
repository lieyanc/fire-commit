package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/llm"
)

// Phase represents the current phase of the TUI.
type Phase int

const (
	PhaseLoading Phase = iota
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

	// Loading: progressive per-message results
	spinner    spinner.Model
	messages   []string
	partial    []string
	slotDone   []bool
	slotFailed []bool
	completed  int
	finished   int
	failed     int
	total      int
	resultCh   <-chan llm.IndexedMessageEvent
	// generationID identifies the active round of LLM generation.
	// It prevents stale events from a previous round from mutating state.
	generationID int

	// Select
	cursor int

	// Edit
	editArea textarea.Model
	editing  bool

	// Confirm
	confirmCursor int
	versionTag    string
	tagInput      textinput.Model
	editingTag    bool
	tagHintBase   string
	tagHintMinor  string
	tagHintPatch  string

	// Result
	committed  bool
	pushed     bool
	commitErr  error
	pushErr    error
	tagged     bool
	tagErr     error
	tagPushed  bool
	tagPushErr error

	// Window
	width  int
	height int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Whether user chose to push
	wantPush bool
}

// messageReadyMsg is a streamed event for one LLM request.
type messageReadyMsg struct {
	generationID int
	index        int
	delta        string
	content      string
	done         bool
	err          error
}

// allDoneMsg signals the result channel was closed (all requests finished).
type allDoneMsg struct{ generationID int }

// commitDoneMsg signals the commit operation completed.
type commitDoneMsg struct{ err error }

// pushDoneMsg signals the push operation completed.
type pushDoneMsg struct{ err error }

// tagDoneMsg signals the tag creation completed.
type tagDoneMsg struct{ err error }

// tagPushDoneMsg signals the tag push completed.
type tagPushDoneMsg struct{ err error }

// NewModel creates a new TUI model.
func NewModel(cfg *config.Config, diff, stat string) Model {
	s := newSpinner()

	ta := textarea.New()
	ta.Placeholder = "Edit commit message..."
	ta.Focus()
	ta.CharLimit = 500
	ta.SetWidth(60)
	ta.SetHeight(5)

	ti := textinput.New()
	initialTagHints := buildTagHints("")
	ti.Placeholder = initialTagHints.base
	ti.CharLimit = 50
	ti.Width = 30

	ctx, cancel := context.WithCancel(context.Background())

	n := cfg.Generation.NumSuggestions
	if n <= 0 {
		n = 3
	}

	model := Model{
		phase:         PhaseLoading,
		cfg:           cfg,
		diff:          diff,
		stat:          stat,
		spinner:       s,
		messages:      make([]string, 0, n),
		partial:       make([]string, n),
		slotDone:      make([]bool, n),
		slotFailed:    make([]bool, n),
		total:         n,
		generationID:  1,
		editArea:      ta,
		tagInput:      ti,
		tagHintBase:   initialTagHints.base,
		tagHintMinor:  initialTagHints.minor,
		tagHintPatch:  initialTagHints.patch,
		confirmCursor: confirmCommitOnly,
		ctx:           ctx,
		cancel:        cancel,
	}

	model.resizeInputs()
	return model
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle
	return s
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startGeneration())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, keys.CtrlC) {
			m.cancel()
			return m, tea.Quit
		}
	}

	// Handle generation stream events globally so we can keep receiving
	// suggestions while the user is already on Select/Edit/Confirm screens.
	switch msg := msg.(type) {
	case startResultsMsg:
		if msg.generationID != m.generationID {
			return m, nil
		}
		m.resultCh = msg.ch
		return m, waitForMessage(m.resultCh, msg.generationID)

	case messageReadyMsg:
		if msg.generationID != m.generationID {
			return m, nil
		}
		return m.handleGenerationMessage(msg)

	case allDoneMsg:
		if msg.generationID != m.generationID {
			return m, nil
		}
		m.resultCh = nil
		if m.completed == 0 {
			if m.failed > 0 {
				m.commitErr = fmt.Errorf("all LLM requests failed")
			} else {
				m.commitErr = fmt.Errorf("LLM returned no commit messages")
			}
			m.phase = PhaseDone
		}
		return m, nil
	}

	switch m.phase {
	case PhaseLoading:
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
	case PhaseLoading:
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
			return messageReadyMsg{
				generationID: m.generationID,
				index:        -1,
				err:          err,
			}
		}

		opts := llm.GenerateOptions{
			Language: m.cfg.Generation.Language,
		}

		ch := llm.GenerateMultiple(m.ctx, provider, m.diff, opts, m.total)
		return startResultsMsg{
			generationID: m.generationID,
			ch:           ch,
		}
	}
}

// startResultsMsg delivers the message-event channel to the model.
type startResultsMsg struct {
	generationID int
	ch           <-chan llm.IndexedMessageEvent
}

// waitForMessage reads the next IndexedMessageEvent from the channel.
func waitForMessage(ch <-chan llm.IndexedMessageEvent, generationID int) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return allDoneMsg{generationID: generationID}
		}
		return messageReadyMsg{
			generationID: generationID,
			index:        msg.Index,
			delta:        msg.Delta,
			content:      msg.Content,
			done:         msg.Done,
			err:          msg.Err,
		}
	}
}

func (m Model) handleGenerationMessage(msg messageReadyMsg) (tea.Model, tea.Cmd) {
	// Provider construction failures are fatal and happen before any channel exists.
	if msg.err != nil && m.resultCh == nil {
		m.commitErr = msg.err
		m.phase = PhaseDone
		return m, nil
	}

	var next tea.Cmd
	if m.resultCh != nil {
		next = waitForMessage(m.resultCh, msg.generationID)
	}

	if msg.index < 0 || msg.index >= len(m.partial) {
		return m, next
	}
	if m.slotDone[msg.index] {
		return m, next
	}

	if msg.err != nil {
		m.slotDone[msg.index] = true
		m.slotFailed[msg.index] = true
		m.finished++
		m.failed++
		if m.completed == 0 && m.finished >= m.total {
			m.commitErr = fmt.Errorf("all LLM requests failed: %w", msg.err)
			m.phase = PhaseDone
			return m, nil
		}
		return m, next
	}

	if msg.delta != "" {
		m.partial[msg.index] += msg.delta
	}

	if msg.done {
		m.slotDone[msg.index] = true
		m.finished++
		if msg.content == "" {
			m.slotFailed[msg.index] = true
			m.failed++
		} else {
			m.partial[msg.index] = msg.content
			m.messages = append(m.messages, msg.content)
			m.completed++
			if m.phase == PhaseLoading {
				m.phase = PhaseSelect
			}
			if m.cursor >= len(m.messages) {
				m.cursor = len(m.messages) - 1
			}
		}
	}

	return m, next
}

func (m Model) pendingCount() int {
	pending := m.total - m.finished
	if pending < 0 {
		return 0
	}
	return pending
}

// Run starts the TUI program.
func Run(cfg *config.Config, diff, stat string) error {
	m := NewModel(cfg, diff, stat)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
