# 统一交互 TUI 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 cc-env 的交互层重构为单一 Bubble Tea TUI（`cc-env` 无参进入），删除 `list/add/edit/remove/rename/status` 子命令，保留 `cc-env <profile>` 直达启动与 `cc-env current`。

**Architecture:** 新增 `internal/tui` 包承载 Bubble Tea 应用（Elm 架构：`Model`/`Update`/`View`，单一 `state` 枚举驱动 list/form/confirm 三态）。把可测的纯逻辑（排序、遮罩、保留名校验、表单→profile 构建、增删改持久化）与 Bubbles 组件渲染分离，保证测试不依赖 Bubbles 版本。`internal/cli` 瘦身为「参数分发 + 直达启动 + current + 非 TTY 兜底」，`internal/profile` 作为持久化/校验核心被复用，几乎不动。

**Tech Stack:** Go 1.23、`github.com/charmbracelet/bubbletea`、`.../bubbles`（list、textinput）、`.../lipgloss`；标准库 `os/exec` 启动 `claude`。

**前置条件（执行前确认）：** 本计划首个任务需要联网执行 `go get` 拉取 Bubble Tea 依赖。若环境离线，先准备好模块缓存（`GOFLAGS=-mod=mod`、`GOPROXY`/vendor）再开始。

---

## 文件结构

**新增（`internal/tui/`）：**
- `app.go` — `Run(profilesPath string) (Result, error)`：建 program、运行、返回启动目标。
- `model.go` — `Model`、`state` 枚举、`Init/Update/View`、`reload`、`submitForm`、`deleteSelected`。
- `list.go` — `profileItem`、`buildItems`、`orderProfiles`、`descriptionFor`、`profileNamesSorted`。
- `form.go` — `formModel`、`textFields`/`boolFields`、`newForm`、`build`、`set`、`value`、`reservedName`。
- `preview.go` — 高亮项预览渲染、`maskSecret`。
- `keys.go` — 键位常量。
- `styles.go` — lipgloss 样式。
- 测试：`model_test.go`、`list_test.go`、`form_test.go`、`preview_test.go`。

**修改：**
- `go.mod` / `go.sum` — 新增依赖。
- `internal/cli/app.go` — 重写 `Run` 分发，新增 `runDefault`、`runNonInteractiveStatus`、`isInteractive`、`runTUI` 变量。
- `internal/cli/launch.go` — 接管 `launchClaude` 变量。
- `internal/cli/app_test.go` — 删除过时用例、重写 `TestMain` 与无参用例。
- `README.md`、`CLAUDE.md` — 去掉 zero-dependency、更新命令/架构。

**删除：**
- `internal/cli/status_selector.go` + `status_selector_test.go`
- `internal/cli/list_menu.go` + `list_menu_test.go`
- `internal/cli/interactive.go`
- `internal/cli/prompt.go`
- `internal/cli/commands.go`
- `internal/cli/term_darwin.go` + `internal/cli/term_other.go`

**保留不动：** `internal/cli/display.go`（非 TTY 兜底与 current 复用其 helper）、`internal/cli/parse.go`、`internal/profile/*`、`internal/output/*`、`main.go`。

---

## Task 1: 引入 Bubble Tea 依赖

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 拉取依赖**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go mod tidy
```
Expected: `go.mod` 出现三个 `require`，`go.sum` 更新，无报错。

- [ ] **Step 2: 写一个最小冒烟文件验证依赖可编译**

Create `internal/tui/app.go`:
```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// Result 表示 TUI 退出时用户的选择。
type Result struct {
	Target string // 选中的 profile 名（含 official）
	Launch bool   // true 表示退出后应切换并启动 claude
}

// placeholderCmd 仅用于确认依赖可编译，后续任务会替换。
func placeholderCmd() tea.Cmd { return tea.Quit }
```

- [ ] **Step 3: 编译验证**

Run: `go build ./...`
Expected: 成功，无错误。

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum internal/tui/app.go
git commit -m "build: 引入 bubbletea/bubbles/lipgloss 依赖"
```

---

## Task 2: 纯逻辑——profile 排序与 token 遮罩

**Files:**
- Create: `internal/tui/list.go`
- Create: `internal/tui/preview.go`
- Test: `internal/tui/list_test.go`, `internal/tui/preview_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/tui/list_test.go`:
```go
package tui

import (
	"reflect"
	"testing"
)

func TestOrderProfilesPutsCurrentFirst(t *testing.T) {
	got := orderProfiles([]string{"a", "b", "official"}, "b")
	want := []string{"b", "a", "official"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderProfiles = %v, want %v", got, want)
	}
}

func TestOrderProfilesNoCurrentKeepsOrder(t *testing.T) {
	got := orderProfiles([]string{"a", "b"}, "")
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderProfiles = %v, want %v", got, want)
	}
}

func TestOrderProfilesCurrentMissingIgnored(t *testing.T) {
	got := orderProfiles([]string{"a", "b"}, "ghost")
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderProfiles = %v, want %v", got, want)
	}
}
```

Create `internal/tui/preview_test.go`:
```go
package tui

import "testing"

func TestMaskSecretShort(t *testing.T) {
	if got := maskSecret("abcd"); got != "****" {
		t.Fatalf("maskSecret short = %q, want ****", got)
	}
}

func TestMaskSecretLong(t *testing.T) {
	if got := maskSecret("abcdef"); got != "ab**ef" {
		t.Fatalf("maskSecret long = %q, want ab**ef", got)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run 'TestOrderProfiles|TestMaskSecret' -count=1`
