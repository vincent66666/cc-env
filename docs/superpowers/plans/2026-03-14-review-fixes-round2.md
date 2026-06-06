# Review Fixes Round 2 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the latest reviewed issues in interactive CLI behavior and sync the user-facing docs.

**Architecture:** Keep the existing interactive selector structure. Tighten raw-terminal exit handling for `Ctrl+C`, align interactive status rendering with the existing text renderer for empty model values, and then update the docs to describe the current Chinese/interactive runtime behavior.

**Tech Stack:** Go standard library, existing CLI TTY tests, Markdown docs.

---

## Chunk 1: Safe Interactive Exit

### Task 1: Lock Ctrl+C Behavior With Tests

**Files:**
- Modify: `internal/cli/app_test.go`

- [ ] Add failing TTY tests for `cc-env` and `cc-env list` proving `Ctrl+C` exits cleanly during interactive selection.
- [ ] Run the targeted tests and confirm they fail for the expected reason.

### Task 2: Implement Safe Ctrl+C Handling

**Files:**
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/term_darwin.go`

- [ ] Update raw-terminal handling so interactive selectors treat `Ctrl+C` as a safe local exit instead of leaving the terminal vulnerable to an un-restored signal exit.
- [ ] Run the targeted tests and confirm they pass.

## Chunk 2: Status Rendering Consistency

### Task 3: Lock Empty-Model Rendering With a Failing Test

**Files:**
- Modify: `internal/cli/status_selector_test.go`

- [ ] Add a failing selector render test proving empty `ANTHROPIC_MODEL` should render as `-`.
- [ ] Run the targeted test and confirm it fails.

### Task 4: Implement the Model Fallback

**Files:**
- Modify: `internal/cli/status_selector.go`

- [ ] Add the same `-` fallback used by the non-interactive status renderer.
- [ ] Run the targeted selector tests and confirm they pass.

## Chunk 3: Documentation Sync

### Task 5: Update Runtime Examples

**Files:**
- Modify: `README.md`
- Modify: `docs/usage.md`

- [ ] Update `cc-env` and `cc-env list` examples to match the current Chinese output and interactive TTY behavior.
- [ ] Keep non-interactive examples explicit where needed to avoid ambiguity.

## Chunk 4: Verification

### Task 6: Final Verification

**Files:**
- No code changes

- [ ] Run targeted CLI tests for the new behaviors.
- [ ] Run `go test ./internal/cli -count=1`.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `go build ./...`.
