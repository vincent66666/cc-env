# External Integrations

**Analysis Date:** 2026-03-27

## APIs & External Services

None identified. cc-switch is a local CLI tool that manages configuration files on disk. It does not make any HTTP requests or call external APIs.

## Data Stores

**Local File Storage:**
- **Profiles database:** `~/.claude/cc-switch/profiles.json`
  - Managed by: `internal/profile/store.go`
  - Format: JSON with version field, current profile name, and profile map
  - Override path via: `CC_SWITCH_PROFILES_PATH` env var
- **Claude Code settings:** `~/.claude/settings.json`
  - Managed by: `internal/settings/store.go`
  - Read-modify-write pattern: reads existing JSON, updates `env` field, writes back
  - Atomic writes via temp file + rename (`internal/settings/store.go` lines 45-63)
  - Override path via: `CC_SWITCH_SETTINGS_PATH` env var
- **Settings backups:** `~/.claude/settings.json.bak.*`
  - Managed by: `internal/settings/backup.go`
  - Created before each settings modification for rollback capability

**Databases:**
- None - All persistence is local JSON files

**Caching:**
- None

**Message Queues:**
- None

## Authentication & Authorization

**Auth Provider:**
- None - cc-switch itself has no authentication
- It manages Anthropic API tokens (`ANTHROPIC_AUTH_TOKEN`) as opaque string values stored in profiles (`internal/profile/types.go`)

**Token Management:**
- Profiles store env vars including `ANTHROPIC_AUTH_TOKEN` in `profiles.json`
- Switching profiles copies env vars into `settings.json`'s `env` field
- cc-switch does not validate or use tokens itself; it only passes them to Claude Code's configuration

**Permission Model:**
- File system permissions only
- Profiles file created with default permissions via `os.MkdirAll` with `0o755` for directories

## Monitoring & Observability

**Error Tracking:**
- None - Errors written to stderr via `internal/output/print.go`

**Logs:**
- No structured logging; output via styled terminal printing (`internal/output/style.go`, `internal/output/print.go`)

## CI/CD & Deployment

**Hosting:**
- Local CLI tool, distributed as compiled binary

**CI Pipeline:**
- None detected (no `.github/`, no Makefile, no goreleaser config)

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Integration with Claude Code

cc-switch integrates with Claude Code by modifying its settings file:

1. **Read:** Parses `~/.claude/settings.json` as generic JSON (`map[string]any`)
2. **Modify:** Sets the `env` key to the selected profile's env map
3. **Write:** Atomic write (temp file + rename) with backup before modification
4. **Rollback:** If write fails, backup can be restored (`internal/settings/backup.go`)

**Supported env keys** (defined in `internal/profile/types.go`):
- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`

---

*Integration audit: 2026-03-27*
