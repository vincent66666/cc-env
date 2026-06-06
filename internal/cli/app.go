package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"cc-env/internal/output"
	"cc-env/internal/profile"
)

type Paths struct {
	Profiles string
}

type selectorAction int

const (
	selectorActionUp selectorAction = iota
	selectorActionDown
	selectorActionEnter
	selectorActionEdit
	selectorActionRename
	selectorActionRemove
	selectorActionQuit
)

const (
	clearScreenSequence      = "\x1b[H\x1b[2J"
	enterAlternateScreenMode = "\x1b[?1049h"
	exitAlternateScreenMode  = "\x1b[?1049l"
)

var (
	promptReader      io.Reader = os.Stdin
	promptWriter      io.Writer = os.Stdout
	promptInteractive           = func() bool {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return false
		}

		return stat.Mode()&os.ModeCharDevice != 0
	}
	startInteractiveSession = startInteractiveTerminalSession
	launchClaude            = runClaude
)

func selectorInteractive(stdout io.Writer) bool {
	if !promptInteractive() || !rawTerminalSupported() {
		return false
	}

	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return stat.Mode()&os.ModeCharDevice != 0
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

func runProfileCommand(paths Paths, name string, args []string, stderr io.Writer) int {
	target, claudeArgs := parseProfileCommandArgs(name, args)
	return switchProfile(paths, target, claudeArgs, stderr)
}

func parseProfileCommandArgs(name string, args []string) (string, []string) {
	target := normalizeProfileName(name)
	if len(args) > 0 && args[0] == "--" {
		return target, args[1:]
	}
	return target, args
}

func switchProfile(paths Paths, target string, claudeArgs []string, stderr io.Writer) int {
	data, err := profile.LoadForList(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	if target == "" {
		target = data.Current
	}
	if target == "" {
		target = profile.OfficialProfileName
	}

	var targetEnv map[string]string
	if !profile.IsOfficialName(target) {
		targetProfile, ok := data.Profiles[target]
		if !ok {
			_, _ = fmt.Fprintf(stderr, "未找到配置 %q\n", target)
			return 1
		}

		if err := profile.ValidateProfile(target, targetProfile); err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
		targetEnv = targetProfile.Env
	}

	data.Current = target
	if err := profile.Save(paths.Profiles, data); err != nil {
		_, _ = fmt.Fprintf(stderr, "更新当前配置失败：%v\n", err)
		return 1
	}

	env := buildClaudeEnv(os.Environ(), targetEnv)
	if err := launchClaude(claudeArgs, env); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		_, _ = fmt.Fprintf(stderr, "启动 claude 失败：%v\n", err)
		return 1
	}
	return 0
}

func runClaude(args []string, env []string) error {
	command := exec.Command("claude", args...)
	command.Env = env
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func buildClaudeEnv(base []string, overlay map[string]string) []string {
	managed := map[string]struct{}{}
	for _, key := range profile.ManagedEnvKeys {
		managed[key] = struct{}{}
	}

	values := map[string]string{}
	order := []string{}
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, isManaged := managed[key]; isManaged {
			continue
		}
		if _, seen := values[key]; !seen {
			order = append(order, key)
		}
		values[key] = value
	}

	env := make([]string, 0, len(order)+len(overlay))
	for _, key := range order {
		env = append(env, key+"="+values[key])
	}
	for _, key := range profile.ManagedEnvKeys {
		value := strings.TrimSpace(overlay[key])
		if value == "" {
			continue
		}
		env = append(env, key+"="+value)
	}
	return env
}

func runInteractiveStatus(paths Paths, selector statusSelector, stdout, stderr io.Writer) int {
	stdinFile, ok := promptReader.(*os.File)
	if !ok {
		_, _ = io.WriteString(stdout, selector.render())
		return 0
	}

	closeInteractive, err := startInteractiveSession(stdinFile, stdout)
	if err != nil {
		_, _ = io.WriteString(stdout, selector.render())
		return 0
	}
	defer closeInteractive()

	reader := bufio.NewReader(promptReader)
	for {
		_, _ = io.WriteString(stdout, clearScreenSequence)
		_, _ = io.WriteString(stdout, selector.render())

		action, err := readSelectorAction(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			_, _ = fmt.Fprintf(stderr, "读取交互输入失败：%v\n", err)
			return 1
		}

		switch action {
		case selectorActionUp:
			selector.moveUp()
		case selectorActionDown:
			selector.moveDown()
		case selectorActionEnter:
			closeInteractive()
			return switchProfile(paths, selector.selectedName(), nil, stderr)
		case selectorActionQuit:
			return 0
		}
	}
}

func runInteractiveList(paths Paths, menu listMenu, stdout, stderr io.Writer) int {
	stdinFile, ok := promptReader.(*os.File)
	if !ok {
		_, _ = io.WriteString(stdout, menu.render())
		return 0
	}

	closeInteractive, err := startInteractiveSession(stdinFile, stdout)
	if err != nil {
		_, _ = io.WriteString(stdout, menu.render())
		return 0
	}
	defer func() {
		if closeInteractive != nil {
			closeInteractive()
		}
	}()

	reader := bufio.NewReader(promptReader)
	for {
		_, _ = io.WriteString(stdout, clearScreenSequence)
		_, _ = io.WriteString(stdout, menu.render())

		action, err := readSelectorAction(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			_, _ = fmt.Fprintf(stderr, "读取交互输入失败：%v\n", err)
			return 1
		}

		switch action {
		case selectorActionUp:
			menu.moveUp()
		case selectorActionDown:
			menu.moveDown()
		case selectorActionEnter:
			if menu.mode == listMenuModeProfiles {
				if menuHasMissingCurrentProfile(menu) {
					closeInteractive()
					return switchProfile(paths, menu.selectedProfile(), nil, stderr)
				}
				menu.enterActions()
				continue
			}

			switch menu.mode {
			case listMenuModeActions:
				exitCode, done := executeListAction(paths, &menu, menu.selectedAction(), reader, stdinFile, stdout, stderr, &closeInteractive)
				if done {
					return exitCode
				}
			case listMenuModeDeleteConfirm:
				switch menu.selectedConfirmAction() {
				case listMenuConfirmDelete:
					exitCode, done := executeListDelete(paths, &menu, reader, stdinFile, stdout, stderr, &closeInteractive)
					if done {
						return exitCode
					}
				case listMenuConfirmCancel:
					menu.backToActions()
				}
			}
		case selectorActionEdit:
			if menu.mode == listMenuModeDeleteConfirm {
				continue
			}
			exitCode, done := executeListAction(paths, &menu, listMenuActionEdit, reader, stdinFile, stdout, stderr, &closeInteractive)
			if done {
				return exitCode
			}
		case selectorActionRename:
			if menu.mode == listMenuModeDeleteConfirm {
				continue
			}
			exitCode, done := executeListAction(paths, &menu, listMenuActionRename, reader, stdinFile, stdout, stderr, &closeInteractive)
			if done {
				return exitCode
			}
		case selectorActionRemove:
			if menu.mode == listMenuModeDeleteConfirm {
				continue
			}
			exitCode, done := executeListAction(paths, &menu, listMenuActionRemove, reader, stdinFile, stdout, stderr, &closeInteractive)
			if done {
				return exitCode
			}
		case selectorActionQuit:
			return 0
		}
	}
}

func executeListAction(
	paths Paths,
	menu *listMenu,
	action listMenuAction,
	reader *bufio.Reader,
	stdinFile *os.File,
	stdout, stderr io.Writer,
	closeInteractive *func(),
) (int, bool) {
	switch action {
	case listMenuActionSwitch:
		if selected := menu.selectedProfile(); selected != "" {
			if *closeInteractive != nil {
				(*closeInteractive)()
			}
			return switchProfile(paths, selected, nil, stderr), true
		}
	case listMenuActionEdit:
		selected := menu.selectedProfile()
		if selected == "" || profile.IsOfficialName(selected) {
			return 0, false
		}
		if *closeInteractive != nil {
			(*closeInteractive)()
		}

		exitCode := runEditWithPromptReader(paths, []string{selected}, reader, stdout, stderr)
		if exitCode != 0 {
			return exitCode, true
		}

		return resumeListSession(paths, menu, stdinFile, stdout, stderr, closeInteractive, selected, menu.index)
	case listMenuActionRename:
		selected := menu.selectedProfile()
		if selected == "" || profile.IsOfficialName(selected) {
			return 0, false
		}
		selectedIndex := menu.index

		if *closeInteractive != nil {
			(*closeInteractive)()
		}

		newName, err := promptRenameName(reader)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1, true
		}

		exitCode := runRename(paths, []string{selected, newName}, stdout, stderr)
		if exitCode != 0 {
			return exitCode, true
		}

		return resumeListSession(paths, menu, stdinFile, stdout, stderr, closeInteractive, newName, selectedIndex)
	case listMenuActionRemove:
		selected := menu.selectedProfile()
		if selected == "" || selected == menu.currentName || profile.IsOfficialName(selected) {
			return 0, false
		}
		menu.enterDeleteConfirm()
	case listMenuActionBack:
		menu.backToList()
	}

	return 0, false
}