Expected: 编译失败（`orderProfiles`/`maskSecret` 未定义）。

- [ ] **Step 3: 实现**

Create `internal/tui/list.go`:
```go
package tui

import (
	"sort"

	"cc-env/internal/profile"
)

func profileNamesSorted(profiles map[string]profile.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// orderProfiles 把 current 放到首位，其余保持传入顺序。
func orderProfiles(names []string, current string) []string {
	ordered := make([]string, 0, len(names))
	found := false
	for _, name := range names {
		if name == current {
			found = true
			break
		}
	}
	if found && current != "" {
		ordered = append(ordered, current)
	}
	for _, name := range names {
		if name == current {
			continue
		}
		ordered = append(ordered, name)
	}
	return ordered
}
```

Create `internal/tui/preview.go`:
```go
package tui

import "strings"

// maskSecret 遮罩敏感值，仅保留首尾两位。
func maskSecret(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run 'TestOrderProfiles|TestMaskSecret' -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/list.go internal/tui/preview.go internal/tui/list_test.go internal/tui/preview_test.go
git commit -m "feat(tui): profile 排序与 token 遮罩纯逻辑"
```

---

## Task 3: 列表项与构建

**Files:**
- Modify: `internal/tui/list.go`
- Test: `internal/tui/list_test.go`

- [ ] **Step 1: 写失败测试**

Append to `internal/tui/list_test.go`:
```go
import "cc-env/internal/profile" // 若文件顶部已 import 可省略

func sampleData() profile.ProfilesFile {
	return profile.ProfilesFile{
		Version: 1,
		Current: "kimi",
		Profiles: map[string]profile.Profile{
			"deepseek": {Description: "DeepSeek", Env: map[string]string{
				profile.EnvAuthToken: "tok", profile.EnvBaseURL: "https://d",
			}},
			"kimi": {Description: "Kimi", Env: map[string]string{
				profile.EnvAuthToken: "tok", profile.EnvBaseURL: "https://k",
			}},
		},
	}
}

func TestBuildItemsOrdersCurrentFirstOfficialLast(t *testing.T) {
	items := buildItems(sampleData())
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	first := items[0].(profileItem)
	last := items[2].(profileItem)
	if first.name != "kimi" || !first.current {
		t.Fatalf("first item = %+v, want current kimi", first)
	}
	if last.name != profile.OfficialProfileName || !last.official {
		t.Fatalf("last item = %+v, want official", last)
	}
}

func TestBuildItemsOfficialDescription(t *testing.T) {
	items := buildItems(sampleData())
	last := items[2].(profileItem)
	if last.description != "官方登录态" {
		t.Fatalf("official description = %q", last.description)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run TestBuildItems -count=1`
Expected: 编译失败（`buildItems`/`profileItem` 未定义）。

- [ ] **Step 3: 实现**

Append to `internal/tui/list.go`:
```go
import (
	"github.com/charmbracelet/bubbles/list"
) // 与文件已有 import 合并

type profileItem struct {
	name        string
	description string
	current     bool
	official    bool
}

func (i profileItem) Title() string {
	if i.current {
		return i.name + "（当前）"
	}
	return i.name
}

func (i profileItem) Description() string { return i.description }

func (i profileItem) FilterValue() string { return i.name + " " + i.description }

func descriptionFor(data profile.ProfilesFile, name string) string {
	if profile.IsOfficialName(name) {
		return "官方登录态"
	}
	return data.Profiles[name].Description
}

func buildItems(data profile.ProfilesFile) []list.Item {
	names := profileNamesSorted(data.Profiles)
	names = append(names, profile.OfficialProfileName)
	ordered := orderProfiles(names, data.Current)

	items := make([]list.Item, 0, len(ordered))
	for _, name := range ordered {
		items = append(items, profileItem{
			name:        name,
			description: descriptionFor(data, name),
			current:     name == data.Current,
			official:    profile.IsOfficialName(name),
		})
	}
	return items
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run TestBuildItems -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/list.go internal/tui/list_test.go
git commit -m "feat(tui): 列表项构建与排序"
```

---

## Task 4: 表单字段定义、保留名与 build

**Files:**
- Create: `internal/tui/form.go`
- Test: `internal/tui/form_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/tui/form_test.go`:
```go
package tui

import (
	"testing"

	"cc-env/internal/profile"
)

func TestReservedNameRejectsOfficialAndCurrent(t *testing.T) {
	for _, name := range []string{"official", "current", " current "} {
		if !reservedName(name) {
			t.Fatalf("reservedName(%q) = false, want true", name)
		}
	}
	if reservedName("deepseek") {
		t.Fatalf("reservedName(deepseek) = true, want false")
	}
}

func TestFormBuildRejectsMissingToken(t *testing.T) {
	f := newForm("", profile.Profile{})
	f.set("name", "demo")
	f.set(profile.EnvBaseURL, "https://x")
	// 不填 token
	if _, _, err := f.build(); err == nil {
		t.Fatalf("expected error for missing token")
	}
}

func TestFormBuildBuildsEnvFromFields(t *testing.T) {
	f := newForm("", profile.Profile{})
	f.set("name", "demo")
	f.set("description", "Demo")
	f.set(profile.EnvAuthToken, "tok")
	f.set(profile.EnvBaseURL, "https://x")
	f.set("ANTHROPIC_MODEL", "m1")
	f.bools[0] = true // CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC

	name, p, err := f.build()
	if err != nil {
		t.Fatalf("build err: %v", err)
	}
	if name != "demo" || p.Description != "Demo" {
		t.Fatalf("name/desc = %q/%q", name, p.Description)
	}
	if p.Env[profile.EnvAuthToken] != "tok" || p.Env["ANTHROPIC_MODEL"] != "m1" {
		t.Fatalf("env = %v", p.Env)
	}
	if p.Env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] != "1" {
		t.Fatalf("bool flag not set: %v", p.Env)
	}
	if _, ok := p.Env["description"]; ok {
		t.Fatalf("description leaked into env")
	}
}

func TestFormBuildRejectsReservedName(t *testing.T) {
	f := newForm("", profile.Profile{})
	f.set("name", "current")
	f.set(profile.EnvAuthToken, "tok")
	f.set(profile.EnvBaseURL, "https://x")
	if _, _, err := f.build(); err == nil {
		t.Fatalf("expected reserved-name error")
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run 'TestReservedName|TestFormBuild' -count=1`
Expected: 编译失败（`newForm`/`formModel`/`reservedName` 未定义）。

