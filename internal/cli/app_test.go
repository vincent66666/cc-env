package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cc-env/internal/profile"
	"cc-env/internal/tui"
)

func TestMain(m *testing.M) {
	isInteractive = func(io.Writer) bool { return false }
	runTUI = func(string) (tui.Result, error) { return tui.Result{}, nil }
	launchClaude = func(args []string, env []string) error { return nil }
	os.Exit(m.Run())
}

type claudeLaunchCall struct {
	called bool
	args   []string
	env    []string
}

func stubClaudeLauncher(t *testing.T) *claudeLaunchCall {
	t.Helper()

	original := launchClaude
	call := &claudeLaunchCall{}
	launchClaude = func(args []string, env []string) error {
		call.called = true
		call.args = append([]string(nil), args...)
		call.env = append([]string(nil), env...)
		return nil
	}
	t.Cleanup(func() {
		launchClaude = original
	})
	return call
}

func envValue(env []string, key string) (string, bool) {
	for _, item := range env {
		envKey, value, ok := strings.Cut(item, "=")
		if ok && envKey == key {
			return value, true
		}
	}
	return "", false
}

func TestRun_ProfileCommandLaunchesClaudeWithProfileEnvAndCurrentProfile(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken:                        "token-demo",
					profile.EnvBaseURL:                          "https://demo.example.com",
					"ANTHROPIC_MODEL":                           "glm-5",
					"CLAUDE_CODE_SUBAGENT_MODEL":                "haiku",
					"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC":  "1",
					"CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK": "1",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{
  "model": "opus",
  "enabledPlugins": {
    "demo": true
  },
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "old-token"
  }
}
`)

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)
	t.Setenv(profile.EnvAuthToken, "old-token")
	t.Setenv(profile.EnvBaseURL, "https://old.example.com")
	t.Setenv("ANTHROPIC_MODEL", "old-model")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo", "--print"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected profile command to keep stdout clean, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after switch: %v", err)
	}

	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile demo, got %q", savedProfiles.Current)
	}

	if got := strings.Join(call.args, " "); got != "--print" {
		t.Fatalf("expected claude args to be forwarded, got %q", got)
	}
	if value, ok := envValue(call.env, profile.EnvAuthToken); !ok || value != "token-demo" {
		t.Fatalf("expected launched token env, got %q exists=%v", value, ok)
	}
	if value, ok := envValue(call.env, profile.EnvBaseURL); !ok || value != "https://demo.example.com" {
		t.Fatalf("expected launched base url env, got %q exists=%v", value, ok)
	}
	if value, ok := envValue(call.env, "ANTHROPIC_MODEL"); !ok || value != "glm-5" {
		t.Fatalf("expected launched model env, got %q exists=%v", value, ok)
	}
	if value, ok := envValue(call.env, "CLAUDE_CODE_SUBAGENT_MODEL"); !ok || value != "haiku" {
		t.Fatalf("expected launched subagent model env, got %q exists=%v", value, ok)
	}
	if value, ok := envValue(call.env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"); !ok || value != "1" {
		t.Fatalf("expected launched nonessential traffic flag, got %q exists=%v", value, ok)
	}
	if value, ok := envValue(call.env, "CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK"); !ok || value != "1" {
		t.Fatalf("expected launched nonstreaming fallback flag, got %q exists=%v", value, ok)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings after switch: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("unmarshal settings after switch: %v", err)
	}

	if got["model"] != "opus" {
		t.Fatalf("expected model to remain unchanged, got %#v", got["model"])
	}

	env, ok := got["env"].(map[string]any)
	if !ok {
		t.Fatalf("expected env object, got %#v", got["env"])
	}

	if env[profile.EnvAuthToken] != "old-token" {
		t.Fatalf("expected existing settings token to remain unchanged, got %#v", env[profile.EnvAuthToken])
	}

	if _, exists := env[profile.EnvBaseURL]; exists {
		t.Fatalf("expected settings base url to remain absent, got %#v", env[profile.EnvBaseURL])
	}
}

func TestRun_ProfileCommandRecoversFromMissingCurrentProfile(t *testing.T) {
	profilesPath := writeRawProfilesFixture(t, `{
  "version": 1,
  "current": "ghost",
  "profiles": {
    "demo": {
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token-demo",
        "ANTHROPIC_BASE_URL": "https://demo.example.com"
      }
    },
    "beta": {
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token-beta",
        "ANTHROPIC_BASE_URL": "https://beta.example.com"
      }
    }
  }
}
`)
	settingsPath := writeSettingsFixture(t, `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`)

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected profile command to keep stdout clean, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after recovery switch: %v", err)
	}
	if savedProfiles.Current != "demo" {
		t.Fatalf("expected recovered current profile demo, got %q", savedProfiles.Current)
	}
}

func TestRun_ProfileCommandIgnoresInvalidSettingsAndAdvancesCurrent(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":`)

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected profile command to ignore invalid settings json, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after failed switch: %v", err)
	}

	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to advance to demo, got %q", savedProfiles.Current)
	}
}

