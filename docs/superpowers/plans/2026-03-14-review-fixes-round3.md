# Review Fixes Round 3 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the remaining review issues around platform compatibility, interactive TTY detection, and current-profile deletion traps.

**Architecture:** Keep the current interactive CLI structure, but narrow selector activation to real bidirectional TTY sessions. Add a non-Darwin raw-terminal fallback so unsupported platforms still compile and fall back to text output. Then adjust list action rendering so the current profile never exposes a delete action.

**Tech Stack:** Go standard library, existing CLI tests, GOOS cross-compilation checks.

---

## Chunk 1: Compatibility

### Task 1: Lock current failures

**Files:**
- Modify: `internal/cli/app_test.go`

- [ ] Add a failing test proving `cc-env` stays in plain-text mode when stdout is not interactive.
- [ ] Run a Linux-targeted compile check and confirm the package currently fails because `makeRawTerminal` is missing.

### Task 2: Implement compatibility fixes

**Files:**
- Modify: `internal/cli/app.go`
- Create: `internal/cli/term_other.go`

- [ ] Add a stricter selector-interaction gate that requires both input and output to be TTY files.
- [ ] Add a non-Darwin `makeRawTerminal` stub so unsupported targets compile and naturally fall back to text output.
- [ ] Re-run the targeted tests and cross-platform compile check.

## Chunk 2: Current Profile Action Safety

### Task 3: Lock current-profile action behavior

**Files:**
- Modify: `internal/cli/app_test.go`
- Modify: `internal/cli/list_menu_test.go`

- [ ] Add a failing test proving the current profile does not expose `删除` in the interactive action menu.
- [ ] Add a failing list-menu render test for the current-profile action set.

### Task 4: Implement action filtering

**Files:**
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/list_menu.go`

- [ ] Render a reduced action set for the current profile that omits `删除`.
- [ ] Keep non-current profiles on the existing action set.
- [ ] Re-run the targeted tests and confirm the trap path is gone.

## Chunk 3: Verification

### Task 5: Final verification

**Files:**
- No code changes

- [ ] Run targeted CLI tests for stdout fallback and current-profile action filtering.
- [ ] Run `GOOS=linux GOARCH=amd64 go test ./internal/cli -c`.
- [ ] Run `go test ./internal/cli -count=1`.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `go build ./...`.
