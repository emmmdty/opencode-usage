# Product/Engineering Architecture Review

**Reviewer:** Subagent C (Product/Engineering Architecture)
**Date:** 2026-08-26
**Scope:** Feature prioritization for opencode-usage v0.2.0+

---

## Current State Summary

The project is a Go CLI tool (`opencode-usage`) that manages OpenCode Go plan accounts and queries quota usage. It uses cobra for CLI, lipgloss for terminal styling, keyring for credential storage, and has a basic TUI table renderer.

**Existing commands:** `quota`, `models`, `account` (add/list/remove/export/import), `current`, `alias` (install/uninstall), `version`, `update`

**Existing capabilities:**
- Multi-account quota querying with concurrent requests (semaphore-based)
- Retry logic with exponential backoff (3 retries)
- Color-coded quota display with configurable warning/danger thresholds
- Keyring + AES-GCM encrypted fallback for credential storage
- Master password support
- API key validation on add/import
- Shell alias management
- JSON output (`--json` flag)
- `--account`, `--output`, `--no-color` global flags
- Human-readable reset time formatting

---

## Feature Evaluation

### 1. Default `ou` command showing a dashboard overview

**Category: DO NOW**

**User value:** Currently `ou` with no subcommand shows help. Users expect a bare command to show a useful overview — the "what do I have" summary. This is the single most impactful UX improvement possible. Every CLI tool benefits from a sensible default action.

**Implementation complexity:** Low. The `quota` command already does all the heavy lifting. The root command's `Run` field just needs to be set to a function that calls the same multi-account quota logic. No new packages needed.

**Changes needed:**
- `internal/cmd/root.go`: Add `Run` (or `RunE`) to `rootCmd` that executes the quota query logic
- Refactor the quota query logic out of `quotaCmd.RunE` into a shared function (e.g., `runQuotaOverview(accountFilter string, jsonOut bool, outPath string)`) in `internal/cmd/quota.go`
- `rootCmd.Run` calls `runQuotaOverview("", false, "")`
- The bare `ou` shows quota for all accounts, or a summary with best-account recommendation if #6 is also built

---

### 2. Multi-account quota display with current account marker

**Category: DO NOW**

**User value:** When you have 5+ accounts and the quota table is displayed, you don't know which one is actively being used by opencode. This is high-value for the multi-account use case that is the core product purpose.

**Implementation complexity:** Low-Medium. Requires reading `~/.local/share/opencode/auth.json` (already done in `current.go`) and matching the active token's key ID against the config's stored key IDs.

**Changes needed:**
- `internal/tui/quota.go`: Accept a `currentAccount string` parameter; render a marker (e.g., `*` or `→`) next to the matching account row
- `internal/cmd/quota.go`: Before rendering, read the current opencode auth config and resolve which account name matches the active key ID
- `internal/cmd/root.go`: Pass the current account name through to the quota overview function

---

### 3. Account health status (HEALTHY/WARNING/EXHAUSTED/INVALID/UNKNOWN)

**Category: NEXT**

**User value:** Quickly identifying which accounts are usable vs. errored/exhausted is essential for the multi-account workflow. Currently you see `错误: ...` for failures but no semantic status for working accounts.

**Implementation complexity:** Medium. Requires mapping the `Usage` struct fields (percent values, status strings) to health categories. No new API calls needed — derive from existing data.

**Changes needed:**
- `internal/models/usage.go`: Add a `HealthStatus string` field or a method on `Usage`
- `internal/tui/quota.go`: Display health status column or badge per account
- Logic: `percent >= dangerThreshold` → EXHAUSTED, `percent >= warningThreshold` → WARNING, `status == "ok"` → HEALTHY, error present → INVALID, otherwise → UNKNOWN

---

### 4. Quota warning thresholds

**Category: NEXT**

**User value:** Configurable alert thresholds (already partially implemented in config and TUI rendering). Users want to be notified *before* they're out of quota, not just when displayed. The config already stores `ColorThresholds.Warning` and `ColorThresholds.Danger`, but there's no way to configure them via CLI and no proactive warnings.

**Implementation complexity:** Low. The rendering already uses these thresholds. Need a CLI command or flag to set them.

**Changes needed:**
- `internal/cmd/root.go` or new `config.go` subcommand: `ou config set-threshold warning 70` / `ou config set-threshold danger 90`
- Or a `--warning` / `--danger` flag on the quota command

---

### 5. Best available account recommendation

**Category: NEXT**

**User value:** When all accounts are queried, the user wants to know "which one should I use?" This is the highest-value decision-support feature for multi-account users.

**Implementation complexity:** Medium. Requires ranking accounts by available quota (lowest percent used = best) and excluding errored/invalid accounts. Logic goes in the quota overview or a dedicated function.

**Changes needed:**
- `internal/cmd/quota.go`: After collecting results, sort by remaining quota (ascending percent = most available first), mark the top healthy account as "recommended"
- `internal/tui/quota.go`: Display recommendation indicator
- `internal/cmd/root.go`: In default dashboard, show the recommended account prominently

