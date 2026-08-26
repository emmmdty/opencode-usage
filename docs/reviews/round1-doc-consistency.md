# Documentation ↔ Code ↔ Behavior Consistency Audit

**Reviewer:** E (Documentation ↔ Code ↔ Behavior Consistency)
**Date:** 2026-08-26
**Scope:** Full audit of README.md, --help text, docs/ content, code implementation, test coverage, and runtime behavior

---

## Findings

### [F01] README does not document `doctor` command
- **Severity**: MAJOR
- **Location**: README.md (entire file)
- **Claim**: README features and usage sections omit the `doctor` command entirely
- **Reality**: `ou doctor` is fully implemented (`internal/cmd/doctor.go`), registered in `root.go`, and works at runtime. It checks config, keyring, network, accounts, and opencode auth. It appears in `--help` output.
- **Impact**: Users cannot discover the diagnostics command from the README. This is a useful self-service troubleshooting tool.
- **Fix**: Add a "Diagnostics" section to README with `opencode-usage doctor` usage.

### [F02] README does not document `--output` / `-o` flag
- **Severity**: MAJOR
- **Location**: README.md (entire file)
- **Claim**: README never mentions the `--output` / `-o` global flag
- **Reality**: The flag is defined in `internal/cmd/root.go:80` and functional. `account export` also uses it for file output (`account.go:284-288`). The flag appears in `--help` output.
- **Impact**: Users cannot discover file output capability from the README. Export to file is a documented feature but the mechanism is not explained.
- **Fix**: Add `-o, --output <file>` to the global flags or usage examples section.

### [F03] README does not document `--no-color` flag
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README mentions "NO_COLOR" only in context of env var (Security section does not mention it), but never documents the `--no-color` CLI flag
- **Reality**: `--no-color` is defined in `internal/cmd/root.go:81` and appears in `--help`. The `NO_COLOR` env var is also supported (root.go:31).
- **Impact**: Users cannot discover color disablement via CLI flag from README.
- **Fix**: Add `--no-color` to usage or features section. Also mention `NO_COLOR` env var compliance.

### [F04] README documents exit codes 2-7 but code only uses 0 and 1
- **Severity**: BLOCKER
- **Location**: README.md:131-140 (exit codes table)
- **Claim**: README documents exit codes 0-7 with specific meanings (2=usage error, 3=auth failure, 4=network error, 5=config error, 6=config not found, 7=keyring unavailable)
- **Reality**: `cmd/opencode-usage/main.go:13` calls `os.Exit(1)` for all errors. No code path in the entire codebase uses `os.Exit(2)` through `os.Exit(7)`. Cobra's `SilenceUsage: true` on root means cobra won't exit with code 2 on usage errors. All commands return errors that are caught by `main()` and exit with code 1.
- **Impact**: Users scripting around exit codes will get incorrect behavior. The documented contract is broken. A script checking `if [ $? -eq 3 ]` for auth failure will never trigger.
- **Fix**: Either implement proper exit codes in `main.go` or remove the exit code table from README. Implementing is preferred: catch typed errors and map to exit codes.

### [F05] Design doc says Go 1.22+, go.mod requires Go 1.26.6
- **Severity**: MAJOR
- **Location**: docs/superpowers/specs/2026-08-24-opencode-usage-design.md:26 vs go.mod:3
- **Claim**: Design doc states "Go 1.22+" as tech stack requirement
- **Reality**: `go.mod` specifies `go 1.26.6`. The binary builds and runs on this version. Go 1.26.6 is a much higher requirement than 1.22.
- **Impact**: Users on Go 1.22-1.25 cannot build from source as documented. README's "Build from source" section will fail for these users.
- **Fix**: Update design doc to match go.mod, or update README to state Go 1.26.6+ requirement. Also verify if 1.26.6 is actually needed or if go.mod can be lowered.

### [F06] Delivery report says v0.2.0, version.go says 0.1.0
- **Severity**: MAJOR
- **Location**: docs/DELIVERY.md:1 vs internal/version/version.go:6
- **Claim**: DELIVERY.md title says "opencode-usage v0.2.0 UX Refresh"
- **Reality**: `internal/version/version.go` has `Version = "0.1.0"`. Running `opencode-usage version` outputs `opencode-usage 0.1.0`.
- **Impact**: Version mismatch between documentation and actual binary. Confusing for users and release management.
- **Fix**: Update `version.go` to `0.2.0` if this is indeed the v0.2.0 release, or update DELIVERY.md to reflect the actual version.

