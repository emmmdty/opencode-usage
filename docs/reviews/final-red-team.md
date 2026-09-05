# Final Red Team Review — Reviewer Z

**Reviewer:** Z (Final Red Team)
**Date:** 2026-08-26
**Status:** Adversarial — trying to block release

---

## Build & Test Results

| Check | Result |
|-------|--------|
| `go build` | PASS |
| `go test ./...` | PASS (5 packages, 3 skipped) |
| `go test -race ./...` | PASS |
| `go vet ./...` | PASS |
| Cross-compile linux/amd64 | PASS |
| Cross-compile windows/amd64 | PASS |
| Cross-compile darwin/arm64 | PASS |
| `--help` | PASS |
| `version` | PASS |
| LICENSE exists | PASS |

---

## BUGS FOUND

### BUG-1: Every error is printed TWICE (Medium Severity)

**Reproduction:**
```
$ ./token-usage quota -n "nonexistent" 2>&1
Error: account 'nonexistent' not found
Error: account 'nonexistent' not found
```

**Root Cause:** `SilenceErrors` is NOT set on the root cobra command. Cobra's default behavior prints errors to stderr, then `main.go` prints them again via `fmt.Fprintf(os.Stderr, "Error: %v\n", err)`.

**Location:** `internal/cmd/root.go:24` (missing `SilenceErrors: true`) + `cmd/token-usage/main.go:12`

**Impact:** Every single error case produces doubled output. Looks unprofessional. Breaks any shell script that parses stderr for error counting.

---

### BUG-2: `--output` flag is silently ignored by 5 of 8 commands (Medium Severity)

The `-o, --output` flag is documented as a global flag in help and README. It only works for `quota` and `models`. It does NOT work for:

| Command | Uses `writeOutput()`? | `-o` works? |
|---------|----------------------|-------------|
| `quota` | Yes | Yes |
| `models` | Yes | Yes |
| `version` | No (`fmt.Println`) | **NO** |
| `doctor` | No (`fmt.Printf`) | **NO** |
| `update` | No (`fmt.Printf`) | **NO** |
| `account list` | No (`fmt.Printf`) | **NO** |
| `current` | No (`fmt.Println`) | **NO** |
| `account export` | Special (uses `outputFile` var) | Yes |

**Reproduction:**
```
$ ./token-usage version -o /tmp/test.txt
token-usage 0.2.0 (commit: none, built: unknown)
$ cat /tmp/test.txt
cat: /tmp/test.txt: No such file or directory
```

**Location:** `internal/cmd/root.go:62`, `internal/cmd/doctor.go:68`, `internal/cmd/alias.go`, `internal/cmd/account.go:159`, `internal/cmd/current.go:44`

---

### BUG-3: `--json` flag silently ignored by 4 commands (Low-Medium Severity)

The `-j, --json` flag is accepted but produces identical text output for:

- `doctor` — always text
- `account list` — always text
- `update` — always text
- `version` — always text

**Impact:** Users piping to `jq` or other tools get unparseable text output with no error.

---

### BUG-4: Version format inconsistency (Low Severity)

Two different version formats:

```
$ ./token-usage --version
token-usage version 0.2.0

$ ./token-usage version
token-usage 0.2.0 (commit: none, built: unknown)
```

The `--version` flag uses cobra's built-in format. The `version` subcommand uses `version.GetVersionInfo()`. When built via GoReleaser with ldflags, the subcommand shows useful info (commit, date) but the flag does not.

---

### BUG-5: `doctor` hardcodes raw ANSI escape codes (Low Severity)

`internal/cmd/doctor.go:103` uses:
```go
return doctorTheme{okIcon: "\033[32m✓\033[0m", warnIcon: "\033[33m!\033[0m", failIcon: "\033[31m✗\033[0m"}
```

While it does check `noColor`/`NO_COLOR`/`isTerminal()`, this is inconsistent with the rest of the TUI which uses lipgloss. The `isTerminal()` implementation in `doctor.go:106-112` uses `os.Stdout.Stat()` + `ModeCharDevice`, while `root.go:35` uses `term.IsTerminal()`. These could theoretically disagree.

**Impact:** Maintenance burden. If lipgloss changes behavior, doctor won't follow.

---

## EDGE CASES TESTED

### Empty `secrets.enc` triggers interactive password prompt

When the encrypted secrets file exists but is empty, and `use_master_password` is nil, the tool calls `getMasterPassword()` which calls `doInitMasterPassword()` which attempts `term.ReadPassword()`. In non-interactive contexts (piped, cron, CI), this hangs or fails with `inappropriate ioctl for device`.

**Reproduction:**
```
$ touch ~/.config/token-usage/secrets.enc
$ ./token-usage quota 2>&1
Enter master password:
```

### Corrupt config YAML

Properly returns error (tested):
```
Error: yaml: line 1: did not find expected node content
```

### Corrupt `secrets.enc` (invalid base64)

Gracefully degraded (tested):
```
✗ illegal base64 data at input byte 3
```

### NO_COLOR

Correctly removes all ANSI escape codes. Unicode decorative characters (─, █, ░) remain — this is correct per no-color.org spec.

### Concurrent access

Two instances running simultaneously produce identical output. No corruption detected.

### Narrow terminal (width 50)

Falls back to compact format. Display remains readable.

### JSON output (`--json`)

Quota JSON output is clean — no ANSI contamination. Valid JSON.

---

## ADDITIONAL OBSERVATIONS

### No tests for `cmd`, `models`, or `version` packages

Three packages have zero test files:
- `internal/cmd/` — all command logic, 0 tests
- `internal/models/` — data types, 0 tests
- `internal/version/` — version info, 0 tests

### `account export` JSON is not sorted consistently

`exportData.Accounts` is populated from map iteration (random order), then sorted. This works correctly — confirmed.

### `config.MigrateConfig` loses intentional zero thresholds

If a user sets `color_thresholds.warning: 0` or `color_thresholds.danger: 0`, migration resets to defaults because `old.ColorThresholds.Warning != 0` is checked. Zero is a valid threshold (means "always warning").

### `resolveCurrentAccount` uses last-6-char matching

If two accounts have API keys ending in the same 6 characters, the wrong account could be marked as "active." Low probability but architecturally fragile.

---

## WHAT I COULD NOT BREAK

- Encryption/decryption roundtrip: solid
- Keyring fallback to encrypted file: works correctly
- Multiple concurrent accounts (tested 3): all fetched in parallel
- 100% quota display: renders correctly
- Expired reset timestamps: shows "expired"
- CJK account names: handled correctly via `runewidth`
- Path traversal in account names (`../../etc/passwd`): treated as literal name, not a filesystem risk
- Network failure: gracefully returns error per account
- File permission issues: returns clear error message

---

## Verdict

BLOCK RELEASE

**Must-fix before release:**

1. **BUG-1 (double errors):** Add `SilenceErrors: true` to root command in `internal/cmd/root.go:24`. One-line fix.
2. **BUG-2 (`--output` ignored):** Either wire `writeOutput()` into all commands, or remove `-o` from the global flags and only show it on commands that support it. The current state is misleading documentation.

These two issues are user-facing bugs that will be immediately noticed. The double-error makes the tool look broken, and the `--output` flag being documented globally but only working on 2/8 commands is a usability trap.
