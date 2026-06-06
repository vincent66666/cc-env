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

// shortKey 把受管 env key 映射为预览用短标签，避免长 key 在窄列里折行。
// 新增 ManagedEnvKeys 时需在此补充对应短标签（保持 ≤ previewKey 宽度）。
func shortKey(envKey string) string {
	switch envKey {
	case profile.EnvAuthToken:
		return "token"
	case profile.EnvBaseURL:
		return "base"
	case "ANTHROPIC_MODEL":
		return "model"
	case "ANTHROPIC_DEFAULT_OPUS_MODEL":
		return "opus"
	case "ANTHROPIC_DEFAULT_SONNET_MODEL":
		return "sonnet"
	case "ANTHROPIC_DEFAULT_HAIKU_MODEL":
		return "haiku"
	case "CLAUDE_CODE_SUBAGENT_MODEL":
		return "subagent"
	case "CLAUDE_CODE_EFFORT_LEVEL":
		return "effort"
	case "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":
		return "no-traffic"
	case "CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK":
		return "no-stream"
	default:
		return envKey
	}
}