func executeListDelete(
	paths Paths,
	menu *listMenu,
	reader *bufio.Reader,
	stdinFile *os.File,
	stdout, stderr io.Writer,
	closeInteractive *func(),
) (int, bool) {
	_ = reader
	selected := menu.selectedProfile()
	if selected == "" {
		return 0, false
	}
	selectedIndex := menu.index

	if *closeInteractive != nil {
		(*closeInteractive)()
	}

	exitCode := runRemove(paths, []string{selected}, stdout, stderr)
	if exitCode != 0 {
		return exitCode, true
	}

	return resumeListSession(paths, menu, stdinFile, stdout, stderr, closeInteractive, "", selectedIndex)
}

func resumeListSession(
	paths Paths,
	menu *listMenu,
	stdinFile *os.File,
	stdout, stderr io.Writer,
	closeInteractive *func(),
	selectedName string,
	selectedIndex int,
) (int, bool) {
	var err error
	*closeInteractive, err = startInteractiveSession(stdinFile, stdout)
	if err != nil {
		return 1, true
	}

	*menu, err = reloadListMenu(paths, selectedName, selectedIndex)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1, true
	}

	return 0, false
}

func startInteractiveTerminalSession(stdinFile *os.File, stdout io.Writer) (func(), error) {
	restore, err := makeRawTerminal(stdinFile)
	if err != nil {
		return nil, err
	}

	_, _ = io.WriteString(stdout, enterAlternateScreenMode)

	active := true
	return func() {
		if !active {
			return
		}
		active = false
		restore()
		_, _ = io.WriteString(stdout, exitAlternateScreenMode)
	}, nil
}

