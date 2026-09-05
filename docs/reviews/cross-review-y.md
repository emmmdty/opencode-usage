# Cross Review Y — Missing-Issue Hunter

**Reviewer**: Cross Reviewer Y — Missing-Issue Hunter  
**Date**: 2026-08-26  
**Scope**: What all 6 Round 1 reviewers (A–F) + Cross Reviewer X collectively missed  
**Method**: Read all review files, all source code, ran build/vet/test/mod-tidy, verified `go install`, checked dependency hygiene, tested edge cases

---

## Findings

### [CY-1] Unused direct dependencies `bubbles` and `bubbletea` bloat go.mod
- **Severity**: MAJOR
- **Category**: Dependencies / Release Packaging
- **Description**: `go.mod` lists `charmbracelet/bubbles v1.0.0` and `charmbracelet/bubbletea v1.3.10` as direct `require` dependencies. Neither package is imported by any `.go` file in the project. They appear to be remnants from an earlier TUI design that was never implemented (the project uses lipgloss for rendering, not bubbletea). These unused dependencies pull in ~12 transitive dependencies (`atotto/clipboard`, `erikgeiser/coninput`, `mattn/go-localereader`, `muesli/ansi`, `muesli/cancelreader`, `charmbracelet/x/cellbuf`, etc.) that are never used.
- **Evidence**:
  - `go mod why github.com/charmbracelet/bubbles` → `(main module does not need package ...)`
  - `go mod why github.com/charmbracelet/bubbletea` → same
  - `grep -rn 'bubbles\|bubbletea' --include='*.go' .` → no results
  - `go mod tidy` removes both from go.mod and drops 6 indirect dependencies
- **Impact**: (1) Binary bloat — unused dependencies inflate the module graph. (2) Supply-chain attack surface — more transitive deps = more risk. (3) `go mod tidy` output differs from committed go.mod, meaning CI may fail on `go mod verify` or `go mod tidy` checks. (4) Confuses contributors who wonder why bubbletea is listed but unused.
- **Suggested Fix**: Run `go mod tidy` and commit the result. If bubbletea/bubbles are planned for future use, add a comment in go.mod explaining why.

---

### [CY-2] `go install` is completely broken — both README path and go.mod path fail
- **Severity**: BLOCKER
- **Category**: Installation / Documentation
- **Description**: There is no working `go install` path for this project. The README instructs users to run `go install github.com/emmmdty/token-usage/cmd/token-usage@latest`, but:
  1. The `go.mod` declares `module github.com/token-usage` (no `emmmdty` org prefix)
  2. `go install github.com/emmmdty/token-usage/...` fails with: `module declares its path as: github.com/token-usage but was required as: github.com/emmmdty/token-usage`
  3. `go install github.com/token-usage/cmd/token-usage@latest` fails with: `Repository not found` (no such GitHub repo exists at that path)
- **Evidence**: Both commands executed and confirmed failing:
  ```
  $ go install github.com/emmmdty/token-usage/cmd/token-usage@latest
  go: github.com/emmmdty/token-usage@v0.1.0: version constraints conflict:
      module declares its path as: github.com/token-usage
              but was required as: github.com/emmmdty/token-usage

  $ go install github.com/token-usage/cmd/token-usage@latest
  go: github.com/token-usage/cmd/token-usage@latest: module not found:
      remote: Repository not found.
  ```
- **Impact**: The primary installation method documented in the README does not work. Users who follow the instructions get a confusing error. The only working installation methods are "download binary" and "build from source" (which requires cloning).
- **Suggested Fix**: Either (a) change `go.mod` to `module github.com/emmmdty/token-usage` to match the actual GitHub repo, or (b) update the README to use the correct `go install` path if the module path is correct and the repo needs to be published at that path. Option (a) is simplest.

---

