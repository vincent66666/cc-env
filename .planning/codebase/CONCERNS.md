# Codebase Concerns

**Analysis Date:** 2026-03-27

## Tech Debt

**Committed binary in git:**
- Issue: `cli.test` (4.8MB ELF binary) is tracked by git. `cc-switch` binary exists in working dir but is gitignored.
- Files: `cli.test` (tracked), `cc-switch` (untracked but present)
- Impact: Bloats repository history permanently. Every rebuild that gets committed increases repo size.
- Fix approach: Add `cli.test` to `.gitignore` and remove from tracking with `git rm --cached cli.test`.

**God file - `internal/cli/app.go` at 1173 lines:**
- Issue: Single file contains all CLI command handlers, interactive prompting, profile switching logic, settings snapshot/restore, atomic file writing, and display name formatting. This violates SRP heavily.
- Files: `internal/cli/app.go`
- Impact: Hard to navigate, test in isolation, or modify one command without risk to others. The 3423-line test file (`internal/cli/app_test.go`) is a symptom of this.
- Fix approach: Extract into separate files per command (e.g., `cmd_add.go`, `cmd_edit.go`, `cmd_use.go`). Extract `writeFileAtomically()`, `settingsSnapshot`, and prompt helpers into dedicated packages or files.

**Duplicated `normalizeProfileName` function:**
- Issue: Identical function defined in both `internal/cli/app.go` (line 913) and `internal/profile/store.go` (line 218). Both just call `strings.TrimSpace()`.
- Files: `internal/cli/app.go:913`, `internal/profile/store.go:218`
- Impact: Divergence risk if normalization logic changes. Confusing which is canonical.
- Fix approach: Export the function from `internal/profile` and use it in CLI layer.

**Duplicated atomic write pattern:**
- Issue: The temp-file-write-rename pattern is implemented three times: `internal/settings/store.go:45-64`, `internal/profile/store.go:115-136`, and `internal/cli/app.go:304-331`.
- Files: `internal/settings/store.go`, `internal/profile/store.go`, `internal/cli/app.go`
- Impact: Bug fixes to atomic write must be applied in three places.
- Fix approach: Extract a shared `writeFileAtomically()` utility (one already exists in `app.go` but is not reused by settings/profile packages).

**Duplicated `orderedProfiles` / `prioritizeCurrentProfile` logic:**
- Issue: `statusSelector.orderedNames()` in `internal/cli/status_selector.go:83-97` and `prioritizeCurrentProfile()` in `internal/cli/list_menu.go:248-267` implement the same "current first" ordering logic with slightly different implementations.
- Files: `internal/cli/status_selector.go:83-97`, `internal/cli/list_menu.go:248-267`
- Impact: Two implementations to maintain for the same behavior.
- Fix approach: Unify into a single shared helper function.

**Package-level mutable state for testing:**
- Issue: `promptReader`, `promptWriter`, `promptInteractive`, and `startInteractiveSession` are package-level vars in `internal/cli/app.go` (lines 43-55) mutated by tests. `forcedStyledOutput` in `internal/output/style.go` (line 15) follows the same pattern.
- Files: `internal/cli/app.go:43-55`, `internal/output/style.go:15`
- Impact: Tests that forget to restore state can cause flaky failures. Not safe for parallel test execution.
- Fix approach: Use dependency injection via struct fields or functional options instead of package globals.

## Known Bugs

None identified from static analysis.

## Security Considerations

**Auth tokens stored in plaintext JSON:**
- Risk: `ANTHROPIC_AUTH_TOKEN` values are stored in `~/.claude/cc-switch/profiles.json` with no encryption.
- Files: `internal/profile/types.go:14` (SupportedEnvKeys), `internal/profile/store.go:93` (Save)
- Current mitigation: Token is partially masked during interactive edit (`internal/cli/app.go:1115-1121`). File permissions are 0644 (world-readable).
- Recommendations: Set file permissions to 0600 for profiles.json. Consider OS keychain integration for token storage.

**Backup files retain sensitive data:**
- Risk: Backups of `settings.json` (which contain auth tokens in the `env` block) accumulate in `~/.claude/cc-switch/backups/` with no cleanup or access restrictions.
- Files: `internal/settings/backup.go:21-22`
- Current mitigation: None. Backups grow unbounded.
- Recommendations: Limit backup count (e.g., keep last 5). Set backup file permissions to 0600. Add a cleanup mechanism.

**Settings file permissions too permissive:**
- Risk: `os.CreateTemp` uses default permissions. Atomic writes via temp files may create files readable by other users.
- Files: `internal/settings/store.go:45`, `internal/profile/store.go:115`
- Current mitigation: None.
- Recommendations: Explicitly set 0600 permissions on temp files before writing sensitive content.

## Performance Bottlenecks

None identified. The codebase is a small CLI tool with no long-running operations, network calls, or large data processing.

## Scalability & Performance

**Unbounded backup growth:**
- Problem: Every profile switch creates a backup of `settings.json` with no pruning.
- Files: `internal/settings/backup.go`
- Cause: `BackupFile()` creates timestamped backups but never removes old ones.
- Improvement path: Add a cleanup step that retains only the N most recent backups after each write.