func reloadListMenu(paths Paths, selectedName string, selectedIndex int) (listMenu, error) {
	data, err := profile.LoadForList(paths.Profiles)
	if err != nil {
		return listMenu{}, err
	}

	menu := listMenu{
		profiles:     prioritizeCurrentProfile(profileNames(data.Profiles), data.Current),
		currentName:  data.Current,
		descriptions: profileDescriptions(data.Profiles),
	}

	if selectedName != "" {
		for i, name := range menu.profiles {
			if name == selectedName {
				menu.index = i
				return menu, nil
			}
		}
	}

	if selectedIndex >= len(menu.profiles) {
		selectedIndex = len(menu.profiles) - 1
	}
	if selectedIndex < 0 {
		selectedIndex = 0
	}
	menu.index = selectedIndex
	return menu, nil
}

func readSelectorAction(reader *bufio.Reader) (selectorAction, error) {
	for {
		key, err := reader.ReadByte()
		if err != nil {
			return selectorActionQuit, err
		}

		switch key {
		case 0x03:
			return selectorActionQuit, nil
		case 'e', 'E':
			return selectorActionEdit, nil
		case 'r', 'R':
			return selectorActionRename, nil
		case 'd', 'D':
			return selectorActionRemove, nil
		case 'q', 'Q':
			return selectorActionQuit, nil
		case '\r', '\n':
			return selectorActionEnter, nil
		case 0x1b:
			next, err := reader.ReadByte()
			if err != nil {
				return selectorActionQuit, err
			}
			if next != '[' {
				continue
			}

			arrow, err := reader.ReadByte()
			if err != nil {
				return selectorActionQuit, err
			}

			switch arrow {
			case 'A':
				return selectorActionUp, nil
			case 'B':
				return selectorActionDown, nil
			}
		}
	}
}

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