- [ ] **Step 3: 实现**

Create `internal/tui/form.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"cc-env/internal/profile"
)

type formField struct {
	key      string // "name"、"description" 或 env key
	label    string
	required bool
	secret   bool
}

var textFields = []formField{
	{key: "name", label: "名称", required: true},
	{key: "description", label: "描述"},
	{key: profile.EnvAuthToken, label: "ANTHROPIC_AUTH_TOKEN", required: true, secret: true},
	{key: profile.EnvBaseURL, label: "ANTHROPIC_BASE_URL", required: true},
	{key: "ANTHROPIC_MODEL", label: "ANTHROPIC_MODEL"},
	{key: "ANTHROPIC_DEFAULT_OPUS_MODEL", label: "ANTHROPIC_DEFAULT_OPUS_MODEL"},
	{key: "ANTHROPIC_DEFAULT_SONNET_MODEL", label: "ANTHROPIC_DEFAULT_SONNET_MODEL"},
	{key: "ANTHROPIC_DEFAULT_HAIKU_MODEL", label: "ANTHROPIC_DEFAULT_HAIKU_MODEL"},
	{key: "CLAUDE_CODE_SUBAGENT_MODEL", label: "CLAUDE_CODE_SUBAGENT_MODEL"},
}

var boolFields = []formField{
	{key: "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", label: "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"},
	{key: "CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK", label: "CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK"},
}

type formModel struct {
	original string // 编辑时的原名；新增时为 ""
	inputs   []textinput.Model
	bools    []bool
	focus    int
	err      string
}

func reservedName(name string) bool {
	switch strings.TrimSpace(name) {
	case profile.OfficialProfileName, "current":
		return true
	default:
		return false
	}
}

func newForm(editing string, existing profile.Profile) formModel {
	inputs := make([]textinput.Model, len(textFields))
	for i, fld := range textFields {
		in := textinput.New()
		in.Prompt = ""
		if fld.secret {
			in.EchoMode = textinput.EchoPassword
			in.EchoCharacter = '•'
		}
		inputs[i] = in
	}
	f := formModel{
		original: editing,
		inputs:   inputs,
		bools:    make([]bool, len(boolFields)),
	}
	if editing != "" {
		f.set("name", editing)
		f.set("description", existing.Description)
		for i, fld := range textFields {
			if fld.key == "name" || fld.key == "description" {
				continue
			}
			f.inputs[i].SetValue(existing.Env[fld.key])
		}
		for i, bf := range boolFields {
			f.bools[i] = existing.Env[bf.key] != ""
		}
	}
	f.inputs[0].Focus()
	return f
}

func (f *formModel) set(key, value string) {
	for i, fld := range textFields {
		if fld.key == key {
			f.inputs[i].SetValue(value)
			return
		}
	}
}

func (f formModel) value(key string) string {
	for i, fld := range textFields {
		if fld.key == key {
			return strings.TrimSpace(f.inputs[i].Value())
		}
	}
	return ""
}

// build 把表单内容构造成 (name, profile)，并做保留名 + 校验。
func (f formModel) build() (string, profile.Profile, error) {
	name := f.value("name")

	env := map[string]string{}
	for i, fld := range textFields {
		if fld.key == "name" || fld.key == "description" {
			continue
		}
		if v := strings.TrimSpace(f.inputs[i].Value()); v != "" {
			env[fld.key] = v
		}
	}
	for i, bf := range boolFields {
		if f.bools[i] {
			env[bf.key] = "1"
		}
	}

	p := profile.Profile{Description: f.value("description"), Env: env}

	if reservedName(name) {
		return "", profile.Profile{}, fmt.Errorf("配置 %q 是保留名称，不能使用", name)
	}
	if err := profile.ValidateProfile(name, p); err != nil {
		return "", profile.Profile{}, err
	}
	return name, p, nil
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run 'TestReservedName|TestFormBuild' -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/form.go internal/tui/form_test.go
git commit -m "feat(tui): 表单字段定义与 profile 构建校验"
```

---

## Task 5: 根 Model 与持久化（reload/submitForm/deleteSelected）

**Files:**
- Create: `internal/tui/model.go`
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/tui/model_test.go`:
```go
package tui

