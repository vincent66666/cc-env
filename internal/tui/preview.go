package tui

import (
	"strings"

	"cc-env/internal/profile"
)

// maskSecret 遮罩敏感值，仅保留首尾两位。
func maskSecret(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

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
