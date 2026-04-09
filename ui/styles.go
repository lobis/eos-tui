package ui

import "github.com/charmbracelet/lipgloss"

func newStyles() styles {
	return styles{
		app: lipgloss.NewStyle().
			Padding(0, 1),
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")),
		tab: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("245")),
		tabActive: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")),
		panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
		panelDim: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),
		selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")),
		label: lipgloss.NewStyle().
			Foreground(lipgloss.Color("109")),
		value: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Bold(true),
		status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		splash: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("123")),
		splashDim: lipgloss.NewStyle().
			Foreground(lipgloss.Color("80")),
		splashBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("80")).
			Padding(1, 3),
	}
}
