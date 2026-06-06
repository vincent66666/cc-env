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
