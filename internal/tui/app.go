package tui

import tea "github.com/charmbracelet/bubbletea"

// Result 表示 TUI 退出时用户的选择。
type Result struct {
	Target string // 选中的 profile 名（含 official）
	Launch bool   // true 表示退出后应切换并启动 claude
}

// placeholderCmd 仅用于确认依赖可编译，后续任务会替换。
func placeholderCmd() tea.Cmd { return tea.Quit }
