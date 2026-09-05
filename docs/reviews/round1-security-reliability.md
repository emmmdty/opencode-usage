# Security & Reliability Review — Round 1

**Reviewer:** F (Security / Reliability / Cross-platform)
**Date:** 2026-08-26
**Scope:** Full codebase audit for security vulnerabilities, reliability issues, and cross-platform concerns.

---

## Findings

### [SEC-001] Hardcoded Default Encryption Password

- **Severity**: MAJOR
- **Category**: Security
- **File(s)**: `internal/auth/encrypted.go:27`
- **Description**: The default master password `"token-usage-default"` is hardcoded in the source code. When `useMasterPassword` is nil or `false`, this static password is used for all encryption/decryption operations, making the encrypted secrets file trivially decryptable by anyone with access to the source code or binary.
- **Impact**: If a user does not explicitly enable master password mode, their encrypted secrets file (`secrets.enc`) is protected only by a publicly known password. An attacker with file access can decrypt all stored API keys.
- **Suggested Fix**: Remove the default password fallback entirely. If no master password is set, either: (1) require the user to set one before storing secrets, or (2) generate a random machine-specific key (e.g., from a combination of hostname + username + a random salt stored separately). At minimum, document the risk prominently.

### [SEC-002] No Atomic File Writes for Secrets

- **Severity**: MAJOR
- **Category**: Security / Reliability
- **File(s)**: `internal/auth/encrypted.go:213,306`, `internal/config/config.go:87`
- **Description**: Secrets and config files are written directly using `os.WriteFile()`. If the process crashes or the system loses power mid-write, the file can be left in a corrupted state. For secrets, this means potential data loss of all stored API keys.
- **Impact**: Data loss on crash/power failure. Partial writes could leave an unrecoverable file.
- **Suggested Fix**: Use atomic writes: write to a temp file in the same directory, then `os.Rename()` to the target path. `Rename` is atomic on POSIX filesystems and NTFS.

### [SEC-003] No File Permission Check Before Reading Secrets

- **Severity**: MINOR
- **Category**: Security
- **File(s)**: `internal/auth/encrypted.go:181,222,262`
- **Description**: When reading the encrypted secrets file, the code does not verify that the file permissions are restrictive (e.g., `0600`). A user or misconfigured system could have world-readable permissions on `secrets.enc`, exposing encrypted data. The code should either check permissions and warn, or reset them on read.
- **Impact**: Encrypted secrets could be world-readable on misconfigured systems.
- **Suggested Fix**: After reading the file, check `os.Stat()` and if permissions are not `0600`, either `os.Chmod(path, 0600)` to fix it, or emit a warning.

### [SEC-004] Account Names Used as Keyring Keys Without Namespacing

- **Severity**: MINOR
- **Category**: Security
- **File(s)**: `internal/auth/credential.go:57-59,67-68,77-78`
- **Description**: The `ring.Set()`, `ring.Get()`, and `ring.Remove()` calls use the raw `account` name as the keyring item key. If a user picks a name like `"__token_usage_test__"` (the test key used in `init()`), it could collide with internal keys.
- **Impact**: Account name collisions with internal test keys could cause data corruption or unexpected behavior.
- **Suggested Fix**: Prefix all keyring keys with a namespace, e.g., `"account:" + account`.

### [SEC-005] API Key Logged in Error Context via Doctor Command

- **Severity**: MINOR
- **Category**: Security
- **File(s)**: `internal/cmd/doctor.go:46`
- **Description**: The `doctor` command makes an unauthenticated GET request to `https://opencode.ai/zen/go/v1/usage` to check connectivity. While it doesn't send an API key, it accesses the `/usage` endpoint without auth, which may log a 401 on the server side. This is not a direct leak, but it's unnecessary.
- **Impact**: Low — server-side log noise. Not a direct security issue.
- **Suggested Fix**: Use a health-check endpoint or just verify DNS/TLS connectivity to the host.

### [SEC-006] ExtractKeyID Exposes Last 6 Characters

