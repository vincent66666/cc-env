# Codebase Structure

**Analysis Date:** 2026-03-27

## Directory Layout

```
cc-switch/
├── main.go                  # Application entry point
├── go.mod                   # Go module definition (no external deps)
├── .gitignore               # Ignores compiled binary
├── CLAUDE.md                # Claude Code project instructions
├── README.md                # Project documentation
├── cc-switch                # Compiled binary (gitignored)
├── cli.test                 # Compiled test binary
├── internal/                # All application packages (Go internal convention)
│   ├── cli/                 # Command routing, interactive UI, prompting
│   │   ├── app.go           # Run(), command handlers, interactive sessions
│   │   ├── app_test.go      # Tests for all CLI commands
│   │   ├── parse.go         # Argument parsing (Command struct)
│   │   ├── list_menu.go     # Interactive list menu state machine
│   │   ├── list_menu_test.go # Tests for list menu logic
│   │   ├── status_selector.go    # Interactive status selector state machine
│   │   ├── status_selector_test.go # Tests for status selector
│   │   ├── term_darwin.go   # macOS raw terminal implementation
│   │   └── term_other.go    # Non-Darwin stub (interactive unsupported)
│   ├── profile/             # Profile data model and persistence
│   │   ├── types.go         # Profile, ProfilesFile structs, SupportedEnvKeys
│   │   ├── store.go         # Load/Save/SetCurrent/Remove/Rename operations
│   │   ├── store_test.go    # Tests for profile store
│   │   ├── validate.go      # Profile validation rules
│   │   └── validate_test.go # Tests for validation
│   ├── settings/            # Claude settings.json manipulation
│   │   ├── store.go         # WriteEnv() -- merge env into settings JSON
│   │   ├── store_test.go    # Tests for settings store
│   │   └── backup.go        # BackupFile() -- timestamped backups
│   └── output/              # Terminal output rendering
│       ├── print.go         # RenderStatus(), RenderList() with styled/plain modes
│       ├── print_test.go    # Tests for output rendering
│       └── style.go         # ANSI styling, TTY detection
└── docs/                    # Documentation and planning
    ├── usage.md             # Usage guide
    └── superpowers/         # Development plans and specs
        ├── plans/           # Implementation plans
        └── specs/           # Design specifications
```

## Directory Purposes

**`internal/cli/`:**
- Purpose: All CLI logic -- command dispatch, argument parsing, interactive terminal UI, user prompting
- Contains: The largest package; handles both non-interactive and interactive (TUI) modes
- Key files: `app.go` (~1173 lines, contains all command handlers and interactive session management)

**`internal/profile/`:**
- Purpose: Profile data persistence layer -- CRUD operations on profiles.json
- Contains: Type definitions, file I/O with atomic writes, validation
- Key files: `types.go` (data model), `store.go` (all file operations), `validate.go` (whitelist-based validation)

**`internal/settings/`:**
- Purpose: Manages Claude Code's `settings.json` -- merges env vars, creates backups
- Contains: JSON merge logic, timestamped backup file creation
- Key files: `store.go` (WriteEnv), `backup.go` (BackupFile)

**`internal/output/`:**
- Purpose: Formats CLI output for both TTY (styled with ANSI) and piped (plain text) modes
- Contains: Status and list renderers, TTY detection, ANSI color helpers
- Key files: `print.go` (public render functions), `style.go` (styling infrastructure)

**`docs/`:**
- Purpose: Project documentation and development planning artifacts
- Contains: Usage guide, historical implementation plans and design specs
- Key files: `docs/usage.md`

## Key File Locations

**Entry Points:**
- `main.go`: Application bootstrap -- calls `cli.Run(os.Args[1:], os.Stdout, os.Stderr)`
- `internal/cli/app.go`: `Run()` function (line 75) -- command dispatcher

**Configuration:**
- `go.mod`: Go module config (module `cc-switch`, Go 1.23.0, zero dependencies)
- `CLAUDE.md`: Claude Code project-specific instructions

**Core Logic:**
- `internal/cli/app.go`: Command handlers (`runStatus`, `runList`, `runUse`, `runAdd`, `runEdit`, `runRemove`, `runRename`, `switchProfile`)
- `internal/profile/store.go`: Profile file CRUD (`Load`, `LoadForList`, `Save`, `SetCurrent`, `Remove`, `Rename`)
- `internal/settings/store.go`: Settings env injection (`WriteEnv`)

**Testing:**
- `internal/cli/app_test.go`: CLI integration tests
- `internal/cli/list_menu_test.go`: List menu unit tests
- `internal/cli/status_selector_test.go`: Status selector unit tests
- `internal/profile/store_test.go`: Profile store tests
- `internal/profile/validate_test.go`: Validation tests
- `internal/settings/store_test.go`: Settings store tests
- `internal/output/print_test.go`: Output rendering tests

## Naming Conventions

**Files:**
- `snake_case.go`: All Go source files use snake_case
- `*_test.go`: Test files colocated with source, standard Go convention
- `term_darwin.go` / `term_other.go`: Build-tag-based platform files use `term_<platform>.go`

**Directories:**
- Lowercase, single-word names: `cli`, `profile`, `settings`, `output`
- All under `internal/` to enforce Go's access control

**Functions/Methods:**
- Exported: `PascalCase` (e.g., `Run`, `Load`, `Save`, `WriteEnv`, `RenderStatus`)
- Unexported: `camelCase` (e.g., `runStatus`, `switchProfile`, `readSelectorAction`)
- Command handlers: `run<CommandName>` pattern (e.g., `runAdd`, `runEdit`, `runRemove`)

## Where to Add New Code

**New CLI Command:**
1. Add case to the switch in `Run()` at `internal/cli/app.go` (line 79)
2. Add handler function `run<Name>()` in `internal/cli/app.go`
3. Add tests in `internal/cli/app_test.go`

**New Profile Field:**
1. Add to `SupportedEnvKeys` in `internal/profile/types.go`
2. Add flag in `profileFlags` struct and `parseProfileFlags()` in `internal/cli/app.go`
3. Add to `buildProfileEnv()` in `internal/cli/app.go`
4. Add prompt in `promptAddFields()` and `promptEditFields()` in `internal/cli/app.go`

**New Output Format:**
1. Add render function in `internal/output/print.go`
2. Add styled variant if needed
3. Add tests in `internal/output/print_test.go`

**New Platform Support:**
1. Add `term_<platform>.go` with build tag implementing `rawTerminalSupported()` and `makeRawTerminal()`

**Utilities:**
- Shared helpers go in the most specific package that needs them
- No shared `utils` package exists; follow Go convention of keeping helpers package-local

## Special Directories

**`internal/`:**
- Purpose: All application packages; enforces Go's internal package access restriction
- Generated: No
- Committed: Yes

**`docs/superpowers/`:**
- Purpose: Historical development plans and design specs (not runtime code)
- Generated: No
- Committed: Yes

**`.planning/`:**
- Purpose: GSD planning and codebase analysis documents
- Generated: By tooling
- Committed: Yes

**`.idea/`:**
- Purpose: GoLand/IntelliJ IDE configuration
- Generated: By IDE
- Committed: Yes (but typically should be gitignored)

---

*Structure analysis: 2026-03-27*