func TestRun_ProfileCommandTrimsNameArgument(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`)

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{" demo "}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected profile command with spaced name to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after trimmed-name profile command: %v", err)
	}

	if savedProfiles.Current != "demo" {
		t.Fatalf("expected trimmed-name profile command to switch to demo, got %q", savedProfiles.Current)
	}
}

func TestRun_ProfileCommandStripsOptionalSeparatorBeforeClaudeArgs(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo", "--", "--print"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected profile command with separator to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := strings.Join(call.args, " "); got != "--print" {
		t.Fatalf("expected claude args to be forwarded, got %q", got)
	}
}

func TestRun_OfficialProfileCommandClearsManagedEnv(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	for _, key := range profile.ManagedEnvKeys {
		t.Setenv(key, "old-"+key)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{profile.OfficialProfileName}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected official profile command to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after official profile command: %v", err)
	}
	if savedProfiles.Current != profile.OfficialProfileName {
		t.Fatalf("expected current profile official, got %q", savedProfiles.Current)
	}
	for _, key := range profile.ManagedEnvKeys {
		if value, ok := envValue(call.env, key); ok {
			t.Fatalf("expected official mode to clear %s, got %q", key, value)
		}
	}
}

func TestRun_ProfileCommandIgnoresUnwritableBackupDir(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`)

	homeFile := filepath.Join(t.TempDir(), "home-file")
	if err := os.WriteFile(homeFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fake home file: %v", err)
	}

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)
	t.Setenv("HOME", homeFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected profile command to ignore backup dir, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after profile command: %v", err)
	}

	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to advance to demo, got %q", savedProfiles.Current)
	}
}

func TestRun_ProfileCommandRollsBackSettingsWhenUpdatingCurrentFails(t *testing.T) {
	root := t.TempDir()
	profilesDir := filepath.Join(root, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("create profiles dir: %v", err)
	}

	profilesPath := filepath.Join(profilesDir, "profiles.json")
	if err := profile.Save(profilesPath, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	}); err != nil {
		t.Fatalf("save profiles fixture: %v", err)
	}

	settingsPath := writeSettingsFixture(t, `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "old-token",
    "ANTHROPIC_BASE_URL": "https://old.example.com"
  }
}
`)

	if err := os.Chmod(profilesDir, 0o555); err != nil {
		t.Fatalf("make profiles dir read-only: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(profilesDir, 0o755)
	})

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected profile command to fail when updating current profile cannot persist")
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after failed profile command: %v", err)
	}

	if savedProfiles.Current != "beta" {
		t.Fatalf("expected current profile to remain beta, got %q", savedProfiles.Current)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings after failed profile command: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("unmarshal settings after failed profile command: %v", err)
	}

	env, ok := got["env"].(map[string]any)
	if !ok {
		t.Fatalf("expected env object, got %#v", got["env"])
	}

	if env[profile.EnvAuthToken] != "old-token" {
		t.Fatalf("expected settings token to roll back, got %#v", env[profile.EnvAuthToken])
	}

	if env[profile.EnvBaseURL] != "https://old.example.com" {
		t.Fatalf("expected settings base url to roll back, got %#v", env[profile.EnvBaseURL])
	}
}

func TestRun_ProfileCommandIgnoresCustomSettingsPath(t *testing.T) {
	root := t.TempDir()
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsDirAsFile := filepath.Join(root, "settings-parent")
	if err := os.WriteFile(settingsDirAsFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fake settings parent file: %v", err)
	}
	settingsPath := filepath.Join(settingsDirAsFile, "settings.json")

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected profile command to ignore custom settings path, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after profile command: %v", err)
	}

	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to advance to demo, got %q", savedProfiles.Current)
	}
}

