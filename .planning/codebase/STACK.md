# Technology Stack

**Analysis Date:** 2026-03-27

## Languages

**Primary:**
- Go 1.23.0 - Entire codebase (`go.mod`)

**Secondary:**
- None

## Runtime

**Environment:**
- Go 1.23.0 (native compiled binary)

**Package Manager:**
- Go modules (`go.mod`)
- Lockfile: No `go.sum` present (zero external dependencies)

## Frameworks

**Core:**
- Go standard library only - No external frameworks

**Testing:**
- `testing` (Go stdlib) - All test files use the built-in testing package
- No external test frameworks or assertion libraries

**Build/Dev:**
- `go build` - Compile to binary: `go build -o cc-switch .`
- No Makefile, Dockerfile, or CI/CD configuration detected

## Key Dependencies

**Critical:**
- None - This project has **zero external dependencies**. It relies entirely on the Go standard library.

**Standard Library Packages Used:**
- `encoding/json` - Profile and settings JSON serialization (`internal/profile/store.go`, `internal/settings/store.go`)
- `os` - File I/O, environment, process exit (`main.go`, throughout `internal/`)
- `path/filepath` - Path manipulation (`internal/settings/store.go`, `internal/profile/store.go`)
- `bytes` - Buffer for JSON encoding (`internal/settings/store.go`)
- `fmt` - Formatted output (`internal/output/print.go`)
- `time` - Backup file timestamps (`internal/settings/backup.go`)
- `syscall` - Raw terminal mode (`internal/cli/term_darwin.go`, `internal/cli/term_other.go`)

## Configuration

**Environment:**
- `CC_SWITCH_PROFILES_PATH` - Override default profiles.json location (used in tests)
- `CC_SWITCH_SETTINGS_PATH` - Override default settings.json location (used in tests)
- No `.env` files present

**Build:**
- `go.mod` - Module definition, Go version constraint
- No build configuration files (no Makefile, no goreleaser, no golangci-lint config)

## Platform Requirements

**Development:**
- Go 1.23.0+
- macOS or Linux (platform-specific terminal code in `internal/cli/term_darwin.go` and `internal/cli/term_other.go`)

**Production:**
- Standalone compiled binary, no runtime dependencies
- Requires read/write access to `~/.claude/` directory
- Targets: `~/.claude/cc-switch/profiles.json` (profile storage) and `~/.claude/settings.json` (Claude Code config)

---

*Stack analysis: 2026-03-27*
