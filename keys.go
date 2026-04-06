package main

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Space    key.Binding
	Escape   key.Binding
	Save     key.Binding
	Edit     key.Binding
	AddEntry key.Binding
	Delete   key.Binding
	Find     key.Binding
	Tab      key.Binding
	Profile  key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
	),
	AddEntry: key.NewBinding(
		key.WithKeys("a"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
	),
	Find: key.NewBinding(
		key.WithKeys("f"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
	),
	Profile: key.NewBinding(
		key.WithKeys("p"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}
