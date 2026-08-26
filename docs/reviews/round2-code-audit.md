# Round 2 — Fresh Code Audit (Reviewer G)

**Auditor**: Reviewer G (independent, no prior findings)
**Scope**: All Go source in `internal/`, `cmd/`, `test/`
**Tools**: Manual review, `go test ./...`, `go test -race ./...`, `go vet ./...`
**Date**: 2026-08-26

---

## Test Results

| Command | Result |
|---------|--------|
| `go test ./...` | PASS (all packages) |
| `go test -race -count=1 ./...` | PASS (no races) |
| `go go vet ./...` | PASS (clean) |
| `go build ./...` | PASS |

---

## Findings

### F1 — Race condition on `cachedMasterPassword` (MAJOR)

**File**: `internal/auth/encrypted.go:54-76, 78-96`

`doInitMasterPassword()` writes to the package-level `cachedMasterPassword` variable (lines 57 and 75) **without acquiring `passwordMu`**. Meanwhile, `getMasterPassword()` reads `cachedMasterPassword` under `passwordMu.RLock()` (line 80) and reads it again **without any lock** on line 96.

The `sync.Once` in `passwordOnce` serializes calls to `Do()`, but it does **not** provide happens-before ordering for reads outside `Do()`. If goroutine B reads `cachedMasterPassword` on line 80 (under RLock) while goroutine A is writing it in `doInitMasterPassword()` (no lock), this is a data race.

The race detector did not trigger during testing because the `passwordOnce.Do` path is unlikely to race under single-threaded test execution, but the theoretical window exists in production with concurrent encrypted-key operations.

**Severity**: MAJOR — data race on shared state
**Verdict**: REQUEST CHANGES

**Recommendation**: Acquire `passwordMu.Lock()` inside `doInitMasterPassword()` before writing to `cachedMasterPassword`, or restructure so that `passwordOnce.Do` handles all synchronization atomically (e.g., set the password as the return value of the function passed to `Do`).

---

### F2 — `resolveCurrentAccount` uses hardcoded path (MINOR)

**File**: `internal/cmd/quota.go:149-178`

`resolveCurrentAccount` reads `~/.local/share/opencode/auth.json` with a hardcoded path. This is duplicated from the same path used in `current.go:26`. If the path ever changes, two locations must be updated.

**Severity**: MINOR — maintainability
**Verdict**: APPROVE (with recommendation)

**Recommendation**: Extract the auth path to a shared constant or helper, similar to how `getConfigPath()` is used for the config file.

---

### F3 — `homeDir()` silently swallows errors (MINOR)

**File**: `internal/cmd/doctor.go:114-117`

```go
func homeDir() string {
    h, _ := os.UserHomeDir()
    return h
}
```

If `UserHomeDir()` fails, `h` is `""` and the subsequent `os.Stat(homeDir() + "/.local/share/opencode/auth.json")` will check `/` which will always exist, leading to a misleading "auth.json found" result.

Similarly in `internal/auth/credential.go:17`:
```go
homeDir, _ := os.UserHomeDir()
```

**Severity**: MINOR — silent failure, misleading diagnostics
**Verdict**: REQUEST CHANGES

**Recommendation**: Return the error from `homeDir()`, or at minimum log a warning when `UserHomeDir()` fails.

---

### F4 — `resp.Body.Close()` not deferred in retry loop (MINOR)

**File**: `internal/client/opencode.go:48-57`

```go
resp, err := c.httpClient.Do(req)
if err != nil {
    ...
}
body, err := io.ReadAll(resp.Body)
resp.Body.Close()
```

While the body **is** closed on line 57, it's not using `defer`. If `io.ReadAll` were to panic (unlikely but defensive), the body would leak. More importantly, this breaks the idiomatic Go pattern that `defer resp.Body.Close()` should follow immediately after checking the error from `Do()`.

**Severity**: MINOR — code hygiene / resource safety
**Verdict**: APPROVE

**Recommendation**: Use `defer resp.Body.Close()` immediately after the error check for `client.Do`.

---

### F5 — `accountAddCmd` has no test coverage for concurrent use (MINOR)

**File**: `internal/cmd/account.go:25-98`

The `account add` command performs a non-atomic two-step operation: first saves config, then stores the API key. If `StoreAPIKey` fails, the code rolls back the config (line 85-86). However, there's a TOCTOU window: if another process reads the config between `SaveConfig` and `StoreAPIKey`, it could see an account entry without a corresponding key.

