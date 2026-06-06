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
	current      string
	err          string
	result       Result
}

func newModel(profilesPath string) (Model, error) {
	data, err := profile.LoadForList(profilesPath)
	if err != nil {
		return Model{}, err
	}

	items := buildItems(data)
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "配置"
	l.SetShowHelp(false)

	return Model{
		profilesPath: profilesPath,
		state:        stateList,
		list:         l,
		current:      data.Current,
	}, nil
}

// reload 重新从磁盘读取并重建列表项。
func (m *Model) reload() error {
	data, err := profile.LoadForList(m.profilesPath)
	if err != nil {
		return err
	}
	m.current = data.Current
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
		m.list.SetSize(msg.Width, msg.Height-2)
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
	return m, nil
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
	preview := ""
	if it, ok := m.selectedItem(); ok {
		data, _ := profile.LoadForList(m.profilesPath)
		preview = renderPreview(it.name, data.Profiles[it.name])
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, paneStyle.Render(m.list.View()), preview)
	hint := hintStyle.Render("Enter 切换并启动  a 新建  e 编辑  d 删除  / 过滤  q 退出")
	if m.err != "" {
		hint = errStyle.Render(m.err) + "\n" + hint
	}
	return body + "\n" + hint
}

func (m Model) viewForm() string {
	var b strings.Builder
	title := "新建配置"
	if m.form.original != "" {
		title = "编辑配置：" + m.form.original
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")
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
			mark = "[x]"
		}
		b.WriteString(cursor + mark + " " + fld.label + "\n")
	}
	if m.form.err != "" {
		b.WriteString("\n" + errStyle.Render(m.form.err) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("Tab/↑↓ 切换字段  Space 切换开关  Enter 保存  Esc 取消"))
	return b.String()
}

func (m Model) viewConfirm() string {
	return titleStyle.Render("删除配置") + "\n\n" +
		"确认删除 " + m.confirmName + "？此操作不可恢复。\n\n" +
		hintStyle.Render("y 确认  n 取消")
}
