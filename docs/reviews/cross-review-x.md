# Cross Review X — Conflict Resolution

**Reviewer**: Cross Reviewer X — Conflict Resolver  
**Date**: 2026-08-26  
**Scope**: All 6 Round 1 reviews, conflict analysis, false positive detection, severity correction

---

## Conflict Analysis

### [CX-1] CC-04 vs ARC-04: Hardcoded thresholds in computeSummary
- **Reviewers involved**: A (CC-04), B (ARC-04)
- **Conflict**: Both identify the same root cause — `computeSummary` uses hardcoded `80`/`50` instead of style thresholds. Reviewer A classifies as MINOR, Reviewer B as MAJOR. B also identifies four total locations with hardcoded thresholds.
- **Code evidence**: `internal/tui/quota.go:298-300` — `computeSummary` hardcodes `if maxPercent >= 80` and `else if maxPercent >= 50`. Meanwhile `formatPercent` (line 225-234) and `formatPercentCompact` (line 157-167) correctly use `style.DangerThreshold` and `style.WarningThreshold`. The `printQuotaTable` in `cmd/quota.go:201-210` passes the configured style to `FormatQuotaOverview`, but `computeSummary` at line 93 is called without the style parameter.
- **Resolution**: **Both are correct — same issue.** Reviewer B's severity (MAJOR) is more accurate because the summary text contradicts the visual colors for any user with custom thresholds. This is a functional bug, not just a code quality concern. MINOR is too low.
- **Corrected severity**: MAJOR

---

### [CX-2] T-003 vs ARC-11: migrateConfig drops UseMasterPassword
- **Reviewers involved**: B (ARC-11), C (T-003)
- **Conflict**: Both identify the same bug. Reviewer C says BLOCKER, Reviewer B says MAJOR.
- **Code evidence**: `internal/config/config.go:94-108` — `migrateConfig` copies `Accounts`, `ColorThresholds`, and `MaxConcurrentRequests` but **never copies `UseMasterPassword`**. The field `UseMasterPassword *bool` (line 21) is silently dropped during migration.
- **Resolution**: **Same issue, confirmed real.** Both are right about the bug. The severity debate: This causes silent loss of security configuration (`use_master_password: true` becomes unset). The user would need to re-configure master password mode after any version bump that triggers migration. However, migration only triggers when `cfg.Version != CurrentVersion`, which is a one-time event per config. BLOCKER is arguably too high since it's a one-time data loss that's recoverable by re-enabling the setting. MAJOR is appropriate.
- **Corrected severity**: MAJOR

---

### [CX-3] VQ-03: → marker width — real bug or false positive?
- **Reviewers involved**: D (VQ-03)
- **Conflict**: Reviewer D claims `→` has `runewidth` of 2 in some locales, causing column misalignment.
- **Code evidence**: `internal/tui/quota.go:264` — `computeNameWidth` adds `w += 2` for the current account. `formatQuotaRow` at line 180-183 uses `marker = theme.Active.Render("→ ")` (3 bytes, but `→` is one rune). The name is then padded to `nameWidth` (which includes the +2). `padRight` uses `runewidth.StringWidth` (line 274). In `formatQuotaRow` at line 190-194, the format is `marker + name + colWidth + ...`. If `→` has `runewidth.RuneWidth` of 2 (East Asian ambiguous width), the name field starts at position 3 instead of 2, but `computeNameWidth` only adds 2 to account for it. This shifts the active row's data columns by 1 cell relative to inactive rows.
- **Resolution**: **Real bug, but locale-dependent.** On systems with `LANG=C` or Western locales, `runewidth.RuneWidth('→')` returns 1, and the output aligns correctly. On East Asian locales (`LANG=ja_JP.UTF-8`, `LANG=zh_CN.UTF-8`), `runewidth` may return 2 for `→` (East Asian Ambiguous Width character), causing the misalignment described. The bug is real but only manifests on specific locale configurations. The severity should be MINOR (not MAJOR) since it requires a specific locale to trigger and the tool is primarily targeted at English-speaking users.
- **Corrected severity**: MINOR (not MAJOR)

---