import (
	"os"
	"path/filepath"
	"testing"

	"cc-env/internal/profile"
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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run 'TestSubmitForm|TestDeleteSelected' -count=1`
Expected: 编译失败（`newModel`/`submitForm`/`deleteSelected` 未定义）。

- [ ] **Step 3: 实现**

Create `internal/tui/model.go`:
```go
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
```

> 说明：`Update`/`View` 在 Task 6 加入。本任务只让 `newModel`/`reload`/`submitForm`/`deleteSelected` 可编译可测。`list`/`Result` 字段已被引用以避免 unused。

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run 'TestSubmitForm|TestDeleteSelected' -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat(tui): 根 Model 与增删改持久化"
```

---

## Task 6: list 态 Update——导航、Enter 启动、退出、进入表单/删除

**Files:**
- Modify: `internal/tui/model.go`
- Create: `internal/tui/keys.go`
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: 写失败测试**

Append to `internal/tui/model_test.go`:
```go
import tea "github.com/charmbracelet/bubbletea" // 与已有 import 合并

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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run 'TestEnterOn|TestQuit|TestPressA|TestPressD' -count=1`
Expected: 编译失败（`Update` 未定义）。

- [ ] **Step 3: 实现键位与 list 态 Update**

Create `internal/tui/keys.go`:
```go
package tui

// list 态键位。
const (
	keyAdd     = "a"
	keyEdit    = "e"
	keyDelete  = "d"
	keyQuit    = "q"
)
```

Append to `internal/tui/model.go`:
```go
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
```

> `updateForm`/`updateConfirm` 在 Task 7、Task 8 加入；本步骤先各加一个最小桩以便编译：

```go
func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.state = stateList
	}
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		m.state = stateList
	}
	return m, nil
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run 'TestEnterOn|TestQuit|TestPressA|TestPressD' -count=1`
Expected: PASS。

> 若 `G`/`j` 默认绑定与预期不符导致 `TestPressDOnOfficial`/`TestPressDOnDeletable` 失败，改用直接定位：在测试里 `m.list.Select(idx)` 设置选中项后再发 `d`（`Select` 是 bubbles list 公有方法）。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/keys.go internal/tui/model_test.go
git commit -m "feat(tui): list 态导航/启动/进入表单与删除确认"
```

---

## Task 7: 表单态 Update——字段导航、toggle、提交、校验回显

**Files:**
- Modify: `internal/tui/model.go`, `internal/tui/form.go`
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: 写失败测试**

Append to `internal/tui/model_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run 'TestFormSubmit|TestFormEsc|TestFormToggle' -count=1`
Expected: 编译/断言失败（`toggle`、表单提交逻辑未实现）。

- [ ] **Step 3: 实现表单导航与提交**

Append to `internal/tui/form.go`:
```go
func (f formModel) fieldCount() int { return len(textFields) + len(boolFields) }

func (f formModel) onBool() (int, bool) {
	bi := f.focus - len(textFields)
	if bi >= 0 && bi < len(boolFields) {
		return bi, true
	}
	return 0, false
}

func (f *formModel) toggle() {
	if bi, ok := f.onBool(); ok {
		f.bools[bi] = !f.bools[bi]
	}
}

func (f *formModel) focusActive() {
	for i := range f.inputs {
		if i == f.focus {
			f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
}

func (f *formModel) next() {
	f.focus = (f.focus + 1) % f.fieldCount()
	f.focusActive()
}

func (f *formModel) prev() {
	f.focus = (f.focus - 1 + f.fieldCount()) % f.fieldCount()
	f.focusActive()
}
```

Replace the `updateForm` 桩 in `internal/tui/model.go` with:
```go
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
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run 'TestFormSubmit|TestFormEsc|TestFormToggle' -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/form.go internal/tui/model_test.go
git commit -m "feat(tui): 表单字段导航、toggle 与提交校验"
```

---

## Task 8: 删除确认态 Update

**Files:**
- Modify: `internal/tui/model.go`
- Test: `internal/tui/model_test.go`

- [ ] **Step 1: 写失败测试**

Append to `internal/tui/model_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run TestConfirm -count=1`
Expected: 失败（桩 `updateConfirm` 只处理 Esc）。

- [ ] **Step 3: 实现**

Replace the `updateConfirm` 桩 in `internal/tui/model.go` with:
```go
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
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `go test ./internal/tui/ -run TestConfirm -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git commit -m "feat(tui): 删除确认态处理"
```

---

## Task 9: View 渲染（list/preview/form/confirm）与样式

**Files:**
- Modify: `internal/tui/model.go`, `internal/tui/preview.go`
- Create: `internal/tui/styles.go`
- Test: `internal/tui/preview_test.go`

- [ ] **Step 1: 写失败测试**

Append to `internal/tui/preview_test.go`:
```go
import "cc-env/internal/profile" // 与已有 import 合并

func TestRenderPreviewMasksToken(t *testing.T) {
	p := profile.Profile{Description: "Demo", Env: map[string]string{
		profile.EnvAuthToken: "secrettoken",
		profile.EnvBaseURL:   "https://x",
		"ANTHROPIC_MODEL":    "m1",
	}}
	out := renderPreview("demo", p)
	if !strings.Contains(out, "demo") || !strings.Contains(out, "m1") {
		t.Fatalf("preview missing name/model: %q", out)
	}
	if strings.Contains(out, "secrettoken") {
		t.Fatalf("preview leaked raw token: %q", out)
	}
	if !strings.Contains(out, maskSecret("secrettoken")) {
		t.Fatalf("preview missing masked token: %q", out)
	}
}

func TestRenderPreviewOfficial(t *testing.T) {
	out := renderPreview(profile.OfficialProfileName, profile.Profile{})
	if !strings.Contains(out, "官方登录态") {
		t.Fatalf("official preview = %q", out)
	}
}
```

(确保 `preview_test.go` 顶部已 `import "strings"`。)

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./internal/tui/ -run TestRenderPreview -count=1`
Expected: 失败（`renderPreview` 未定义）。

- [ ] **Step 3: 实现 styles、preview、View**

Create `internal/tui/styles.go`:
```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	paneStyle   = lipgloss.NewStyle().Padding(0, 1)
	titleStyle  = lipgloss.NewStyle().Bold(true)
	hintStyle   = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Bold(true)
	previewKey  = lipgloss.NewStyle().Width(8)
)
```

Append to `internal/tui/preview.go`:
```go
import (
	"strings"

	"cc-env/internal/profile"
) // 与已有 import 合并

// renderPreview 渲染高亮 profile 的字段，token 遮罩。
func renderPreview(name string, p profile.Profile) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("预览") + "\n")
	b.WriteString(previewKey.Render("名称") + name + "\n")

	if profile.IsOfficialName(name) {
		b.WriteString(previewKey.Render("说明") + "官方登录态\n")
		return paneStyle.Render(b.String())
	}

	if p.Description != "" {
		b.WriteString(previewKey.Render("描述") + p.Description + "\n")
	}
	for _, key := range profile.ManagedEnvKeys {
		v := p.Env[key]
		if v == "" {
			continue
		}
		if key == profile.EnvAuthToken {
			v = maskSecret(v)
		}
		b.WriteString(previewKey.Render(shortKey(key)) + v + "\n")
	}
	return paneStyle.Render(b.String())
}

