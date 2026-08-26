# Engineering Review — Subagent E

**Reviewer:** Independent Engineering Reviewer
**Date:** 2026-08-26
**Scope:** `internal/tui/theme.go`, `internal/tui/quota.go`, `internal/tui/quota_test.go`, `internal/cmd/root.go`, `internal/cmd/quota.go`, `internal/cmd/account.go`, `internal/cmd/doctor.go`, `internal/cmd/current.go`, `internal/cmd/alias.go`, `internal/cmd/models.go`

---

## 1. Secret Leakage

| # | Severity | File | Finding |
|---|----------|------|---------|
| 1.1 | **MAJOR** | `internal/cmd/current.go:55-57` | Token is masked as `***` + last 4 chars. This leaks ~4 characters of a bearer token. While not the full key, this is a weaker masking scheme than typical `****` or redaction. If tokens have low entropy in the suffix (e.g., base64), 4 chars may be guessable. |
| 1.2 | **MAJOR** | `internal/cmd/account.go:153` | `accountListCmd` prints `"sk-..." + account.KeyID` where `KeyID` is the last 6 chars of the API key (`auth.ExtractKeyID` returns `apiKey[len(apiKey)-6:]`). Combined with the `sk-...` prefix, this leaks 6 suffix characters of every API key. On a shared terminal or piped output, this is a significant information disclosure. |
| 1.3 | **MAJOR** | `internal/cmd/account.go:343-354` | `accountImportCmd` prints validation errors to stdout which may include token echoes or server error details. The `result.Message` (e.g., "请检查您的API Key") is safe, but the generic `err` on line 346 could contain the request URL which includes the API key as a Bearer token in diagnostic output from the HTTP client. |
| 1.4 | **MINOR** | `internal/cmd/account.go:77` | API keys are stored via `auth.StoreAPIKey` which delegates to keyring or encrypted file. The storage itself is sound (keyring or AES-GCM encrypted). However, the in-memory `apiKey` variable (`line 48`) lives as a Go string that cannot be zeroed — standard Go limitation but worth noting. |
| 1.5 | **NIT** | `internal/tui/quota.go:131` | Error messages from API responses are displayed truncated but unredacted. Errors like "HTTP 401" are safe, but if a server returns a message containing an echo of the token, it would be shown. |

## 2. Race Conditions

| # | Severity | File | Finding |
|---|----------|------|---------|
| 2.1 | **MAJOR** | `internal/tui/theme.go:12-13,100-104` | Global mutable variables `isColorEnabled` and `terminalWidth` are written during `init()` (line 17-26) and again in a second `init()` (line 100-104) without synchronization. While `init()` runs sequentially, `DisableColor()` and `SetTerminalWidth()` (lines 28-36) are exported and could be called concurrently. There is no mutex protecting these globals. |
| 2.2 | **MINOR** | `internal/cmd/quota.go:86-121` | The goroutines write to a buffered channel (`results`) which is safe. The `sync.WaitGroup` + `close(results)` pattern is correct. The semaphore channel limits concurrency. No race here. |

## 3. File Permissions

| # | Severity | File | Finding |
|---|----------|------|---------|
| 3.1 | **MINOR** | `internal/cmd/root.go:52` | `writeOutput` uses `0644` for `--output` files. This is world-readable. For quota data this is acceptable, but if a user pipes JSON output containing account names to a file, 0644 means any user on the system can read it. Consider 0600 or respecting umask. |
| 3.2 | **NIT** | `internal/cmd/account.go:285` | `accountExportCmd` correctly uses `0600` for export files. Good. |
| 3.3 | **NIT** | `internal/cmd/alias.go:50` | RC files are opened with `0644` which matches the typical RC file permission. Acceptable. |

## 4. Context Cancellation