### [CY-3] No LICENSE file exists despite README claiming MIT
- **Severity**: MAJOR
- **Category**: Legal / License Compliance
- **Description**: The README states "MIT" under a `## License` section (line 142-144), but no `LICENSE`, `LICENSE.md`, `LICENSE.txt`, or `COPYING` file exists in the repository. An MIT license requires the full license text to be included. Without the license file, the code is technically "all rights reserved" — anyone using, modifying, or distributing the code has no explicit permission.
- **Evidence**: `ls LICENSE* license* COPYING*` → no matches. `find . -name 'LICENSE*'` → no results. The go.sum references MIT licenses for dependencies but the project itself has no license file.
- **Impact**: (1) Legal risk — downstream users and contributors have no license grant. (2) Go module proxy and pkg.go.dev will not display a license, which may deter adoption. (3) Corporate users may not be allowed to use code without an explicit license file. (4) The `go install` flow (once fixed) would install an unlicensed binary.
- **Suggested Fix**: Add a `LICENSE` file containing the full MIT license text with the correct copyright holder.

---

### [CY-4] `go.mod` is not tidy — committed state diverges from `go mod tidy` output
- **Severity**: MINOR
- **Category**: Dependencies / Build Hygiene
- **Description**: Running `go mod tidy` produces changes to 9 files (go.mod, go.sum, and 7 source files are shown in diff). The committed `go.mod` has unused direct dependencies (bubbles, bubbletea) and transitive dependencies that `go mod tidy` would remove. This means: (1) any CI check running `go mod tidy --diff` would fail, (2) contributors running `go mod tidy` get a dirty working tree, (3) the dependency list is stale.
- **Evidence**: `go mod tidy && git diff --stat` shows:
  ```
   go.mod                  |  12 +-
   go.sum                  |  21 +--
   internal/cmd/account.go | 202 +++++++++++++-------------
   ... (7 files changed, 640 insertions, 309 deletions)
  ```
- **Impact**: Build hygiene issue. The source file diffs suggest the code was modified after go.mod was last tidied. `go mod verify` passes (all checksums match), but the module graph is not minimal.
- **Suggested Fix**: Run `go mod tidy` and commit the result. Establish a CI check that verifies `go mod tidy` produces no diff.

---

### [CY-5] `service` parameter is dead code in all auth credential functions
- **Severity**: MINOR
- **Category**: Code Quality / API Design
- **Description**: `auth.StoreAPIKey(service, account, apiKey)`, `auth.GetAPIKey(service, account)`, and `auth.DeleteAPIKey(service, account)` all accept a `service` string parameter that is never used inside the function body. The keyring is opened with `ServiceName: "token-usage"` in `init()` (credential.go:20), which is the only place the service name matters. Every caller passes `"token-usage"` as the service argument, making the parameter pure dead code.
- **Evidence**:
  - `credential.go:55-62`: `StoreAPIKey` body uses `ring.Set(keyring.Item{Key: account, ...})` — no reference to `service`
  - `credential.go:65-73`: `GetAPIKey` body uses `ring.Get(account)` — no reference to `service`
  - `credential.go:76-80`: `DeleteAPIKey` body uses `ring.Remove(account)` — no reference to `service`
  - All callers: `auth.StoreAPIKey("token-usage", name, apiKey)`, `auth.GetAPIKey("token-usage", name)`, etc.
- **Impact**: Misleading API surface. A future developer might think different service names provide isolation, but they don't. The parameter creates a false sense of configurability.
- **Suggested Fix**: Remove the `service` parameter from the function signatures, or actually use it (e.g., prefix keyring keys with `service + ":"`).

---

### [CY-6] `doctor` command ignores `--no-color` CLI flag — uses raw ANSI regardless
- **Severity**: MAJOR
- **Category**: UX / CLI Flags
- **Description**: The `doctor` command creates its own theme via `newDoctorTheme()` (doctor.go:99-104) which only checks `os.Getenv("NO_COLOR")` and `isTerminal()`. It does NOT check the `noColor` global variable set by the `--no-color` CLI flag. When a user runs `token-usage --no-color doctor`, the PersistentPreRun in root.go disables lipgloss color, but doctor's own ANSI escape codes (`\033[32m`, `\033[33m`, `\033[31m`) are emitted unconditionally.
- **Evidence**:
  - `root.go:30-33`: PersistentPreRun checks `noColor` flag and calls `tui.DisableColor()`
  - `doctor.go:99-104`: `newDoctorTheme()` checks `os.Getenv("NO_COLOR")` (env var only, not the flag) and `isTerminal()`
  - `doctor.go:103`: Returns raw ANSI `\033[32m✓\033[0m` etc.
  - The `noColor` variable (root.go:21) is package-level but doctor.go never reads it