### [CX-4] F04: Exit codes 2-7 — is this a BLOCKER?
- **Reviewers involved**: E (F04)
- **Conflict**: Reviewer E classifies as BLOCKER. The code only uses `os.Exit(1)` for all errors.
- **Code evidence**: `cmd/opencode-usage/main.go:13` — `os.Exit(1)` is the only exit path. README lines 131-140 document exit codes 2-7 with specific meanings. No code in the entire codebase calls `os.Exit(2)` through `os.Exit(7)`. The `SilenceUsage: true` on root command prevents Cobra from exiting with code 2 on usage errors.
- **Resolution**: **Real issue, but BLOCKER is too high.** The documented contract is broken — scripts checking `$? -eq 3` for auth failure will never trigger. However, this is a documentation/feature gap, not a crash or data loss. Users who don't rely on specific exit codes are unaffected. The tool works correctly; it just doesn't distinguish error types via exit codes. MAJOR is the correct severity — it's a missing feature that contradicts documentation.
- **Corrected severity**: MAJOR (not BLOCKER)

---

### [CX-5] XP-001/XP-002/XP-003: Windows issues — MAJOR or known limitation?
- **Reviewers involved**: F (XP-001, XP-002, XP-003)
- **Conflict**: Reviewer F classifies all three as MAJOR cross-platform issues.
- **Code evidence**:
  - **XP-001**: `internal/cmd/quota.go:154` — `homeDir + "/.local/share/opencode/auth.json"` uses `/` separator. However, Go's `os` package on Windows handles mixed separators (`/` works in paths). The real issue is not the separator but that `.local/share` is an XDG convention that doesn't exist on Windows.
  - **XP-002**: `internal/cmd/alias.go:21-32` — `os.Getenv("SHELL")` is empty on Windows. The alias command falls back to `.bashrc` which typically doesn't exist on Windows. The command won't crash; it will just fail to find a shell RC file.
  - **XP-003**: `internal/cmd/current.go:26`, `internal/cmd/quota.go:154` — `~/.local/share/opencode/auth.json` path doesn't exist on Windows. `resolveCurrentAccount` returns `""` silently (line 157: `os.ReadFile` returns error, function returns empty string).
- **Resolution**: These are **real cross-platform issues, but MAJOR is appropriate** — the tool builds and runs on Windows (cross-compilation confirmed by Reviewer F). The core quota functionality works. The issues are: (1) `current` command can't detect the active opencode account on Windows, (2) `alias install` does nothing useful on Windows, (3) paths use Unix conventions. These are legitimate MAJOR issues for Windows users, not just "known limitations." A "known limitation" would be if the tool explicitly said "Windows not supported." It doesn't — it claims Windows support in the README.
- **Corrected severity**: MAJOR (confirmed as stated)

---

### [CX-6] T-011: No cmd tests — BLOCKER or gap?
- **Reviewers involved**: C (T-011)
- **Conflict**: Reviewer C classifies as BLOCKER.
- **Code evidence**: The `internal/cmd/` directory has zero `*_test.go` files. This package contains: concurrent quota fetching with goroutines and semaphore (`quota.go:86-147`), account CRUD with keyring/encrypted storage interaction (`account.go`), shell alias file manipulation (`alias.go`), JSON output mode, file output mode, current account resolution from auth.json.
- **Resolution**: **BLOCKER is too high, but MAJOR is correct.** The absence of tests is a significant quality gap, but the code does work — it's been manually tested and used. A BLOCKER implies the product cannot ship. The product ships and functions. However, the complexity of the concurrent fetching, file I/O, and multiple output modes makes this a high-priority MAJOR. The lack of tests for the alias file manipulation is particularly concerning since incorrect pattern matching could corrupt user shell RC files.
- **Corrected severity**: MAJOR (not BLOCKER)

---

### [CX-7] CC-01: Race condition on cachedMasterPassword — real or theoretical?
- **Reviewers involved**: A (CC-01)
- **Conflict**: Reviewer A flags this as MAJOR.
- **Code evidence**: `internal/auth/encrypted.go:77-89` — `getMasterPassword()`:
  ```go
  if cachedMasterPassword != "" {
      return cachedMasterPassword, nil
  }
  if useMasterPassword != nil && !*useMasterPassword {
      cachedMasterPassword = defaultPassword  // WRITE without sync
      return cachedMasterPassword, nil
  }
  passwordOnce.Do(doInitMasterPassword)
  ```
  When `useMasterPassword` is explicitly set to `false`, line 83 writes `cachedMasterPassword = defaultPassword` without synchronization. Multiple goroutines in `quota.go:96-117` call `auth.GetAPIKey` concurrently, which calls `getMasterPassword`. Two goroutines can both pass the line 78 check (both see empty string), both enter the line 82 path, and both write to `cachedMasterPassword` simultaneously.
