package tui

import (
	"strings"
	"testing"

	"cc-env/internal/profile"
)

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

// TestRenderPreviewKeysDoNotWrap 防止长 env key 在预览窄列里折行成多行碎片。
func TestRenderPreviewKeysDoNotWrap(t *testing.T) {
	p := profile.Profile{Description: "d", Env: map[string]string{}}
	for _, k := range profile.ManagedEnvKeys {
		p.Env[k] = "v"
	}

	out := renderPreview("demo", p)

	nonEmpty := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}

	// 预览标题 + 名称 + 描述 + 每个 env 字段，各占一行。
	want := 3 + len(profile.ManagedEnvKeys)
	if nonEmpty != want {
		t.Fatalf("preview produced %d non-empty lines, want %d (key wrapping?):\n%s", nonEmpty, want, out)
	}
}
