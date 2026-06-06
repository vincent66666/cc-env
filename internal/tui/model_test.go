package tui

import (
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