This is acceptable for a single-user CLI tool but worth noting.

**Severity**: MINOR — theoretical TOCTOU
**Verdict**: APPROVE

---

### F6 — `accountsToQuery` is a map alias, not a copy (MINOR)

**File**: `internal/cmd/quota.go:62`

```go
accountsToQuery = cfg.Accounts
```

When no account filter is specified, `accountsToQuery` is a reference to the same map as `cfg.Accounts`, not a copy. Any mutation of `cfg.Accounts` afterward would also modify `accountsToQuery`. In the current code, this doesn't happen, but it's fragile.

**Severity**: MINOR — maintainability hazard
**Verdict**: APPROVE

**Recommendation**: Use a copy: `for k, v := range cfg.Accounts { accountsToQuery[k] = v }`.

---

### F7 — `encrypted.go` delimiter `\x00` in account data (MINOR)

**File**: `internal/auth/encrypted.go:195-201, 253-258, 293-300`

The encrypted store uses `\x00` as a field delimiter between account name and API key. If an account name or API key contains a null byte, parsing would break. While null bytes in API keys are extremely unlikely, the code doesn't validate against this.

**Severity**: NIT — extremely unlikely edge case
**Verdict**: APPROVE

---

### F8 — `doctor.go:50` — HTTP response body close not deferred (NIT)

**File**: `internal/cmd/doctor.go:45-51`

```go
resp, err := client.Get("https://opencode.ai/zen/go/v1/usage")
if err != nil {
    checks = append(checks, ...)
} else {
    resp.Body.Close()
    checks = append(checks, ...)
}
```

Not using `defer` is fine here since nothing follows, but it's inconsistent with the pattern used in `validator.go:45`.

**Severity**: NIT
**Verdict**: APPROVE

---

### F9 — `import` command silently skips validation failures (MINOR)

**File**: `internal/cmd/account.go:333-381`

The `account import` command validates each API key before importing. If validation fails, it prints a message and continues. The final summary shows `imported` vs `skipped` counts but doesn't distinguish between "key invalid", "already exists", or "config save failed". This could confuse users trying to debug a partial import.

**Severity**: MINOR — UX
**Verdict**: APPROVE (with recommendation)

**Recommendation**: Consider printing more detailed skip reasons or a categorized summary.

---

### F10 — `aliasExists` does substring match, not exact match (MINOR)

**File**: `internal/cmd/alias.go:99-115`

```go
if strings.Contains(line, "alias "+alias+"=") {
    return true
}
```

This matches any line containing `alias ou=` as a substring. A line like `# alias out='something'` would also match since it contains `alias ou`. This could lead to false positives.

**Severity**: MINOR — edge case false positive
**Verdict**: REQUEST CHANGES

**Recommendation**: Use a more precise match, e.g., check that the line starts with `alias ou=` or `alias ou='` or `alias ou="`.

---

### F11 — `formatResetTime` off-by-one for exact days (NIT)

**File**: `internal/tui/quota.go:253-273`

```go
days := int(duration.Hours() / 24)
hours := int(duration.Hours()) % 24
```

When duration is exactly 24 hours, `days = 1`, `hours = 0`, output: `1d0h`. This is arguably correct but could be simplified to `1d` when hours are 0.

**Severity**: NIT — cosmetic
**Verdict**: APPROVE

---

### F12 — `extractKeyID` returns full key if ≤ 6 chars (NIT)

**File**: `internal/auth/credential.go:83-87`

```go
func ExtractKeyID(apiKey string) string {
    if len(apiKey) > 6 {
        return apiKey[len(apiKey)-6:]
    }
    return apiKey
}
```

For very short keys (≤6 chars), the full key is returned, which could be a partial exposure. The test confirms this behavior is intentional.

**Severity**: NIT — security hygiene (full short key displayed in `account list`)
**Verdict**: APPROVE

---

## Summary

| Severity | Count | Action Required |
|----------|-------|----------------|
| BLOCKER | 0 | — |
| MAJOR | 1 | F1 — fix race on `cachedMasterPassword` |
| MINOR | 6 | F2, F3, F5, F6, F9, F10 |
| NIT | 5 | F4, F7, F8, F11, F12 |

### Overall Verdict: **REQUEST CHANGES** (1 MAJOR)

The codebase is well-structured and tests pass cleanly. The primary concern is the race condition in `encrypted.go` (F1), which should be addressed before the next release. The remaining MINOR findings are maintainability and UX improvements that can be addressed incrementally.