func menuHasMissingCurrentProfile(menu listMenu) bool {
	if menu.currentName == "" {
		return false
	}

	for _, name := range menu.profiles {
		if name == menu.currentName {
			return false
		}
	}

	return true
}

type profileFlags struct {
	description                 string
	token                       string
	baseURL                     string
	model                       string
	defaultOpus                 string
	defaultSonnet               string
	defaultHaiku                string
	subagentModel               string
	disableNonessentialTraffic  bool
	disableNonstreamingFallback bool
}

func runAdd(paths Paths, args []string, stdout, stderr io.Writer) int {
	name, input, err := parseProfileFlags(args, false)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	var promptSession *bufio.Reader
	if promptInteractive() {
		promptSession = bufio.NewReader(promptReader)
	}

	data, err := profile.Load(paths.Profiles)
	if err != nil && !os.IsNotExist(err) {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}
	if os.IsNotExist(err) {
		data = profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{}}
	}

	if promptInteractive() && strings.TrimSpace(name) == "" {
		name, err = promptAddName(promptSession)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
	}

	if strings.TrimSpace(name) == "" {
		_, _ = io.WriteString(stderr, "必须提供配置名称\n")
		return 1
	}
	if isReservedProfileName(name) {
		_, _ = fmt.Fprintf(stderr, "配置 %q 是内置配置名称，不能作为普通 profile\n", name)
		return 1
	}

	if _, exists := data.Profiles[name]; exists {
		_, _ = fmt.Fprintf(stderr, "配置 %q 已存在\n", name)
		return 1
	}

	if promptInteractive() {
		input, err = promptAddFields(promptSession, input)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
	}

	newProfile := profile.Profile{
		Description: input.description,
		Env:         buildProfileEnv(input, nil),
	}

	if err := profile.ValidateProfile(name, newProfile); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	data.Profiles[name] = newProfile
	if err := profile.Save(paths.Profiles, data); err != nil {
		_, _ = fmt.Fprintf(stderr, "保存配置失败：%v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已添加配置：%s\n", name)
	return 0
}

func runEdit(paths Paths, args []string, stdout, stderr io.Writer) int {
	var promptSession *bufio.Reader
	if promptInteractive() {
		promptSession = bufio.NewReader(promptReader)
	}

	return runEditWithPromptReader(paths, args, promptSession, stdout, stderr)
}

func runEditWithPromptReader(paths Paths, args []string, promptSession *bufio.Reader, stdout, stderr io.Writer) int {
	name, input, err := parseProfileFlags(args, true)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}
	if profile.IsOfficialName(name) {
		_, _ = fmt.Fprintf(stderr, "内置配置 %q 不支持编辑\n", name)
		return 1
	}

	data, err := profile.Load(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	existing, ok := data.Profiles[name]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "未找到配置 %q\n", name)
		return 1
	}

	if input.description != "" {
		existing.Description = input.description
	}
	if promptInteractive() {
		existing, err = promptEditFields(promptSession, existing, input)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
	} else {
		existing.Env = buildProfileEnv(input, existing.Env)
	}

	if err := profile.ValidateProfile(name, existing); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	data.Profiles[name] = existing
	if err := profile.Save(paths.Profiles, data); err != nil {
		_, _ = fmt.Fprintf(stderr, "保存配置失败：%v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已更新配置：%s\n", name)
	return 0
}

