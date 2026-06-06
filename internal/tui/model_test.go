package tui

import (
	"path/filepath"
	"testing"

	"cc-env/internal/profile"
	tea "github.com/charmbracelet/bubbletea"
)

func writeProfiles(t *testing.T, data profile.ProfilesFile) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := profile.Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}
	return path
}

func loadProfiles(t *testing.T, path string) profile.ProfilesFile {
	t.Helper()
	data, err := profile.LoadForList(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return data
}

func TestSubmitFormAddPersistsProfile(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{}})
	m, err := newModel(path)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.form = newForm("", profile.Profile{})
	m.form.set("name", "demo")
	m.form.set(profile.EnvAuthToken, "tok")
	m.form.set(profile.EnvBaseURL, "https://x")

	if err := m.submitForm(); err != nil {
		t.Fatalf("submitForm: %v", err)
	}
	data := loadProfiles(t, path)
	if _, ok := data.Profiles["demo"]; !ok {
		t.Fatalf("demo not persisted: %v", data.Profiles)
	}
}

func TestSubmitFormAddRejectsDuplicate(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{
		"demo": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
	}})
	m, _ := newModel(path)
	m.form = newForm("", profile.Profile{})
	m.form.set("name", "demo")
	m.form.set(profile.EnvAuthToken, "tok")
	m.form.set(profile.EnvBaseURL, "https://x")
	if err := m.submitForm(); err == nil {
		t.Fatalf("expected duplicate error")
	}
}

func TestSubmitFormEditRenameMovesCurrent(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Current: "demo", Profiles: map[string]profile.Profile{
		"demo": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
	}})
	m, _ := newModel(path)
	existing := loadProfiles(t, path).Profiles["demo"]
	m.form = newForm("demo", existing)
	m.form.set("name", "demo2")
	if err := m.submitForm(); err != nil {
		t.Fatalf("submitForm: %v", err)
	}
	data := loadProfiles(t, path)
	if _, ok := data.Profiles["demo2"]; !ok {
		t.Fatalf("rename target missing")
	}
	if _, ok := data.Profiles["demo"]; ok {
		t.Fatalf("old name still present")
	}
	if data.Current != "demo2" {
		t.Fatalf("current not moved: %q", data.Current)
	}
}

func TestDeleteSelectedRemovesProfile(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Current: "keep", Profiles: map[string]profile.Profile{
		"keep": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
		"gone": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
	}})
	m, _ := newModel(path)
	if err := m.deleteSelected("gone"); err != nil {
		t.Fatalf("deleteSelected: %v", err)
	}
	if _, ok := loadProfiles(t, path).Profiles["gone"]; ok {
		t.Fatalf("gone still present")
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func update(m Model, k tea.KeyMsg) Model {
	next, _ := m.Update(k)
	return next.(Model)
}

func TestEnterOnProfileSetsLaunchResult(t *testing.T) {
	path := writeProfiles(t, sampleData())
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m = update(m, key("enter"))
	if !m.result.Launch || m.result.Target != "kimi" {
		t.Fatalf("result = %+v, want launch kimi", m.result)
	}
}

func TestQuitDoesNotLaunch(t *testing.T) {
	path := writeProfiles(t, sampleData())
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m = update(m, key("q"))
	if m.result.Launch {
		t.Fatalf("quit should not launch")
	}
}

func TestPressAEntersForm(t *testing.T) {
	path := writeProfiles(t, sampleData())
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m = update(m, key("a"))
	if m.state != stateForm || m.form.original != "" {
		t.Fatalf("state = %v original = %q, want form/add", m.state, m.form.original)
	}
}

func TestPressDOnDeletableEntersConfirm(t *testing.T) {
	path := writeProfiles(t, sampleData()) // current kimi; deepseek deletable
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m = update(m, key("j")) // 移动到 deepseek（第二项）
	m = update(m, key("d"))
	if m.state != stateConfirm || m.confirmName != "deepseek" {
		t.Fatalf("state = %v confirm = %q", m.state, m.confirmName)
	}
}

func TestPressDOnCurrentIgnored(t *testing.T) {
	path := writeProfiles(t, sampleData()) // 首项 kimi 是 current
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m = update(m, key("d"))
	if m.state != stateList {
		t.Fatalf("delete on current should stay in list, got %v", m.state)
	}
}

func TestPressDOnOfficialIgnored(t *testing.T) {
	path := writeProfiles(t, sampleData())
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m = update(m, key("G")) // 跳到末项 official（list 默认绑定 G=末尾）
	m = update(m, key("d"))
	if m.state != stateList {
		t.Fatalf("delete on official should stay in list, got %v", m.state)
	}
}

func TestFormSubmitInvalidStaysInFormWithError(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{}})
	m, _ := newModel(path)
	m.state = stateForm
	m.form = newForm("", profile.Profile{}) // 全空，缺 name/token/base
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.state != stateForm || m.form.err == "" {
		t.Fatalf("invalid submit should stay in form with err, state=%v err=%q", m.state, m.form.err)
	}
}

func TestFormSubmitValidReturnsToList(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{}})
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m.state = stateForm
	m.form = newForm("", profile.Profile{})
	m.form.set("name", "demo")
	m.form.set(profile.EnvAuthToken, "tok")
	m.form.set(profile.EnvBaseURL, "https://x")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.state != stateList {
		t.Fatalf("valid submit should return to list, got %v (err=%q)", m.state, m.form.err)
	}
	if _, ok := loadProfiles(t, path).Profiles["demo"]; !ok {
		t.Fatalf("demo not persisted")
	}
}

func TestFormEscCancels(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{}})
	m, _ := newModel(path)
	m.state = stateForm
	m.form = newForm("", profile.Profile{})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	if m.state != stateList {
		t.Fatalf("esc should cancel form")
	}
}

func TestFormToggleBoolField(t *testing.T) {
	f := newForm("", profile.Profile{})
	// 焦点移动到第一个 bool 字段并空格切换
	idx := len(textFields) // 第一个 bool 的焦点序号
	f.focus = idx
	f.toggle()
	if !f.bools[0] {
		t.Fatalf("toggle did not flip bool[0]")
	}
}

func TestConfirmYesDeletes(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Current: "keep", Profiles: map[string]profile.Profile{
		"keep": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
		"gone": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
	}})
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m.state = stateConfirm
	m.confirmName = "gone"
	next, _ := m.Update(key("y"))
	m = next.(Model)
	if m.state != stateList {
		t.Fatalf("confirm yes should return to list, got %v", m.state)
	}
	if _, ok := loadProfiles(t, path).Profiles["gone"]; ok {
		t.Fatalf("gone not deleted")
	}
}

func TestConfirmNoCancels(t *testing.T) {
	path := writeProfiles(t, profile.ProfilesFile{Version: 1, Current: "keep", Profiles: map[string]profile.Profile{
		"keep": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
		"gone": {Env: map[string]string{profile.EnvAuthToken: "t", profile.EnvBaseURL: "https://x"}},
	}})
	m, _ := newModel(path)
	m.list.SetSize(40, 10)
	m.state = stateConfirm
	m.confirmName = "gone"
	next, _ := m.Update(key("n"))
	m = next.(Model)
	if m.state != stateList {
		t.Fatalf("confirm no should return to list")
	}
	if _, ok := loadProfiles(t, path).Profiles["gone"]; !ok {
		t.Fatalf("gone should still exist after cancel")
	}
}