func shortKey(envKey string) string {
	switch envKey {
	case profile.EnvAuthToken:
		return "token"
	case profile.EnvBaseURL:
		return "base"
	case "ANTHROPIC_MODEL":
		return "model"
	default:
		return envKey
	}
}
```

Append `View` to `internal/tui/model.go`:
```go
import "github.com/charmbracelet/lipgloss" // 与已有 import 合并

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
```

(确保 `model.go` 顶部 `import "strings"`。)

- [ ] **Step 4: 运行测试，确认通过 + 全包测试**

Run: `go test ./internal/tui/ -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/tui/styles.go internal/tui/preview.go internal/tui/model.go internal/tui/preview_test.go
git commit -m "feat(tui): list/preview/form/confirm 渲染与样式"
```

---

## Task 10: `tui.Run` 入口

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: 实现 Run，替换占位**

Replace `internal/tui/app.go` 全文:
```go
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
```

- [ ] **Step 2: 编译 + 全包测试**

Run: `go build ./... && go test ./internal/tui/ -count=1`
Expected: 成功，PASS。

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): Run 入口装配 program"
```

---

## Task 11: cli 切换——重写 Run 分发 + relocate launchClaude（生产代码）

> 本任务只改生产代码并让 `go build` 通过；测试编译会在 Task 12 修好。**Task 11 与 Task 12 必须在同一个开发回合内连续完成**，中间不要停在红灯状态。

**Files:**
- Modify: `internal/cli/app.go`, `internal/cli/launch.go`
- Delete: `internal/cli/commands.go`, `internal/cli/prompt.go`, `internal/cli/interactive.go`, `internal/cli/status_selector.go`, `internal/cli/list_menu.go`, `internal/cli/term_darwin.go`, `internal/cli/term_other.go`

- [ ] **Step 1: 把 launchClaude 变量移到 launch.go**

In `internal/cli/launch.go`, 在 `runClaude` 函数定义之后新增:
```go
// launchClaude 可在测试中替换。
var launchClaude = runClaude
```

- [ ] **Step 2: 重写 app.go 的 Run 与分发**

Replace `internal/cli/app.go` 中 `Run` 函数为:
```go
func Run(args []string, stdout, stderr io.Writer) int {
	command := Parse(args)
	paths := defaultPaths()

	switch command.Name {
	case "":
		return runDefault(paths, stdout, stderr)
	case "current":
		return runCurrent(paths, stdout, stderr)
	default:
		return runProfileCommand(paths, command.Name, command.Args, stderr)
	}
}
```

In `internal/cli/app.go`, 新增分发辅助（放在 `Run` 之后）:
```go
// runTUI 可在测试中替换。
var runTUI = tui.Run

// isInteractive 判定 stdin 与 stdout 是否均为终端；可在测试中替换。
var isInteractive = func(stdout io.Writer) bool {
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	if err != nil || stat.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	inStat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return inStat.Mode()&os.ModeCharDevice != 0
}

func runDefault(paths Paths, stdout, stderr io.Writer) int {
	if !isInteractive(stdout) {
		return runNonInteractiveStatus(paths, stdout, stderr)
	}

	result, err := runTUI(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "启动交互界面失败：%v\n", err)
		return 1
	}
	if !result.Launch {
		return 0
	}
	return switchProfile(paths, result.Target, nil, stderr)
}
```

In `internal/cli/app.go`, 重命名/改造 `runStatus` 为非交互兜底（删掉其交互分支）。把原 `runStatus` 函数体替换为 `runNonInteractiveStatus`:
```go
func runNonInteractiveStatus(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		if shouldRenderUnknownForProfileLoadError(err) {
			_, _ = io.WriteString(stdout, "当前配置：未知\n")
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	currentProfile, currentDisplay, _, _, ok := currentStatus(data)
	if !ok {
		_, _ = io.WriteString(stdout, "当前配置：未知\n")
		return 0
	}

	names := availableNames(data.Profiles, data.Current)
	return output.RenderStatus(
		stdout,
		currentDisplay,
		currentProfile,
		displayNamesForProfiles(data.Profiles, names),
	)
}
```

