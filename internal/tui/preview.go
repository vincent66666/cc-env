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

// renderPreview 渲染高亮 profile 的字段（token 遮罩）。
// 标题用选中 profile 名（卡片由调用方加边框），以呼应左侧选中行。
func renderPreview(name string, current bool, p profile.Profile) string {
	var b strings.Builder
	title := name
	if current {
		title += "（当前）"
	}
	b.WriteString(previewTitle.Render(title) + "\n\n")

	if profile.IsOfficialName(name) {
		b.WriteString("官方登录态")
		return b.String()
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
	return strings.TrimRight(b.String(), "\n")
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