- **Severity**: NIT
- **Category**: Security
- **File(s)**: `internal/auth/credential.go:83-88`, `internal/cmd/account.go:153`
- **Description**: The `ExtractKeyID` function returns the last 6 characters of the API key. The `account list` command displays `"sk-..." + account.KeyID`, effectively showing `"sk-" + last_6_chars`. For keys with low entropy in the suffix, this could be a partial information leak. However, 6 hex characters = 24 bits of entropy, which is borderline.
- **Impact**: Minor — last 6 chars of key are displayed in `account list` output and stored in config YAML. The full key is never displayed.
- **Suggested Fix**: Consider hashing the key ID (e.g., first 8 chars of SHA-256) rather than exposing a suffix. Or document that the key suffix is intentionally visible for identification.

### [SEC-007] Encrypted File Stores Secrets in Colon-Delimited Plaintext (Inside Encryption)

- **Severity**: NIT
- **Category**: Security
- **File(s)**: `internal/auth/encrypted.go:187-192,203-205`
- **Description**: The internal format of the encrypted file is `account:apikey\n` lines. If the master password is weak (or the default from SEC-001), the colon-delimited format is trivially parseable. Even with encryption, a structured format with a magic header could help detect corruption.
- **Impact**: Low — encryption protects this at rest when the password is strong.
- **Suggested Fix**: Consider using a structured format (JSON/TOML) inside the encryption envelope for robustness, and add a magic header for corruption detection.

### [SEC-008] No Retry-After Header Respect in Client

- **Severity**: MINOR
- **Category**: Reliability
- **File(s)**: `internal/client/opencode.go:66-69`
- **Description**: When receiving HTTP 429 (Too Many Requests), the client retries with exponential backoff but ignores the `Retry-After` header from the server. The server may specify a longer wait time than the client's backoff schedule.
- **Impact**: Could lead to continued rate-limit violations and extended lockout.
- **Suggested Fix**: Parse the `Retry-After` header (which can be seconds or an HTTP-date) and use `max(serverRetryAfter, clientBackoff)` as the wait duration.

### [SEC-009] Retry Amplification on Network Errors

- **Severity**: MINOR
- **Category**: Reliability
- **File(s)**: `internal/client/opencode.go:48-53`
- **Description**: Network errors (DNS failure, connection refused, TLS handshake failure) are retried up to 3 times. These errors are typically not transient and retrying adds latency without benefit. Additionally, the error is only returned after all retries, increasing total wait time by up to 7 seconds (1+2+4).
- **Impact**: User-facing latency on persistent network failures.
- **Suggested Fix**: Only retry on timeout and connection-reset errors. For DNS and TLS errors, fail immediately.

### [SEC-010] Non-Atomic Config Migration

- **Severity**: MINOR
- **Category**: Reliability
- **File(s)**: `internal/config/config.go:65-69`
- **Description**: When the config version doesn't match, `migrateConfig()` creates a new config and saves it. If the save fails, the original file is unchanged but the in-memory `cfg` is the migrated version. Subsequent operations use the migrated config, but the file still has the old version, leading to repeated migration attempts.
- **Impact**: Repeated migration attempts on every run if save fails.
- **Suggested Fix**: Return an error if migration save fails, rather than proceeding with the migrated in-memory config.

### [SEC-011] Doctor Command Makes Unauthenticated Request Without User Consent