- **Resolution**: **Real race, but low practical impact.** The race exists in the Go memory model sense — two goroutines writing to the same string variable without synchronization is undefined behavior. The race detector would flag this. However, both goroutines write the same value (`"opencode-usage-default"`), so in practice on amd64 the corruption window is tiny and the value is always the same. The main risk is the race detector flagging it in tests, which would make `-race` builds fail. MAJOR is correct for a library that claims to be safe for concurrent use.
- **Corrected severity**: MAJOR (confirmed)

---

### [CX-8] CC-10 vs T-005 vs REL-002: Colon in account name — overlap
- **Reviewers involved**: A (CC-10), C (T-005), F (REL-002)
- **Conflict**: Three reviewers flag the same issue with different IDs and severities: A says MINOR, C says MAJOR, F says MINOR.
- **Code evidence**: `internal/auth/encrypted.go:189` — `strings.SplitN(line, ":", 2)` splits on the first colon. Account `prod:us` with key `sk-abc` becomes `prod:us:sk-abc`, which splits to `["prod", "us:sk-abc"]` — the account name becomes `prod` and the key becomes `us:sk-abc`. This is a real data corruption bug.
- **Resolution**: **Same issue, all three are correct about the bug.** MAJOR is the right severity — this corrupts stored API keys, making them unretrievable. The encrypted file is silently corrupted and the user loses access to their API key.
- **Corrected severity**: MAJOR (Reviewer C correct, A and F too low)

---

### [CX-9] REL-004 vs CC-01: Password in global string — overlap or distinct?
- **Reviewers involved**: A (CC-01), F (REL-004)
- **Conflict**: CC-01 is about race condition on `cachedMasterPassword`. REL-004 is about the password being stored in an immutable `string` that can't be zeroed. Different issues on the same variable.
- **Code evidence**: `internal/auth/encrypted.go:21` — `cachedMasterPassword string` is a package-level var. REL-004 is about memory scraping: Go strings are immutable and can't be zeroed after use, so the password remains in process memory. CC-01 is about concurrent access without synchronization.
- **Resolution**: **Distinct issues.** REL-004 is a security hardening suggestion (use `[]byte` and zero it). CC-01 is a correctness bug (data race). Both are valid. REL-004 as MINOR is correct — it's a defense-in-depth measure, not a vulnerability in most threat models.
- **Corrected severity**: MINOR (confirmed)

---

### [CX-10] VQ-01/VQ-02 vs CC-06: Table overflow at narrow widths
- **Reviewers involved**: D (VQ-01, VQ-02), A (CC-06)
- **Conflict**: Reviewer D flags table overflow as MAJOR. Reviewer A doesn't mention this at all (different scope). No conflict between reviewers, but D's finding is significant and should be verified.
- **Code evidence**: `internal/tui/quota.go:56-66` — `usable = width - 2`, `colWidth = availCols / 3`, minimum `colWidth = 15`. At width=60: `usable=58`, `fixedTotal=nameWidth+10` (minimum 17), `availCols=41`, `colWidth=13` → bumped to 15. `sepLen = nameWidth + 3*15 + 10 = nameWidth + 55`. With `nameWidth=7` (minimum), `sepLen=62`, exceeding `usable=58`. The separator and rows overflow.
- **Resolution**: **Real bug, confirmed.** Reviewer D's analysis is correct. The minimum `colWidth=15` enforcement can cause layout to exceed terminal width. MAJOR is appropriate.
- **Corrected severity**: MAJOR (confirmed)

---

### [CX-11] F01 vs F19: Duplicate findings
- **Reviewers involved**: E (F01, F19)
- **Conflict**: F01 and F19 are the same finding — doctor command not in README.
- **Code evidence**: Both reference the same gap.
- **Resolution**: **Duplicate within the same reviewer.** F19 should be merged into F01. Not a cross-reviewer conflict, but noted for deduplication.
- **Corrected severity**: MAJOR (F01/F19 combined)

---