### [F07] Remaining Chinese error messages contradict "English error messages" claim
- **Severity**: MAJOR
- **Location**: docs/DELIVERY.md:26, internal/cmd/alias.go:23, internal/auth/validator.go:43-75, internal/auth/encrypted.go:62-74
- **Claim**: DELIVERY.md states "English error messages — consistent English across all user-facing text"
- **Reality**: Multiple files contain Chinese text:
  - `alias.go:23`: `"获取用户主目录失败"` (failed to get home directory)
  - `validator.go:43`: `"网络连接失败，请检查网络"` (network connection failed)
  - `validator.go:51`: `"API Key有效"` (API Key valid)
  - `validator.go:57`: `"请检查您的API Key"` (please check your API Key)
  - `validator.go:63`: `"请订阅OpenCode Go计划"` (please subscribe to OpenCode Go plan)
  - `validator.go:69`: `"请求过于频繁，请稍后重试"` (too many requests, try later)
  - `validator.go:75`: `"服务器错误: HTTP %d"` (server error)
  - `encrypted.go:62`: `"请输入主密码: "` (enter master password)
  - `encrypted.go:65`: `"读取主密码失败"` (failed to read master password)
  - `encrypted.go:70`: `"主密码不能为空"` (master password cannot be empty)
  - `encrypted.go:133,153`: `"数据格式无效"` (invalid data format)
- **Impact**: International users cannot understand error messages. Inconsistent with the claimed English-only user-facing text. The `account add` interactive prompt for master password is in Chinese.
- **Fix**: Translate all user-facing strings to English. Use `go generate` with `xgo` or similar for i18n if Chinese support is desired later.

### [F08] README does not document `completion` command
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README lists all commands but omits `completion`
- **Reality**: Cobra auto-generates a `completion` command for shell autocompletion (bash/zsh/fish/powershell). It appears in `--help` output.
- **Impact**: Minor discoverability gap. Shell completions are useful for power users.
- **Fix**: Optionally mention in README. Low priority since cobra handles it automatically.

### [F09] `account list` does not support `--json` flag despite it being global
- **Severity**: MAJOR
- **Location**: README.md:68 (JSON output feature), internal/cmd/account.go:98-161
- **Claim**: README says "JSON output — Machine-readable output with `--json`" as a general feature
- **Reality**: `account list` ignores the `--json` flag. Its `RunE` always produces formatted text output. Only `quota` and `models` commands honor `--json`.
- **Impact**: Users expect `ou account list --json` to produce JSON but get formatted text. Inconsistent behavior across commands.
- **Fix**: Add JSON output support to `account list` or clarify in README that `--json` only works with `quota` and `models`.

### [F10] Root command default behavior not documented
- **Severity**: MAJOR
- **Location**: README.md
- **Claim**: README does not mention that running bare `opencode-usage` shows the quota dashboard
- **Reality**: The root command's `RunE` (`internal/cmd/root.go:41`) calls `runQuotaOverview()`, displaying the full quota dashboard. This is the primary use case.
- **Impact**: Users may not realize they can just run `opencode-usage` without any subcommand. The most important UX behavior is undocumented.
- **Fix**: Add a "Quick Start" section showing `opencode-usage` or `ou` as the primary way to check quota.

### [F11] JSON output field name inconsistency: "quota" vs "usage"
- **Severity**: MINOR
- **Location**: internal/cmd/quota.go:22 (`json:"quota,omitempty"`) vs internal/models/usage.go:14 (`json:"rolling"`)
- **Claim**: No documentation specifies the JSON schema field names
- **Reality**: The JSON output uses `"quota"` as the key for usage data per account (from `accountResult` struct tag), but the internal model is called `Usage`. The visual QA and product review both show `"quota"` in JSON output. This is not documented anywhere.
- **Impact**: JSON consumers must inspect actual output to know field names. The `"version"` field exists but schema is unspecified.
- **Fix**: Document the JSON schema in README or a separate docs/json-schema.md.

