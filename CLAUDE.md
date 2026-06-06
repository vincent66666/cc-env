# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

cc-env is a Go CLI tool for managing Claude Code API launch profiles. It persists profiles in `~/.claude/cc-env/profiles.json` and switches modes by launching `claude` with a cleaned environment, so third-party API variables do not pollute the official Claude login state.

## Commands

```bash
go build -o cc-env .        # Build binary
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
  settings/          → Legacy settings.json integration, no longer called by CLI use flow
  output/            → Terminal styling and formatted output helpers
```

## Key Design Details

- **Zero external dependencies** — stdlib only (Go 1.23)
- `cc-env use` saves the current mode, clears managed Claude API variables, overlays the selected profile env, and runs `claude`
- Built-in `official` mode clears managed third-party variables and uses Claude's native login state
- Environment override: `CC_ENV_PROFILES_PATH`; legacy `CC_SWITCH_PROFILES_PATH` is still accepted as a fallback
- Interactive menus use raw terminal mode with platform-specific implementations (darwin vs other)
- All commands support both interactive prompts and CLI flags (e.g. `--name`, `--description`, `--env`)
- Profile env fields: `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_OPUS_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `CLAUDE_CODE_SUBAGENT_MODEL`, `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`, `CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK`