---

### 6. Concurrency with semaphore

**Category: OUT OF SCOPE (already implemented)**

Already in `quota.go:55-93`. The semaphore is `maxConcurrent` (default 5). No work needed unless we want to expose it as a CLI flag.

---

### 7. Cache with stale fallback

**Category: NEXT**

**User value:** Avoids hitting the API on every `ou` invocation. Quota data changes slowly — caching for 30-60 seconds with stale-while-revalidate pattern would make the tool feel instant for repeated usage.

**Implementation complexity:** Medium. Write results to a cache file with timestamp; on subsequent invocations, serve from cache if fresh enough, or serve stale + fetch in background.

**Changes needed:**
- `internal/config/config.go`: Add cache file path logic
- New `internal/cache/cache.go`: Simple file-based cache with TTL
- `internal/cmd/quota.go`: Check cache first, serve stale if within grace period

---

### 8. Partial failure handling

**Category: OUT OF SCOPE (already implemented)**

The current code in `quota.go:98-111` already handles partial failures — each account's error is captured individually and displayed in the table. No work needed.

---

### 9. Retry logic

**Category: OUT OF SCOPE (already implemented)**

`internal/client/opencode.go:34-76` implements 3-retry with exponential backoff for 429 and 5xx errors. No work needed.

---

### 10. Watch/refresh mode

**Category: NEXT**

**User value:** Users want to monitor quota consumption in real-time during heavy usage sessions (e.g., while running opencode). A `--watch` flag that refreshes every N seconds would be valuable.

**Implementation complexity:** Medium-High. Requires a terminal loop with clear/redraw, signal handling for graceful exit. Lipgloss already supports this pattern (bubbletea), but a simple loop with `time.Ticker` would work.

**Changes needed:**
- `internal/cmd/quota.go`: Add `--watch` / `--interval` flags, implement refresh loop
- Handle SIGINT/SIGTERM for clean exit
- Consider using bubbletea for animated refresh if lipgloss is already a dependency

---

### 11. Doctor/diagnostics command

**Category: NEXT**

**User value:** When something goes wrong (network issues, keyring failures, config corruption), users need a single command to diagnose the problem. Essential for support and self-service troubleshooting.

**Implementation complexity:** Low-Medium. A new cobra command that checks: keyring availability, config file parseability, network connectivity, API key validity, and outputs a summary.

**Changes needed:**
- New `internal/cmd/doctor.go`: Checks each subsystem, reports OK/WARN/FAIL
- Uses existing `auth.IsKeyringAvailable()`, `auth.ValidateAPIKey()`, `config.LoadOrCreateConfig()`
- Could be a new `ou doctor` subcommand

---

### 12. JSON output stability

**Category: NEXT**

**User value:** JSON output is consumed by scripts and other tools. Currently the JSON schema is implicit — adding a `schemaVersion` field and documenting the contract prevents breaking changes.

**Implementation complexity:** Low. Add a version wrapper object to JSON output.

**Changes needed:**
- `internal/cmd/quota.go`: Wrap JSON output in `{"version": "1.0", "accounts": [...]}`
- Document the JSON schema in README

---

### 13. Shell integration improvements

**Category: OUT OF SCOPE**

Shell alias install/uninstall already works for bash and zsh. Fish, nushell, and completion generation are nice-to-haves but not blocking. The existing `ou` alias is sufficient.

---

### 14. Import/export enhancements

**Category: OUT OF SCOPE (already implemented)**

Account export/import with validation is already in `account.go`. The current implementation validates API keys on import and skips duplicates. Sufficient for v1.

---

### 15. Credential security improvements

**Category: OUT OF SCOPE**

The dual-path (keyring + AES-GCM encrypted file) with master password support is already solid. The encrypted.go implementation uses PBKDF2 with 100K iterations, AES-256-GCM, random salt/nonce — standard practice. The keyring integration tests availability. No critical gaps.

---

### 16. Account switching

**Category: NEXT**

