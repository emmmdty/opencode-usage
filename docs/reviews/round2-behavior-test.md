# Round 2 — Fresh Behavior/Test Audit

**Reviewer:** H — Fresh Behavior/Test Audit  
**Date:** 2026-08-26  
**Scope:** Runtime behavior, test coverage, correctness of logic  
**Codebase:** `/home/tjk/myProjects/tools/opencode-usage`

---

## Build & Test Results

| Check | Result |
|---|---|
| `go build` | PASS — binary produced cleanly |
| `go test ./...` | PASS — all 7 test files pass (4 packages have no tests) |
| `go test -race ./...` | PASS — no race conditions detected |

---

## CLI Commands (manual execution)

| Command | Result | Notes |
|---|---|---|
| `./opencode-usage --help` | PASS | Shows all commands, flags, aliases (`ou`) |
| `./opencode-usage version` | PASS | Prints `opencode-usage 0.2.0 (commit: none, built: unknown)` |
| `./opencode-usage quota --help` | PASS | Correct subcommand help |
| `./opencode-usage account --help` | PASS | Shows add/list/remove/export/import subcommands |
| `./opencode-usage doctor --help` | PASS | Correct subcommand help |
| `./opencode-usage models --help` | PASS | Correct subcommand help |
| `NO_COLOR=1 ./opencode-usage quota --help` | PASS | Works correctly |
| `./opencode-usage --no-color quota --help` | PASS | Flag-based color disable works |
| `./opencode-usage --json quota` (no valid keys) | PASS | Returns valid JSON with `version:"1"` and `accounts` array |

---

## Finding F-01: Error message printed twice on unknown account

**Severity:** Low  
**Verdict:** BUG  

Running `./opencode-usage quota -n nonexistent-account` outputs:

```
Error: account 'nonexistent-account' not found
Error: account 'nonexistent-account' not found
EXIT: 1
```

The error is printed identically twice. `runQuotaOverview` in `internal/cmd/quota.go:58` returns `fmt.Errorf("account '%s' not found", ...)`. Cobra should print this once. The double-print suggests an error propagation issue — likely the root command's `RunE` (`internal/cmd/root.go:40`) is also invoking `runQuotaOverview` after the subcommand error bubbles up. Root's `RunE` should be `nil` or guarded, since the root command delegates to `quota` when no subcommand is given.

**Fix:** Either set `rootCmd.RunE = nil` and handle the default case in `RunE` only when no subcommand matches, or add `SilenceErrors: true` and handle output manually.

---

## Finding F-02: `doctor` ignores `--json` and `--no-color` flags

**Severity:** Medium  
**Verdict:** BUG  

`internal/cmd/doctor.go:68` uses `fmt.Printf("  %s %-20s %s\n", ...)` with raw ANSI escape codes in `newDoctorTheme()` (line 103: `"\033[32m✓\033[0m"`). This bypasses both the `--json` output mode and the `--no-color` / `NO_COLOR` flag for icon rendering. The `newDoctorTheme` function checks `noColor` and `NO_COLOR` env, but the `--json` flag is completely ignored — doctor never produces machine-readable output.

**Fix:** Add JSON output path when `jsonOutput` is true. Use lipgloss or the TUI theme instead of raw ANSI.

---

## Finding F-03: `account list` doesn't respect `--no-color` / `--json`

**Severity:** Low  
**Verdict:** BUG  

