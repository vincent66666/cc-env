package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// runCmdFast 执行 cmd 并返回其消息，用于在测试里模拟 Bubble Tea 运行时的消息回灌。
// 慢命令（光标闪烁、tick）会超时跳过；BatchMsg 递归展开。
func runCmdFast(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() {
		defer func() { _ = recover() }()
		ch <- cmd()
	}()
	select {
	case msg := <-ch:
		switch v := msg.(type) {
		case tea.BatchMsg:
			var out []tea.Msg
			for _, c := range v {
				out = append(out, runCmdFast(c)...)
			}
			return out
		case nil:
			return nil
		default:
			return []tea.Msg{msg}
		}
	case <-time.After(40 * time.Millisecond):
		return nil
	}
}

// drive 把一条消息送进 Update，并把命令产出的后续消息继续回灌，直到队列清空。
func drive(m Model, msg tea.Msg) Model {
	queue := []tea.Msg{msg}
	for steps := 0; len(queue) > 0 && steps < 300; steps++ {
		cur := queue[0]
		queue = queue[1:]
		next, cmd := m.Update(cur)
		m = next.(Model)
		queue = append(queue, runCmdFast(cmd)...)
	}
	return m
}

// TestFilterNarrowsVisibleItems 守护回归：/ 过滤需真正缩小可见项。
// 之前 Update 丢弃了非按键消息（含 list 的 FilterMatchesMsg），导致过滤无效。
func TestFilterNarrowsVisibleItems(t *testing.T) {
	path := writeProfiles(t, sampleData()) // official, kimi, deepseek
	m, _ := newModel(path)
	m.list.SetSize(60, 20)

	m = drive(m, key("/"))
	if m.list.FilterState() != list.Filtering {
		t.Fatalf("'/' did not enter filtering, state=%v", m.list.FilterState())
	}

	for _, r := range "deep" {
		m = drive(m, key(string(r)))
	}

	visible := m.list.VisibleItems()
	if len(visible) != 1 || visible[0].(profileItem).name != "deepseek" {
		names := make([]string, 0, len(visible))
		for _, it := range visible {
			names = append(names, it.(profileItem).name)
		}
		t.Fatalf("filter 'deep' should narrow to [deepseek], got %v", names)
	}
}