In `internal/cli/app.go` 的 import 块，新增 `"cc-env/internal/tui"`，并删除不再使用的旧 `runList` 函数（整段删除）。

- [ ] **Step 3: 删除过时生产文件**

Run:
```bash
git rm internal/cli/commands.go internal/cli/prompt.go internal/cli/interactive.go \
       internal/cli/status_selector.go internal/cli/list_menu.go \
       internal/cli/term_darwin.go internal/cli/term_other.go
```

- [ ] **Step 4: 处理编译残留**

Run: `go build ./...`
Expected：可能报 `display.go` 中 `prioritizeCurrentProfile`、`profileListDisplayName` 等已无人引用（它们随旧文件被引用而存在）。逐个处理：
- 若某 helper 仅被已删文件引用 → 从 `display.go` 删除该函数。
- 保留仍被 `runNonInteractiveStatus`/`runCurrent`/`currentStatus`/`displayNamesForProfiles` 使用的 helper。
反复 `go build ./...` 直到通过。

> 预期需从 `display.go` 删除：`profileListDisplayName`、`profileDescriptions`、`prioritizeCurrentProfile`（后者本就在已删的 `list_menu.go`）。`profileNames`、`modeNames`、`availableNames`、`displayNamesForProfiles`、`profileDisplayName`、`currentDescription`、`officialProfileDescription` 保留。`runCurrent`、`currentStatus`、`shouldRenderUnknownForProfileLoadError`、`normalizeProfileName`、`formatCLIError` 保留。

- [ ] **Step 5: 确认生产代码编译通过**

Run: `go build ./...`
Expected: 成功。（此时 `go test` 仍会因测试引用旧符号而编译失败——Task 12 修复。）

- [ ] **Step 6: 暂不 commit，直接进入 Task 12**（保持工作树连续修复测试后一并提交）

---

## Task 12: cli 测试清理与重写

**Files:**
- Delete: `internal/cli/status_selector_test.go`, `internal/cli/list_menu_test.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: 删除整文件级过时测试**

Run:
```bash
git rm internal/cli/status_selector_test.go internal/cli/list_menu_test.go
```

- [ ] **Step 2: 重写 app_test.go 的 TestMain 与 TTY 辅助**

In `internal/cli/app_test.go`, 用下面内容替换 `TestMain` 和 `TestTTYHelperProcess` 两个函数（删除 `TestTTYHelperProcess` 与其用到的 `helperArgs` 等 TTY 辅助）：
```go
func TestMain(m *testing.M) {
	isInteractive = func(io.Writer) bool { return false }
	runTUI = func(string) (tui.Result, error) { return tui.Result{}, nil }
	launchClaude = func(args []string, env []string) error { return nil }
	os.Exit(m.Run())
}
```
并在 import 块加入 `"cc-env/internal/tui"`。

- [ ] **Step 3: 删除针对已移除命令的测试函数**

删除 `app_test.go` 中下列测试函数（按名搜索整段删除）：

```
TestRun_OfficialNameIsReservedForProfileManagement
TestRun_CommandNamesAreReservedForProfileManagement
TestRun_AddPersistsProfile
TestRun_AddTrimsNameArgumentAndRejectsNormalizedDuplicate
TestRun_EditUpdatesExistingProfile
TestRun_EditTrimsNameArgument
TestRun_AddRejectsMissingRequiredFields
TestRun_AddRejectsMissingNameNonInteractive
TestRun_AddRejectsMissingBaseURLNonInteractive
TestRun_AddRejectsDuplicateName
TestRun_AddInteractivePromptsForAllFields
TestRun_AddInteractiveRejectsDuplicateNameBeforeFurtherPrompts
TestRun_AddInteractiveInterruptedInputFails
TestRun_AddRealTTYCtrlDAbortsWithoutWrite
TestRun_AddRealTTYCtrlCAbortsWithoutWrite
TestRun_AddRealTTYBlankInputFails
TestRun_EditInteractivePromptsAndKeepsExistingValuesOnBlank
TestRun_EditInteractiveBlankOptionalFieldKeepsMissingKey
TestRun_EditInteractiveInterruptedInputFailsWithoutMutation
TestRun_EditRealTTYMultipleEntersKeepValues
TestRun_EditInteractiveSkipsPromptForExplicitFields
TestRun_EditInteractiveMasksShortToken
TestRun_CustomPathsSupportAddProfileCommandAndCurrentFlow
TestRun_RemoveRejectsCurrentProfile
TestRun_RemoveDeletesNonCurrentProfile
TestRun_RemoveTrimsNameArgument
TestRun_RenameMovesProfileAndCurrentPointer
TestRun_RenameTrimsNameArguments
TestRun_StatusInteractiveArrowSelectionSwitchesProfile
TestRun_StatusInteractiveQuitLeavesCurrentUnchanged
TestRun_StatusInteractiveUsesAlternateScreen
TestRun_StatusInteractiveAlternateScreenWrapsMultipleMoves
TestRun_StatusInteractiveCtrlCLeavesCurrentUnchanged
TestRun_StatusInteractiveWithoutAlternativesPrintsStatusOnly
TestRun_StatusInteractiveEOFClosesAlternateScreen
TestRun_StatusFallsBackToPlainTextWhenStdoutIsNotTTY
TestRun_StatusFallsBackToPlainTextWhenRawTerminalUnavailable
TestRun_StatusFallsBackToPlainTextWhenStdoutIsTTYAndStdinIsFile
TestStatusSelectorRenderIncludesQuitHint
TestRun_StatusPrintsUnknownWhenCurrentProfileIsMissing
TestRun_StatusReportsLoadErrorWhenProfilesFileIsInvalid
TestRun_ListPrintsOfficialWhenProfilesFileIsMissing
TestRun_ListCommandWinsOverExistingListProfile
TestRun_ListPrintsProfilesWhenCurrentProfileIsMissing
TestRun_ListReportsLoadErrorWhenProfilesFileIsInvalid
TestRun_ListPrintsProfiles
TestRun_ListFallsBackToPlainTextWhenStdoutIsNotTTY
TestRun_ListFallsBackToPlainTextWhenStdinIsNotTTY
TestRun_ListFallsBackToPlainTextWhenStdoutIsTTYAndStdinIsFile
TestRun_ListPlainTextOmitsSeparatorWhenDescriptionIsEmpty
TestRun_ListPlainTextTrimsDescriptionWhitespace
TestListMenuRenderIncludesQuitHintInAllModes
TestRun_ListInteractiveSwitchesSelectedProfile
TestRun_ListInteractiveSwitchesSelectedProfileWhenCurrentProfileIsMissing
TestRun_ListInteractiveBackLeavesCurrentUnchanged
TestRun_ListInteractiveUsesAlternateScreen
TestRun_ListInteractiveActionMenuQuitClosesAlternateScreen
TestRun_ListInteractiveCurrentProfileActionsHideRemove
TestRun_ListInteractiveDeleteConfirmQuitClosesAlternateScreen
TestRun_ListInteractiveCtrlCLeavesCurrentUnchanged
TestRun_ListInteractiveEditUpdatesProfileAndReturnsToList
TestRun_ListInteractiveEditReentersAlternateScreenAfterSuccess
TestRun_ListInteractiveRenameUpdatesProfileAndReturnsToList
TestRun_ListInteractiveRenameCurrentProfileUpdatesCurrentAndReentersList
TestRun_ListInteractiveRemoveConfirmsAndRefreshesList
TestRun_ListInteractiveRemoveReentersAlternateScreenAfterSuccess
TestRun_ListInteractiveRemoveLastProfileShowsEmptyState
TestRun_ListInteractiveEmptyStateQuitClosesAlternateScreen
```

- [ ] **Step 4: 改写无参默认行为测试**

删除 `TestRun_NoArgsDefaultsToOfficialAndClearsManagedEnv` 与 `TestRun_NoArgsShowsBaseURLAndModel`，新增（放在文件末尾）：
```go
func TestRun_NoArgsNonInteractivePrintsStatusWithoutLaunch(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {Description: "Demo", Env: map[string]string{
				profile.EnvAuthToken: "tok",
				profile.EnvBaseURL:   "https://demo.example.com",
				"ANTHROPIC_MODEL":    "m1",
			}},
		},
	})
	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr=%q", code, stderr.String())
	}
	if call.args != nil {
		t.Fatalf("no-args non-interactive should not launch claude")
	}
	if !strings.Contains(stdout.String(), "demo") {
		t.Fatalf("status output missing profile: %q", stdout.String())
	}
}