### [CX-12] ARC-04 threshold defaults vs CC-05: Zero-value threshold
- **Reviewers involved**: A (CC-05), B (ARC-04)
- **Conflict**: ARC-04 notes hardcoded fallbacks. CC-05 notes that `0` is treated as "unset" in `cmd/quota.go:205-210`. These are related but distinct issues.
- **Code evidence**: `internal/cmd/quota.go:205-210` — `if style.WarningThreshold == 0 { style.WarningThreshold = 50 }`. This means a user who configures `color_thresholds.warning: 0` gets silently overridden to 50. The zero value for `int` is indistinguishable from "not set."
- **Resolution**: **Distinct but related issues.** ARC-04 is about hardcoded constants being scattered. CC-05 is about the zero-value ambiguity making it impossible to set a threshold of 0. Both are real. CC-05 as MINOR is correct — it's an edge case that affects very few users.
- **Corrected severity**: MINOR (confirmed)

---

### [CX-13] SEC-001: Hardcoded default password — overlap with CC-01?
- **Reviewers involved**: A (CC-01), F (SEC-001)
- **Conflict**: SEC-001 flags the hardcoded default password `"opencode-usage-default"` as a MAJOR security issue. CC-01 uses the same variable but focuses on the race condition. Different issues.
- **Code evidence**: `internal/auth/encrypted.go:27` — `const defaultPassword = "opencode-usage-default"`. This is the encryption key used when master password mode is disabled. Anyone with the source code or binary can decrypt the secrets file.
- **Resolution**: **Distinct issues.** SEC-001 is about weak default encryption. CC-001 is about concurrent access. SEC-001 is a real security concern but the severity depends on threat model: the encrypted file is only used when the keyring is unavailable, and the file is stored with 0600 permissions. If an attacker has file access, they likely also have the binary. MAJOR is appropriate.
- **Corrected severity**: MAJOR (confirmed)

---

### [CX-14] F06: Version mismatch — BLOCKER or MAJOR?
- **Reviewers involved**: E (F06)
- **Conflict**: Reviewer E classifies as MAJOR.
- **Code evidence**: `internal/version/version.go:6` — `Version = "0.1.0"`. `docs/DELIVERY.md` title says "v0.2.0 UX Refresh".
- **Resolution**: **Real issue, MAJOR is correct.** The version in the binary doesn't match the delivery documentation. This affects release management and user confusion.
- **Corrected severity**: MAJOR (confirmed)

---

### [CX-15] ARC-07 vs XP-004: doctor.go ANSI codes vs isTerminal inconsistency
- **Reviewers involved**: B (ARC-07), F (XP-004)
- **Conflict**: ARC-07 flags doctor.go using raw ANSI codes instead of the theme system. XP-004 flags doctor.go using `os.Stdout.Stat()` instead of `term.IsTerminal()`. Different aspects of the same file.
- **Code evidence**: `internal/cmd/doctor.go:99-104` — `newDoctorTheme()` uses raw `\033[32m` escape codes. Line 106-112 — `isTerminal()` uses `os.Stdout.Stat()` check. The rest of the codebase uses lipgloss themes and `term.IsTerminal()`.
- **Resolution**: **Both are correct, complementary findings.** ARC-07 is about style inconsistency. XP-004 is about terminal detection inconsistency. Both are MINOR.
- **Corrected severity**: MINOR each (confirmed)

---

## New Issues Found During Cross-Review

### [CX-16] F07/F23 Chinese strings: MAJOR confirmed
- **Code evidence**: `internal/auth/validator.go:42-75` — All validation messages are in Chinese. `internal/auth/encrypted.go:61-62` — `"请输入主密码: "` is a user-facing interactive prompt. `internal/cmd/alias.go:23` — Chinese error message.
- **Resolution**: MAJOR is correct. User-facing text in Chinese while the project claims English-only.

### [CX-17] SEC-002 Non-atomic file writes: MAJOR confirmed
- **Code evidence**: `internal/auth/encrypted.go:213` — `os.WriteFile(path, ..., 0600)` writes directly. `internal/config/config.go:87` — same pattern. A crash mid-write corrupts the file.
- **Resolution**: MAJOR is correct for the secrets file. Less critical for config (recreated from defaults).

---

## Consolidated Severity List (All Unique Issues)

