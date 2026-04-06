package main

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up   key.Binding
	Down key.Binding
	Quit key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
	),
}
