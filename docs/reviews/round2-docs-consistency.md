# Round 2 — Fresh Docs Consistency Audit

**Reviewer:** Reviewer I — Fresh Docs Consistency Audit  
**Date:** 2026-08-26  
**Scope:** README.md vs code behavior, help output, config, env vars, tests  
**Binary built:** `go build ./cmd/opencode-usage/` — succeeds  
**Tests:** `go test ./...` — all pass  

---

## Feature Traceability Matrix

| Feature | README Says | Help Says | Code Implements | Test Covers | Runtime Verified |
|---------|------------|-----------|-----------------|-------------|------------------|
| Root command (quota dashboard) | Default action shows quota dashboard | `RunE: runQuotaOverview(...)` | `internal/cmd/root.go:40` | `quota_test.go` (TUI formatting) | Yes — running `opencode-usage` with no args invokes quota |
| `account` | Documented with add/list/remove/export/import | `account` alias `a` | `internal/cmd/account.go` | No dedicated cmd tests | Yes |
| `account add` | Interactive prompt | `add` alias `aa` | `internal/cmd/account.go:26` | No cmd test | Yes |
| `account list` | Lists accounts | `list` alias `al` | `internal/cmd/account.go:101` | No cmd test | Yes |
| `account remove <name>` | Takes name arg | `remove` alias `ar`, `ExactArgs(1)` | `internal/cmd/account.go:194` | No cmd test | Yes |
| `account export` | Exports names+key IDs only, no secrets | `export` alias `ae` | `internal/cmd/account.go:250` | No cmd test | Yes |
| `account export -o` | `-o` flag documented | Inherits global `-o` | `internal/cmd/account.go:287` | No cmd test | Yes |
| `account import <file>` | Imports from JSON file | `import` alias `ai`, `ExactArgs(1)` | `internal/cmd/account.go:299` | No cmd test | Yes |
| `quota` | View quota for all accounts | `quota` alias `q` | `internal/cmd/quota.go:32` | TUI formatting tests | Yes |
| `quota -n <account>` | Filter by account | Inherits global `-n` | `internal/cmd/quota.go:54` | No cmd test | Yes |
| `quota --json` | JSON output | Inherits global `-j` | `internal/cmd/quota.go:141` | No cmd test | Yes |
| `quota -o <file>` | Output to file | Inherits global `-o` | `internal/cmd/quota.go:146` | No cmd test | Yes |
| `models` | List available models | `models` alias `m` | `internal/cmd/models.go:14` | No cmd test | Yes |
| `current` | Show active opencode configuration | `current` alias `cc` | `internal/cmd/current.go:17` | No cmd test | Yes |
| `doctor` | Check configuration and connectivity | `doctor` | `internal/cmd/doctor.go:14` | No cmd test | Yes |
| `alias install` | Install `ou` shell alias | `alias install` | `internal/cmd/alias.go:17` | No cmd test | Yes |
| `alias uninstall` | Uninstall `ou` shell alias | `alias uninstall` | `internal/cmd/alias.go:67` | No cmd test | Yes |
| `version` | Show version information | `version` | `internal/cmd/root.go:58` | No cmd test | Yes |
| `update` | Check for new releases | `update` | `internal/cmd/root.go:66` | No cmd test | Yes |
| `completion` | **NOT documented** | `completion` (cobra auto-gen) | Cobra built-in | No | Yes |
| `--json` / `-j` | Documented as global flag | `-j, --json` | `root.go:78` | No | Yes |
| `--account` / `-n` | Documented as global flag | `-n, --account` | `root.go:79` | No | Yes |
| `--output` / `-o` | Documented as global flag | `-o, --output` | `root.go:80` | No | Yes |
| `--no-color` | Documented as global flag | `--no-color` | `root.go:81` | No | Yes |
| `ou` shell alias | Documented in aliases table | `root.go` `Aliases: []string{"ou"}` | Yes | No | Yes |
| Short aliases (a, aa, al, ar, ae, ai, q, m, cc) | All documented in aliases table | All present in `--help` | All in code | No | Yes |

---

## Detailed Findings

### F1 — Go Version Mismatch: README vs go.mod (MAJOR)

| | Value |
|---|---|
| README says | "Requires Go 1.22+" |
| go.mod says | `go 1.26.6` |
| Impact | Users following README may try Go 1.22–1.25 and get cryptic build failures. `go 1.26.6` means Go 1.26.6 is the **minimum** version required by the module. |

**Verdict:** README must say "Requires Go 1.26+" (or whatever the actual minimum is). The current claim of "Go 1.22+" is **incorrect**.

### F2 — Undocumented `completion` Command (MINOR)