`internal/cmd/account.go:159` uses `fmt.Printf` with no color awareness. The `--json` flag is not handled for `account list` (the list subcommand doesn't check `jsonOutput`). Also, the `-> ` current-account marker is hardcoded plain text with no color, unlike the TUI theme.

**Fix:** Add JSON output path and color-aware rendering for `account list`.

---

## Finding F-04: `formatCompact` includes summary and active account — VERIFIED CORRECT

**Severity:** N/A  
**Verdict:** PASS  

`internal/tui/quota.go:152-159` in `formatCompact`:
- Line 152: calls `computeSummary(results, style)` and writes it
- Lines 155-159: iterates results to find `IsCurrent` and writes `"Active: " + r.Name`

Both summary and active account are present in compact mode. Confirmed by reading the code.

---

## Finding F-05: `computeSummary` uses `style` thresholds — VERIFIED CORRECT

**Severity:** N/A  
**Verdict:** PASS  

`internal/tui/quota.go:297-343`: `computeSummary` accepts a `QuotaStyle` parameter and uses `style.DangerThreshold` (line 315) and `style.WarningThreshold` (line 317) for classification. Thresholds are configurable via `config.yaml` `color_thresholds` field (`internal/config/config.go:16-19`), with defaults of 50/80 set in `getDefaultConfig` (line 37-38). No hardcoded values in the classification logic.

---

## Finding F-06: `findNextReset` checks all three quota windows — VERIFIED CORRECT

**Severity:** N/A  
**Verdict:** PASS  

`internal/tui/quota.go:375`:
```go
for _, resetTime := range []time.Time{r.Usage.Rolling.ResetsAt, r.Usage.Weekly.ResetsAt, r.Usage.Monthly.ResetsAt} {
```

All three windows (Rolling/5H, Weekly, Monthly) are checked. Zero times are skipped (line 377). The earliest non-zero reset across all accounts is returned.

---

## Finding F-07: Four packages have zero test files

**Severity:** Medium  
**Verdict:** MISSING COVERAGE  

No test files exist for:
- `internal/cmd/` — all command logic (quota, account, doctor, models, alias, current, root)
- `internal/models/` — data models
- `internal/version/` — version info
- `cmd/opencode-usage/` — main entry point (expected, but worth noting)

The `cmd` package contains significant logic: `runQuotaOverview`, `resolveCurrentAccount`, `printQuotaTable`, `getAPIKeyForCommand`, `formatRelativeTime`, `aliasExists`, `removeAlias`. None of this is unit-tested. The `TestCLIIntegration` in `test/integration_test.go` only tests `--help` output.

---

## Finding F-08: `TestFormatQuotaOverviewCompactWidth` doesn't verify compact-specific output

**Severity:** Low  
**Verdict:** INCOMPLETE TEST  

`internal/tui/quota_test.go:229-245` sets terminal width to 50 to trigger compact mode, but only asserts the account name is present. It does not verify:
- Summary line is present
- Active account marker is present
- Compact format header ("OpenCode Go" without table formatting)
- The simplified `5H: xx%  W: xx%  M: xx%` format

**Fix:** Add assertions for summary, active account, and compact-specific formatting.

---

## Finding F-09: `TestComputeSummary` doesn't test critical or all-healthy edge cases

**Severity:** Low  
**Verdict:** INCOMPLETE TEST  

`internal/tui/quota_test.go:161-188` tests 1 healthy + 1 warning + 1 error. Missing test cases:
- All accounts healthy (0% usage)
- All accounts critical (90%+ usage)
- All accounts with errors
- Single account at exactly the threshold boundary (e.g., 50% = warning, 80% = critical)
- Pluralization: single account should say "1 account" not "1 accounts"

**Fix:** Add boundary and edge-case tests.

---

## Finding F-10: `getAPIKeyForCommand` silently uses first alphabetical account

**Severity:** Low  
**Verdict:** BEHAVIORAL CONCERN  

`internal/cmd/models.go:67-73`: When no `--account` flag is specified, the function picks the first account alphabetically. This is undocumented and could surprise users with multiple accounts. The `quota` command queries all accounts; `models` queries only one. This inconsistency may confuse users.

---

## Finding F-11: `ExtractKeyID` collision risk with short keys

**Severity:** Low  
**Verdict:** BEHAVIORAL CONCERN  

`internal/auth/credential.go:83-87`: `ExtractKeyID` returns the full key if length <= 6. Two different short keys could produce the same "key ID", causing `resolveCurrentAccount` (`internal/cmd/quota.go:171`) to match the wrong account. In practice, API keys are long, so this is unlikely but the edge case exists.

---

## Finding F-12: `resolveCurrentAccount` depends on file path, no error handling

**Severity:** Low  
**Verdict:** FRAGILE  

`internal/cmd/quota.go:154`: Hardcoded path `~/.local/share/opencode/auth.json`. If the file is corrupted, has unexpected JSON structure, or the token format changes, `resolveCurrentAccount` silently returns `""` (no current account highlighted). No logging or user feedback.

---

## Finding F-13: Integration test is minimal

**Severity:** Medium  
**Verdict:** INSUFFICIENT  

`test/integration_test.go` only tests that `--help` contains "OpenCode Go". Missing integration tests:
- `quota --json` returns valid JSON
- `version` outputs version string
- `account list` with no accounts
- Exit code verification on errors
- `--no-color` / `NO_COLOR` behavior
- Subcommand alias resolution (`q` for `quota`, `a` for `account`)

---

## Finding F-14: Race detector clean — no concurrency issues

**Severity:** N/A  
**Verdict:** PASS  

`go test -race ./...` passed cleanly. The concurrent goroutines in `runQuotaOverview` (`internal/cmd/quota.go:93-118`) use a semaphore channel and `sync.WaitGroup` correctly. Results are collected after `wg.Wait()` and `close(results)`.

---

## Finding F-15: `formatResetTime` handles zero time

**Severity:** N/A  
**Verdict:** PASS (indirect)  

`internal/tui/quota.go:253-273`: `formatResetTime` handles negative duration (returns "expired"). However, the caller `findNextReset` skips zero times (line 377), so `formatResetTime` is never called with zero time from the normal flow. This is correct.

---

## Summary

| Severity | Count | Items |
|---|---|---|
| Bug (High) | 0 | — |
| Bug (Medium) | 2 | F-02, F-07 |
| Bug (Low) | 3 | F-01, F-03, F-08 |
| Behavioral Concern | 2 | F-10, F-11 |
| Fragile | 1 | F-12 |
| Incomplete Test | 2 | F-08, F-09 |
| Insufficient Coverage | 1 | F-13 |
| Pass | 5 | F-04, F-05, F-06, F-14, F-15 |

**Overall Assessment:** Core TUI rendering logic is correct and well-tested. Thresholds are configurable, all three quota windows are checked, and compact mode includes summary + active account. The main gaps are: (1) the `cmd` package has zero unit tests, (2) `doctor` and `account list` ignore `--json`/`--no-color` flags, (3) the error message double-prints on invalid account names.