func TestRun_UnknownCommandIsTreatedAsProfileName(t *testing.T) {
	t.Setenv("CC_ENV_PROFILES_PATH", filepath.Join(t.TempDir(), "profiles.json"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"import", "--from", "/tmp/legacy"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected import to fail, got %d", exitCode)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}

	if got := stderr.String(); got != "未找到配置 \"import\"\n" {
		t.Fatalf("expected missing profile error, got %q", got)
	}
}

func TestRun_CurrentPrintsCurrentProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token",
					profile.EnvBaseURL:   "https://example.com",
				},
			},
		},
	})

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"current"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stdout.String(); got != "demo\n" {
		t.Fatalf("expected current profile output, got %q", got)
	}
}

func TestRun_LegacyProfilesPathEnvFallback(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token",
					profile.EnvBaseURL:   "https://example.com",
				},
			},
		},
	})

	t.Setenv("CC_ENV_PROFILES_PATH", "")
	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"current"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected legacy profiles path fallback to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "demo\n" {
		t.Fatalf("expected current profile output through legacy path, got %q", got)
	}
}

func TestRun_CurrentPrintsUnknownWhenProfilesFileIsMissing(t *testing.T) {
	t.Setenv("CC_ENV_PROFILES_PATH", filepath.Join(t.TempDir(), "profiles.json"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"current"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stdout.String(); got != "未知\n" {
		t.Fatalf("expected missing profiles file to print unknown, got %q", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}
}

func TestRun_CurrentPrintsUnknownWhenCurrentProfileIsMissing(t *testing.T) {
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	content := `{
  "version": 1,
  "current": "ghost",
  "profiles": {
    "demo": {
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token",
        "ANTHROPIC_BASE_URL": "https://example.com"
      }
    }
  }
}
`
	if err := os.WriteFile(profilesPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write profiles fixture: %v", err)
	}

	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"current"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stdout.String(); got != "未知\n" {
		t.Fatalf("expected missing current profile to print unknown, got %q", got)
	}
}

func TestRun_CurrentReportsLoadErrorWhenProfilesFileIsInvalid(t *testing.T) {
	profilesPath := writeRawProfilesFixture(t, "{")
	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"current"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "加载配置失败：") {
		t.Fatalf("expected load error stderr, got %q", got)
	}
}

func TestRun_NoArgsNonInteractivePrintsUnknownWhenCurrentProfileIsMissing(t *testing.T) {
	call := stubClaudeLauncher(t)
	profilesPath := filepath.Join(t.TempDir(), "profiles.json")
	content := `{
  "version": 1,
  "current": "ghost",
  "profiles": {
    "demo": {
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token",
        "ANTHROPIC_BASE_URL": "https://example.com"
      }
    }
  }
}
`
	if err := os.WriteFile(profilesPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write profiles fixture: %v", err)
	}
	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got != "当前配置：未知\n" {
		t.Fatalf("expected unknown current status, got %q", got)
	}
	if call.called {
		t.Fatalf("no-args non-interactive must not launch claude")
	}
}

func TestRun_NoArgsNonInteractiveReportsLoadErrorWhenProfilesFileIsInvalid(t *testing.T) {
	profilesPath := writeRawProfilesFixture(t, "{")
	t.Setenv("CC_ENV_PROFILES_PATH", profilesPath)

	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "加载配置失败：") {
		t.Fatalf("expected load error stderr, got %q", got)
	}
}

func writeProfilesFixture(t *testing.T, data profile.ProfilesFile) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := profile.Save(path, data); err != nil {
		t.Fatalf("save profiles fixture: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}

	return path
}

func writeRawProfilesFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write raw profiles fixture: %v", err)
	}

	return path
}

func writeSettingsFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write settings fixture: %v", err)
	}

	return path
}

func loadProfilesFixture(t *testing.T, path string) profile.ProfilesFile {
	t.Helper()

	data, err := profile.LoadForList(path)
	if err != nil {
		t.Fatalf("load profiles fixture: %v", err)
	}
	return data
}

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
	if call.called {
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
	if !call.called {
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
	if call.called {
		t.Fatalf("quit should not launch claude")
	}
}
