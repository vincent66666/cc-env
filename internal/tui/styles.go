package tui

import "github.com/charmbracelet/lipgloss"

var (
	paneStyle  = lipgloss.NewStyle().Padding(0, 1)
	titleStyle = lipgloss.NewStyle().Bold(true)
	hintStyle  = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Bold(true)
	previewKey = lipgloss.NewStyle().Width(12)
)
