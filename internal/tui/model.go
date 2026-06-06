package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cc-env/internal/profile"
)

type state int

const (
	stateList state = iota
	stateForm
	stateConfirm
)

type Model struct {
	profilesPath string
	state        state
	list         list.Model
	form         formModel
	confirmName  string
	err          string
	result       Result
	width        int
	height       int
}

func newModel(profilesPath string) (Model, error) {
	data, err := profile.LoadForList(profilesPath)
	if err != nil {
		return Model{}, err
	}

	// 选中行用强调色，与右侧预览卡片边框一致，建立左右关联。
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(accent).BorderForeground(accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(accent).BorderForeground(accent)

	items := buildItems(data)
	l := list.New(items, delegate, 0, 0)
	l.Title = "配置"
	l.SetShowHelp(false)

	return Model{
		profilesPath: profilesPath,
		state:        stateList,
		list:         l,
	}, nil
}

// paneHeight 返回卡片内容高度（终端高度减去边框与底部提示行）。
func (m Model) paneHeight() int {
	return max(m.height-4, 3)
}

// twoPaneMinWidth 是显示「列表+预览」双栏所需的最小终端宽度，低于此值退化为单栏。
const twoPaneMinWidth = 60

func (m Model) twoPane() bool { return m.width >= twoPaneMinWidth }

// listWidth 返回列表卡片内容宽度：双栏用固定侧栏宽，单栏占满终端。
func (m Model) listWidth() int {
	if m.twoPane() {
		return 24
	}
	return max(m.width-4, 10)
}

// reload 重新从磁盘读取并重建列表项。
func (m *Model) reload() error {
	data, err := profile.LoadForList(m.profilesPath)
	if err != nil {
		return err
	}
	m.list.SetItems(buildItems(data))
	return nil
}

// submitForm 持久化新增或编辑（含改名）。
func (m *Model) submitForm() error {
	name, p, err := m.form.build()
	if err != nil {
		return err
	}

	data, err := profile.LoadForList(m.profilesPath)
	if err != nil {
		return err
	}

	switch {
	case m.form.original == "":
		if _, exists := data.Profiles[name]; exists {
			return fmt.Errorf("配置 %q 已存在", name)
		}
	case name != m.form.original:
		if _, exists := data.Profiles[name]; exists {
			return fmt.Errorf("配置 %q 已存在", name)
		}
		delete(data.Profiles, m.form.original)
		if data.Current == m.form.original {
			data.Current = name
		}
	}

	data.Profiles[name] = p
	if err := profile.Save(m.profilesPath, data); err != nil {
		return err
	}
	return m.reload()
}

// deleteSelected 删除指定 profile（profile.Remove 内部拦截当前配置）。
func (m *Model) deleteSelected(name string) error {
	if err := profile.Remove(m.profilesPath, name); err != nil {
		return err
	}
	return m.reload()
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) selectedItem() (profileItem, bool) {
	it, ok := m.list.SelectedItem().(profileItem)
	return it, ok
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.listWidth(), m.paneHeight())
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case stateList:
			return m.updateList(msg)
		case stateForm:
			return m.updateForm(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		}
	}
	// 非按键消息（list 过滤的 FilterMatchesMsg、光标闪烁等）转发给当前活动子组件，
	// 否则 list 收不到过滤结果，/ 过滤将无效。
	return m.forwardMsg(msg)
}