## Fragile Areas

**Interactive terminal session lifecycle:**
- Files: `internal/cli/app.go:333-467` (runInteractiveStatus, runInteractiveList), `internal/cli/app.go:564-586` (resumeListSession)
- Why fragile: The `closeInteractive` function pointer is passed around and can be nil. It is reassigned in `resumeListSession`. The deferred cleanup in `runInteractiveList` checks for nil (line 389), indicating awareness of the fragility. Terminal raw mode must be correctly restored or the user's terminal is left in a broken state.
- Safe modification: Always test interactive flows manually on a real terminal after changes. Ensure every code path through the interactive loop correctly calls `closeInteractive`.
- Test coverage: Interactive terminal paths are difficult to test; `app_test.go` mocks `startInteractiveSession` but real terminal state transitions are untested.

**Non-Darwin platform support:**
- Files: `internal/cli/term_other.go`
- Why fragile: The `!darwin` build tag simply returns `false` for `rawTerminalSupported()` and errors on `makeRawTerminal()`. This means the interactive TUI (arrow keys, live selection) is completely unavailable on Linux and Windows.
- Safe modification: Implement `golang.org/x/term` or equivalent for cross-platform raw terminal support.
- Test coverage: No tests for non-darwin terminal behavior.

**Profile switch is a two-step non-atomic operation:**
- Files: `internal/cli/app.go:229-270` (switchProfile)
- Why fragile: Switching writes `settings.json` first, then updates `profiles.json`. If the process is killed between these two writes, `settings.json` has the new profile's env but `profiles.json` still points to the old profile. The rollback mechanism (lines 260-265) only handles `profile.Save` errors, not process crashes.
- Safe modification: Consider writing both files in a single transaction or using a "pending switch" marker.
- Test coverage: The rollback path is tested, but crash-between-writes is not.

## Maintainability

**`app.go` complexity:**
- The 1173-line `internal/cli/app.go` handles 8 commands, interactive sessions, prompting, display formatting, and file I/O. This is the single biggest maintainability concern.
- Files: `internal/cli/app.go`
- Recommendation: Split into one file per command and extract shared utilities.

**`unsafe` usage in terminal handling:**
- `internal/cli/term_darwin.go` uses `unsafe.Pointer` for direct syscall ioctl calls.
- Files: `internal/cli/term_darwin.go:40,48`
- Recommendation: Replace with `golang.org/x/term` package which provides safe, cross-platform terminal handling. This would also fix the non-darwin platform limitation.

**No `go.sum` file:**
- The project has zero external dependencies (`go.mod` only declares the module). This is a strength for maintainability -- zero dependency risk. However, the trade-off is reimplementing terminal handling and CLI parsing manually.

## Risk Areas

**No CI/CD pipeline detected:**
- No GitHub Actions, Makefile, or CI configuration files found.
- Risk: No automated test runs on push. Regressions can be merged unchecked.
- Files: None (absence is the issue)

**No `help` or `--help` command:**
- The CLI has no help/usage output. Unknown commands just print an error.
- Files: `internal/cli/app.go:96-98`
- Risk: Poor user experience for new users. No self-documentation.

## Dependencies at Risk

None. The project has zero external dependencies, using only the Go standard library. This is a notable strength.

## Missing Critical Features

**No backup cleanup:**
- Problem: Backups accumulate forever in `~/.claude/cc-switch/backups/`.
- Blocks: Long-running usage will consume disk space without limit.

**No cross-platform interactive mode:**
- Problem: Interactive TUI only works on macOS (darwin).
- Blocks: Linux users (common for dev/server environments) cannot use arrow-key navigation or the list action menu.
- Files: `internal/cli/term_other.go`

## Test Coverage Gaps

**No tests for `internal/cli/term_darwin.go`:**
- What's not tested: Raw terminal mode setup, termios get/set, terminal restore on cleanup.
- Files: `internal/cli/term_darwin.go`
- Risk: Terminal could be left in raw mode if restore fails silently.
- Priority: Medium

**No tests for `internal/settings/backup.go` edge cases:**
- What's not tested: Disk full scenarios, permission denied on backup directory, backup name collision loop.
- Files: `internal/settings/backup.go:27-53`
- Risk: The infinite loop on name collision (line 27) has no upper bound -- though practically unlikely, it is unbounded.
- Priority: Low

**No tests for `internal/output/style.go` TTY detection:**
- What's not tested: `writerIsTTY()` and `stdinIsTTY()` behavior with actual terminal file descriptors.
- Files: `internal/output/style.go:33-54`
- Risk: Styled output could be emitted to pipes/files, breaking downstream parsing.
- Priority: Low

**Strengths worth noting:**
- Test-to-source ratio is excellent (4985 test lines vs 2231 source lines).
- Core logic (profile CRUD, settings write, output rendering) has thorough test coverage.
- Atomic file writes prevent data corruption on normal errors.
- Settings rollback on profile switch failure is implemented and tested.

---

*Concerns audit: 2026-03-27*
