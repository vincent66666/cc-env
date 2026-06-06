package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"cc-env/internal/profile"
)

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
