# Round 1 — Code Correctness Review

**Reviewer**: Reviewer A — Code Correctness  
**Date**: 2026-08-26  
**Scope**: All Go source files in `cmd/`, `internal/auth/`, `internal/client/`, `internal/config/`, `internal/models/`, `internal/tui/`, `internal/version/`

---

## Findings

### [CC-01] Data race on `cachedMasterPassword` global variable
- **Severity**: MAJOR
- **File**: `internal/auth/encrypted.go:78-88`
- **Description**: `getMasterPassword()` reads and writes `cachedMasterPassword` (a package-level `string`) without synchronization. When `useMasterPassword` is not nil and `*useMasterPassword == false`, the function takes the early-return path (line 83) that writes `cachedMasterPassword = defaultPassword` without going through `passwordOnce.Do()`. If two goroutines call `getMasterPassword()` concurrently (which happens in `quota.go` line 100 when multiple accounts are queried), both can enter this path simultaneously, creating a data race on the global string variable.
- **Impact**: Undefined behavior per Go memory model. On amd64 the string pointer+length pair may be torn, causing one goroutine to read a partially-written string. In practice the value is always `"opencode-usage-default"` so the corruption window is small, but the race detector will flag this and it violates Go's memory safety guarantees.
- **Reproduction**: Run `go test -race ./internal/auth/` with `useMasterPassword` set to `false` and multiple concurrent callers of `GetAPIKey`.
- **Suggested Fix**: Protect the entire `getMasterPassword` function body with a `sync.Once` or `sync.RWMutex`, or restructure so the `useMasterPassword == false` path also goes through `passwordOnce.Do`.

---

### [CC-02] Non-atomic account add: key stored before config saved
- **Severity**: MAJOR
- **File**: `internal/cmd/account.go:77-91`
- **Description**: `accountAddCmd.RunE` stores the API key in the keyring/encrypted file (line 77) *before* saving the config (line 88). If `config.SaveConfig` fails (e.g., disk full, permissions error), the key is stored but the config file doesn't reflect the new account. On next run, the account is invisible but the key persists as an orphan in the keyring.
- **Impact**: Data inconsistency. The user sees no error about the account being added (since the command returned an error), but the key is permanently stored. Subsequent `account add` with the same name fails with "already exists" because the config check passes, but the key is already in the keyring.
- **Reproduction**: Make the config directory read-only after the keyring store succeeds (e.g., `chmod 555 ~/.config/opencode-usage/`) and run `account add`.
- **Suggested Fix**: Validate config save *before* storing the key, or add a cleanup step that removes the key if config save fails.

---

### [CC-03] Non-atomic account remove: key deleted before config saved
- **Severity**: MAJOR
- **File**: `internal/cmd/account.go:214-222`
- **Description**: `accountRemoveCmd.RunE` deletes the key from the keyring (line 214) *before* saving the config (line 220). If `config.SaveConfig` fails, the key is deleted but the config still lists the account. On next run, the account appears in the list but queries fail because the key is gone.
- **Impact**: Data inconsistency opposite of CC-02. Account appears configured but is non-functional.
- **Reproduction**: Make the config directory read-only and run `account remove <name>`.
- **Suggested Fix**: Save the config first, then delete the key. If key deletion fails, attempt to restore the config.

---

### [CC-04] `computeSummary` uses hardcoded thresholds instead of style thresholds
- **Severity**: MINOR
- **File**: `internal/tui/quota.go:298-300`
- **Description**: `computeSummary` classifies accounts as healthy/warning/critical using hardcoded `80` and `50` thresholds, but the visual rendering in `formatPercent`, `formatPercentCompact`, and `renderBar` uses `style.WarningThreshold` and `style.DangerThreshold`. If a user configures custom thresholds (e.g., warning=40, danger=70), the summary text says "2 healthy" while the colors show yellow/red.
- **Impact**: Summary text contradicts visual color indicators when custom thresholds are configured.
- **Reproduction**: Set `color_thresholds.warning: 40` and `color_thresholds.danger: 70` in config. Add an account with rolling=45%, weekly=45%, monthly=45%. The bars render yellow but the summary counts it as "healthy".
- **Suggested Fix**: Pass the `QuotaStyle` to `computeSummary` and use `style.WarningThreshold` / `style.DangerThreshold` instead of hardcoded 50/80.

