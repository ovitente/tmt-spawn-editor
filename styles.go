package main

import "github.com/charmbracelet/lipgloss"

var (
	borderColor       = lipgloss.Color("#3a3a3a")
	activeBorderColor = lipgloss.Color("#606060")
	reorderMoveColor  = lipgloss.Color("214")
	commentColor      = lipgloss.Color("241")

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(activeBorderColor).
				Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true)

	appTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("110")).
			PaddingLeft(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248"))

	helpSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
