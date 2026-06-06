package tui

import (
	"sort"

	"cc-env/internal/profile"
	"github.com/charmbracelet/bubbles/list"
)

func profileNamesSorted(profiles map[string]profile.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type profileItem struct {
	name        string
	description string
	current     bool
	official    bool
}

func (i profileItem) Title() string {
	if i.current {
		return i.name + "（当前）"
	}
	return i.name
}

func (i profileItem) Description() string { return i.description }

func (i profileItem) FilterValue() string { return i.name + " " + i.description }

func descriptionFor(data profile.ProfilesFile, name string) string {
	if profile.IsOfficialName(name) {
		return "官方登录态"
	}
	return data.Profiles[name].Description
}

func buildItems(data profile.ProfilesFile) []list.Item {
	names := profileNamesSorted(data.Profiles)
	names = append(names, profile.OfficialProfileName)
	ordered := orderProfiles(names, data.Current)

	items := make([]list.Item, 0, len(ordered))
	for _, name := range ordered {
		items = append(items, profileItem{
			name:        name,
			description: descriptionFor(data, name),
			current:     name == data.Current,
			official:    profile.IsOfficialName(name),
		})
	}
	return items
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
