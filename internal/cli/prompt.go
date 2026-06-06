package cli

import (
	"bufio"
	"fmt"
	"strings"

	"cc-env/internal/profile"
)

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
		existing.Description, _, err = promptEditValue(reader, "描述", existing.Description, false, false)
		if err != nil {
			return profile.Profile{}, err
		}
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
