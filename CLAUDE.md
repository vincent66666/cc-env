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

`cc-env`（no args）enters the interactive TUI. `list`, `add`, `edit`, `remove`, `rename`, `status` subcommands have been removed. Only `cc-env <profile>` direct launch and `cc-env current` are retained as CLI subcommands.

## Architecture

```
main.go              → Entry point, calls cli.Run()
internal/
  cli/               → Argument dispatch, direct launch, current, non-TTY status fallback
    app.go           → Run dispatch: "" → TUI or non-TTY status; "current" → runCurrent; <profile> → direct launch
    launch.go        → launchClaude variable + runClaude implementation
    parse.go         → CLI argument parser
    display.go       → Output helpers (currentStatus, displayNames, renderStatus)
  tui/               → Bubble Tea interactive TUI (Elm architecture)
    app.go           → Run entry point (builds program, returns Result)
    model.go         → State machine: stateList / stateForm / stateConfirm; Init/Update/View
    list.go          → profileItem, buildItems, orderProfiles, profileNamesSorted
    form.go          → formModel, textFields/boolFields, newForm, build, reservedName
    preview.go       → renderPreview, maskSecret
    keys.go          → Key constants (keyAdd, keyEdit, keyDelete, keyQuit)
    styles.go        → lipgloss styles
  profile/           → Profile CRUD and persistence (profiles.json)
    types.go         → Profile & ProfilesFile structs
    store.go         → Read/write profiles.json
    validate.go      → Profile validation rules
  output/            → Terminal styling and formatted output helpers
```

## Key Design Details

- **Interactive TUI uses Bubble Tea** (bubbletea/bubbles/lipgloss); the rest of the codebase uses stdlib only (Go 1.24.2)
- `cc-env <profile|official>` saves the current mode, clears managed Claude API variables, overlays the selected profile env, and runs `claude`
- `cc-env` (no args) enters the interactive TUI; in a non-TTY context it prints the current status and exits without launching claude
- Built-in `official` mode clears managed third-party variables and uses Claude's native login state
- Environment override: `CC_ENV_PROFILES_PATH`; legacy `CC_SWITCH_PROFILES_PATH` is still accepted as a fallback
- Profile env fields: `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_OPUS_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `CLAUDE_CODE_SUBAGENT_MODEL`, `CLAUDE_CODE_EFFORT_LEVEL`, `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`, `CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK`