| # | Severity | File | Finding |
|---|----------|------|---------|
| 4.1 | **MAJOR** | `internal/client/opencode.go:41` | `http.NewRequest` is called without a context. Requests are not cancellable via `Ctrl+C`. The `http.Client.Timeout` (10s) provides a ceiling, but if the user kills the process mid-request, the goroutine will leak until the timeout fires. The `doctor.go:46` call `client.Get(url)` also has no context. |
| 4.2 | **MINOR** | `internal/cmd/quota.go:93-118` | The concurrent quota fetching goroutines have no cancellation. If the user presses Ctrl+C, goroutines continue running until HTTP timeouts expire. Should use `signal.NotifyContext` or pass `cmd.Context()`. |

## 5. Error Handling

| # | Severity | File | Finding |
|---|----------|------|---------|
| 5.1 | **MINOR** | `internal/cmd/doctor.go:114-116` | `homeDir()` silently swallows `os.UserHomeDir()` error, returning empty string. This leads to silent path construction failures (e.g., line 54 checks `homeDir() + "/.local/share/..."` which becomes `"/.local/share/..."`). |
| 5.2 | **NIT** | `internal/tui/quota.go:55` | `_ = nameWidth` — the computed `nameWidth` is computed but immediately discarded with `_ =`. This is dead code (line 55 assigns, then line 66 recomputes `totalWidth` without using it). Appears to be a leftover from refactoring. |
| 5.3 | **NIT** | `internal/cmd/quota.go:130` | `result.err.Error()` — error string is stored in JSON output. If an error message contains sensitive info (e.g., file path to encrypted keyring), it leaks to JSON consumers. Not currently exploitable but fragile. |

## 6. Unicode/ANSI Safety

| # | Severity | File | Finding |
|---|----------|------|---------|
| 6.1 | **MINOR** | `internal/cmd/doctor.go:103` | Uses raw ANSI escape codes (`\033[32m✓\033[0m`) instead of `lipgloss` or `termenv`. While functional, this bypasses the `NO_COLOR` check done via `lipgloss.SetColorProfile` in root.go's `PersistentPreRun`. The `doctorTheme` has its own color check (line 100) which is good, but the approach is inconsistent with the rest of the codebase. |
| 6.2 | **MINOR** | `internal/tui/quota.go:209-210` | Uses `█` (U+2588) and `░` (U+2591) Unicode block characters for progress bars. These are safe on modern terminals but may render as `?` or boxes on very old/limited terminals. The code does fall back to no-color mode but doesn't fall back to ASCII bar characters (e.g., `#` and `-`). |
| 6.3 | **NIT** | `internal/cmd/alias.go:23` | Error message is in Chinese (`"获取用户主目录失败"`) while most other error messages in the codebase are in English. Inconsistent locale handling. |

## 7. JSON Stability

| # | Severity | File | Finding |
|---|----------|------|---------|
| 7.1 | **MAJOR** | `internal/cmd/quota.go:27-30,141-143` | JSON output schema includes `"version": "1"` and `"accounts": [...]` with `quotaResponse`. The `Usage` model uses `json:"rolling"`, `json:"weekly"`, `json:"monthly"` but the top-level response uses `"quota"` (`json:"quota,omitempty"` in `accountResult`). This means JSON output shows `"quota"` key for account data, not `"usage"`. If the API or internal model changes field names, all JSON consumers break. There is no documented schema versioning beyond the `"version"` field. |
| 7.2 | **MINOR** | `internal/cmd/quota.go:142` | The `"version"` field is always `"1"` but is never checked or validated on the consumer side. Without documentation, consumers cannot know what fields may appear or disappear between versions. |

## 8. Test Quality

