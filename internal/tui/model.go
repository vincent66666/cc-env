package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

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
	if msg.Type == tea.KeyEsc {
		m.state = stateList
	}
	return m, nil
}

func (m Model) View() string { return "" }
