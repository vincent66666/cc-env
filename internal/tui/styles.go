package tui

import "github.com/charmbracelet/lipgloss"

// accent 是选中项与预览卡片共用的强调色，用来在视觉上把左侧选中行和右侧预览关联起来。
var accent = lipgloss.Color("212")

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	hintStyle  = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	previewKey = lipgloss.NewStyle().Width(12).Faint(true)

	// listBox 左侧配置列表卡片（中性边框）。
	listBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// previewBox 右侧预览卡片，边框用强调色呼应选中行。
	previewBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)

	// previewTitle 预览卡片标题（选中 profile 名），强调色加粗。
	previewTitle = lipgloss.NewStyle().Bold(true).Foreground(accent)
)
