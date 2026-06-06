package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestViewListBodyFitsTerminalWidth 守护回归：卡片主体在任意窗口宽度下都不溢出终端。
// 窄终端会退化为单栏，双栏只在足够宽时出现。
func TestViewListBodyFitsTerminalWidth(t *testing.T) {
	path := writeProfiles(t, sampleData())
	m, _ := newModel(path)

	for _, w := range []int{40, 50, 58, 60, 80, 120} {
		nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 20})
		mm := nm.(Model)
		for _, line := range strings.Split(mm.View(), "\n") {
			// 仅校验含边框字符的卡片主体行；底部提示行允许软换行。
			if strings.ContainsAny(line, "╭╮╰╯│─") && lipgloss.Width(line) > w {
				t.Fatalf("width %d: 卡片行溢出 (%d>%d): %q", w, lipgloss.Width(line), w, line)
			}
		}
	}
}