- **Impact**: Users who set `--no-color` (e.g., for scripting, CI, or accessibility) get color output in doctor. This breaks the expected contract that `--no-color` disables all color. Also affects piping — `isTerminal()` returns false when piped, but the `--no-color` flag in a terminal still shows colors.
- **Suggested Fix**: Pass the `noColor` state to `newDoctorTheme()`, or check `noColor` in addition to `NO_COLOR` env var. Better yet, use the `tui.Theme` system instead of raw ANSI.

---

### [CY-7] Compact mode (narrow terminals) missing summary line and active-account footer
- **Severity**: MAJOR
- **Category**: TUI / UX
- **Description**: When the terminal is narrower than 60 columns, `FormatQuotaOverview` switches to `formatCompact()` (quota.go:46). The compact mode omits two pieces of information that the table mode always shows:
  1. **Summary line**: Table mode shows `"3 accounts  2 healthy  1 warning"` (line 93-94). Compact mode has no equivalent — users on narrow terminals cannot see the health overview at a glance.
  2. **Active account footer**: Table mode shows `"Active: work"` (line 96-101). Compact mode does not indicate which account is the current/active one. The only clue is the `->` marker on the active row, but this is easy to miss in the compact layout.
- **Evidence**:
  - `formatTable` (quota.go:92-101): includes `computeSummary` + active account footer
  - `formatCompact` (quota.go:121-155): includes "Best:" and "Reset:" but NOT summary or active
- **Impact**: Narrow-terminal users lose important context. The health summary is the key insight ("how many accounts are OK vs at risk"), and the active account is critical for multi-account users. This is an information regression triggered by terminal width.
- **Suggested Fix**: Add `computeSummary` output and active-account line to `formatCompact`.

---

### [CY-8] `findNextReset` only considers Rolling resets, ignoring Weekly and Monthly
- **Severity**: MINOR
- **Category**: TUI / Accuracy
- **Description**: The `findNextReset` function (quota.go:344-360) iterates all accounts and finds the earliest `Rolling.ResetsAt` time. It completely ignores `Weekly.ResetsAt` and `Monthly.ResetsAt`. If an account's Rolling window resets in 7 hours but its Weekly window resets in 30 minutes, the user sees `"Next reset: work · 7h0m"` instead of the more urgent `"Next reset: work · 30m"`.
- **Evidence**:
  - `quota.go:351`: `if earliest.IsZero() || r.Usage.Rolling.ResetsAt.Before(earliest)` — only checks Rolling
  - `quota.go:352`: `earliest = r.Usage.Rolling.ResetsAt` — only stores Rolling time
  - The `models.QuotaWindow` struct has `ResetsAt` on all three windows (Rolling, Weekly, Monthly)
- **Impact**: The "Next reset" line is misleading when a Weekly or Monthly reset is sooner than the Rolling reset. Users may not realize their weekly quota is about to refresh.
- **Suggested Fix**: Consider all three `ResetsAt` fields when computing the next reset, or show the next reset per window type (e.g., "Next 5H reset: work · 30m").

---

### [CY-9] `account import` partial failure orphans multiple API keys without cleanup
- **Severity**: MAJOR
- **Category**: Reliability / Data Integrity
- **Description**: The `account import` command (account.go:296-378) loops through accounts and for each one: (1) validates via HTTP, (2) stores the key in the keyring/encrypted file, (3) adds to in-memory config. After the loop, it saves the config (line 372). If the config save fails after some keys were successfully stored, those keys become orphaned — they exist in the keyring but not in the config. Unlike CC-02 (single account add), import can orphan MULTIPLE keys in one operation. Additionally, if the import is interrupted (Ctrl+C) mid-loop, already-stored keys have no config entries and no cleanup mechanism.
- **Evidence**:
  - `account.go:356`: `auth.StoreAPIKey("token-usage", account.Name, account.APIKey)` — stores key
  - `account.go:362-367`: Adds to in-memory config
  - `account.go:372`: `config.SaveConfig(cfg, configPath)` — saves after loop
  - Between lines 356 and 372, multiple keys can be stored with no rollback if save fails