---

### [CC-05] Threshold value of 0 in config silently becomes default
- **Severity**: MINOR
- **File**: `internal/cmd/quota.go:205-210`
- **Description**: `printQuotaTable` treats a threshold of `0` as "unset" and replaces it with the default (50/80). This means a user cannot configure a threshold of 0 (which would mean "always warn" or "always danger"). The YAML zero value for `int` is `0`, making it impossible to distinguish "not set" from "explicitly set to 0".
- **Impact**: User-configured threshold of 0 is silently overridden. Minor UX issue.
- **Reproduction**: Set `color_thresholds.warning: 0` in config.yaml and run `quota`. The threshold becomes 50.
- **Suggested Fix**: Use `*int` (pointer) for threshold fields in the config struct, or use a sentinel value like -1 for "unset".

---

### [CC-06] No context cancellation for concurrent HTTP requests
- **Severity**: MINOR
- **File**: `internal/cmd/quota.go:93-118`, `internal/client/opencode.go:33-76`
- **Description**: The goroutines in `runQuotaOverview` have no context. `doRequest` retries up to 3 times with exponential backoff (1s, 2s) plus 10s timeout each attempt. Worst case: ~37 seconds of continued network activity after Ctrl+C.
- **Impact**: User pressing Ctrl+C sees the program hang for up to 37 seconds while HTTP requests complete and retry.
- **Reproduction**: Run `quota` with a slow/unreachable server. Press Ctrl+C immediately. Observe the process continues running.
- **Suggested Fix**: Accept `context.Context` in `NewClient`/`doRequest` and propagate it via `http.NewRequestWithContext`. Cancel the context on SIGINT/SIGTERM.

---

### [CC-07] Doctor check reports "OK" for non-200 HTTP responses
- **Severity**: MINOR
- **File**: `internal/cmd/doctor.go:46-52`
- **Description**: The network connectivity check in `doctorCmd` considers any non-error HTTP response as "OK", including 401, 403, 500, etc. While the intent is a connectivity check, a 500 from the server is reported as "opencode.ai reachable" with green checkmark.
- **Impact**: Misleading diagnostic output. Server-side issues appear as network health.
- **Reproduction**: Block port 443 and run `doctor` → FAIL (correct). Have a proxy return 500 → OK (misleading).
- **Suggested Fix**: Check `resp.StatusCode == http.StatusOK` or at least `resp.StatusCode < 500`.

---

### [CC-08] Alias removal doesn't handle alternative shell alias syntax
- **Severity**: MINOR
- **File**: `internal/cmd/alias.go:99-115`, `internal/cmd/alias.go:130-142`
- **Description**: `aliasExists` checks for `alias ou=` in a line. Shell aliases can also be written as `alias ou 'command'` (without `=`). A user with this syntax won't have their alias detected or removed.
- **Impact**: `alias install` adds a duplicate alias. `alias uninstall` fails to find and remove the existing alias.
- **Reproduction**: Add `alias ou 'opencode-usage'` to `.bashrc`. Run `alias install`. A duplicate is added. Run `alias uninstall`. The original alias is not removed.
- **Suggested Fix**: Match on `alias ou` followed by either `=` or whitespace, and use a more flexible regex or split-based matching.

---

### [CC-09] Alias removal matches commented-out aliases
- **Severity**: MINOR
- **File**: `internal/cmd/alias.go:130-142`
- **Description**: `removeAlias` uses `strings.Contains(line, aliasLine)` which matches the pattern anywhere in the line. A commented-out line like `# alias ou=old-command` would be removed.
- **Impact**: Commented-out alias lines are silently deleted.
- **Reproduction**: Add `# alias ou=old-version` as a comment in `.bashrc`. Run `alias uninstall`. The comment line is removed.
- **Suggested Fix**: Trim the line and check if it starts with `alias ou=` (not preceded by `#`).

---

### [CC-10] Account names/API keys with colon or newline break encrypted storage
- **Severity**: MINOR
- **File**: `internal/auth/encrypted.go:188-193`, `internal/auth/encrypted.go:246-251`
- **Description**: The encrypted storage format is `account:apikey\n`. If an account name contains a colon, `SplitN(line, ":", 2)` splits on the wrong delimiter, truncating the account name. If an API key contains a newline, the line-based parsing breaks entirely.
- **Impact**: Corrupted storage. API key cannot be retrieved. Account becomes inaccessible.
- **Reproduction**: Store an account named `work:prod` and an API key containing `\n`. Attempt to retrieve them.
- **Suggested Fix**: Use a different delimiter (e.g., null byte) or JSON-based format for the encrypted file content.

