# Architecture

**Analysis Date:** 2026-03-27

## Pattern Overview

**Overall:** CLI application with layered package architecture (no framework, pure Go standard library)

**Key Characteristics:**
- Zero external dependencies -- only Go standard library
- Command-line tool for managing Claude Code (Anthropic CLI) configuration profiles
- Reads/writes JSON files on the local filesystem (`~/.claude/` directory)
- Supports both non-interactive (piped) and interactive (TUI) terminal modes
- Platform-specific terminal handling via Go build tags

## Layers

**Entry Point (`main.go`):**
- Purpose: Minimal bootstrap; delegates immediately to `cli.Run`
- Location: `main.go`
- Contains: Only `main()` function
- Depends on: `internal/cli`
- Used by: Go runtime

**CLI Layer (`internal/cli/`):**
- Purpose: Command parsing, routing, interactive terminal UI, user prompting, orchestration of profile operations
- Location: `internal/cli/`
- Contains: Command dispatcher (`app.go`), argument parser (`parse.go`), interactive list menu (`list_menu.go`), status selector (`status_selector.go`), platform terminal code (`term_darwin.go`, `term_other.go`)
- Depends on: `internal/profile`, `internal/settings`, `internal/output`
- Used by: `main.go`

**Profile Layer (`internal/profile/`):**
- Purpose: Profile data model, CRUD operations on the profiles JSON file, validation
- Location: `internal/profile/`
- Contains: Type definitions (`types.go`), file I/O and CRUD (`store.go`), validation rules (`validate.go`)
- Depends on: Go standard library only
- Used by: `internal/cli`, `internal/output`

**Settings Layer (`internal/settings/`):**
- Purpose: Writing environment variables into Claude's `settings.json`, backup management
- Location: `internal/settings/`
- Contains: JSON merge-write logic (`store.go`), timestamped backup creation (`backup.go`)
- Depends on: Go standard library only
- Used by: `internal/cli`

**Output Layer (`internal/output/`):**
- Purpose: Rendering status and list output with optional ANSI styling for TTY
- Location: `internal/output/`
- Contains: Styled/plain text rendering (`print.go`), TTY detection and ANSI helpers (`style.go`)
- Depends on: `internal/profile` (for `Profile` type and `EnvBaseURL` constant)
- Used by: `internal/cli`

## Data Flow

**Profile Switch Flow (core operation):**

1. User runs `cc-switch use <name>` or selects a profile in interactive mode
2. `cli.Run` parses args via `Parse()`, dispatches to `runUse()` or `switchProfile()`
3. `profile.LoadForList()` reads `~/.claude/cc-switch/profiles.json` into `ProfilesFile` struct
4. Target profile is validated via `profile.ValidateProfile()`
5. Current `settings.json` is snapshotted for rollback (`readSettingsSnapshot()`)
6. `settings.WriteEnv()` backs up existing `settings.json` to `~/.claude/cc-switch/backups/`, then merges the profile's `env` map into the JSON and writes atomically
7. `profile.Save()` updates `current` field in `profiles.json` (atomic write via temp file + rename)
8. If step 7 fails, `restoreSettingsSnapshot()` rolls back `settings.json`

**Interactive List Flow:**

1. `runList()` detects TTY via `selectorInteractive()`
2. If interactive, enters `runInteractiveList()` which sets raw terminal mode (`makeRawTerminal()`) and alternate screen buffer
3. Renders `listMenu` state, reads keypresses byte-by-byte via `readSelectorAction()`
4. Supports nested menu modes: profile selection -> action menu -> delete confirmation
5. For edit/rename actions, exits raw mode, runs line-based prompts, then resumes interactive session via `resumeListSession()`

**State Management:**
- All state is persisted in two JSON files on disk:
  - `~/.claude/cc-switch/profiles.json` -- profile definitions and current selection
  - `~/.claude/settings.json` -- Claude Code's active settings (env vars injected here)
- No in-memory state persists between invocations
- Interactive mode uses local struct state (`listMenu`, `statusSelector`) during a session

## Key Abstractions

**Command:**
- Purpose: Represents a parsed CLI command with name and arguments
- Examples: `internal/cli/parse.go`
- Pattern: Simple struct, no interface

**ProfilesFile / Profile:**
- Purpose: Data model for the profiles configuration file
- Examples: `internal/profile/types.go`
- Pattern: JSON-serializable structs with a version field for future migration

**listMenu:**
- Purpose: State machine for the interactive list UI with three modes (profiles, actions, delete confirm)
- Examples: `internal/cli/list_menu.go`
- Pattern: Struct with mode enum, renders itself via `render()` method

**statusSelector:**
- Purpose: State machine for the interactive status view with profile quick-switch
- Examples: `internal/cli/status_selector.go`
- Pattern: Same pattern as listMenu but simpler (single mode)

**Paths:**
- Purpose: Holds file paths for profiles and settings, overridable via env vars
- Examples: `internal/cli/app.go` (line 20-23)
- Pattern: Struct initialized in `defaultPaths()`, supports `CC_SWITCH_PROFILES_PATH` and `CC_SWITCH_SETTINGS_PATH` overrides

## Entry Points

**`main.go`:**
- Location: `main.go`
- Triggers: Direct CLI invocation (`cc-switch [command] [args]`)
- Responsibilities: Passes args to `cli.Run()`, exits with returned code

**`cli.Run()`:**
- Location: `internal/cli/app.go` (line 75)
- Triggers: Called by `main()`
- Responsibilities: Parses command, dispatches to handler, returns exit code

## Error Handling

**Strategy:** Return error values up the call stack; CLI layer formats and writes to stderr, returns non-zero exit code

**Patterns:**
- Profile/settings layers return `error` values; never print directly
- CLI layer uses `fmt.Fprintf(stderr, ...)` for all error output
- Sentinel errors with `errors.Is()` for specific cases (e.g., `ErrCurrentProfileMissing`, `os.ErrNotExist`)
- Custom error type `currentProfileMissingError` implements `Is()` for sentinel matching
- Atomic file writes with cleanup on failure (temp file removed if rename fails)
- Settings rollback on profile save failure (`restoreSettingsSnapshot`)

## Cross-Cutting Concerns

**Logging:** None. No logging framework. Errors go to stderr, output to stdout.

**Validation:** Centralized in `internal/profile/validate.go`. Validates profile name (non-empty), required env keys (`ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`), and rejects unsupported env keys against a whitelist (`SupportedEnvKeys`).

**Authentication:** Not applicable -- this tool manages auth tokens for Claude Code but does not authenticate itself.

**File Safety:** All file writes use atomic temp-file-then-rename pattern. Settings writes create timestamped backups in `~/.claude/cc-switch/backups/`.

**Platform Compatibility:** Build-tag-separated terminal handling: `term_darwin.go` (raw terminal via syscall) and `term_other.go` (interactive mode unsupported on non-Darwin).

---

*Architecture analysis: 2026-03-27*