| # | Severity | File | Finding |
|---|----------|------|---------|
| 8.1 | **MINOR** | `internal/tui/quota_test.go:229-231` | `SetTerminalWidth(50)` mutates global state without synchronization. If tests run in parallel (`t.Parallel()`), this creates flaky tests. None of the tests use `t.Parallel()` currently, so this is safe for now but fragile. |
| 8.2 | **MINOR** | `internal/tui/quota_test.go:248` | `DisableColor()` is called at test scope without restoring state. Tests that run after this test will have color permanently disabled. The `TestProgressBars` test calls `DisableColor()` at line 248 but other tests (e.g., `TestFormatQuotaOverviewSingleAccount`) don't restore color state, so their output may vary depending on test execution order. |
| 8.3 | **MINOR** | Coverage gaps: No tests for `formatTable` with narrow terminal (barWidth adjustment logic), no tests for `findNextReset`, no tests for edge cases like 0 accounts vs 1 account in `findBestAccount`, no tests for error wrapping in `formatQuotaCell`. |
| 8.4 | **NIT** | `TestPadRight` (line 134) — the assertion logic at lines 147-154 manually computes runewidth instead of using the library, making the test logic mirror the implementation rather than independently verify it. |

## 9. Breaking Changes

| # | Severity | File | Finding |
|---|----------|------|---------|
| 9.1 | **MINOR** | `internal/cmd/root.go:27-28` | The root command now has `Aliases: []string{"ou"}` and the root `RunE` calls `runQuotaOverview` directly. Previously (pre-change), `opencode-usage` with no subcommand likely showed help. Now it shows quota output. This is a **behavioral breaking change** for users who run bare `opencode-usage`. |
| 9.2 | **NIT** | JSON output format: The JSON schema fields use `"quota"` (from `accountResult`) not `"usage"`. Any prior consumers expecting different field names will break. Since this appears to be the first release, this may be acceptable. |

## 10. Security Surface

| # | Severity | File | Finding |
|---|----------|------|---------|
| 10.1 | **MAJOR** | `internal/auth/encrypted.go:27` | `defaultPassword = "opencode-usage-default"` is a hardcoded encryption fallback password used when master password is not enabled. This means anyone who can read the encrypted file (`secrets.enc`) can decrypt it with this static password. The encrypted file is better than plaintext, but this provides minimal security against an attacker with file access. |
| 10.2 | **MAJOR** | `internal/cmd/account.go:48` | After `term.ReadPassword`, the raw API key is stored as a Go string (`apiKey := strings.TrimSpace(string(apiKeyBytes))`). The original `apiKeyBytes` slice from `term.ReadPassword` is not zeroed. While Go's GC will eventually reclaim the memory, the key remains in the process heap indefinitely. |
| 10.3 | **MINOR** | `internal/auth/credential.go:42` | The keyring test writes `"test"` as a plaintext value during `init()`. If the keyring backend has any logging or persistence side effects, this could leak. Minor risk. |
| 10.4 | **NIT** | `internal/cmd/alias.go:48-50` | The alias command appends `alias ou='opencode-usage'` to RC files. If the binary path is not in `$PATH`, this alias will fail silently. More importantly, if an attacker can modify the RC file before the user runs `alias install`, they could inject arbitrary commands. This is a standard risk for RC file modification tools. |

---

## Summary

| Severity | Count |
|----------|-------|
| BLOCKER  | 0     |
| MAJOR    | 7     |
| MINOR    | 12    |
| NIT      | 7     |

### Key MAJOR Issues

1. **API key suffix leakage** (1.2) — `account list` prints last 6 chars of every API key
2. **Global mutable state without synchronization** (2.1) — `isColorEnabled`/`terminalWidth` unprotected
3. **Hardcoded encryption password** (10.1) — `defaultPassword` makes encrypted storage trivially bypassable
4. **Uncancellable HTTP requests** (4.1) — No context propagation for graceful shutdown
5. **Token masking is weak** (1.1) — Only 4 suffix chars shown, still partially leaks token
6. **No input sanitization on error output** (1.3) — Error messages could echo sensitive data
7. **JSON schema undocumented** (7.1) — `"version"` field exists but contract is unspecified

### Final Verdict

**REQUEST CHANGES**

The code is functional and well-structured overall. However, the API key suffix leakage in `account list` (MAJOR 1.2) and the hardcoded encryption default password (MAJOR 10.1) are security concerns that should be addressed before release. The context cancellation gap (MAJOR 4.1) will cause goroutine leaks on Ctrl+C. The remaining MAJOR issues (race conditions on globals, weak token masking) are lower risk but should be tracked.
