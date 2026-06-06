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

func TestBuildItemsOrdersCurrentFirstOfficialLast(t *testing.T) {
	items := buildItems(sampleData())
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	first := items[0].(profileItem)
	last := items[2].(profileItem)
	if first.name != "kimi" || !first.current {
		t.Fatalf("first item = %+v, want current kimi", first)
	}
	if last.name != profile.OfficialProfileName || !last.official {
		t.Fatalf("last item = %+v, want official", last)
	}
}

func TestBuildItemsOfficialDescription(t *testing.T) {
	items := buildItems(sampleData())
	last := items[2].(profileItem)
	if last.description != "官方登录态" {
		t.Fatalf("official description = %q", last.description)
	}
}