| ID | Severity | Source | Summary |
|----|----------|--------|---------|
| ARC-01 | MAJOR | B | Duplicated base URL across three packages |
| ARC-02 | MAJOR | B | Duplicated account loading boilerplate |
| ARC-03 | MAJOR | B | resolveCurrentAccount reads auth.json directly |
| ARC-04 / CC-04 | MAJOR | A, B | computeSummary uses hardcoded thresholds (corrected from MINOR) |
| ARC-05 | MAJOR | B | Scattered global mutable state in auth/tui |
| ARC-06 | MINOR | B | Duplicate accountResult struct |
| ARC-07 / XP-004 | MINOR | B, F | doctor.go bypasses theme system / inconsistent terminal detection |
| ARC-08 | MINOR | B | Alias incomplete shell support |
| ARC-09 | MINOR | B | Hardcoded fmt.Println — no unified output |
| ARC-10 | MINOR | B | Chinese/English mixed — no i18n strategy |
| ARC-11 / T-003 | MAJOR | B, C | migrateConfig silently drops UseMasterPassword (corrected from BLOCKER) |
| ARC-12 | MAJOR | B | No testable interfaces for client/auth |
| ARC-13 | MINOR | B | getAPIKeyForCommand duplicates config loading |
| ARC-14 / CC-13 | NIT | B, A | max function shadows Go built-in |
| ARC-15 | MINOR | B | doctor.go creates own HTTP client |
| ARC-16 | NIT | B | writeOutput is a free function |
| CC-01 | MAJOR | A | Data race on cachedMasterPassword |
| CC-02 | MAJOR | A | Non-atomic account add: key before config |
| CC-03 | MAJOR | A | Non-atomic account remove: delete before config |
| CC-05 | MINOR | A | Threshold value of 0 silently becomes default |
| CC-06 | MINOR | A | No context cancellation for HTTP requests |
| CC-07 | MINOR | A | Doctor reports "OK" for non-200 HTTP |
| CC-08 | MINOR | A | Alias removal doesn't handle alternative syntax |
| CC-09 | MINOR | A | Alias removal matches commented-out aliases |
| CC-10 / T-005 / REL-002 | MAJOR | A, C, F | Colon in account name breaks encrypted storage (corrected from MINOR) |
| CC-11 | MINOR | A | account list doesn't support -o flag |
| CC-12 | NIT | A | account import saves config when nothing imported |
| CC-14 | NIT | A | truncateError test too lenient |
| CC-15 | NIT | A | formatQuotaOverview mutates input slice |
| CC-16 | NIT | A | IsCurrent set redundantly |
| CC-17 | NIT | A | HTTP body not drained before close |
| CC-18 | NIT | A | homeDir() silently discards error |
| CC-19 | NIT | A | account export doesn't truncate file (correct behavior noted) |
| F01 / F19 | MAJOR | E | README missing doctor command |
| F02 | MAJOR | E | README missing --output / -o flag |
| F03 | MINOR | E | README missing --no-color flag |
| F04 | MAJOR | E | Exit codes 2-7 documented but only 0/1 used (corrected from BLOCKER) |
| F05 | MAJOR | E | Go version mismatch design doc vs go.mod |
| F06 | MAJOR | E | Version mismatch DELIVERY.md vs version.go |
| F07 / F23 | MAJOR | E | Chinese error messages contradict English claim |
| F08 | MINOR | E | README missing completion command |
| F09 | MAJOR | E | account list ignores --json flag |
| F10 | MAJOR | E | Root command default behavior undocumented |
| F11 | MINOR | E | JSON output field name inconsistency |
| F12 | MINOR | E | account export description incomplete |
| F13 | NIT | E | formatResetTime shows <1m correctly (stale audit) |
| F14 | MINOR | E | Go module path mismatch |
| F15 | MINOR | E | README missing env var documentation |
| F16 | MINOR | E | README missing configurable thresholds |
| F17 | MINOR | E | README missing max_concurrent_requests |
| F18 | MINOR | E | README missing use_master_password config |
| F20 | NIT | E | version output format undocumented |
| F21 | MAJOR | E | update command doesn't actually update |
| F24 | MINOR | E | Design doc version planning stale |
| F25 | MINOR | E | README missing Go version requirement |
| SEC-001 | MAJOR | F | Hardcoded default encryption password |
| SEC-002 | MAJOR | F | Non-atomic file writes for secrets |
| SEC-003 | MINOR | F | No file permission check on secrets read |
| SEC-004 | MINOR | F | Account names as keyring keys without namespace |
| SEC-005 | MINOR | F | API key logged via doctor command |
| SEC-006 | NIT | F | ExtractKeyID exposes last 6 chars |
| SEC-007 | NIT | F | Colon-delimited format inside encryption |
| SEC-008 | MINOR | F | Retry-After header ignored |
| SEC-009 | MINOR | F | Retry amplification on non-transient errors |
| SEC-010 | MINOR | F | Non-atomic config migration |
| SEC-011 | NIT | F | Doctor makes unsolicited HTTP request |
| XP-001 | MAJOR | F | Hardcoded Unix path separators |
| XP-002 | MAJOR | F | Alias command assumes Unix shell |
| XP-003 | MAJOR | F | .local/share path doesn't exist on Windows |
| REL-001 | MINOR | F | No panic recovery in goroutines |
| REL-003 | MINOR | F | Non-atomic read-modify-write for deletes |
| REL-004 | MINOR | F | Master password cached in immutable string |
| REL-005 | NIT | F | Inconsistent HTTP client timeouts |
| T-001 | BLOCKER | C | Client retry logic completely untested |
| T-002 | MAJOR | C | Client malformed response not tested |
| T-004 | MAJOR | C | Config corrupt YAML not tested |
| T-006 | MAJOR | C | Encrypted store concurrent access untested |
| T-007 | MAJOR | C | Keyring test skipped in most CI |
| T-008 | MINOR | C | ExtractKeyID edge cases not tested |
| T-009 | MAJOR | C | Validator env var fallback not tested |
| T-010 | MINOR | C | Config permission error not tested |
| T-011 | MAJOR | C | No cmd-level tests (corrected from BLOCKER) |
| T-012 | MAJOR | C | Alias file manipulation not tested |
| T-013 | MINOR | C | TUI NO_COLOR mode not verified |
| T-014 | MINOR | C | Narrow terminal compact mode not tested properly |
| T-015 | MINOR | C | TestPadRight has weak assertions |
| T-016 | MAJOR | C | Integration test only tests --help |
| T-017 | NIT | C | formatResetTime zero-minutes edge case |
| T-018 | MINOR | C | computeSummary edge cases not tested |
| T-019 | MINOR | C | findBestAccount all-errored not tested |
| T-020 | MAJOR | C | Concurrent quota fetching untested |
| T-021 | NIT | C | Unreachable code in doRequest |
| T-022 | MINOR | C | ValidateAPIKey test tests useless behavior |
| T-023 | NIT | C | SetTerminalWidth with 0 not guarded |
| T-024 | MINOR | C | No test for renderBar width=0 |
| T-025 | NIT | C | formatResetTime exact 24h boundary |
| T-026 | MAJOR | C | Global state pollution between tests |
| VQ-01 | MAJOR | D | Table overflows terminal at narrow widths |
| VQ-02 | MAJOR | D | Separator line not constrained to width |
| VQ-03 | MINOR | D | Active account column misalignment (corrected from MAJOR) |
| VQ-04 | MINOR | D | Grammar: "1 accounts" |
| VQ-05 | NIT | D | Compact/table mode marker inconsistency |
| VQ-06 | NIT | D | truncateError over-truncates exact-width |
| VQ-07 | NIT | D | Empty account name produces bare "Active:" |
| VQ-08 | MINOR | D | NO_COLOR footer same symbol for healthy/critical |

---

## Severity Count Summary

| Severity | Count | Change from Round 1 |
|----------|-------|-------------------|
| BLOCKER | 1 | -3 from original 4 (T-003→MAJOR, T-011→MAJOR, F04→MAJOR) |
| MAJOR | 37 | +3 (CC-04 promoted MINOR→MAJOR, CC-10 promoted MINOR→MAJOR, VQ-03 demoted MAJOR→MINOR net -1) |
| MINOR | 33 | -3 (CC-04, CC-10 out; VQ-03 in) |
| NIT | 14 | unchanged |

**Net change from Round 1**: BLOCKER reduced from 4 to 1 (only T-001 client retry untested remains). Two findings promoted from MINOR to MAJOR (CC-04 hardcoded thresholds, CC-10 colon in account name). Three findings demoted from BLOCKER to MAJOR (T-003/ARC-11 migration bug, F04 exit codes, T-011 no cmd tests). One finding demoted from MAJOR to MINOR (VQ-03 → marker width, locale-dependent).
