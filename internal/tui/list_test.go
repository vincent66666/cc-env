package tui

import (
	"reflect"
	"testing"
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