// forwardMsg 把非按键消息转发给当前活动子组件。
func (m Model) forwardMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.state == stateForm {
		if m.form.focus < len(textFields) {
			m.form.inputs[m.form.focus], cmd = m.form.inputs[m.form.focus].Update(msg)
		}
		return m, cmd
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 过滤输入态下，字母交给 list 处理，不触发快捷键。
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", keyQuit:
		return m, tea.Quit
	case "enter":
		if it, ok := m.selectedItem(); ok {
			m.result = Result{Target: it.name, Launch: true}
			return m, tea.Quit
		}
	case keyAdd:
		m.form = newForm("", profile.Profile{})
		m.state = stateForm
		m.err = ""
		return m, nil
	case keyEdit:
		if it, ok := m.selectedItem(); ok && !it.official {
			data, err := profile.LoadForList(m.profilesPath)
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.form = newForm(it.name, data.Profiles[it.name])
			m.state = stateForm
			m.err = ""
		}
		return m, nil
	case keyDelete:
		if it, ok := m.selectedItem(); ok && !it.official && !it.current {
			m.confirmName = it.name
			m.state = stateConfirm
			m.err = ""
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = stateList
		return m, nil
	case tea.KeyTab, tea.KeyDown:
		m.form.next()
		return m, nil
	case tea.KeyShiftTab, tea.KeyUp:
		m.form.prev()
		return m, nil
	case tea.KeyEnter:
		if err := m.submitForm(); err != nil {
			m.form.err = err.Error()
			return m, nil
		}
		m.state = stateList
		return m, nil
	}

	if msg.Type == tea.KeySpace {
		if _, ok := m.form.onBool(); ok {
			m.form.toggle()
			return m, nil
		}
	}

	// 文本字段：把按键交给当前聚焦的 textinput。
	if m.form.focus < len(textFields) {
		var cmd tea.Cmd
		m.form.inputs[m.form.focus], cmd = m.form.inputs[m.form.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if err := m.deleteSelected(m.confirmName); err != nil {
			m.err = err.Error()
		}
		m.confirmName = ""
		m.state = stateList
		return m, nil
	case "n", "N", "esc", "q":
		m.confirmName = ""
		m.state = stateList
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateForm:
		return m.viewForm()
	case stateConfirm:
		return m.viewConfirm()
	default:
		return m.viewList()
	}
}

func (m Model) viewList() string {
	h := m.paneHeight()
	left := listBox.Height(h).Render(m.list.View())

	hint := hintStyle.Render("Enter 切换并启动  a 新建  e 编辑  d 删除  / 过滤  q 退出")
	if m.err != "" {
		hint = errStyle.Render(m.err) + "\n" + hint
	}

	// 窄终端（含首帧 width==0）退化为单栏，只显示列表，避免双栏宽度溢出错乱。
	if !m.twoPane() {
		return left + "\n" + hint
	}

	preview := ""
	if it, ok := m.selectedItem(); ok {
		data, _ := profile.LoadForList(m.profilesPath)
		preview = renderPreview(it.name, it.current, data.Profiles[it.name])
	}
	// 预览卡片宽度由剩余终端宽度决定，保持稳定（不随选中项内容长短跳动）。
	previewInner := max(m.width-lipgloss.Width(left)-5, 24)
	right := previewBox.Width(previewInner).Height(h).Render(preview)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	return body + "\n" + hint
}

func (m Model) viewForm() string {
	var b strings.Builder
	title := "新建配置"
	if m.form.original != "" {
		title = "编辑配置：" + m.form.original
	}
	b.WriteString(previewTitle.Render(title) + "\n\n")
	for i, fld := range textFields {
		cursor := "  "
		if m.form.focus == i {
			cursor = "> "
		}
		b.WriteString(cursor + fld.label + "：" + m.form.inputs[i].View() + "\n")
	}
	for i, fld := range boolFields {
		cursor := "  "
		if m.form.focus == len(textFields)+i {
			cursor = "> "
		}
		mark := "[ ]"
		if m.form.bools[i] {
			mark = "[✓]"
		}
		b.WriteString(cursor + mark + " " + fld.label + "\n")
	}
	if m.form.err != "" {
		b.WriteString("\n" + errStyle.Render(m.form.err))
	}

	card := formBox.Render(strings.TrimRight(b.String(), "\n"))
	hint := hintStyle.Render("Tab/↑↓ 切换字段  Space 切换开关  Enter 保存  Esc 取消")
	return card + "\n" + hint
}

func (m Model) viewConfirm() string {
	body := errStyle.Render("删除配置") + "\n\n" +
		"确认删除 " + m.confirmName + "？此操作不可恢复。"
	card := confirmBox.Render(body)
	hint := hintStyle.Render("y 确认  n 取消")
	return card + "\n" + hint
}
