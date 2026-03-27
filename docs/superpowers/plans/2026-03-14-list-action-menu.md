# List Action Menu Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `cc-switch list` enter an interactive two-level menu in TTY mode, allowing profile selection and actions for switch, edit, remove, and back.

**Architecture:** Reuse the existing TTY interaction pattern added for `cc-switch` status mode, but add a dedicated list selector state machine for a profile list screen and an action menu screen. Keep non-interactive `cc-switch list` unchanged so scripts still receive plain text output.

**Tech Stack:** Go standard library, existing Darwin raw-terminal helper, ANSI escape sequences, existing CLI tests.

---

## Chunk 1: Pure List Menu State

### Task 1: Add pure selector/action-menu state and rendering

**Files:**
- Create: `internal/cli/list_menu.go`
- Test: `internal/cli/list_menu_test.go`

- [ ] **Step 1: Write the failing tests**

Add tests for:
- rendering the profile list with a highlighted row
- entering an action menu with `切换 / 修改 / 删除 / 返回`
- moving inside the action menu and returning to the profile list

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run 'TestListMenu' -count=1`
Expected: FAIL because list menu types/functions do not exist.

- [ ] **Step 3: Write minimal implementation**

Implement:
- list menu state (`list` vs `actions`)
- highlighted profile index
- highlighted action index
- render functions for both screens

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run 'TestListMenu' -count=1`
Expected: PASS

## Chunk 2: TTY Integration For Switch/Back

### Task 2: Connect interactive `list` flow in app.go

**Files:**
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add TTY tests for:
- `cc-switch list` showing the interactive list in TTY mode
- selecting a profile, entering the action menu, choosing `切换`, and switching successfully
- entering the action menu and choosing `返回` without changing current profile

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run 'TestRun_ListInteractive' -count=1`
Expected: FAIL because `list` still prints plain text only.

- [ ] **Step 3: Write minimal implementation**

Implement:
- TTY branch in `runList`
- action-menu read loop using existing raw-terminal input handling
- integration with existing `switchProfile`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run 'TestRun_ListInteractive' -count=1`
Expected: PASS

## Chunk 3: Remove Confirm + Edit Refresh

### Task 3: Add remove confirmation and edit refresh flow

**Files:**
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Write the failing tests**

Add TTY tests for:
- choosing `删除` enters a confirmation step
- confirming deletion removes the profile and refreshes the list
- choosing `修改` enters existing edit flow, saves changes, and returns to the list

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run 'TestRun_ListInteractive(Edit|Remove)' -count=1`
Expected: FAIL because the action menu does not yet support these actions.

- [ ] **Step 3: Write minimal implementation**

Implement:
- delete confirmation sub-state
- edit handoff that reuses existing edit prompts
- list refresh after edit/remove

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run 'TestRun_ListInteractive(Edit|Remove)' -count=1`
Expected: PASS

## Chunk 4: Verification

### Task 4: Regression verification

**Files:**
- Modify: `internal/cli/app_test.go`

- [ ] **Step 1: Verify non-interactive behavior remains**

Run: `go test ./internal/cli -count=1`
Expected: PASS

- [ ] **Step 2: Verify full repo**

Run: `go test ./... -count=1`
Expected: PASS

Run: `go build ./...`
Expected: PASS
