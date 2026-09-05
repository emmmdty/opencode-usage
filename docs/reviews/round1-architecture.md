# Architecture & Maintainability Review — Round 1

**Reviewer**: B (Architecture & Maintainability)
**Date**: 2026-08-26

---

### [ARC-01] Duplicated base URL across three packages
- **Severity**: MAJOR
- **File(s)**: `internal/client/opencode.go:22`, `internal/auth/validator.go:11`, `internal/cmd/doctor.go:46`
- **Description**: The API base URL `https://opencode.ai/zen/go/v1` is hardcoded in three separate locations. `client.NewClient` defaults to it, `auth.ValidateAPIKey` defaults to it, and `doctorCmd` hardcodes the full URL for its network check.
- **Impact**: If the API endpoint changes, three files must be updated in lockstep. Miss one and the tool silently hits stale endpoints or fails.
- **Suggested Improvement**: Define the base URL in a single place (e.g., `config` or a shared `api` package). Have `client`, `auth`, and `doctor` all reference that single constant.

---

### [ARC-02] Duplicated account loading boilerplate in every command
- **Severity**: MAJOR
- **File(s)**: `internal/cmd/quota.go:42-51`, `internal/cmd/models.go:47-57`, `internal/cmd/account.go:62-71,103-112,199-208,255-261`
- **Description**: The same 6-line sequence (`getConfigPath` → `config.LoadOrCreateConfig` → `configureAuthFromConfig`) is copy-pasted into every single command handler. Five commands repeat this identical setup.
- **Impact**: Adding a new command requires copying this boilerplate. If the initialization sequence changes (e.g., adding a migration step), every copy must be found and updated.
- **Suggested Improvement**: Extract a `loadConfig() (*config.Config, error)` helper in the `cmd` package that encapsulates path resolution, loading, and auth configuration. Each command calls one function.

---

### [ARC-03] `resolveCurrentAccount` reads auth.json directly — violates package boundaries
- **Severity**: MAJOR
- **File(s)**: `internal/cmd/quota.go:149-178`, `internal/cmd/current.go:17-61`
- **Description**: The `cmd` package directly reads `~/.local/share/opencode/auth.json` and parses its JSON structure. This is opencode's internal auth format, yet the `cmd` layer is doing raw file I/O and JSON unmarshaling of another application's internals. The same logic is partially duplicated in `currentCmd`.
- **Impact**: If opencode changes its auth.json format, two cmd files break. Business logic (resolving which account is "current") lives in the wrong layer — it should be in `auth` or `client`.
- **Suggested Improvement**: Move `resolveCurrentAccount` into the `auth` package (e.g., `auth.GetCurrentOpencodeAccount() string`). The `cmd` package should not know the auth.json schema.

---

### [ARC-04] Threshold defaults (50/80) hardcoded in four places
- **Severity**: MAJOR
- **File(s)**: `internal/cmd/quota.go:205-209`, `internal/tui/quota.go:24-28`, `internal/tui/quota.go:298-301`, `internal/config/config.go:37-38`
- **Description**: The warning threshold `50` and danger threshold `80` are hardcoded in `config.getDefaultConfig`, `cmd.printQuotaTable` (as fallback), `tui.DefaultQuotaStyle`, and `tui.computeSummary` (hardcoded `80`/`50` without using any style parameter at all).
- **Impact**: `computeSummary` ignores the user's configured thresholds entirely — a user who sets danger to 90 will see incorrect "critical" counts. Other fallbacks mean config values can be silently overridden.
- **Suggested Improvement**: Define threshold defaults as exported constants in the `config` package. Have all consumers reference those constants. Fix `computeSummary` to accept and use `QuotaStyle`.

---

### [ARC-05] Scattered global mutable state in `auth` and `tui` packages
- **Severity**: MAJOR
- **File(s)**: `internal/auth/credential.go:11-14`, `internal/auth/encrypted.go:20-25`, `internal/tui/theme.go:11-13`
- **Description**: Both packages rely on package-level `var` globals initialized in `init()` and mutated at runtime: `ring`, `keyringAvailable` (auth), `cachedMasterPassword`, `passwordOnce`, `useMasterPassword` (auth), `isColorEnabled`, `terminalWidth` (tui).
- **Impact**: Global mutable state makes testing unreliable (test order matters), prevents concurrent use, and creates hidden coupling. `tui.init()` runs at import time, but `cmd/root.go` later calls `tui.DisableColor()` — two different mechanisms fighting over the same state.
- **Suggested Improvement**: Convert globals into struct fields. `auth` should return a configured `Auth` or `Keyring` struct. `tui` should return a configured `Theme` or `Renderer` struct. This eliminates init-order dependencies and enables testing.