- **Impact**: Importing 10 accounts where the 8th key succeeds but config save fails → 7 orphaned keys in the keyring. These keys consume space and could cause "already exists" errors on re-import (though the code checks `cfg.Accounts[name]` not the keyring). The encrypted file grows with each orphaned entry and has no garbage collection.
- **Suggested Fix**: (1) Save config incrementally after each successful key store, or (2) use a transaction-like pattern where keys are stored in a staging area until config save succeeds, or (3) add a cleanup function that removes keyring entries not referenced by config.

---

### [CY-10] Auth JSON path (`~/.local/share/opencode/auth.json`) duplicated in 3 files without a shared constant
- **Severity**: MINOR
- **Category**: Code Quality / DRY
- **Description**: The path `~/.local/share/opencode/auth.json` is hardcoded in three separate files using string concatenation:
  1. `internal/cmd/quota.go:154`: `authPath := homeDir + "/.local/share/opencode/auth.json"`
  2. `internal/cmd/current.go:26`: `authPath := homeDir + "/.local/share/opencode/auth.json"`
  3. `internal/cmd/doctor.go:54`: `os.Stat(homeDir() + "/.local/share/opencode/auth.json")`
  
  Each uses a slightly different pattern (variable assignment vs inline expression), and none uses `filepath.Join`. If opencode changes this path, three files must be updated.
- **Evidence**: Three occurrences confirmed via grep. ARC-01 covers the base URL duplication but not this specific path.
- **Impact**: Maintenance burden. If the path changes, missing one location causes silent failure (quota and current commands fail to detect the active account, doctor shows wrong status).
- **Suggested Fix**: Define a shared constant like `const OpencodeAuthPath = ".local/share/opencode/auth.json"` in a central location (e.g., `config` or `auth` package) and use `filepath.Join(homeDir, OpencodeAuthPath)`.

---

### [CY-11] `computeSummary` counts both error accounts and high-usage accounts as "critical" — no distinction
- **Severity**: NIT
- **Category**: TUI / UX
- **Description**: `computeSummary` (quota.go:281-319) uses a single `errors` counter for two very different conditions: (1) accounts where the API call failed (line 288-290: `if r.Error != "" { errors++ }`), and (2) accounts where usage exceeds 80% (line 298-299: `if maxPercent >= 80 { errors++ }`). The output says `"1 critical"` for both cases. A user seeing "1 critical" doesn't know if their account is broken (API error) or just heavily used (80% quota). The visual rendering distinguishes them (error rows show `✗` prefix, high-usage rows show red bars), but the summary merges them.
- **Evidence**: `quota.go:288-290` increments `errors` for error accounts. `quota.go:298-299` increments `errors` for high-percent accounts. `quota.go:315`: `fmt.Sprintf("%d critical", errors)`.
- **Impact**: Ambiguous summary. Users cannot quickly determine if "critical" means a problem or just high usage.
- **Suggested Fix**: Split into separate counters: `errorCount` for API failures and `criticalCount` for high usage. Output: `"1 error  1 critical"` or `"2 accounts  1 healthy  1 error"`.

---

### [CY-12] `version` command output includes placeholder values (`commit: none, built: unknown`) in release builds
- **Severity**: MINOR
- **Category**: Release Packaging
- **Description**: `internal/version/version.go:6-8` defines `Version = "0.1.0"`, `Commit = "none"`, `Date = "unknown"`. These are set at compile time via `-ldflags` (standard Go practice), but the `.goreleaser.yml` does not configure ldflags to inject version/commit/date. The goreleaser config (line 3-5) only specifies `binary` and `main` — no `ldflags` section. This means release binaries will always report `commit: none, built: unknown`.
- **Evidence**:
  - `version.go:6-8`: `Version = "0.1.0"`, `Commit = "none"`, `Date = "unknown"` — default values
  - `.goreleaser.yml:3-5`: `builds: - binary: token-usage` — no ldflags configured
  - Running `./token-usage version` outputs: `token-usage 0.1.0 (commit: none, built: unknown)`
