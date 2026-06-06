# CLI UI Refresh Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve the readability and interaction quality of `cc-env` terminal UI while keeping non-TTY output stable and preserving current business behavior.

**Architecture:** Extend the existing rendering code instead of introducing a new TUI framework. Add a small ANSI style layer for TTY output, then incrementally enhance `list` and `status` interactive renderers and key handling with test-first changes.

**Tech Stack:** Go 1.23, standard library, existing `go test` suite

---

## File Map

- `internal/output/print.go`
  - 负责普通 `list/current/status` 输出。
- `internal/output/style.go`
  - 新增。负责 TTY 检测和 ANSI 样式封装。
- `internal/output/print_test.go`
  - 新增。覆盖普通输出在 TTY/非 TTY 的渲染差异。
- `internal/cli/app.go`
  - 负责交互输入分发和 `list/status` 会话控制。
- `internal/cli/list_menu.go`
  - 负责 `list` 交互界面的状态和渲染。
- `internal/cli/list_menu_test.go`
  - 覆盖 `list` 交互页面的结构与动作。
- `internal/cli/status_selector.go`
  - 负责 `status` 交互界面的状态和渲染。
- `internal/cli/status_selector_test.go`
  - 覆盖 `status` 交互页面的渲染与选择行为。

Do not create commits unless the user explicitly asks for them.

## Chunk 1: Output Styling

### Task 1: Add ANSI-aware output styling for normal commands

**Files:**
- Create: `internal/output/style.go`
- Create: `internal/output/print_test.go`
- Modify: `internal/output/print.go`

- [ ] **Step 1: Write a failing test for non-TTY plain rendering**

Add a test in `internal/output/print_test.go` that writes to a `bytes.Buffer` and verifies `RenderStatus` still returns plain text without ANSI escapes.

```go
func TestRenderStatusPlainOutputWhenWriterIsNotTTY(t *testing.T) {
    var buf bytes.Buffer
    code := RenderStatus(&buf, "demo - 正式环境", profile.Profile{
        Env: map[string]string{
            profile.EnvBaseURL: "https://example.com",
            "ANTHROPIC_MODEL":  "glm-5",
        },
    }, []string{"beta - 测试环境"})

    if code != 0 {
        t.Fatalf("expected exit code 0, got %d", code)
    }
    if strings.Contains(buf.String(), "\x1b[") {
        t.Fatalf("expected plain output, got %q", buf.String())
    }
}
```

- [ ] **Step 2: Run the test and verify it passes or exposes the current baseline**

Run: `go test ./internal/output -run TestRenderStatusPlainOutputWhenWriterIsNotTTY -v`
Expected: PASS with current behavior.

- [ ] **Step 3: Write a failing test for TTY-styled rendering**

Introduce a test seam in the output package, then add a test that forces TTY mode and asserts `RenderList` or `RenderStatus` includes ANSI sequences and grouped labels.

```go
func TestRenderListUsesStyledSectionsWhenTTY(t *testing.T) {
    restore := forceOutputTTYForTest(true)
    defer restore()

    var buf bytes.Buffer
    RenderList(&buf, []string{"demo（当前） - 正式环境", "beta - 测试环境"})

    got := buf.String()
    if !strings.Contains(got, "\x1b[") {
        t.Fatalf("expected ANSI styled output, got %q", got)
    }
    if !strings.Contains(got, "当前配置") {
        t.Fatalf("expected grouped current section, got %q", got)
    }
}
```

- [ ] **Step 4: Run the new TTY rendering test and verify it fails**

Run: `go test ./internal/output -run TestRenderListUsesStyledSectionsWhenTTY -v`
Expected: FAIL because style detection/separated sections do not exist yet.

- [ ] **Step 5: Implement the minimal style layer**

Add `internal/output/style.go` with:

```go
type styler struct {
    enabled bool
}

func detectStyledOutput(w io.Writer) styler
func (s styler) heading(text string) string
func (s styler) current(text string) string
func (s styler) muted(text string) string
func (s styler) warning(text string) string
```

Implementation notes:
- Enable styles only when writer is an `*os.File` attached to a char device.
- Add a package-level test hook to force style on/off in tests.
- Keep the palette minimal: one primary color, one muted color, one warning color, plus bold.

- [ ] **Step 6: Update normal command renderers**

In `internal/output/print.go`:
- Refactor `RenderStatus` to print a compact titled block with `当前配置` / `接口地址` / `模型`.
- Refactor `RenderList` to separate current profile from the rest when styled output is enabled.
- Preserve deterministic plain text when styling is disabled.

- [ ] **Step 7: Run the output tests**

Run: `go test ./internal/output -v`
Expected: PASS

## Chunk 2: List Interactive UI

### Task 2: Improve interactive `list` rendering and direct shortcuts

**Files:**
- Modify: `internal/cli/app.go`
- Modify: `internal/cli/list_menu.go`
- Modify: `internal/cli/list_menu_test.go`

- [ ] **Step 1: Write a failing render test for the new list layout**

Add assertions in `internal/cli/list_menu_test.go` for:
- top title text
- current profile badge/marker
- bottom shortcut hint line
- delete confirmation warning text

Example:

