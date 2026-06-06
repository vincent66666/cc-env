package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cc-env/internal/output"
	"cc-env/internal/profile"
	"cc-env/internal/tui"
)

type Paths struct {
	Profiles string
}

func Run(args []string, stdout, stderr io.Writer) int {
	command := Parse(args)
	paths := defaultPaths()

	switch command.Name {
	case "":
		return runDefault(paths, stdout, stderr)
	case "current":
		return runCurrent(paths, stdout, stderr)
	default:
		return runProfileCommand(paths, command.Name, command.Args, stderr)
	}
}

// runTUI 可在测试中替换。
var runTUI = tui.Run

// isInteractive 判定 stdin 与 stdout 是否均为终端；可在测试中替换。
var isInteractive = func(stdout io.Writer) bool {
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	if err != nil || stat.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	inStat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return inStat.Mode()&os.ModeCharDevice != 0
}

func runDefault(paths Paths, stdout, stderr io.Writer) int {
	if !isInteractive(stdout) {
		return runNonInteractiveStatus(paths, stdout, stderr)
	}

	result, err := runTUI(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "启动交互界面失败：%v\n", err)
		return 1
	}
	if !result.Launch {
		return 0
	}
	return switchProfile(paths, result.Target, nil, stderr)
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

func runNonInteractiveStatus(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		if shouldRenderUnknownForProfileLoadError(err) {
			_, _ = io.WriteString(stdout, "当前配置：未知\n")
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	currentProfile, currentDisplay, _, _, ok := currentStatus(data)
	if !ok {
		_, _ = io.WriteString(stdout, "当前配置：未知\n")
		return 0
	}

	names := availableNames(data.Profiles, data.Current)
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
