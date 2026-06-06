package tui

import (
	"reflect"
	"testing"

	"cc-env/internal/profile"
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

func TestBuildItemsPutsOfficialFirstThenCurrent(t *testing.T) {
	items := buildItems(sampleData())
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	first := items[0].(profileItem)
	second := items[1].(profileItem)
	if first.name != profile.OfficialProfileName || !first.official {
		t.Fatalf("first item = %+v, want official first", first)
	}
	if second.name != "kimi" || !second.current {
		t.Fatalf("second item = %+v, want current kimi second", second)
	}
}

func TestBuildItemsOfficialDescription(t *testing.T) {
	items := buildItems(sampleData())
	first := items[0].(profileItem)
	if first.description != "官方登录态" {
		t.Fatalf("official description = %q", first.description)
	}
}
