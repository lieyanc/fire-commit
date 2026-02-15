package tui

import (
	"context"

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
	spinner   spinner.Model
	messages  []string
	completed int
	total     int
	resultCh  <-chan llm.IndexedMessage

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

// messageReadyMsg signals a single LLM request completed.
type messageReadyMsg struct {
	index   int
	content string
	err     error
}

// allDoneMsg signals the result channel was closed (all requests finished).
type allDoneMsg struct{}

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
	ti.Placeholder = "v1.0.0"
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
		messages:      make([]string, n),
		total:         n,
		editArea:      ta,
		tagInput:      ti,
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
			return messageReadyMsg{index: 0, err: err}
		}

		opts := llm.GenerateOptions{
			Language: m.cfg.Generation.Language,
		}

		ch := llm.GenerateMultiple(m.ctx, provider, m.diff, opts, m.total)
		return startResultsMsg{ch: ch}
	}
}

// startResultsMsg delivers the IndexedMessage channel to the model.
type startResultsMsg struct {
	ch <-chan llm.IndexedMessage
}

// waitForMessage reads the next IndexedMessage from the channel.
func waitForMessage(ch <-chan llm.IndexedMessage) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return allDoneMsg{}
		}
		return messageReadyMsg{
			index:   msg.Index,
			content: msg.Content,
			err:     msg.Err,
		}
	}
}

// Run starts the TUI program.
func Run(cfg *config.Config, diff, stat string) error {
	m := NewModel(cfg, diff, stat)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
