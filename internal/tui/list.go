package tui

import (
	"sort"

	"cc-env/internal/profile"
)

func profileNamesSorted(profiles map[string]profile.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// orderProfiles 把 current 放到首位，其余保持传入顺序。
func orderProfiles(names []string, current string) []string {
	ordered := make([]string, 0, len(names))
	found := false
	for _, name := range names {
		if name == current {
			found = true
			break
		}
	}
	if found && current != "" {
		ordered = append(ordered, current)
	}
	for _, name := range names {
		if name == current {
			continue
		}
		ordered = append(ordered, name)
	}
	return ordered
}
