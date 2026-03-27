# Testing Patterns

**Analysis Date:** 2026-03-27

## Test Framework

**Runner:**
- Go standard `testing` package (no third-party test framework)
- Config: None (uses `go test` defaults)

**Assertion Library:**
- Standard library only. All assertions use manual `if` checks with `t.Fatalf()`.
- No `testify`, `gomega`, or other assertion libraries.

**Run Commands:**
```bash
go test ./... -count=1         # Run all tests
go test ./internal/cli/ -run TestFoo -count=1  # Run single test
go build -o cc-switch .        # Build binary
```

## Test File Organization

**Location:**
- Co-located with source files in the same package (white-box testing)

**Naming:**
- `{source}_test.go` pattern
- Test functions: `Test{Function}_{Scenario}` with underscore-separated descriptions

**Test Files:**
```
internal/
  cli/
    app_test.go              # 3423 lines - CLI command integration tests
    list_menu_test.go        # 265 lines - List menu rendering/navigation unit tests
    status_selector_test.go  # 160 lines - Status selector rendering/navigation unit tests
  output/
    print_test.go            # 161 lines - Output rendering tests (plain + styled)
  profile/
    store_test.go            # 711 lines - Profile CRUD and persistence tests
    validate_test.go         # 40 lines - Profile validation tests
  settings/
    store_test.go            # 230 lines - Settings write and backup tests
```

## Test Structure

**Suite Organization:**
```go
// Flat test functions with descriptive names (no subtests by default)
func TestRun_UseUpdatesSettingsEnvAndCurrentProfile(t *testing.T) {
    // 1. Create fixtures
    profilesPath := writeProfilesFixture(t, profile.ProfilesFile{...})
    settingsPath := writeSettingsFixture(t, `{...}`)

    // 2. Set environment
    t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
    t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

    // 3. Execute
    var stdout, stderr bytes.Buffer
    exitCode := Run([]string{"use", "demo"}, &stdout, &stderr)

    // 4. Assert exit code, stdout, stderr, and persisted state
    if exitCode != 0 {
        t.Fatalf("expected exit code 0, got %d", exitCode)
    }
}
```

**Subtests used sparingly:**
```go
// internal/profile/validate_test.go
func TestValidateProfile_RequiresTokenAndBaseURL(t *testing.T) {
    t.Run("missing token", func(t *testing.T) { ... })
    t.Run("missing base url", func(t *testing.T) { ... })
}
```

**Setup Patterns:**
- `t.TempDir()` for isolated filesystem per test
- `t.Setenv()` for environment variable overrides (auto-restored)
- `t.Helper()` in fixture helper functions
- `t.Cleanup()` for teardown of global state mutations

**Assertion Pattern:**
```go
if got := stdout.String(); got != "已切换到配置：demo\n" {
    t.Fatalf("expected switch success output, got %q", got)
}
```

## Mocking

**Framework:** None. Uses dependency injection via package-level variables.

**Patterns:**

1. **Package-level variable replacement** (in `internal/cli/app_test.go`):
```go
func TestMain(m *testing.M) {
    promptInteractive = func() bool { return false }
    os.Exit(m.Run())
}
```

2. **Prompt session mocking** via `withPromptSession()` helper:
```go
func withPromptSession(t *testing.T, input string) *bytes.Buffer {
    t.Helper()
    oldPromptReader := promptReader
    oldPromptWriter := promptWriter
    oldPromptInteractive := promptInteractive
    t.Cleanup(func() {
        promptReader = oldPromptReader
        promptWriter = oldPromptWriter
        promptInteractive = oldPromptInteractive
    })
    var output bytes.Buffer
    promptReader = strings.NewReader(input)
    promptWriter = &output
    promptInteractive = func() bool { return true }
    return &output
}
```

3. **Style output forcing** for testing ANSI output:
```go
restore := forceStyledOutputForTest(true)
defer restore()
```

4. **Injectable time** for deterministic backup filenames:
```go
now := func() time.Time {
    return time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)
}
settings.WriteEnv(path, newEnv, now)
```

**What to Mock:**
- `promptReader` / `promptWriter` / `promptInteractive` for interactive I/O
- `startInteractiveSession` for raw terminal mode
- Time functions for deterministic file naming
- File paths via environment variables

**What NOT to Mock:**
- Filesystem operations (use `t.TempDir()` for real file I/O)
- JSON encoding/decoding
- Profile store operations (test through real files)

## Fixtures and Factories

**Test Data Helpers (in `internal/cli/app_test.go`):**

```go
// Structured fixture - validates on save
func writeProfilesFixture(t *testing.T, data profile.ProfilesFile) string {
    t.Helper()
    path := filepath.Join(t.TempDir(), "profiles.json")
    if err := profile.Save(path, data); err != nil {
        t.Fatalf("save profiles fixture: %v", err)
    }
    return path
}

// Raw fixture - for testing invalid/edge-case JSON
func writeRawProfilesFixture(t *testing.T, content string) string {
    t.Helper()
    path := filepath.Join(t.TempDir(), "profiles.json")
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        t.Fatalf("write raw profiles fixture: %v", err)
    }
    return path
}

// Settings fixture - raw JSON string
func writeSettingsFixture(t *testing.T, content string) string {
    t.Helper()
    path := filepath.Join(t.TempDir(), "settings.json")
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        t.Fatalf("write settings fixture: %v", err)
    }
    return path
}
```

