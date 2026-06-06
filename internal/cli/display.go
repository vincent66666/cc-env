package cli

import (
	"sort"
	"strings"

	"cc-env/internal/profile"
)

func profileNames(profiles map[string]profile.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func modeNames(profiles map[string]profile.Profile) []string {
	names := make([]string, 0, len(profiles)+1)
	names = append(names, profileNames(profiles)...)
	names = append(names, profile.OfficialProfileName)
	return names
}

func availableNames(profiles map[string]profile.Profile, current string) []string {
	names := make([]string, 0, len(profiles))
	for _, name := range modeNames(profiles) {
		if name == current {
			continue
		}
		names = append(names, name)
	}
	return names
}

func displayNamesForProfiles(profiles map[string]profile.Profile, names []string) []string {
	displayNames := make([]string, 0, len(names))
	for _, name := range names {
		if profile.IsOfficialName(name) {
			displayNames = append(displayNames, profileDisplayName(name, officialProfileDescription()))
			continue
		}
		displayNames = append(displayNames, profileDisplayName(name, profiles[name].Description))
	}
	return displayNames
}

func profileDisplayName(name, description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return name
	}

	return name + " - " + description
}

func profileListDisplayName(name, description string, current bool) string {
	if current {
		return profileDisplayName(name+"（当前）", description)
	}

	return profileDisplayName(name, description)
}

func profileDescriptions(profiles map[string]profile.Profile) map[string]string {
	descriptions := make(map[string]string, len(profiles))
	for name, item := range profiles {
		descriptions[name] = item.Description
	}
	descriptions[profile.OfficialProfileName] = officialProfileDescription()
	return descriptions
}

func currentDescription(name string, currentProfile profile.Profile) string {
	if profile.IsOfficialName(name) {
		return officialProfileDescription()
	}
	return currentProfile.Description
}

func officialProfileDescription() string {
	return "官方登录态"
}