`--help` shows `completion` (Cobra's auto-generated shell completion command). The README does not mention it. This is standard Cobra behavior and arguably not worth documenting, but it is a discrepancy: every command visible in `--help` should ideally be mentioned in README.

### F3 — Config Path: Match (OK)

| | Value |
|---|---|
| README says | `~/.config/opencode-usage/config.yaml` |
| Code (`getConfigPath()`) | `homeDir + "/.config/opencode-usage/config.yaml"` |

### F4 — Config Defaults: Match (OK)

| Option | README Default | Code Default (`getDefaultConfig()`) |
|--------|---------------|--------------------------------------|
| `color_thresholds.warning` | 50 | 50 (`config.go:37`) |
| `color_thresholds.danger` | 80 | 80 (`config.go:38`) |
| `max_concurrent_requests` | 5 | 5 (`config.go:35`) |
| `use_master_password` | false | `nil` (pointer, omitempty) — functionally false |

Note: `use_master_password` in code is `*bool` with `omitempty`, so a fresh config file won't contain it at all. README shows `use_master_password: false` in the example YAML, which is fine — it's a valid serialization, just not the default output.

### F5 — Environment Variables: Match (OK)

| Variable | README | Code Location |
|----------|--------|---------------|
| `NO_COLOR` | Documented | `root.go:31`, `doctor.go:100`, `tui/theme.go:17` |
| `OPENCODE_USAGE_MASTER_PASSWORD` | Documented | `auth/encrypted.go:56` |
| `OPENCODE_USAGE_KEYRING_PASSWORD` | Documented | `auth/credential.go:22` |
| `OPENCODE_USAGE_BASE_URL` | Documented | `auth/validator.go:23` |

### F6 — Exit Codes: Partially Documented (MINOR)

| | README | Code |
|---|---|---|
| 0 | Success | `main.go` — implicit |
| 1 | Error (usage, auth, network, config, or keyring) | `main.go:13` — `os.Exit(1)` for any error |

README correctly captures the behavior. The code only ever exits 0 or 1 (single error code for all failures). Documentation is accurate.

### F7 — Module Path: Match (OK)

| | Value |
|---|---|
| go.mod | `github.com/emmmdty/opencode-usage` |
| README install | `go install github.com/emmmdty/opencode-usage/cmd/opencode-usage@latest` |

These match. The `cmd/opencode-usage/` subdirectory is correct per the directory structure.

### F8 — Installation Instructions: Match (OK)

- `go install` path matches module path + binary entry point
- `go build -o opencode-usage ./cmd/opencode-usage/` matches directory structure
- Binary builds successfully with Go 1.26.6

### F9 — `account export` Security Claim: Accurate (OK)

README says "names + key IDs only, no secrets". Code (`account.go:272`) exports `ExportAccount{Name, KeyID}` — no API key field in the struct. Confirmed: no secrets in export.

### F10 — LICENSE: Correct (OK)

MIT License, copyright 2026 emmmdty. Matches README "MIT" claim.

### F11 — Short Aliases Table: Accurate (OK)

All 10 aliases listed in README match the code:

| README Alias | Code `Aliases` field | File:Line |
|---|---|---|
| `account` → `a` | `Aliases: []string{"a"}` | `account.go:21` |
| `account add` → `aa` | `Aliases: []string{"aa"}` | `account.go:27` |
| `account list` → `al` | `Aliases: []string{"al"}` | `account.go:103` |
| `account remove` → `ar` | `Aliases: []string{"ar"}` | `account.go:196` |
| `account export` → `ae` | `Aliases: []string{"ae"}` | `account.go:252` |
| `account import` → `ai` | `Aliases: []string{"ai"}` | `account.go:301` |
| `quota` → `q` | `Aliases: []string{"q"}` | `quota.go:34` |
| `models` → `m` | `Aliases: []string{"m"}` | `models.go:16` |
| `current` → `cc` | `Aliases: []string{"cc"}` | `current.go:19` |
| `opencode-usage` → `ou` | `Aliases: []string{"ou"}` | `root.go:26` |

### F12 — Build from Source Path: Match (OK)

README: `go build -o opencode-usage ./cmd/opencode-usage/`  
Actual path: `cmd/opencode-usage/main.go` — correct.

### F13 — Version Output: Consistent (OK)

- `internal/version/version.go`: `Version = "0.2.0"`
- `--help` shows version from `rootCmd.Version: version.Version` → `0.2.0`
- `version` command output: `opencode-usage 0.2.0 (commit: none, built: unknown)` — correct

### F14 — Goreleaser Platforms: Match (OK)

README: "Linux, macOS, and Windows (amd64/arm64)"  
`.goreleaser.yml`: `goos: [linux, darwin, windows]`, `goarch: [amd64, arm64]` — matches.

---

## Issues Summary

| ID | Severity | Description |
|----|----------|-------------|
| F1 | **MAJOR** | Go version requirement mismatch: README says "1.22+", go.mod requires `1.26.6` |
| F2 | MINOR | `completion` command appears in `--help` but is undocumented in README |

---

## Verdict

**Overall:** Documentation is highly consistent with code. One **major** issue (Go version mismatch) must be fixed. The undocumented `completion` command is a minor cosmetic gap common to Cobra CLIs.

**Recommendation:** Fix the Go version claim in README from "Go 1.22+" to match the actual `go 1` directive in `go.mod`.
