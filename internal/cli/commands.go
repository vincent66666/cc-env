package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"cc-env/internal/profile"
)

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