func TestRun_NoArgsInteractiveLaunchesSelectedTarget(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "official",
		Profiles: map[string]profile.Profile{
			"demo": {Env: map[string]string{
				profile.EnvAuthToken: "tok", profile.EnvBaseURL: "https://demo.example.com",
			}},
		},
	})
	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	originalInteractive := isInteractive
	originalTUI := runTUI
	isInteractive = func(io.Writer) bool { return true }
	runTUI = func(string) (tui.Result, error) { return tui.Result{Target: "demo", Launch: true}, nil }
	t.Cleanup(func() { isInteractive = originalInteractive; runTUI = originalTUI })

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d stderr=%q", code, stderr.String())
	}
	if call.args == nil {
		t.Fatalf("expected claude launch for selected target")
	}
	if got := loadProfilesFixture(t, profilesPath).Current; got != "demo" {
		t.Fatalf("current = %q, want demo", got)
	}
}

func TestRun_NoArgsInteractiveQuitDoesNotLaunch(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1, Current: "official", Profiles: map[string]profile.Profile{},
	})
	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	originalInteractive := isInteractive
	originalTUI := runTUI
	isInteractive = func(io.Writer) bool { return true }
	runTUI = func(string) (tui.Result, error) { return tui.Result{Launch: false}, nil }
	t.Cleanup(func() { isInteractive = originalInteractive; runTUI = originalTUI })

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if call.args != nil {
		t.Fatalf("quit should not launch claude")
	}
}
```

> 若 `writeProfilesFixture` / `loadProfilesFixture` 辅助不存在或签名不符，沿用 app_test.go 现有的 fixture helper（搜索 `func writeProfilesFixture`），或用 Task 5 风格的 `profile.Save` 直接写入。

- [ ] **Step 5: 处理删除测试后产生的未用辅助**

Run: `go vet ./internal/cli/` 与 `go test ./internal/cli/ -count=1`
若报某些测试辅助函数（如 `helperArgs`、real-TTY 启动器）未使用 → 删除这些辅助。反复修复直到编译通过。

- [ ] **Step 6: 全量构建 + 测试**

Run: `go build ./... && go test ./... -count=1`
Expected: 全部 PASS。

- [ ] **Step 7: Commit（Task 11 + Task 12 一并提交）**

```bash
git add -A
git commit -m "refactor(cli): 无参进 TUI、直达启动与 current 保留，删除 CRUD 子命令

