package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Result 表示 TUI 退出时用户的选择。
type Result struct {
	Target string // 选中的 profile 名（含 official）
	Launch bool   // true 表示退出后应切换并启动 claude
}

// Run 启动交互式 TUI，返回用户选择的启动目标。
func Run(profilesPath string) (Result, error) {
	m, err := newModel(profilesPath)
	if err != nil {
		return Result{}, err
	}

	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return Result{}, err
	}
	return final.(Model).result, nil
}