**Location:**
- Fixtures are inline within test files (no separate fixtures directory)
- Helper functions are defined at the bottom of the test file

## Coverage

**Requirements:** None enforced. No coverage thresholds configured.

**View Coverage:**
```bash
go test ./... -count=1 -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Types

**Unit Tests:**
- `internal/profile/store_test.go` - Profile CRUD operations (Load, Save, SetCurrent, Remove, Rename)
- `internal/profile/validate_test.go` - Profile validation rules
- `internal/settings/store_test.go` - Settings file write and backup
- `internal/output/print_test.go` - Output rendering (plain and styled)
- `internal/cli/list_menu_test.go` - Menu state machine (render, navigate, mode transitions)
- `internal/cli/status_selector_test.go` - Status selector state (render, navigate, wrap-around)

**Integration Tests:**
- `internal/cli/app_test.go` - End-to-end CLI command tests that exercise the full stack (parse -> load profiles -> write settings -> verify persisted state)
- Tests verify both stdout/stderr output AND file system mutations

**Real TTY Tests:**
- Uses `script` command to allocate a real TTY for interactive tests
- `TestTTYHelperProcess` serves as a subprocess entry point via `GO_WANT_TTY_HELPER=1` env var
- `runWithTTY()`, `runWithTTYBestEffort()`, `runWithTTYStdoutAndFileStdin()` helpers in `internal/cli/app_test.go`
- Tests arrow key navigation, alternate screen mode, Ctrl+C/Ctrl+D handling
- Gracefully skipped when `script` command unavailable: `t.Skip("script command not available")`

**E2E Tests:**
- Not applicable. No separate E2E test suite.

## Common Patterns

**Filesystem Isolation:**
```go
func TestStore_LoadProfilesFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "profiles.json")
    // Write fixture, test, verify - all in temp dir
}
```

**Environment Override for Path Isolation:**
```go
t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)
```

**Error Testing:**
```go
err := Remove(path, "demo")
if err == nil || err.Error() != "不能删除当前正在使用的配置" {
    t.Fatalf("expected active-profile remove error, got %v", err)
}
```

**Sentinel Error Testing:**
```go
if !errors.Is(err, ErrCurrentProfileMissing) {
    t.Fatalf("expected missing current load error to match sentinel, got %v", err)
}
```

**Mutation Safety Testing (verify no write on failure):**
```go
// Write invalid JSON, attempt operation, verify file unchanged
original := `{"env":`
os.WriteFile(path, []byte(original), 0o644)
err := WriteEnv(path, newEnv, now)
if err == nil { t.Fatal("expected failure") }
content, _ := os.ReadFile(path)
if string(content) != original {
    t.Fatalf("expected file to remain unchanged")
}
```

**Rollback Testing:**
```go
// Verify settings.json is restored when profile save fails
// internal/cli/app_test.go TestRun_UseRollsBackSettingsWhenUpdatingCurrentFails
```

**Interactive Input Simulation:**
```go
promptOutput := withPromptSession(t, "demo\ndescription\ntoken\nhttps://example.com\n\n\n\n\n")
exitCode := Run([]string{"add"}, &stdout, &stderr)
// Verify prompts appeared in promptOutput
```

## Test Coverage Gaps

**Well-tested areas:**
- Profile CRUD operations (load, save, set current, remove, rename) with edge cases
- Settings write with backup creation and rollback on failure
- All CLI commands (use, add, edit, remove, rename, current, list, status)
- Interactive prompts with input simulation
- Real TTY interaction (arrow keys, alternate screen, Ctrl+C)
- Output rendering (plain text and styled ANSI)
- Input normalization (whitespace trimming)
- Error paths (invalid JSON, missing files, duplicate names, blank names)

**Undertested areas:**
- `internal/cli/term_darwin.go` / `term_other.go` - Raw terminal syscalls not directly unit tested (tested implicitly through TTY integration tests)
- `main.go` - No test (trivial entry point, acceptable)
- Concurrent access to profiles.json (no locking mechanism exists or is tested)
- `internal/settings/backup.go` - Tested only through `settings.WriteEnv()`, not independently

**Critical paths without dedicated tests:**
- `parseProfileFlags()` in `internal/cli/app.go` - Tested implicitly through `runAdd`/`runEdit` but no dedicated flag parsing tests
- `buildProfileEnv()` in `internal/cli/app.go` - Tested implicitly, no isolated unit tests
- `maskValue()` in `internal/cli/app.go` - Tested implicitly through edit interactive mask test

---

*Testing analysis: 2026-03-27*
