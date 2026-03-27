# Coding Conventions

**Analysis Date:** 2026-03-27

## Naming Patterns

**Files:**
- Use lowercase `snake_case` for Go source files: `store.go`, `list_menu.go`, `status_selector.go`
- Test files use `_test.go` suffix co-located with source: `store_test.go`, `app_test.go`
- Platform-specific files use build-tag naming: `term_darwin.go`, `term_other.go`

**Functions:**
- Exported functions use `PascalCase`: `Load()`, `Save()`, `ValidateProfile()`, `RenderStatus()`
- Unexported functions use `camelCase`: `runCurrent()`, `switchProfile()`, `normalizeProfileName()`
- Command handlers follow `run{Command}` pattern: `runCurrent()`, `runList()`, `runStatus()`, `runUse()`, `runAdd()`, `runEdit()`, `runRemove()`, `runRename()`
- Helper constructors use `new` prefix: `newStyler()`

**Variables:**
- Use `camelCase` for all variables
- Constants use `camelCase` for unexported: `ansiReset`, `clearScreenSequence`
- Constants use `PascalCase` for exported: `EnvAuthToken`, `EnvBaseURL`
- Sentinel errors use `Err` prefix: `ErrCurrentProfileMissing`

**Types:**
- Structs use `PascalCase` for exported: `Profile`, `ProfilesFile`, `Paths`
- Structs use `camelCase` for unexported: `styler`, `statusSelector`, `listMenu`, `settingsSnapshot`
- Enum-like types use `camelCase` with descriptive suffixes: `selectorAction`, `listMenuMode`, `listMenuAction`
- Iota constants follow `{type}{Value}` pattern: `selectorActionUp`, `listMenuModeProfiles`

**Packages:**
- Short, lowercase, single-word: `cli`, `profile`, `settings`, `output`
- All under `internal/` to prevent external imports

## Code Style

**Formatting:**
- Standard `gofmt` formatting (no custom config detected)
- Tab indentation (Go default)
- No `.editorconfig`, `.golangci.yml`, or linter config files present

**Linting:**
- No explicit linter configuration. Rely on `gofmt` and `go vet` defaults.

## Import Organization

**Order:**
1. Standard library imports grouped together
2. Blank line separator
3. Internal module imports (`cc-switch/internal/...`)

**Example from `internal/cli/app.go`:**
```go
import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cc-switch/internal/output"
	"cc-switch/internal/profile"
	"cc-switch/internal/settings"
)
```

**Path Aliases:**
- None. The module path is `cc-switch` (from `go.mod`).

## Error Handling

**Patterns:**
- Return `error` as the last return value from functions that can fail
- Use `fmt.Errorf()` for creating error messages (Chinese language)
- Use sentinel errors with `errors.New()` for well-known error conditions: `ErrCurrentProfileMissing` in `internal/profile/store.go`
- Use custom error types implementing `Is()` for error matching with context: `currentProfileMissingError` in `internal/profile/store.go`
- CLI commands return `int` exit codes (0 for success, 1 for failure) instead of errors
- Write errors to `stderr` via `fmt.Fprintf(stderr, ...)` in CLI layer
- Silently discard write errors on output with `_, _ =` pattern:
  ```go
  _, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
  ```
- Use `errors.Is()` for error type checking, never type assertions
- File operation errors: clean up temp files on failure with `_ = os.Remove(tempPath)`

**Error Message Language:**
- All user-facing error messages are in **Chinese**: `"必须提供配置名称"`, `"加载配置失败"`, `"未找到配置"`
- Use `%q` for quoting profile names in errors: `fmt.Errorf("未找到配置 %q", name)`

## Logging

**Framework:** None. No logging framework is used.

**Patterns:**
- User-facing messages go to `stdout` for success output
- Error messages go to `stderr`
- No debug/trace logging exists anywhere in the codebase
- All output messages are in Chinese

## Comments

**When to Comment:**
- Comments are minimal throughout the codebase
- No JSDoc/GoDoc comments on exported functions or types
- Build tags used for platform-specific files: `//go:build darwin`

**JSDoc/TSDoc:**
- Not applicable (Go codebase). No GoDoc comments are used.

## Function Design

**Size:**
- Functions are generally small (10-40 lines)
- Largest functions are the interactive loop handlers in `internal/cli/app.go` (~50 lines)

**Parameters:**
- Use `io.Writer` for stdout/stderr instead of `*os.File` for testability
- Use `io.Reader` for stdin
- Pass file paths as `string` parameters
- Use `func() time.Time` for injectable time (e.g., `settings.WriteEnv()`)

**Return Values:**
- CLI commands return `int` exit codes
- Data operations return `(value, error)` tuples
- Use named struct types for complex return values: `settingsSnapshot`

## Module Design

**Exports:**
- Each package exports a small, focused API
- `profile` package exports: `Load()`, `LoadForList()`, `Save()`, `SetCurrent()`, `Remove()`, `Rename()`, `ValidateProfile()`, types `Profile`, `ProfilesFile`
- `settings` package exports: `WriteEnv()`, `BackupFile()`
- `output` package exports: `RenderStatus()`, `RenderList()`
- `cli` package exports: `Run()`, `Parse()`, `Command`

**Barrel Files:**
- Not applicable (Go uses package-level exports, not barrel files)

## Atomic File Writes

Use the write-to-temp-then-rename pattern consistently for data safety:
```go
tempFile, err := os.CreateTemp(filepath.Dir(path), ".profiles-*.json")
// write to tempFile
os.Rename(tempPath, path)
```
This pattern appears in:
- `internal/profile/store.go` (`Save()`)
- `internal/settings/store.go` (`WriteEnv()`)
- `internal/cli/app.go` (`writeFileAtomically()`)

## Dependency Injection for Testing

**Package-level function variables** allow test overrides:
```go
// internal/cli/app.go
var (
	promptReader      io.Reader = os.Stdin
	promptWriter      io.Writer = os.Stdout
	promptInteractive           = func() bool { ... }
	startInteractiveSession = startInteractiveTerminalSession
)
```

**Environment variable overrides** for file paths in tests:
- `CC_SWITCH_PROFILES_PATH` overrides profiles.json location
- `CC_SWITCH_SETTINGS_PATH` overrides settings.json location

**Style toggle** for output testing:
```go
// internal/output/style.go
var forcedStyledOutput *bool
func forceStyledOutputForTest(enabled bool) func() { ... }
```

## UI/Output Conventions

**Interactive TUI:**
- Uses alternate screen mode (`\x1b[?1049h` / `\x1b[?1049l`)
- Raw terminal mode for key-by-key input reading
- Arrow keys for navigation, Enter for selection, q/Ctrl+C to quit
- Shortcut keys: e (edit), r (rename), d (delete)

**Styled Output:**
- ANSI escape codes for bold, color (cyan=primary, gray=muted)
- Automatically disabled when stdout is not a TTY
- Falls back to plain text for piped output

**Chinese UI Text:**
- All user-facing strings are in Chinese
- Menu labels: `"切换"`, `"修改"`, `"重命名"`, `"删除"`, `"返回"`
- Status labels: `"当前配置"`, `"接口地址"`, `"模型"`, `"可用配置"`

---

*Convention analysis: 2026-03-27*
