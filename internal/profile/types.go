package profile

type Profile struct {
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env"`
}

type ProfilesFile struct {
	Version  int                `json:"version"`
	Current  string             `json:"current,omitempty"`
	Profiles map[string]Profile `json:"profiles"`
}

const OfficialProfileName = "official"

var ManagedEnvKeys = []string{
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_BASE_URL",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"CLAUDE_CODE_SUBAGENT_MODEL",
	"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
	"CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK",
}

var SupportedEnvKeys = func() map[string]struct{} {
	keys := make(map[string]struct{}, len(ManagedEnvKeys))
	for _, key := range ManagedEnvKeys {
		keys[key] = struct{}{}
	}
	return keys
}()

func IsOfficialName(name string) bool {
	return name == OfficialProfileName
}
