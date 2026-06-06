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
