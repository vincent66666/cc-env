# Status Selector Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `cc-switch` show current status plus an arrow-key selectable profile list in interactive terminals, and switch on Enter.

**Architecture:** Keep the existing non-interactive text output path intact. Add a small terminal selector in `internal/cli` for interactive TTY sessions only, with a pure rendering/navigation layer and a thin raw-terminal integration layer.

**Tech Stack:** Go standard library, ANSI escape sequences, Unix terminal ioctls, existing CLI tests.

---

## Chunk 1: Selector Core

### Task 1: Add pure selector state and rendering

**Files:**
- Create: `internal/cli/status_selector.go`
- Test: `internal/cli/status_selector_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests for:
- rendering current status plus a multiline selectable list
- moving selection up/down with wrap-around
- returning no-op when there are no available profiles

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli -run 'TestStatusSelector' -count=1`
Expected: FAIL because selector types/functions do not exist.

- [ ] **Step 3: Write minimal implementation**

Implement:
- a selector state struct containing current profile summary, available names, and highlighted index
- pure methods for `moveUp`, `moveDown`, `selectedName`
- a renderer that emits the full screen text for current state

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli -run 'TestStatusSelector' -count=1`
Expected: PASS

## Chunk 2: Terminal Integration

### Task 2: Add raw terminal read loop for interactive status mode

**Files:**
- Modify: `internal/cli/app.go`
- Create: `internal/cli/term_darwin.go`
- Test: `internal/cli/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add TTY tests for:
- `cc-switch` showing a selectable list when alternatives exist
- pressing ArrowDown + Enter switching to the highlighted profile
- pressing `q` exiting without changing current profile

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli -run 'TestRun_StatusInteractive' -count=1`
Expected: FAIL because `cc-switch` still prints plain text only.

- [ ] **Step 3: Write minimal implementation**

Implement in `app.go`:
- detect interactive TTY in `runStatus`
- if there are available profiles, enter selector mode instead of plain output
- on Enter, call the same switching path used by `use`
- on `q`, exit cleanly without modifying current profile

Implement in `term_darwin.go`:
- minimal raw-mode enter/restore helpers for reading arrow-key escape sequences

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli -run 'TestRun_StatusInteractive' -count=1`
Expected: PASS

## Chunk 3: Regression Verification

### Task 3: Keep existing behavior stable

**Files:**
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Add/adjust regression assertions**

Cover:
- no available profiles still prints status only
- non-interactive `cc-switch` still returns plain text output

- [ ] **Step 2: Run targeted tests**

Run: `go test ./internal/cli -count=1`
Expected: PASS

- [ ] **Step 3: Run full verification**

Run: `go test ./... -count=1`
Expected: PASS

Run: `go build ./...`
Expected: PASS