- cc-env 无参进入 TUI（非 TTY 打印状态，不启动 claude）
- 删除 list/add/edit/remove/rename/status 子命令及其交互/表单代码
- launchClaude 迁移至 launch.go，新增 isInteractive/runTUI 注入点
- 清理 cli 测试，新增无参分发用例"
```

---

## Task 13: 文档更新（README + CLAUDE.md）

**Files:**
- Modify: `README.md`, `CLAUDE.md`

- [ ] **Step 1: 更新 README.md**

执行以下内容编辑：
- 删除/改写 "它不再写入 settings.json…零依赖" 相关表述中关于 **zero-dependency** 的卖点；保留"清理第三方变量、不污染官方登录态"的说明。
- "功能概览" 中：
  - 把 `用 cc-env <profile|official> …` 保留；
  - 把"支持新增、编辑、删除、重命名第三方 profile"改为"通过 `cc-env` 进入交互界面完成新增/编辑/删除/重命名/切换"。
- "使用方法" 章节：删除 `cc-env status` / `cc-env list` 小节；新增"交互界面"小节，说明 `cc-env`（无参）进入 TUI，键位 `↑/↓·j/k` `Enter 切换并启动` `a 新建` `e 编辑` `d 删除` `/ 过滤` `q 退出`；保留 `cc-env current` 与 `cc-env <profile>` 直达说明；补一句"非交互终端下 `cc-env` 打印当前状态、不启动 claude"。

- [ ] **Step 2: 更新 CLAUDE.md**

- "Project" 段：删除/改写 stdlib-only 暗示（如有），保持准确。
- "Key Design Details" 段：把 `Zero external dependencies — stdlib only` 改为 `交互式 TUI 基于 Bubble Tea（bubbletea/bubbles/lipgloss）；其余仅用标准库`。
- "Architecture" 目录树：
  - `cli/` 下删除 `status_selector.go / list_menu.go / term_darwin.go / term_other.go / commands.go / prompt.go / interactive.go` 的描述；保留 `app.go`（参数分发 + 直达 + current + 非 TTY 兜底）、`launch.go`、`parse.go`、`display.go`。
  - 新增 `tui/` 段：`app.go`（Run 入口）、`model.go`（状态机）、`list.go`、`form.go`、`preview.go`、`keys.go`、`styles.go`。
- 命令说明：注明 `cc-env` 无参进 TUI；`list/add/edit/remove/rename/status` 已移除；`cc-env <profile>` 与 `cc-env current` 保留。

- [ ] **Step 3: 校验文档无残留旧命令**

Run: `grep -nE "cc-env (list|add|edit|remove|rename|status)" README.md CLAUDE.md || echo "无残留"`
Expected: `无残留`（或仅出现在"已移除"说明语境中——人工确认）。

- [ ] **Step 4: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: 更新 README/CLAUDE 至单一 TUI 与 Bubble Tea 依赖"
```

---

## Task 14: 最终验证

**Files:** 无（仅校验）

- [ ] **Step 1: 格式化**

Run: `gofmt -l . ` （应无输出）；若有文件列出，运行 `gofmt -w .` 并 `git add -A`。

- [ ] **Step 2: vet + 全量测试**

Run: `go vet ./... && go test ./... -count=1`
Expected: 无 vet 报错，全部 PASS。

- [ ] **Step 3: 构建产物冒烟**

Run:
```bash
go build -o cc-env .
printf '' | ./cc-env            # 非 TTY（管道）应打印状态、退出 0、不启动 claude
./cc-env current                # 应打印当前 profile 名或"未知"
```
Expected: 第一条输出"当前配置：…"且立即返回；第二条输出当前名。

- [ ] **Step 4: 若 Step 1 有格式化改动则提交**

```bash
git add -A
git commit -m "chore: gofmt 与最终校验"
```

---

## 自检对照（Spec 覆盖）

- 合并为单一 TUI（无参进入）：Task 2–10（tui）、Task 11（无参分发）。✅
- 删除 list/add/edit/remove/rename/status：Task 11（生产）+ Task 12（测试）。✅
- 保留 `cc-env <profile>`/`official` 直达：Task 11 保留 `runProfileCommand` 分发 + launch.go 不动。✅
- 保留 `cc-env current`：Task 11 保留 `runCurrent`。✅
- Bubble Tea 框架：Task 1 依赖、Task 2–10 实现。✅
- Enter→exec claude（TUI 退出后由 cli 启动）：Task 6（result）+ Task 11（switchProfile）。✅
- token 遮罩、表单内编辑：Task 4/7/9（`EchoPassword` + maskSecret）。✅
- 重命名并入编辑表单：Task 5（submitForm 改名分支）+ Task 7。✅
- 保留名仅 official/current：Task 4（reservedName）。✅
- 非 TTY 打印状态不启动：Task 11（runNonInteractiveStatus）+ Task 12 测试。✅
- profiles.json 缺失可进入并新增：`LoadForList` + Task 5/6。✅
- 校验失败内联回显：Task 7（form.err）。✅
- 删除 term_*.go、交给 Bubble Tea：Task 11。✅
- 文档去 zero-dependency：Task 13。✅
- 测试策略（tui 单测 + cli 清理）：Task 2–9、Task 12。✅
