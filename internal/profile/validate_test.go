package profile

import "testing"

func TestValidateProfile_RequiresTokenAndBaseURL(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		err := ValidateProfile("demo", Profile{
			Env: map[string]string{
				"ANTHROPIC_BASE_URL": "https://example.com",
			},
		})
		if err == nil || err.Error() != "配置 \"demo\" 缺少必填字段：ANTHROPIC_AUTH_TOKEN" {
			t.Fatalf("expected missing token error, got %v", err)
		}
	})

	t.Run("missing base url", func(t *testing.T) {
		err := ValidateProfile("demo", Profile{
			Env: map[string]string{
				"ANTHROPIC_AUTH_TOKEN": "token",
			},
		})
		if err == nil || err.Error() != "配置 \"demo\" 缺少必填字段：ANTHROPIC_BASE_URL" {
			t.Fatalf("expected missing base url error, got %v", err)
		}
	})
}

func TestValidateProfile_AllowsMissingOptionalModels(t *testing.T) {
	err := ValidateProfile("demo", Profile{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "token",
			"ANTHROPIC_BASE_URL":   "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("expected optional models to be optional, got %v", err)
	}
}

func TestValidateProfile_AllowsClaudeCodeFields(t *testing.T) {
	err := ValidateProfile("demo", Profile{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN":                      "token",
			"ANTHROPIC_BASE_URL":                        "https://example.com",
			"CLAUDE_CODE_SUBAGENT_MODEL":                "haiku",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":  "1",
			"CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK": "1",
		},
	})
	if err != nil {
		t.Fatalf("expected Claude Code fields to be supported, got %v", err)
	}
}

func TestValidateProfile_RejectsOfficialProfileName(t *testing.T) {
	err := ValidateProfile(OfficialProfileName, Profile{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "token",
			"ANTHROPIC_BASE_URL":   "https://example.com",
		},
	})
	if err == nil {
		t.Fatal("expected official profile name to be rejected")
	}
}
