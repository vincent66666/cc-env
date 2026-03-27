# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

cc-switch is a Go CLI tool for managing multiple Claude API configuration profiles. It persists profiles in `~/.claude/cc-switch/profiles.json` and switches the active profile by updating `~/.claude/settings.json`'s env field.

## Commands

```bash
go build -o cc-switch .        # Build binary
go test ./... -count=1         # Run all tests
go test ./internal/cli/ -run TestFoo -count=1  # Run single test
```

## Architecture

```
main.go              → Entry point, calls cli.Run()
internal/
  cli/               → Command dispatch, argument parsing, interactive TUI menus
    app.go           → Core command handler (current/list/status/use/add/edit/remove/rename)
    parse.go         → CLI argument parser
    status_selector.go / list_menu.go → Interactive terminal menus
    term_darwin.go / term_other.go    → Platform-specific raw terminal mode
  profile/           → Profile CRUD and persistence (profiles.json)
    types.go         → Profile & ProfilesFile structs
    store.go         → Read/write profiles.json
    validate.go      → Profile validation rules
  settings/          → Claude settings.json integration with backup/rollback
  output/            → Terminal styling and formatted output helpers
```

## Key Design Details

- **Zero external dependencies** — stdlib only (Go 1.23)
- Profile switching writes env vars into `settings.json` and backs up before write; rollback on failure
- Environment overrides: `CC_SWITCH_PROFILES_PATH` and `CC_SWITCH_SETTINGS_PATH` for custom file paths (used in tests)
- Interactive menus use raw terminal mode with platform-specific implementations (darwin vs other)
- All commands support both interactive prompts and CLI flags (e.g. `--name`, `--description`, `--env`)
- Profile env fields: `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_OPUS_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`