func parseProfileFlags(args []string, requireName bool) (string, profileFlags, error) {
	name := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = normalizeProfileName(args[0])
		flagArgs = args[1:]
	}

	if requireName && strings.TrimSpace(name) == "" {
		return "", profileFlags{}, fmt.Errorf("必须提供配置名称")
	}

	flags := flag.NewFlagSet("profile", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var input profileFlags
	flags.StringVar(&input.description, "description", "", "profile description")
	flags.StringVar(&input.token, "token", "", "anthropic auth token")
	flags.StringVar(&input.baseURL, "base-url", "", "anthropic base url")
	flags.StringVar(&input.model, "model", "", "anthropic model")
	flags.StringVar(&input.defaultOpus, "default-opus-model", "", "default opus model")
	flags.StringVar(&input.defaultSonnet, "default-sonnet-model", "", "default sonnet model")
	flags.StringVar(&input.defaultHaiku, "default-haiku-model", "", "default haiku model")
	flags.StringVar(&input.subagentModel, "subagent-model", "", "claude code subagent model")
	flags.BoolVar(&input.disableNonessentialTraffic, "disable-nonessential-traffic", false, "disable Claude Code nonessential traffic")
	flags.BoolVar(&input.disableNonstreamingFallback, "disable-nonstreaming-fallback", false, "disable Claude Code nonstreaming fallback")

	if err := flags.Parse(flagArgs); err != nil {
		return "", profileFlags{}, err
	}

	return name, input, nil
}

func normalizeProfileName(name string) string {
	return strings.TrimSpace(name)
}

func buildProfileEnv(input profileFlags, existing map[string]string) map[string]string {
	env := map[string]string{}
	for key, value := range existing {
		env[key] = value
	}

	if input.token != "" {
		env[profile.EnvAuthToken] = input.token
	}
	if input.baseURL != "" {
		env[profile.EnvBaseURL] = input.baseURL
	}
	if input.model != "" {
		env["ANTHROPIC_MODEL"] = input.model
	}
	if input.defaultOpus != "" {
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = input.defaultOpus
	}
	if input.defaultSonnet != "" {
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = input.defaultSonnet
	}
	if input.defaultHaiku != "" {
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = input.defaultHaiku
	}
	if input.subagentModel != "" {
		env["CLAUDE_CODE_SUBAGENT_MODEL"] = input.subagentModel
	}
	if input.disableNonessentialTraffic {
		env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = "1"
	}
	if input.disableNonstreamingFallback {
		env["CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK"] = "1"
	}

	return env
}

func promptAddName(reader *bufio.Reader) (string, error) {
	return promptAddValue(reader, "名称", "", true, false)
}

func promptRenameName(reader *bufio.Reader) (string, error) {
	return promptAddValue(reader, "新名称", "", true, false)
}