- **Impact**: Users cannot verify which commit a binary was built from. This makes bug reports harder to triage and release verification impossible.
- **Suggested Fix**: Add ldflags to `.goreleaser.yml`:
  ```yaml
  builds:
    - ldflags:
        - -s -w
        - -X github.com/token-usage/internal/version.Version={{.Version}}
        - -X github.com/token-usage/internal/version.Commit={{.ShortCommit}}
        - -X github.com/token-usage/internal/version.Date={{.Date}}
  ```

---

### [CY-13] `account add` success message says "Run 'tu' to view" but alias may not be installed
- **Severity**: NIT
- **Category**: UX / Copy
- **Description**: After successfully adding an account, `accountAddCmd` prints `"Run 'tu' to view all accounts."` (account.go:93). However, `tu` is a shell alias that must be explicitly installed via `token-usage alias install`. If the user hasn't installed the alias, running `tu` gives "command not found". The message should reference the full binary name `token-usage` (or `token-usage quota`).
- **Evidence**: `account.go:93`: `fmt.Println("Run 'tu' to view all accounts.")`
- **Impact**: Minor UX confusion. New users who just ran `account add` haven't necessarily installed the alias.
- **Suggested Fix**: Change to `"Run 'token-usage' to view all accounts."` or `"Run 'token-usage quota' to view all accounts."`.

---

## Summary

| ID | Severity | Category | Description |
|----|----------|----------|-------------|
| CY-1 | MAJOR | Dependencies | Unused `bubbles`/`bubbletea` in go.mod |
| CY-2 | BLOCKER | Installation | `go install` completely broken for both module paths |
| CY-3 | MAJOR | Legal | No LICENSE file despite README claiming MIT |
| CY-4 | MINOR | Build Hygiene | go.mod not tidy — diverges from `go mod tidy` |
| CY-5 | MINOR | Code Quality | `service` parameter is dead code in auth functions |
| CY-6 | MAJOR | UX / CLI | `doctor` ignores `--no-color` flag, emits raw ANSI |
| CY-7 | MAJOR | TUI / UX | Compact mode missing summary + active-account footer |
| CY-8 | MINOR | TUI / Accuracy | `findNextReset` ignores Weekly/Monthly resets |
| CY-9 | MAJOR | Reliability | `account import` partial failure orphans multiple keys |
| CY-10 | MINOR | Code Quality | Auth JSON path duplicated in 3 files |
| CY-11 | NIT | TUI / UX | `computeSummary` merges errors + high-usage as "critical" |
| CY-12 | MINOR | Release | `version` command shows placeholder values in releases |
| CY-13 | NIT | UX / Copy | Success message references `tu` alias that may not exist |

**Severity counts**: BLOCKER: 1 | MAJOR: 5 | MINOR: 5 | NIT: 2

**Total new issues found**: 13

---

## What All Reviewers Collectively Missed

1. **Installation is completely broken** (CY-2): No reviewer actually tried `go install`. F14 noted the module path mismatch but framed it as "may fail" — it definitively fails for both paths.

2. **No LICENSE file** (CY-3): All reviewers focused on code quality, testing, and documentation accuracy. None checked for the existence of a license file despite the README claiming MIT.

3. **Unused dependencies** (CY-1): All reviewers examined the code but none ran `go mod why` or `go mod tidy` to check dependency hygiene. The bubbletea/bubbles dependencies are completely unused.

4. **Doctor bypasses `--no-color`** (CY-6): ARC-07 and XP-04 noted doctor's raw ANSI and inconsistent terminal detection, but neither tested whether the `--no-color` CLI flag affects doctor output.

5. **Compact mode information loss** (CY-7): VQ-01 through VQ-08 focused on overflow, alignment, and grammar issues in the TUI. None checked whether the compact mode (narrow terminal fallback) includes all the same information as table mode.

6. **`findNextReset` only checks Rolling** (CY-8): All reviewers accepted the "Next reset" line at face value. No one verified that it considers all three quota windows.

7. **Import orphans multiple keys** (CY-9): CC-02 and CC-03 cover single-key atomicity in add/remove. No reviewer analyzed the import loop's multi-key failure mode.

8. **go.mod is stale** (CY-4): The fact that `go mod tidy` produces significant changes was missed by everyone.
