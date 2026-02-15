package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Edit    key.Binding
	Regen   key.Binding
	Push    key.Binding
	Save    key.Binding
	Tab     key.Binding
	Quit    key.Binding
	Escape  key.Binding
	CtrlC   key.Binding
	Version key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Regen: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "regenerate"),
	),
	Push: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "push"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	CtrlC: key.NewBinding(
		key.WithKeys("ctrl+c"),
	),
	Version: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "version tag"),
	),
}