---

### [CC-11] `account list` doesn't support `-o` flag
- **Severity**: MINOR
- **File**: `internal/cmd/account.go:135-158`
- **Description**: `accountListCmd.RunE` uses `fmt.Printf` directly instead of `writeOutput`. The global `-o` flag is silently ignored for this command, while it works for `quota` and `models`.
- **Impact**: User expects `ou account list -o out.txt` to write to file; it doesn't.
- **Reproduction**: Run `ou account list -o /tmp/test.txt`. The output goes to stdout, not the file.
- **Suggested Fix**: Use `writeOutput` in `accountListCmd` for consistency.

---

### [CC-12] `account import` saves config even when nothing was imported
- **Severity**: NIT
- **File**: `internal/cmd/account.go:372-374`
- **Description**: After the import loop, `config.SaveConfig` is called unconditionally. If all accounts were skipped (imported == 0), the config is written back unchanged (triggering a full YAML marshal and disk write for no reason).
- **Impact**: Unnecessary disk I/O and potential config migration trigger on unchanged data.
- **Reproduction**: Import a file where all accounts already exist.
- **Suggested Fix**: Only call `SaveConfig` if `imported > 0`.

---

### [CC-13] Dead `max` function in tui/quota.go
- **Severity**: NIT
- **File**: `internal/tui/quota.go:381-386`
- **Description**: The `max(a, b int) int` function is defined but never called. Go 1.21+ provides a builtin `max` function, making this redundant even if it were used.
- **Impact**: Dead code.
- **Reproduction**: `grep -rn 'max(' internal/tui/quota.go` shows no call sites.
- **Suggested Fix**: Remove the function.

---

### [CC-14] `truncateError` test is too lenient
- **Severity**: NIT
- **File**: `internal/tui/quota_test.go:217-222`
- **Description**: `TestTruncateError` counts runes (via `for range truncated`) and checks `charCount > 31`. But `truncateError` truncates based on `runewidth`, not rune count. For ASCII text these are equivalent, but the test doesn't verify width-based behavior for CJK characters.
- **Impact**: Test doesn't catch width-based truncation bugs.
- **Reproduction**: Add a test case with a CJK string that exceeds `maxLen` in width but not in rune count.
- **Suggested Fix**: Assert `runewidth.StringWidth(truncated) <= maxLen` instead of counting runes.

---

### [CC-15] `formatQuotaOverview` mutates input slice
- **Severity**: NIT
- **File**: `internal/tui/quota.go:39-43`
- **Description**: `formatQuotaOverview` mutates `results[i].IsCurrent` in place. While the current caller creates a fresh slice, this is a side effect that could surprise future callers.
- **Impact**: Unexpected mutation if the same slice is reused.
- **Reproduction**: Pass the same slice to `formatQuotaOverview` twice with different `currentAccount` values.
- **Suggested Fix**: Work on a copy of the slice, or set `IsCurrent` before calling `FormatQuotaOverview`.

---

### [CC-16] `IsCurrent` set redundantly in two places
- **Severity**: NIT
- **File**: `internal/cmd/quota.go:127`, `internal/tui/quota.go:40-42`
- **Description**: `IsCurrent` is set when creating `accountResult` in `runQuotaOverview` (line 127) and again in `FormatQuotaOverview` (line 40-42). The TUI overwrites the data model's value.
- **Impact**: Confusing code duplication. If the logic diverges, the TUI wins.
- **Reproduction**: Read both call sites.
- **Suggested Fix**: Remove the assignment in `FormatQuotaOverview` and rely on the caller to set it correctly, or document that `FormatQuotaOverview` normalizes the field.

---

### [CC-17] HTTP body not drained before close on retryable errors
- **Severity**: NIT
- **File**: `internal/client/opencode.go:56-57`
- **Description**: `doRequest` reads the body via `io.ReadAll` and closes it, but on 429/5xx retryable errors (line 66-69), the response is discarded. The body *is* read and closed (lines 56-57), so no leak occurs. However, `resp.Body.Close()` is called without defer, meaning if `io.ReadAll` panics (theoretically possible on OOM), the body leaks.
- **Impact**: Negligible in practice. A panic during ReadAll would leak the body until GC.
- **Reproduction**: Difficult to reproduce in normal conditions.
- **Suggested Fix**: Use `defer resp.Body.Close()` immediately after checking `err != nil`.

