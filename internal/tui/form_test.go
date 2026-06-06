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