### [F12] README `account export` description is accurate but incomplete
- **Severity**: MINOR
- **Location**: README.md:53
- **Claim**: "Export accounts (names + key IDs only, no secrets)"
- **Reality**: Correct — `ExportAccount` struct has `Name` and `KeyID` fields only. The export command also supports `-o <file>` for file output, which is not mentioned.
- **Impact**: Users don't know they can export to a file.
- **Fix**: Mention `-o` flag in the export example.

### [F13] `formatResetTime` shows "<1m" correctly (UX audit claim of "0m" is wrong)
- **Severity**: NIT
- **Location**: internal/tui/quota.go:253-254
- **Claim**: UX audit (docs/research/ux-audit.md:289) says `formatResetTime` shows "0m" for zero duration
- **Reality**: Code at line 253-254 shows `if minutes == 0 { return "<1m" }`. The UX audit was written before the fix was applied.
- **Impact**: None — the audit document is stale but the code is correct.
- **Fix**: No code fix needed. The UX audit document should be updated to reflect the current behavior.

### [F14] Go module path mismatch with README install path
- **Severity**: MINOR
- **Location**: go.mod:1 vs README.md:21
- **Claim**: README says `go install github.com/emmmdty/opencode-usage/cmd/opencode-usage@latest`
- **Reality**: `go.mod` declares module as `github.com/opencode-usage` (no `emmmdty` org prefix). The `go install` command may fail because the module path doesn't match the repository path.
- **Impact**: `go install` from README will fail unless the actual GitHub repo is at `github.com/emmmdty/opencode-usage` and the go.mod path is overridden. If the repo is at `github.com/opencode-usage`, the README path is wrong.
- **Fix**: Align go.mod module path with the actual GitHub repository URL, or update README install path to match go.mod.

### [F15] README does not document `OPENCODE_USAGE_BASE_URL` env var
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README does not mention environment variables at all
- **Reality**: Code supports:
  - `NO_COLOR` — disable colors (root.go:31, theme.go:17)
  - `OPENCODE_USAGE_KEYRING_PASSWORD` — keyring file password (credential.go:22)
  - `OPENCODE_USAGE_MASTER_PASSWORD` — master password for encrypted storage (encrypted.go:55)
  - `OPENCODE_USAGE_BASE_URL` — custom API base URL (validator.go:23)
  - `SHELL` — determines which RC file to modify for alias install (alias.go:27)
- **Impact**: Power users and enterprise deployments cannot discover configuration via environment variables.
- **Fix**: Add an "Environment Variables" section to README.

### [F16] README does not document configurable color thresholds
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README does not mention configurable warning/danger thresholds
- **Reality**: `config.go` supports `color_thresholds.warning` (default 50) and `color_thresholds.danger` (default 80) in config.yaml. The TUI rendering uses these thresholds.
- **Impact**: Users cannot discover that color thresholds are configurable.
- **Fix**: Add configuration options section to README showing config.yaml structure.

### [F17] README does not document `max_concurrent_requests` config
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README mentions "concurrent queries" feature but not configuration
- **Reality**: `config.go` supports `max_concurrent_requests` (default 5). Used in `quota.go:80-83`.
- **Impact**: Users with many accounts cannot tune concurrency.
- **Fix**: Document in configuration section.

### [F18] README does not document `use_master_password` config option
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README mentions "Master password option for encrypted config" but not the config key
- **Reality**: `config.go:21` defines `UseMasterPassword *bool`. When set to `false` in config, the default password is used instead of prompting.
- **Impact**: Users don't know how to configure master password behavior.
- **Fix**: Document in security or configuration section.

### [F19] `doctor` command is not in the README command table but is in help
- **Severity**: MAJOR
- **Location**: README.md (no doctor mention) vs --help output
- **Claim**: README lists commands but omits `doctor`
- **Reality**: `doctor` is a fully implemented command with 5 diagnostic checks. It appears in `--help` output.
- **Impact**: Duplicate of F01 — the diagnostics feature is invisible to README readers.
- **Fix**: Same as F01.

### [F20] `version` command output format not documented
- **Severity**: NIT
- **Location**: README.md:99
- **Claim**: README shows `opencode-usage version` with no expected output
- **Reality**: Outputs `opencode-usage 0.1.0 (commit: none, built: unknown)`
- **Impact**: Very minor. Users can see it by running the command.
- **Fix**: Optionally show expected output in README.