**User value:** After seeing which account is best (#2, #5), users want `ou account use <name>` to switch the active opencode account. This is the natural action after quota inspection.

**Implementation complexity:** Medium. Requires writing to `~/.local/share/opencode/auth.json` with the selected account's token and provider info.

**Changes needed:**
- `internal/cmd/account.go`: New `account use` subcommand
- Read the API key from credential store, validate it, write to opencode's auth.json
- Need to know the opencode auth.json schema (partially known from `current.go`)

---

### 17. Auto switching

**Category: OUT OF SCOPE**

Automatic switching when quota is exhausted is a significant behavior change with side effects. Users need to opt in carefully. This should be behind a `--auto-switch` flag and require explicit consent. Too risky for initial releases.

---

### 18. Daemon mode

**Category: OUT OF SCOPE**

A background daemon that watches quota and auto-switches or notifies is a major architectural change. Requires process management, IPC, and significant complexity. Not worth building until the core CLI is stable and widely used.

---

### 19. Proxy/router

**Category: OUT OF SCOPE**

An HTTP proxy that sits in front of the OpenCode API and routes to the best account is a fundamentally different product. This is a server, not a CLI tool. Would require a separate binary and deployment model.

---

### 20. Historical usage tracking

**Category: OUT OF SCOPE**

Storing historical quota snapshots would require a local database (SQLite) and charting. High complexity for moderate value — the API doesn't expose historical data, only current snapshots. Can revisit when the API adds history endpoints.

---

### 21. Notifications

**Category: OUT OF SCOPE**

System notifications (desktop/mobile) when quota hits thresholds require OS-specific integration or a notification service. High complexity, marginal value compared to the TUI display.

---

### 22. Progress bar visualization for quota

**Category: NEXT**

**User value:** A visual progress bar (e.g., `[████████░░░░░░░░] 52%`) is more intuitive than a bare percentage. Common pattern in quota/progress CLIs.

**Implementation complexity:** Low. Pure rendering change in the TUI. Use unicode block characters or lipgloss's built-in bar renderer.

**Changes needed:**
- `internal/tui/quota.go`: Replace or supplement the percentage text with a progress bar
- Keep it conditional on terminal width (responsive)

---

### 23. Human-readable reset times

**Category: OUT OF SCOPE (already implemented)**

`internal/tui/quota.go:82-99` already formats reset times as `XdXh`, `XhXm`, `Xm`. Sufficient.

---

### 24. Responsive terminal width handling

**Category: NEXT**

**User value:** On narrow terminals, the quota table overflows and wraps badly. Detecting terminal width and adjusting column widths or switching to a vertical layout prevents display corruption.

**Implementation complexity:** Low-Medium. Use `golang.org/x/term` (already a dependency) to get terminal width, then adjust column sizing or switch layout mode.

**Changes needed:**
- `internal/tui/quota.go`: Accept terminal width, adapt layout (compact vs. full)
- If width < 80: single-column layout; if width >= 80: current table

---

### 25. NO_COLOR / non-TTY support

**Category: DO NOW**

**User value:** The `--no-color` flag exists but doesn't respect the `NO_COLOR` environment variable (standard: https://no-color.org/). CI/CD pipelines, piping output, and non-TTY contexts break when ANSI escape codes are present. This is a correctness fix.

**Implementation complexity:** Trivial. Add an env var check in `PersistentPreRun`.

**Changes needed:**
- `internal/cmd/root.go`: In `PersistentPreRun`, also check `os.Getenv("NO_COLOR")` and if set, disable color. Also auto-detect non-TTY with `os.Getenv("TERM")` or checking if stdout is a terminal.

---

## Priority Summary

### DO NOW (v0.2.0 — immediate)
| # | Feature | Effort | Impact |
|---|---------|--------|--------|
| 1 | Default `ou` dashboard | Low | High |
| 2 | Multi-account current-account marker | Low-Med | High |
| 25 | NO_COLOR / non-TTY support | Trivial | Medium |

### NEXT (v0.3.0 — short-term)
| # | Feature | Effort | Impact |
|---|---------|--------|--------|
| 3 | Account health status | Medium | High |
| 5 | Best available account recommendation | Medium | High |
| 22 | Progress bar visualization | Low | Medium |
| 24 | Responsive terminal width | Low-Med | Medium |
| 12 | JSON output stability | Low | Medium |
| 11 | Doctor/diagnostics command | Low-Med | Medium |
| 7 | Cache with stale fallback | Medium | Medium |
| 16 | Account switching | Medium | High |
| 10 | Watch/refresh mode | Med-High | Medium |
| 4 | Quota warning thresholds (configurable) | Low | Low |

### OUT OF SCOPE (deferred / won't build)
| # | Feature | Reason |
|---|---------|--------|
| 6 | Concurrency with semaphore | Already implemented |
| 8 | Partial failure handling | Already implemented |
| 9 | Retry logic | Already implemented |
| 13 | Shell integration improvements | Already sufficient |
| 14 | Import/export enhancements | Already implemented |
| 15 | Credential security improvements | Already solid |
| 17 | Auto switching | Too risky, needs opt-in design |
| 18 | Daemon mode | Major architecture change |
| 19 | Proxy/router | Different product entirely |
| 20 | Historical usage tracking | API doesn't support it |
| 21 | Notifications | High complexity, marginal value |
| 23 | Human-readable reset times | Already implemented |

---

## Architecture Notes

**Good patterns to preserve:**
- Semaphore-based concurrency in quota.go (clean WaitGroup + channel pattern)
- Dual credential storage (keyring with encrypted fallback)
- TUI rendering is cleanly separated in `internal/tui/` (easy to extend)
- Config migration system with version field

**Technical debt to address:**
- `getConfigPath()` is duplicated across commands — should be a shared utility
- The `account` aliases (`aa`, `al`, `ar`, `ae`, `ai`) are registered as both subcommands and root aliases — this creates confusion (e.g., `ou aa` works but `ou account aa` also works via subcommand)
- `timeNow()` in account.go exists only for testability — should use an interface or clock abstraction
- No tests for the TUI rendering layer (would help with #22, #24 changes)