---

### [ARC-06] Duplicate `accountResult` struct between `cmd` and `tui`
- **Severity**: MINOR
- **File(s)**: `internal/cmd/quota.go:20-25`, `internal/tui/quota.go:12-17`
- **Description**: Two nearly identical structs exist: `cmd.accountResult` (with JSON tags) and `tui.AccountResult` (without JSON tags). Both carry Name, Usage, Error, IsCurrent. `printQuotaTable` manually copies between them field-by-field.
- **Impact**: Any new field added to the account result must be added to both structs and the copy loop. Two sources of truth for the same domain concept.
- **Suggested Improvement**: Define `AccountResult` once in `models` (or a shared domain package) and have both `cmd` and `tui` use it.

---

### [ARC-07] `cmd/doctor.go` uses raw ANSI escape codes instead of theme system
- **Severity**: MINOR
- **File(s**: `internal/cmd/doctor.go:99-104`
- **Description**: `newDoctorTheme()` emits raw ANSI escape sequences (`\033[32m`, `\033[33m`, `\033[31m`) for check icons, completely bypassing the `tui.Theme` system. It also reimplements `isTerminal()` instead of using `term.IsTerminal`.
- **Impact**: Inconsistent styling: doctor output uses raw ANSI while all other output uses lipgloss. If color profiles change, doctor output won't adapt. Duplicated terminal detection logic.
- **Suggested Improvement**: Either use the `tui.Theme` for doctor icons, or extract `isTerminal()` into a shared utility.

---

### [ARC-08] `cmd/alias.go` has incomplete shell support
- **Severity**: MINOR
- **File(s)**: `internal/cmd/alias.go:28-32,77-82`
- **Description**: Shell detection only handles `zsh` and `bash`. Fish, PowerShell, and other shells silently get bash-style treatment, which is incorrect.
- **Impact**: Users on unsupported shells get a broken or non-functional alias installation. Silent failure is worse than an explicit "unsupported shell" error.
- **Suggested Improvement**: Explicitly detect and support supported shells, and return an error for unsupported ones.

---

### [ARC-09] Hardcoded `fmt.Println` output in commands — no unified output channel
- **Severity**: MINOR
- **File(s)**: `internal/cmd/current.go:29-58`, `internal/cmd/doctor.go:60-82`, `internal/cmd/account.go:115,156,224,288-291,332-376`, `internal/cmd/models.go:39-42`
- **Description**: Most commands write output directly to `os.Stdout` via `fmt.Println`/`fmt.Printf`, while the quota command uses `writeOutput()`. The `writeOutput` function respects `--output` file redirection, but direct `fmt.Println` calls bypass it entirely.
- **Impact**: `--output` flag only works for quota output. Running `token-usage account list --output foo.txt` silently ignores the flag. Inconsistent output behavior confuses users and complicates testing.
- **Suggested Improvement**: All commands should use `writeOutput()` (or a similar abstraction) for user-facing text. Direct `fmt.Println` should only be used for debug/development output.

---

### [ARC-10] Chinese strings mixed with English — no i18n strategy
- **Severity**: MINOR
- **File(s)**: `internal/auth/validator.go:42-75`, `internal/auth/encrypted.go:55-67`, `internal/config/config.go:43-70`, `internal/cmd/alias.go:23`
- **Description**: Error messages and comments alternate between Chinese and English with no consistent language. `auth/validator.go` returns Chinese messages like `"网络连接失败，请检查网络"` while `client/opencode.go` returns English `"API error: HTTP %d"`. Comments in `config.go` and `encrypted.go` are Chinese.
- **Impact**: CLI output mixes languages, confusing non-Chinese users. No i18n infrastructure means adding translations later requires finding every hardcoded string.
- **Suggested Improvement**: Standardize on English for all user-facing strings. If Chinese support is needed, add an i18n framework. Comments should also be English for broader contributor accessibility.

---

### [ARC-11] `config.migrateConfig` silently drops `UseMasterPassword`
- **Severity**: MAJOR
- **File(s)**: `internal/config/config.go:94-108`
- **Description**: `migrateConfig` creates a fresh default config and selectively copies `Accounts`, `ColorThresholds`, and `MaxConcurrentRequests` from the old config. It does not copy `UseMasterPassword`. If a user had `use_master_password: true` in their config and the version number changes, that setting is silently lost.
- **Impact**: Users lose security configuration without notification. The migration is destructive and silent.
- **Suggested Improvement**: Copy all fields from old config to new, or at minimum log a warning for fields that are not migrated.

---

### [ARC-12] No testable interfaces for `client` and `auth`
- **Severity**: MAJOR
- **File(s)**: `internal/client/opencode.go`, `internal/auth/credential.go`
- **Description**: `client.NewClient` returns a concrete `*Client` type. `auth.GetAPIKey`, `auth.StoreAPIKey` are package-level functions operating on globals. Neither exposes an interface.
- **Impact**: Commands cannot be unit tested with mock API calls or mock keyring. Tests must hit real endpoints or rely on integration test infrastructure.
- **Suggested Improvement**: Define interfaces (`APIClient`, `Keyring`) that commands accept via dependency injection. Default implementations wire to the real HTTP client and keyring.

---

### [ARC-13] `getAPIKeyForCommand` in `models.go` duplicates config-loading logic
- **Severity**: MINOR
- **File(s)**: `internal/cmd/models.go:47-74`
- **Description**: `getAPIKeyForCommand` re-implements config loading and first-account selection logic that partially overlaps with what `runQuotaOverview` does. It also duplicates the `configureAuthFromConfig` call found everywhere else.
- **Impact**: The "pick first account" logic exists in two places. If selection priority changes (e.g., prefer current account over alphabetical first), two places need updating.
- **Suggested Improvement**: Consolidate account selection logic into a shared helper. When [ARC-02] is fixed, this becomes simpler.

---

### [ARC-14] `max` function shadows Go built-in
- **Severity**: NIT
- **File(s)**: `internal/tui/quota.go:381-385`
- **Description**: A custom `max(a, b int) int` function is defined, but since Go 1.21 `max` is a built-in. The `go.mod` requires Go 1.26.6.
- **Impact**: Unnecessary code that will confuse readers who expect the built-in. Triggers linter warnings.
- **Suggested Improvement**: Delete the custom `max` function and use the built-in.

---

### [ARC-15] `doctor.go` creates its own HTTP client — bypasses `client` package
- **Severity**: MINOR
- **File(s)**: `internal/cmd/doctor.go:45-51`
- **Description**: Doctor creates a raw `http.Client` with a 5-second timeout for its connectivity check, instead of using the `client.Client` type which has retry logic and configurable timeouts.
- **Impact**: Two different HTTP clients with different timeout and retry behaviors in the same codebase. Doctor uses 5s timeout; `client.Client` uses 10s with retries. Behavioral inconsistency.
- **Suggested Improvement**: Either use the `client` package's HTTP infrastructure, or if doctor needs a simple ping, document why it's different.

---

### [ARC-16] `writeOutput` is a free function in `cmd` — unclear ownership
- **Severity**: NIT
- **File(s)**: `internal/cmd/root.go:50-56`
- **Description**: `writeOutput` reads the global `outputFile` variable to decide between file and stdout output. It's a utility function but lives as a package-level function in `cmd`, tightly coupled to the global flag state.
- **Impact**: If a subcommand wants different output behavior (e.g., binary vs text), the global variable approach doesn't scale.
- **Suggested Improvement**: Pass output destination as a parameter rather than reading from a global.

---

## Verdict

**REQUEST CHANGES**

The codebase has a solid foundation with clean package separation and a sensible project layout. However, several architectural issues will compound over time:

1. **Duplicated boilerplate** (ARC-02, ARC-01, ARC-04) means every new feature requires copying multiple blocks of code, increasing the chance of drift.
2. **Package boundary violations** (ARC-03) and **scattered globals** (ARC-05) make testing difficult and create hidden coupling.
3. **Config migration bug** (ARC-11) is a silent data loss risk that should be fixed before any version bump.
4. **Inconsistent output handling** (ARC-09) means the `--output` flag is partially broken.

The most impactful fixes are: extract a config-loading helper (ARC-02), move auth.json reading into the `auth` package (ARC-03), centralize the base URL (ARC-01), and fix the migration to preserve all fields (ARC-11). These four changes would eliminate the majority of maintenance friction.