### [F21] `update` command does not actually update
- **Severity**: MAJOR
- **Location**: README.md:100, internal/cmd/root.go:67-75
- **Claim**: README shows `opencode-usage update` as a command, implying it performs an update
- **Reality**: `updateCmd.RunE` only prints the current version and a URL to check manually. It does not download or install anything.
- **Impact**: Users expect `update` to auto-update but it's just a redirect to GitHub releases.
- **Fix**: Either implement actual update logic or rename to `update --check` / document that it only shows the update URL.

### [F22] `account remove` error message for missing account is in English but design doc shows Chinese
- **Severity**: NIT
- **Location**: internal/cmd/account.go:211
- **Claim**: Design doc examples show Chinese error messages
- **Reality**: `account.go:211` returns `fmt.Errorf("account '%s' not found", accountName)` in English. This is correct per v0.2 goals.
- **Impact**: None — code is correct. Design doc is stale.
- **Fix**: No code fix needed.

### [F23] `account add` Chinese error message in encrypted.go
- **Severity**: MAJOR
- **Location**: internal/auth/encrypted.go:62
- **Claim**: DELIVERY.md says "English error messages — consistent English across all user-facing text"
- **Reality**: `encrypted.go:62` prints `"请输入主密码: "` (Chinese) when prompting for master password. This is user-facing interactive text.
- **Impact**: Non-Chinese users cannot understand the password prompt.
- **Fix**: Translate to "Enter master password: ".

### [F24] Design doc version planning is stale
- **Severity**: MINOR
- **Location**: docs/superpowers/specs/2026-08-24-opencode-usage-design.md:8-16
- **Claim**: Design doc says import/export is v0.3 feature, models is v0.2
- **Reality**: Import, export, models, and doctor are all implemented in the current codebase (v0.1.0 per version.go).
- **Impact**: Design doc doesn't reflect actual implementation state. Misleading for future contributors.
- **Fix**: Update design doc to reflect current state or mark as historical.

### [F25] README Go version requirement not stated
- **Severity**: MINOR
- **Location**: README.md
- **Claim**: README's "Build from source" section shows `go build` but does not state Go version requirement
- **Reality**: `go.mod` requires Go 1.26.6. Users with older Go versions will get build errors.
- **Impact**: Users may waste time attempting to build with incompatible Go versions.
- **Fix**: Add "Requires Go 1.26.6+" to the build from source section.

---

## Feature Traceability Matrix