```go
func TestListMenuRenderShowsSectionTitleAndShortcutHints(t *testing.T) {
    menu := listMenu{
        profiles:    []string{"demo", "beta"},
        currentName: "demo",
        descriptions: map[string]string{"demo": "正式环境"},
    }

    rendered := menu.render()
    for _, fragment := range []string{
        "配置列表",
        "当前配置",
        "↑/↓ 选择",
        "e 编辑",
        "r 重命名",
        "d 删除",
        "q 退出",
    } {
        if !strings.Contains(rendered, fragment) {
            t.Fatalf("expected %q in %q", fragment, rendered)
        }
    }
}
```

- [ ] **Step 2: Run the list-menu render tests and verify the new assertion fails**

Run: `go test ./internal/cli -run 'TestListMenu' -v`
Expected: FAIL on the new layout assertions.

- [ ] **Step 3: Add failing tests for shortcut-key action parsing**

In `internal/cli/list_menu_test.go` or `internal/cli/app_test.go`, cover that pressing `e`, `r`, and `d` maps to distinct actions without breaking arrow key and `Enter` handling.

```go
func TestReadSelectorActionSupportsDirectListShortcuts(t *testing.T) {
    cases := map[string]selectorAction{
        "e": selectorActionEdit,
        "r": selectorActionRename,
        "d": selectorActionRemove,
    }
    // use a bufio.Reader per case and assert action mapping
}
```

- [ ] **Step 4: Run the shortcut test and verify it fails**

Run: `go test ./internal/cli -run TestReadSelectorActionSupportsDirectListShortcuts -v`
Expected: FAIL because these actions are not implemented.

- [ ] **Step 5: Extend selector action parsing in `app.go`**

Add new action constants and update `readSelectorAction` to recognize:
- `e` / `E`
- `r` / `R`
- `d` / `D`

Keep existing handling for arrows, `Enter`, `q`, and `Ctrl+C`.

- [ ] **Step 6: Handle shortcut actions in `runInteractiveList`**

Map direct actions to the existing business flows:
- `e` => `runEditWithPromptReader`
- `r` => `promptRenameName` then `runRename`
- `d` => enter delete confirmation

Guard rails:
- if the current mode is delete confirmation, `d` should not bypass the confirmation state
- current profile cannot go straight to remove

- [ ] **Step 7: Update `list_menu.go` rendering**

Refactor `render()` to output:
- title/header section
- list/actions/confirmation body
- consistent bottom hint line

Use restrained ANSI styling only if the existing interactive output path already supports it safely; otherwise render the improved layout with plain characters.

- [ ] **Step 8: Run the list-specific test suite**

Run: `go test ./internal/cli -run 'TestListMenu|TestReadSelectorActionSupportsDirectListShortcuts' -v`
Expected: PASS

## Chunk 3: Status Interactive UI

### Task 3: Align interactive `status` with the new visual language

**Files:**
- Modify: `internal/cli/status_selector.go`
- Modify: `internal/cli/status_selector_test.go`

- [ ] **Step 1: Write a failing test for the new status layout**

Update `internal/cli/status_selector_test.go` to assert:
- titled header
- structured detail fields
- current profile appears first and remains marked
- bottom hint line is present

```go
func TestStatusSelectorRenderShowsHeaderAndHints(t *testing.T) {
    selector := statusSelector{
        currentName: "demo",
        currentDescription: "正式环境",
        baseURL: "https://example.com",
        model: "glm-5",
        names: []string{"beta"},
        descriptions: map[string]string{"beta": "测试环境"},
    }

    rendered := selector.render()
    for _, fragment := range []string{
        "当前配置",
        "接口地址",
        "模型",
        "可用配置",
        "↑/↓ 选择",
        "Enter 切换",
        "q 退出",
    } {
        if !strings.Contains(rendered, fragment) {
            t.Fatalf("expected %q in %q", fragment, rendered)
        }
    }
}
```

- [ ] **Step 2: Run the status-selector tests and verify the new assertions fail**

Run: `go test ./internal/cli -run 'TestStatusSelector' -v`
Expected: FAIL on the new layout assertions.

- [ ] **Step 3: Update `status_selector.go` with the new structure**

Refactor `render()` so that it:
- shows the same top-level title style used by `list`
- prints current profile details as a compact field block
- renders the selectable profile list below
- appends a concise bottom hint line

- [ ] **Step 4: Keep edge-case behavior intact**

Verify the implementation still renders:
- `模型：-` when model is empty
- an empty selection safely when no profiles exist
- current profile first even when `names` excludes it

- [ ] **Step 5: Run the status-selector tests**

Run: `go test ./internal/cli -run 'TestStatusSelector' -v`
Expected: PASS

## Final Verification

- [ ] Run the focused CLI test set:

Run: `go test ./internal/cli ./internal/output -v`
Expected: PASS

- [ ] Run the broader regression suite:

Run: `go test ./...`
Expected: PASS

## Edge Cases To Cover During Implementation

- TTY 样式启用时不能污染非 TTY 输出。
- 当前配置缺失时，`list` 仍可切换，且界面文案不能误导用户。
- 当前配置不能直接进入删除动作。
- 删除确认页不能被快捷键绕过。
- 空 description、空 model、空列表时界面仍然稳定。

Plan complete and saved to `docs/superpowers/plans/2026-03-14-cli-ui-refresh.md`. Ready to execute?
