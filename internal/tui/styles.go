package tui

import "github.com/charmbracelet/lipgloss"

// accent 是选中项与卡片共用的强调色，用来在视觉上把左侧选中行和右侧预览关联起来。
var accent = lipgloss.Color("212")

var (
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

	// formBox 新建/编辑表单卡片，强调色边框。
	formBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(0, 1)

	// confirmBox 删除确认卡片，红色边框示警。
	confirmBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(0, 1)

	// previewTitle 卡片标题（选中 profile 名 / 表单标题），强调色加粗。
	previewTitle = lipgloss.NewStyle().Bold(true).Foreground(accent)
)