| Feature | README Says | Help Says | Code Implements | Test Covers | Runtime Verified |
|---------|------------|-----------|-----------------|-------------|------------------|
| account add | ✓ (line 44) | ✓ (alias: aa) | ✓ (account.go:25-96) | ✗ no unit test | ✓ interactive prompt works |
| account list | ✓ (line 47) | ✓ (alias: al) | ✓ (account.go:98-161) | ✗ no unit test | ✓ shows accounts with status |
| account remove | ✓ (line 50) | ✓ (alias: ar) | ✓ (account.go:191-227) | ✗ no unit test | ✓ removes account |
| account import | ✓ (line 56) | ✓ (alias: ai) | ✓ (account.go:296-379) | ✗ no unit test | ✓ imports from JSON |
| account export | ✓ (line 53) | ✓ (alias: ae) | ✓ (account.go:247-294) | ✗ no unit test | ✓ exports names+key IDs |
| quota | ✓ (line 63) | ✓ (alias: q) | ✓ (quota.go:32-38) | ✓ (tui/quota_test.go) | ✓ shows quota table |
| quota --account | ✓ (line 66) | ✓ (global -n) | ✓ (quota.go:54-59) | ✗ no unit test | ✓ filters by account |
| quota --json | ✓ (line 69) | ✓ (global -j) | ✓ (quota.go:141-143) | ✗ no unit test | ✓ outputs versioned JSON |
| models | ✓ (line 76) | ✓ (alias: m) | ✓ (models.go:14-44) | ✓ (client/opencode_test.go) | ✓ lists models |
| current | ✓ (line 83) | ✓ (alias: cc) | ✓ (current.go:17-62) | ✗ no unit test | ✓ shows auth config |
| doctor | ✗ NOT in README | ✓ | ✓ (doctor.go:14-85) | ✗ no unit test | ✓ runs diagnostics |
| alias install | ✓ (line 90) | ✓ | ✓ (alias.go:17-64) | ✗ no unit test | ✓ modifies RC file |
| alias uninstall | ✓ (line 93) | ✓ | ✓ (alias.go:67-97) | ✗ no unit test | ✓ removes from RC file |
| version | ✓ (line 99) | ✓ | ✓ (root.go:58-64) | ✗ no unit test | ✓ shows version info |
| update | ✓ (line 100) | ✓ | ✓ (root.go:66-75) | ✗ no unit test | ✓ prints URL only |
| NO_COLOR | ✗ NOT documented | ✓ (--no-color flag) | ✓ (root.go:31, theme.go:17) | ✗ no unit test | ✓ disables colors |
| config | ✓ (line 120) | N/A | ✓ (config/config.go) | ✓ (config/config_test.go) | ✓ loads/creates config |
| concurrency | ✓ (line 14) | N/A | ✓ (quota.go:80-84) | ✗ no unit test | ✓ limits parallel requests |
| security storage | ✓ (line 123-127) | N/A | ✓ (auth/credential.go, encrypted.go) | ✓ (auth/*_test.go) | ✓ keyring + encrypted fallback |
| exit codes | ✓ (lines 131-140) | N/A | ✗ only 0 and 1 used | ✗ no unit test | ✗ all errors exit 1 |
| --output / -o | ✗ NOT documented | ✓ (global flag) | ✓ (root.go:80) | ✗ no unit test | ✓ writes to file |
| --no-color | ✗ NOT documented | ✓ (global flag) | ✓ (root.go:81) | ✗ no unit test | ✓ disables colors |
| completion | ✗ NOT documented | ✓ (auto-generated) | ✓ (cobra built-in) | ✗ no unit test | ✓ generates completions |
| bare `ou` → dashboard | ✗ NOT documented | N/A | ✓ (root.go:41) | ✗ no unit test | ✓ shows quota dashboard |

---

## Summary

| Severity | Count |
|----------|-------|
| BLOCKER  | 1     |
| MAJOR    | 10    |
| MINOR    | 9     |
| NIT      | 5     |

### BLOCKER Issues

1. **F04**: Exit codes 2-7 documented in README but code only uses 0 and 1. The documented contract is entirely broken.

### MAJOR Issues

1. **F01/F19**: `doctor` command fully implemented but completely absent from README
2. **F02**: `--output` / `-o` flag functional but undocumented in README
3. **F05**: Go version mismatch — design doc says 1.22+, go.mod requires 1.26.6
4. **F06**: Version mismatch — DELIVERY.md says v0.2.0, version.go says 0.1.0
5. **F07/F23**: Chinese error messages remain despite "English error messages" claim
6. **F09**: `account list` ignores `--json` flag despite it being a global feature
7. **F10**: Root command default behavior (quota dashboard) not documented
8. **F14**: Go module path mismatch with README install instructions
9. **F20**: `update` command doesn't actually update — just prints URL
10. **F03**: `--no-color` flag undocumented in README

### Positive Findings (Things That Match)

- All documented commands exist and work
- All documented aliases are correct
- All documented flags are correct (for commands that support them)
- Config file path is correct
- Security description is accurate (keyring + encrypted fallback)
- Export format description is accurate (names + key IDs only)
- Quota features (5h/weekly/monthly, progress bars, best account, next reset) all work
- JSON output format matches what is actually produced
- Installation instructions (build from source, go install) are syntactically correct
- Short aliases table in README matches actual registered aliases
- `NO_COLOR` env var support works (despite not being documented)
- Non-TTY auto-detection works
- CJK width calculation is fixed (uses runewidth)

---

## Verdict: **REQUEST CHANGES**

The documentation is substantially correct for what it covers, but has significant gaps:

1. **1 BLOCKER**: Exit codes table documents non-existent behavior
2. **10 MAJORs**: Missing feature documentation (doctor, --output, --no-color), version mismatches, Chinese text contradicting English-only claims, incomplete JSON support on account list, undocumented default behavior, module path mismatch, misleading update command
3. **9 MINORs**: Missing env var docs, config options, Go version requirement, JSON schema, stale design docs

The README needs a pass to: (a) add doctor, --output, --no-color, env vars, config options, Go version requirement, (b) fix exit codes table, (c) document default `ou` → dashboard behavior, (d) clarify update command is informational only.