---

### [CC-18] `homeDir()` silently discards error
- **Severity**: NIT
- **File**: `internal/cmd/doctor.go:114-117`
- **Description**: `homeDir()` calls `os.UserHomeDir()` and discards the error with `_`. If `UserHomeDir` fails, it returns an empty string, causing the `os.Stat` check on line 54 to always fail (file not found).
- **Impact**: The "OpenCode auth" check would report WARN instead of OK, even if the file exists.
- **Reproduction**: Set `HOME` to a non-existent path.
- **Suggested Fix**: Handle the error explicitly, or use the same `getConfigPath()` pattern used elsewhere.

---

### [CC-19] `accountExportCmd` doesn't truncate file on write
- **Severity**: NIT
- **File**: `internal/cmd/account.go:285`
- **Description**: `os.WriteFile` with `0600` truncates by default (the `O_TRUNC` flag is implied by `WriteFile`). This is correct, but worth noting that if the export file already exists with different permissions, the permissions are preserved (WriteFile doesn't change permissions of existing files).
- **Impact**: None. Correct behavior.
- **Reproduction**: N/A
- **Suggested Fix**: No change needed.

---

## Edge Case Analysis

### 0 accounts
- `runQuotaOverview`: Returns early with "No accounts configured" message or empty JSON. **Correct**.
- `accountListCmd`: Returns early with "No accounts configured" message. **Correct**.
- `modelsCmd`: Returns error "no accounts configured". **Correct**.

### 20 accounts
- `runQuotaOverview`: Spawns 20 goroutines with semaphore limiting concurrency. Channel buffer is 20. **Correct**.
- `computeNameWidth`: Iterates all accounts, finds max width. **Correct**.

### Quota at boundary values (0%, 50%, 80%, 100%)
- `renderBar`: `filled = (percent * width) / 100`. At 100%, `filled = width`, `empty = 0`. **Correct**.
- `formatPercent`: At 80%, `percent >= DangerThreshold(80)` → red. At 79%, `percent >= WarningThreshold(50)` → yellow. **Correct**.
- `computeSummary`: At 80%, `maxPercent >= 80` → "critical". At 79%, `maxPercent >= 50` → "warning". **Correct** (but uses hardcoded thresholds, see CC-04).

### Past reset time
- `formatResetTime`: `duration < 0` → "expired". **Correct**.
- `findNextReset`: If all resets are in the past, picks the earliest (most negative). `formatResetTime` returns "expired". **Correct**.

### HTTP 401, 429, 500
- `doRequest`: 401 → immediate error. 429/500 → retry up to 3 times. **Correct**.
- `ValidateAPIKey`: 401 → "invalid_api_key". 429 → "rate_limited". 500 → "server_error". **Correct**.

### Empty account name
- `accountAddCmd`: Checks `name == ""` and returns error. **Correct**.
- `accountImportCmd`: Skips entries with empty name. **Correct**.

### CJK account names
- `computeNameWidth` and `padRight` use `runewidth.StringWidth` for display width. **Correct**.
- `formatQuotaOverview` checks for CJK names in tests. **Correct**.

### Narrow terminal (50 cols)
- `formatQuotaOverview`: `width < 60` → uses `formatCompact`. **Correct**.
- `formatCompact`: No width-dependent layout. **Correct**.

### Wide terminal (200 cols)
- `formatTable`: `usable = width - 2 = 198`. `fixedTotal = nameWidth + 10`. `colWidth = (198 - fixedTotal) / 3`. **Correct**.

---

## Verdict

**REQUEST CHANGES**

The three data-inconsistency bugs (CC-02, CC-03) and the race condition (CC-01) are the most impactful findings. CC-02 and CC-03 create orphaned keys or phantom accounts on any config-save failure. CC-01 violates Go's memory model when master password mode is disabled and multiple accounts are queried concurrently. These should be fixed before release.

The remaining MINOR findings (CC-04 through CC-11) are lower priority but represent real correctness gaps that affect users with custom configurations or non-standard shell setups.