func promptAddFields(reader *bufio.Reader, input profileFlags) (profileFlags, error) {
	var err error
	if input.description == "" {
		input.description, err = promptAddValue(reader, "描述", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.token == "" {
		input.token, err = promptAddValue(reader, profile.EnvAuthToken, "", true, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.baseURL == "" {
		input.baseURL, err = promptAddValue(reader, profile.EnvBaseURL, "", true, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.model == "" {
		input.model, err = promptAddValue(reader, "ANTHROPIC_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultOpus == "" {
		input.defaultOpus, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_OPUS_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultSonnet == "" {
		input.defaultSonnet, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_SONNET_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultHaiku == "" {
		input.defaultHaiku, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_HAIKU_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}

	return input, nil
}

func promptEditFields(reader *bufio.Reader, existing profile.Profile, input profileFlags) (profile.Profile, error) {
	if reader == nil {
		reader = bufio.NewReader(promptReader)
	}

	var err error
	if input.description == "" {
		var keepCurrent bool
		existing.Description, keepCurrent, err = promptEditValue(reader, "描述", existing.Description, false, false)
		if err != nil {
			return profile.Profile{}, err
		}
		_ = keepCurrent
	} else {
		existing.Description = input.description
	}

	existing.Env, err = applyEditPrompt(reader, existing.Env, profile.EnvAuthToken, input.token, true, true)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, profile.EnvBaseURL, input.baseURL, true, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_MODEL", input.model, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_DEFAULT_OPUS_MODEL", input.defaultOpus, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_DEFAULT_SONNET_MODEL", input.defaultSonnet, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_DEFAULT_HAIKU_MODEL", input.defaultHaiku, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env = buildProfileEnv(profileFlags{
		subagentModel:               input.subagentModel,
		disableNonessentialTraffic:  input.disableNonessentialTraffic,
		disableNonstreamingFallback: input.disableNonstreamingFallback,
	}, existing.Env)

	return existing, nil
}

func applyEditPrompt(reader *bufio.Reader, env map[string]string, field, explicit string, required, sensitive bool) (map[string]string, error) {
	if explicit != "" {
		env[field] = explicit
		return env, nil
	}

	currentValue, exists := env[field]
	value, keepCurrent, err := promptEditValue(reader, field, currentValue, required, sensitive)
	if err != nil {
		return nil, err
	}

	if keepCurrent {
		if !exists {
			delete(env, field)
			return env, nil
		}
		env[field] = currentValue
		return env, nil
	}

	env[field] = value
	return env, nil
}

func promptAddValue(reader *bufio.Reader, label, suffix string, required, sensitive bool) (string, error) {
	_ = sensitive
	_, _ = fmt.Fprintf(promptWriter, "%s%s：", label, suffix)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	value = strings.TrimSpace(value)
	if required && value == "" {
		return "", fmt.Errorf("缺少必填字段：%s", label)
	}

	return value, nil
}

func promptEditValue(reader *bufio.Reader, label, current string, required, sensitive bool) (string, bool, error) {
	display := current
	if sensitive {
		display = maskValue(current)
	}

	if display != "" {
		_, _ = fmt.Fprintf(promptWriter, "%s [%s]（直接回车保留当前值）：", label, display)
	} else {
		_, _ = fmt.Fprintf(promptWriter, "%s（直接回车保留当前值）：", label)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", false, err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		if required && strings.TrimSpace(current) == "" {
			return "", false, fmt.Errorf("缺少必填字段：%s", label)
		}
		return current, true, nil
	}

	return value, false, nil
}

func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}

	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func runRemove(paths Paths, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, "必须提供配置名称\n")
		return 1
	}

	name := normalizeProfileName(args[0])
	if name == "" {
		_, _ = io.WriteString(stderr, "必须提供配置名称\n")
		return 1
	}
	if profile.IsOfficialName(name) {
		_, _ = fmt.Fprintf(stderr, "内置配置 %q 不支持删除\n", name)
		return 1
	}

	if err := profile.Remove(paths.Profiles, name); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已删除配置：%s\n", name)
	return 0
}

func runRename(paths Paths, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		_, _ = io.WriteString(stderr, "必须提供旧配置名称和新配置名称\n")
		return 1
	}

	oldName := normalizeProfileName(args[0])
	newName := normalizeProfileName(args[1])
	if oldName == "" || newName == "" {
		_, _ = io.WriteString(stderr, "必须提供旧配置名称和新配置名称\n")
		return 1
	}
	if profile.IsOfficialName(oldName) || profile.IsOfficialName(newName) {
		_, _ = fmt.Fprintf(stderr, "配置 %q 是内置配置名称，不能作为普通 profile\n", profile.OfficialProfileName)
		return 1
	}
	if isReservedCommandName(newName) {
		_, _ = fmt.Fprintf(stderr, "配置 %q 是内置配置名称，不能作为普通 profile\n", newName)
		return 1
	}

	if err := profile.Rename(paths.Profiles, oldName, newName); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已将配置 %s 重命名为 %s\n", oldName, newName)
	return 0
}

func formatCLIError(err error) string {
	if errors.Is(err, io.EOF) {
		return "输入已结束"
	}

	return err.Error()
}

func isReservedProfileName(name string) bool {
	return profile.IsOfficialName(name) || isReservedCommandName(name)
}

func isReservedCommandName(name string) bool {
	switch name {
	case "current", "list", "status", "add", "edit", "remove", "rename":
		return true
	default:
		return false
	}
}
