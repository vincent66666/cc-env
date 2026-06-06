package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cc-env/internal/output"
	"cc-env/internal/profile"
)

type Paths struct {
	Profiles string
}

func Run(args []string, stdout, stderr io.Writer) int {
	command := Parse(args)
	paths := defaultPaths()

	switch command.Name {
	case "":
		return switchProfile(paths, profile.OfficialProfileName, nil, stderr)
	case "current":
		return runCurrent(paths, stdout, stderr)
	case "list":
		return runList(paths, stdout, stderr)
	case "status":
		return runStatus(paths, stdout, stderr)
	case "add":
		return runAdd(paths, command.Args, stdout, stderr)
	case "edit":
		return runEdit(paths, command.Args, stdout, stderr)
	case "remove":
		return runRemove(paths, command.Args, stdout, stderr)
	case "rename":
		return runRename(paths, command.Args, stdout, stderr)
	default:
		return runProfileCommand(paths, command.Name, command.Args, stderr)
	}
}

func defaultPaths() Paths {
	profilesPath := os.Getenv("CC_ENV_PROFILES_PATH")
	if profilesPath == "" {
		profilesPath = os.Getenv("CC_SWITCH_PROFILES_PATH")
	}
	if profilesPath == "" {
		profilesPath = os.ExpandEnv("$HOME/.claude/cc-env/profiles.json")
	}

	return Paths{
		Profiles: profilesPath,
	}
}

func runCurrent(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		if shouldRenderUnknownForProfileLoadError(err) {
			_, _ = io.WriteString(stdout, "未知\n")
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	if data.Current == "" {
		_, _ = io.WriteString(stdout, "未知\n")
		return 0
	}

	if profile.IsOfficialName(data.Current) {
		_, _ = fmt.Fprintf(stdout, "%s\n", data.Current)
		return 0
	}

	if _, ok := data.Profiles[data.Current]; !ok {
		_, _ = io.WriteString(stdout, "未知\n")
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "%s\n", data.Current)
	return 0
}

func runList(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.LoadForList(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	names := modeNames(data.Profiles)
	if selectorInteractive(stdout) && len(names) > 0 {
		return runInteractiveList(paths, listMenu{
			profiles:     prioritizeCurrentProfile(names, data.Current),
			currentName:  data.Current,
			descriptions: profileDescriptions(data.Profiles),
		}, stdout, stderr)
	}

	currentDisplay := ""
	if profile.IsOfficialName(data.Current) {
		currentDisplay = profileDisplayName(data.Current, officialProfileDescription())
	} else if currentProfile, ok := data.Profiles[data.Current]; ok {
		currentDisplay = profileDisplayName(data.Current, currentProfile.Description)
	}

	return output.RenderList(stdout, currentDisplay, displayNamesForProfiles(data.Profiles, names))
}

func runStatus(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		if shouldRenderUnknownForProfileLoadError(err) {
			_, _ = io.WriteString(stdout, "当前配置：未知\n")
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	currentProfile, currentDisplay, currentBaseURL, currentModel, ok := currentStatus(data)
	if !ok {
		_, _ = io.WriteString(stdout, "当前配置：未知\n")
		return 0
	}

	names := availableNames(data.Profiles, data.Current)
	if selectorInteractive(stdout) && len(names) > 0 {
		selector := statusSelector{
			currentName:        data.Current,
			currentDescription: currentDescription(data.Current, currentProfile),
			baseURL:            currentBaseURL,
			model:              currentModel,
			names:              names,
			descriptions:       profileDescriptions(data.Profiles),
		}
		return runInteractiveStatus(paths, selector, stdout, stderr)
	}

	return output.RenderStatus(
		stdout,
		currentDisplay,
		currentProfile,
		displayNamesForProfiles(data.Profiles, names),
	)
}

func currentStatus(data profile.ProfilesFile) (profile.Profile, string, string, string, bool) {
	if profile.IsOfficialName(data.Current) {
		currentProfile := profile.Profile{
			Description: officialProfileDescription(),
			Env: map[string]string{
				profile.EnvBaseURL: "官方登录态",
			},
		}
		return currentProfile,
			profileDisplayName(data.Current, currentProfile.Description),
			"官方登录态",
			"-",
			true
	}

	currentProfile, ok := data.Profiles[data.Current]
	if !ok {
		return profile.Profile{}, "", "", "", false
	}
	return currentProfile,
		profileDisplayName(data.Current, currentProfile.Description),
		currentProfile.Env[profile.EnvBaseURL],
		currentProfile.Env["ANTHROPIC_MODEL"],
		true
}

func shouldRenderUnknownForProfileLoadError(err error) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}

	return errors.Is(err, profile.ErrCurrentProfileMissing)
}

func normalizeProfileName(name string) string {
	return strings.TrimSpace(name)
}

func formatCLIError(err error) string {
	if errors.Is(err, io.EOF) {
		return "输入已结束"
	}

	return err.Error()
}