- **Severity**: NIT
- **Category**: Security
- **File(s**: `internal/cmd/doctor.go:46`
- **Description**: The `doctor` command automatically makes an HTTP request to `https://opencode.ai/zen/go/v1/usage` without any API key. While this is just a connectivity check, it sends a request to a third-party server. In some environments (corporate proxies, restricted networks), this could be unexpected.
- **Impact**: Low — privacy concern in restricted environments.
- **Suggested Fix**: Make the network check opt-in with a flag, or use a lighter-weight check (DNS resolution + TLS handshake without full HTTP request).

### [XP-001] Hardcoded Unix Path Separators in Config/Auth Paths

- **Severity**: MAJOR
- **Category**: Cross-platform
- **File(s)**: `internal/cmd/root.go:95`, `internal/cmd/current.go:26`, `internal/cmd/quota.go:154`, `internal/cmd/doctor.go:54`
- **Description**: Path construction uses string concatenation with `/` separators: `homeDir + "/.config/token-usage/config.yaml"` and `homeDir + "/.local/share/opencode/auth.json"`. On Windows, `os.UserHomeDir()` returns `C:\Users\<name>`, and the resulting path `C:\Users\<name>/.config/...` contains mixed separators. While Go's `os` package handles mixed separators on Windows, `filepath.Join` is the idiomatic and safe approach.
- **Impact**: Potential path resolution issues on Windows. The paths also assume XDG-like layout which doesn't exist on Windows.
- **Suggested Fix**: Use `filepath.Join()` for all path construction. For Windows, consider using `%APPDATA%` or `%USERPROFILE%` for config paths.

### [XP-002] Alias Command Assumes Unix Shell

- **Severity**: MAJOR
- **Category**: Cross-platform
- **File(s)**: `internal/cmd/alias.go:21-32,70-82`
- **Description**: The `alias install` command assumes a Unix shell (zsh/bash) and writes to `.zshrc` or `.bashrc`. On Windows, this is meaningless. The command also uses `os.Getenv("SHELL")` which is not set on Windows.
- **Impact**: `alias install` will always write to `.bashrc` on Windows (even if it exists under WSL), which is confusing and potentially harmful.
- **Suggested Fix**: Detect the platform (runtime.GOOS) and either: (1) skip alias installation on Windows with a helpful message, or (2) support PowerShell profile installation on Windows.

### [XP-003] `.local/share` Path Assumption on Windows

- **Severity**: MAJOR
- **Category**: Cross-platform
- **File(s)**: `internal/cmd/current.go:26`, `internal/cmd/quota.go:154`, `internal/cmd/doctor.go:54`
- **Description**: The code reads `~/.local/share/opencode/auth.json`, which is an XDG Base Directory path. This path does not exist on Windows (where `%APPDATA%` or `%LOCALAPPDATA%` would be used). The `resolveCurrentAccount` function will silently fail on Windows, and `current` command will always show "No opencode configuration found."
- **Impact**: The `current` and `quota` commands cannot detect the active opencode account on Windows.
- **Suggested Fix**: Use platform-appropriate paths. On Windows, check `%APPDATA%\opencode\auth.json` or similar.

### [XP-004] Doctor `isTerminal()` Uses `os.Stdout.Stat()` Instead of `term.IsTerminal()`

- **Severity**: MINOR
- **Category**: Cross-platform
- **File(s)**: `internal/cmd/doctor.go:106-112`
- **Description**: The `isTerminal()` function checks `os.ModeCharDevice` on stdout, while the rest of the codebase uses `term.IsTerminal()`. These can behave differently. `term.IsTerminal()` is the more reliable cross-platform check.
- **Impact**: Inconsistent terminal detection; `doctor` may show color codes when piped on some platforms.
- **Suggested Fix**: Use `term.IsTerminal(int(os.Stdout.Fd()))` consistently, or call `tui.DisableColor()` through a shared helper.

### [REL-001] No Panic Recovery in Goroutines

- **Severity**: MINOR
- **Category**: Reliability
- **File(s)**: `internal/cmd/quota.go:96-117`
- **Description**: The concurrent quota fetching spawns goroutines without `recover()`. If a panic occurs in any goroutine (e.g., nil pointer dereference from unexpected API response), the entire process crashes.
- **Impact**: Process crash on unexpected data.
- **Suggested Fix**: Add `defer recover()` in each goroutine, or use a shared error channel to capture panics.

### [REL-002] `storeEncrypted` Does Not Validate Account Name

- **Severity**: MINOR
- **Category**: Reliability
- **File(s)**: `internal/auth/encrypted.go:162`
- **Description**: The `storeEncrypted` function does not validate the `account` parameter. If `account` contains a `:` character, the colon-delimited format breaks: `strings.SplitN(line, ":", 2)` will split incorrectly, potentially corrupting the stored data.
- **Impact**: Data corruption if account names contain `:`.
- **Suggested Fix**: Validate or reject account names containing `:`, `\n`, or other delimiters.

### [REL-003] `deleteEncrypted` Non-Atomic Read-Modify-Write

- **Severity**: MINOR
- **Category**: Reliability
- **File(s)**: `internal/auth/encrypted.go:256-306`
- **Description**: `deleteEncrypted` reads the file, decrypts, removes the entry, re-encrypts, and writes back. This read-modify-write cycle is not atomic. If two instances run concurrently (e.g., parallel `account remove` commands), one deletion can overwrite the other.
- **Impact**: Race condition: concurrent deletes can lose data.
- **Suggested Fix**: Use file locking (`syscall.Flock`) for the encrypted file, or accept the limitation and document it.

### [REL-004] `cachedMasterPassword` Stored in Global Variable

- **Severity**: MINOR
- **Category**: Security
- **File(s)**: `internal/auth/encrypted.go:21`
- **Description**: The master password is cached in a global `string` variable (`cachedMasterPassword`). Go strings are immutable but not zeroed after use. The password remains in process memory for the lifetime of the process and is not scrubbed.
- **Impact**: Memory-scraping tools could extract the password from process memory.
- **Suggested Fix**: Use `[]byte` instead of `string` for the cached password, and zero the slice when no longer needed. Note: Go's garbage collector may still leave copies, but this reduces the window.

### [REL-005] Doctor Command Uses Different HTTP Client Than Main Client

- **Severity**: NIT
- **Category**: Reliability
- **File(s)**: `internal/cmd/doctor.go:45`
- **Description**: The `doctor` command creates its own `http.Client` with a 5-second timeout, separate from the 10-second timeout used in `client.NewClient`. This inconsistency could lead to different behavior for connectivity checks vs. actual API calls.
- **Impact**: Low — diagnostic command may timeout before real usage does.
- **Suggested Fix**: Reuse the same client configuration or make the timeout configurable.

---

## Cross-Platform Build Results

All three platform builds succeeded:

| Platform | Binary | Status |
|----------|--------|--------|
| Linux amd64 | `/tmp/tu-linux` | ✅ |
| Windows amd64 | `/tmp/tu-windows.exe` | ✅ |
| macOS arm64 | `/tmp/tu-darwin` | ✅ |

---

## Summary

| ID | Severity | Category | Summary |
|----|----------|----------|---------|
| SEC-001 | MAJOR | Security | Hardcoded default encryption password |
| SEC-002 | MAJOR | Security/Reliability | Non-atomic file writes for secrets |
| SEC-003 | MINOR | Security | No file permission check on secrets read |
| SEC-004 | MINOR | Security | Account names as keyring keys without namespace |
| SEC-005 | MINOR | Security | Unauthenticated request in doctor command |
| SEC-006 | NIT | Security | Key ID suffix exposed in output |
| SEC-007 | NIT | Security | Colon-delimited format inside encryption |
| SEC-008 | MINOR | Reliability | Retry-After header ignored |
| SEC-009 | MINOR | Reliability | Retry amplification on non-transient errors |
| SEC-010 | MINOR | Reliability | Non-atomic config migration |
| SEC-011 | NIT | Security | Doctor makes unsolicited HTTP request |
| XP-001 | MAJOR | Cross-platform | Hardcoded Unix path separators |
| XP-002 | MAJOR | Cross-platform | Alias command assumes Unix shell |
| XP-003 | MAJOR | Cross-platform | `.local/share` path doesn't exist on Windows |
| XP-004 | MINOR | Cross-platform | Inconsistent terminal detection |
| REL-001 | MINOR | Reliability | No panic recovery in goroutines |
| REL-002 | MINOR | Reliability | Account name not validated for delimiter |
| REL-003 | MINOR | Reliability | Non-atomic read-modify-write for deletes |
| REL-004 | MINOR | Security | Master password cached in immutable string |
| REL-005 | NIT | Reliability | Inconsistent HTTP client timeouts |

**Severity counts:** BLOCKER: 0 | MAJOR: 5 | MINOR: 10 | NIT: 5

---

## Verdict

**REQUEST CHANGES**

The 5 MAJOR findings (hardcoded default password, non-atomic writes, path construction, alias assumptions, Windows path support) should be addressed before release. The MAJOR cross-platform issues (XP-001, XP-002, XP-003) will cause broken behavior on Windows. SEC-001 (hardcoded default password) is the most critical security issue and should be prioritized.

The MINOR items are worth addressing but not blockers. The NITs are suggestions for improvement.
