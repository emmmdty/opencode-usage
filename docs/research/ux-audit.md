# UX Audit: opencode-usage CLI

**Auditor:** Subagent B — CLI/TUI UX Expert
**Date:** 2026-08-26
**Scope:** Full CLI surface — information architecture, visual system, terminal correctness, error handling

---

## 1. Information Architecture

### 1.1 Root Command Does Nothing Useful [P0]

Running `opencode-usage` (no subcommand) just dumps cobra help. The #1 use case — checking quota — requires `ou q`. A first-time user who runs the tool after setup gets zero value.

**Recommendation:** Default action should be `quota` (i.e., `RunE: quotaCmd.RunE` on root). Most CLIs do this: `docker` → `docker ps`, `gh` → shows dashboard, `kubectl` → shows help + suggested first command.

### 1.2 Command Alias Overload and Confusion [P1]

The alias system is internally inconsistent and confusing:

| Path | Works? | Notes |
|------|--------|-------|
| `ou quota` | Yes | canonical |
| `ou q` | Yes | alias on `quotaCmd` |
| `ou account add` | Yes | canonical |
| `ou a aa` | Yes | alias `a` on `account`, alias `aa` on `add` |
| `ou aa` | Yes | **separate root-level command** `accountAddAliasCmd` |
| `ou al` | Yes | **separate root-level command** `accountListAliasCmd` |
| `ou ar` | Yes | **separate root-level command** `accountRemoveAliasCmd` |

There are **two duplicate paths** to the same action: `ou aa` and `ou a aa`. The root-level `accountAddAliasCmd` / `accountListAliasCmd` / etc. (lines 336-376 in `account.go`) are redundant with the aliases already defined on the subcommands. This bloats `--help` output with near-duplicate entries and confuses users about which form is "correct."

Additionally, `ou a` is the alias for `account`, but `account` itself has sub-commands that take `a` as alias too — so `ou a a` vs `ou aa` vs `ou account add` are three ways to do the same thing.

### 1.3 Chinese-Only Help Text, English Tool Name [P1]

Every `Short`, `Long`, error message, and user-facing string is in Chinese. The tool name `opencode-usage`, the README, flag names (`--json`, `--no-color`), and the binary name are all English. This is inconsistent. International users can't use the tool at all; Chinese users see English in some places and Chinese in others.

### 1.4 No Default Behavior Value [P0]

`ou` (root command) → help text
`ou q` → quota (the thing you always want)
`ou cc` → current config

The alias `cc` for `current` is non-obvious. Users will type `ou current` which works, but `cc` as a shortcut is not mnemonic.

### 1.5 State Information Redundancy [P2]

When viewing quota, the reset time is shown inline: `"80% (剩余5d12h)"`. This is good — it's compact. But `account list` shows `LastVerified` as a full timestamp `2006-01-02 15:04:05` which is hard to scan. Should be relative like the quota reset times: "3h ago", "2d ago".

---

## 2. Visual System

### 2.1 Quota Table Layout [P1]

**File:** `internal/tui/quota.go`

```
  OpenCode Go 配额概览

  账号      滚动配额              周配额                月配额
  ──────────────────────────────────────────────────────────
  work      80% (剩余5d12h)      45% (剩余3d8h)        12% (剩余18d)
  personal  20% (剩余1d3h)       90% (剩余6h)          67% (剩余12d)
```

Issues:

- **No progress bars.** A simple `████████░░░░ 80%` is immediately scannable. Percentages require reading.
- **Column header spacing is awkward.** The three quota columns use the same label width (`quotaWidth = 22`) but headers are short Chinese ("滚动配额", "周配额", "月配额") while data is long. Headers look left-aligned, data looks left-aligned, but the gap is inconsistent visually.
- **Separator line doesn't match content width.** `strings.Repeat("─", nameWidth+quotaWidth*3+3)` doesn't account for the 2-space leading indent. The separator is shorter than the content.
- **No totals/summary row** when showing multiple accounts.
- **No color legend** — green/yellow/red colors are used but never explained.

### 2.2 Account List Formatting [P1]

**File:** `internal/cmd/account.go:139`

```go
fmt.Printf("账号: %-12s Key ID: sk-...%-6s 状态: %-8s 上次验证: %s\n",
    name, account.KeyID, status, lastVerified)
```

- `%-12s` for name: breaks with names >12 chars or CJK names.
- `%-8s` for status: `"未验证"` is 3 CJK chars = 6 display columns, but 9 bytes. `%-8s` pads based on bytes, not display width, causing misalignment.
- Doesn't use the TUI package at all — inconsistent with quota output.
- No visual differentiation for "可能过期" status (should be yellow/warning).

### 2.3 Color System [P2]

- Uses ANSI256 colors (196=red, 214=orange, 46=green). Terminals with only 16 colors will show wrong or no colors.
- No bold/dim for emphasis on headers or key values.
- No Unicode box-drawing characters for table borders (uses `─` which is fine, but could use full box drawing).

### 2.4 Terminal Width Handling [P2]

The table is fixed-width (~76 chars minimum). No terminal width detection. On narrow terminals (80 cols), long account names cause line wrapping. On wide terminals (160 cols), the table is tiny left-aligned.

**Recommendation:** Detect terminal width, center or left-align with max-width, and truncate long names with `…` if needed.

### 2.5 Loading States [P1]

When fetching quota for multiple accounts concurrently, there is **zero visual feedback** until all results arrive. For 5+ accounts or slow networks, the user stares at a blank screen for seconds with no indication the tool is working.

**Recommendation:** Add a spinner or progress indicator: `"Fetching quota for 5 accounts..."` with a spinner.

### 2.6 Error States [P1]

Error rows in the quota table:
```
  work      错误: API error: HTTP 401
```

- No color differentiation — errors look like normal data rows.
- No emoji/icon prefix (e.g., `✗` or `⚠`).
- The error string is raw Go error text (`API error: HTTP 401`), not user-friendly.

### 2.7 Empty States [P2]

- `ou q` with no accounts configured: `"暂无配置的账号，请先运行 'opencode-usage account add' 添加账号"` — good, actionable.
- `ou models` with no models: just prints `"可用模型:"` with nothing after — feels broken.
- `ou cc` with no auth.json: `"未找到opencode配置文件"` — okay but could suggest what to do.

---

## 3. Critical Bug Check

### 3.1 CJK Column Width Calculation — BROKEN [P0]

**File:** `internal/tui/quota.go:23`

```go
nameWidth := 5
for _, r := range results {
    if len(r.Name) > nameWidth {
        nameWidth = len(r.Name)
    }
}
```

`len(r.Name)` returns **byte count**, not display width. For CJK characters:
- Each CJK char = 3 bytes UTF-8, but occupies 2 terminal columns.
- `"测试"` → `len()` = 6, display width = 4.
- `"生产环境"` → `len()` = 12, display width = 8.

This miscalculates column width, causing misaligned tables for any CJK account name.

**Fix:** Use `runewidth.StringWidth()` from `github.com/mattn/go-runewidth` (or `eastasianwidth`).

### 3.2 CJK Padding in Quota Columns — BROKEN [P1]

**File:** `internal/tui/quota.go:61`

```go
rollingColored, strings.Repeat(" ", quotaWidth-len(rollingStr)),
```

`len(rollingStr)` counts bytes. The raw `rollingStr` is `"80% (剩余5d12h)"` which contains `"剩余"` (6 bytes, 4 columns). So `len()` overestimates the display width by 2, causing extra padding and misaligned columns.

This is a secondary effect of bug 3.1 but occurs in a different code path.

### 3.3 `NO_COLOR` Environment Variable Ignored [P1]

**File:** `internal/cmd/root.go:28-31`

```go
PersistentPreRun: func(cmd *cobra.Command, args []string) {
    if noColor {
        lipgloss.SetColorProfile(termenv.Ascii)
    }
},
```

Only checks the `--no-color` CLI flag. Does **not** check:
- `NO_COLOR` env var (https://no-color.org/ standard)
- `TERM=dumb` detection
- Isatty detection for piped output

Per the [NO_COLOR spec](https://no-color.org/), any tool respecting it must check `$NO_COLOR`.

### 3.4 Pipe/Redirect Outputs ANSI Escape Codes [P1]

When running `ou q | less` or `ou q > output.txt`, ANSI color codes are still emitted unless `--no-color` is explicitly passed. Most modern CLI tools auto-detect `os.Stdout` is not a terminal and disable colors.

**File:** `internal/cmd/root.go` — no `isatty` check exists anywhere in the codebase.

### 3.5 `account list` Status Column Misalignment [P1]

**File:** `internal/cmd/account.go:139`

```go
fmt.Printf("账号: %-12s Key ID: sk-...%-6s 状态: %-8s 上次验证: %s\n", ...)
```

Status values:
- `"未验证"` = 9 bytes, 6 display columns → `%-8s` pads 8 bytes → 2 extra bytes → misaligned
- `"正常"` = 6 bytes, 4 display columns → `%-8s` pads 8 bytes → 2 extra bytes → misaligned
- `"可能过期"` = 12 bytes, 8 display columns → `%-8s` pads 8 bytes → already wider than 8 → no padding → misaligned

All three statuses produce different visual alignments. This is broken for any non-ASCII status.

### 3.6 Fixed `quotaWidth` Doesn't Adapt [P2]

**File:** `internal/tui/quota.go:30`

```go
quotaWidth := 22
```

This is hardcoded. The formatted string `"80% (剩余5d12h)"` can be up to ~20 chars, so 22 is tight but works for current content. However:
- If the API ever returns "100%" (3 digits), the string is `"100% (剩余5d12h)"` = 18 bytes, still fits.
- But the Chinese "剩余" contributes 4 display columns, so the visual width is smaller than the byte width, meaning there's wasted visual space that could be used for better alignment.

Should be calculated dynamically based on actual content width.

### 3.7 Token Masking Edge Case [P2]

**File:** `internal/cmd/current.go:53`

```go
masked := "..." + authCfg.Token[len(authCfg.Token)-min(6, len(authCfg.Token)):]
```

For a token shorter than 6 characters (unlikely but possible), `min(6, len(token))` = `len(token)`, so the entire token is revealed. For a 6-char token, all 6 chars are shown. For 7 chars, 6 of 7 are shown.

---

## 4. Other Issues

### 4.1 No `--watch`/`--follow` Mode [P2]

Monitoring quota during heavy usage requires re-running `ou q` manually. A `--watch 30s` flag would be very useful for power users.

### 4.2 `alias install` Only Supports Bash/Zsh [P2]

**File:** `internal/cmd/alias.go:28`

```go
if strings.Contains(shell, "zsh") {
    rcFile = homeDir + ".zshrc"
} else {
    rcFile = homeDir + "/.bashrc"
}
```

No support for fish (`~/.config/fish/config.fish`), PowerShell, or Nushell. Falls back silently to `.bashrc` for any non-zsh shell.

### 4.3 Config File Path Not Overridable [P2]

**File:** `internal/cmd/account.go:392-398`

```go
func getConfigPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return homeDir + "/.config/opencode-usage/config.yaml", nil
}
```

Hardcoded. No `$XDG_CONFIG_HOME` support, no `--config` flag. For users with non-standard home directories or multi-config setups, this is inflexible.

### 4.4 `current` Reads Hardcoded Path [P2]

**File:** `internal/cmd/current.go:26`

```go
authPath := homeDir + "/.local/share/opencode/auth.json"
```

Hardcoded to `~/.local/share/opencode/auth.json`. Should respect `$XDG_DATA_HOME` or accept a flag.

### 4.5 Duplicated Config Loading [P2]

**File:** `internal/cmd/quota.go:29-37` and `internal/cmd/quota.go:145-153`

The quota command loads config **twice**: once at the start of the `RunE` function, and again in `printQuotaTable`. The second load is unnecessary and wasteful.

### 4.6 Missing `--account` / `-n` on `models` and `current` [P2]

The `-n` flag is defined as a global persistent flag, so it's available everywhere, but `models` silently ignores it if multiple accounts exist (it just picks the first alphabetically). Should either enforce it or show a warning.

### 4.7 No JSON Output for `account list` [P2]

The `--json` flag is defined globally but `account list` doesn't honor it. `ou account list --json` just prints the formatted text.

### 4.8 `formatResetTime` Shows "0m" for Zero Duration [P2]

**File:** `internal/tui/quota.go:98`

When the reset time is imminent (< 1 minute), it shows "0m". Should show something like "<1m" or "即将重置".

---

## 5. Summary Table

| # | Issue | Severity | File:Line |
|---|-------|----------|-----------|
| 1.1 | Root command does nothing — should default to quota | **P0** | `cmd/root.go:22` |
| 1.2 | Duplicate alias paths (root-level + subcommand aliases) | P1 | `cmd/account.go:336-389` |
| 1.3 | Chinese-only text with English tool name | P1 | All files |
| 1.4 | `cc` alias not mnemonic for `current` | P2 | `cmd/current.go:19` |
| 1.5 | `account list` shows absolute timestamps not relative | P2 | `cmd/account.go:132` |
| 2.1 | No progress bars in quota table | P1 | `tui/quota.go` |
| 2.2 | Account list formatting inconsistent with quota table | P1 | `cmd/account.go:139` |
| 2.3 | ANSI256 colors only, no 16-color fallback | P2 | `tui/quota.go:74` |
| 2.4 | No terminal width detection | P2 | `tui/quota.go` |
| 2.5 | No loading/spinner state during concurrent fetch | P1 | `cmd/quota.go:68-93` |
| 2.6 | Error rows not visually distinct | P1 | `tui/quota.go:44-45` |
| 2.7 | Empty state for `models` is broken | P2 | `cmd/models.go:36` |
| 3.1 | CJK `len(r.Name)` byte-count for column width | **P0** | `tui/quota.go:23` |
| 3.2 | CJK padding in quota columns `len(rollingStr)` | P1 | `tui/quota.go:61` |
| 3.3 | `NO_COLOR` env var not checked | P1 | `cmd/root.go:28` |
| 3.4 | ANSI codes emitted to pipes/redirects | P1 | `cmd/root.go` (no isatty) |
| 3.5 | `account list` status column misaligned for CJK | P1 | `cmd/account.go:139` |
| 3.6 | Fixed `quotaWidth` doesn't adapt | P2 | `tui/quota.go:30` |
| 3.7 | Token masking reveals full token if ≤6 chars | P2 | `cmd/current.go:53` |
| 4.1 | No `--watch` mode | P2 | — |
| 4.2 | `alias install` only supports bash/zsh | P2 | `cmd/alias.go:28` |
| 4.3 | Config path not overridable, no `$XDG_CONFIG_HOME` | P2 | `cmd/account.go:392` |
| 4.4 | `current` reads hardcoded path | P2 | `cmd/current.go:26` |
| 4.5 | Config loaded twice in quota command | P2 | `cmd/quota.go:29,145` |
| 4.6 | `-n` flag silently ignored by `models` | P2 | `cmd/models.go:55` |
| 4.7 | `account list` ignores `--json` flag | P2 | `cmd/account.go:100` |
| 4.8 | `formatResetTime` shows "0m" near expiry | P2 | `tui/quota.go:98` |

**P0 (3):** Root does nothing; CJK column width broken; — these are the most impactful.
**P1 (9):** Alias overload, Chinese/English mix, no progress bars, no spinner, error states not distinct, CJK padding, NO_COLOR ignored, pipe colors, account list misalignment.
**P2 (12):** Timestamps, terminal width, empty states, token masking, XDG, config paths, etc.

---

## 6. Recommended Priority Fixes

### Immediate (P0)
1. **Default root to `quota`** — Make `ou` show quota table directly.
2. **Fix CJK width** — Add `runewidth.StringWidth()` for all column width and padding calculations.

### Short-term (P1)
3. **Check `NO_COLOR` + isatty** — Auto-disable colors for pipes and when `NO_COLOR` is set.
4. **Add spinner during concurrent fetch** — At minimum, print "Fetching..." before the goroutines.
5. **Add progress bars to quota table** — Replace or supplement percentages with `████░░░░` visual bars.
6. **Color error rows** — Make errors visually distinct (red + prefix).
7. **Remove duplicate root-level alias commands** — `ou aa` etc. are redundant with `ou a aa`.
8. **Fix `account list` CJK alignment** — Use `runewidth` for status and name columns.

### Medium-term (P2)
9. Relative timestamps in `account list`.
10. Terminal width detection and table adaptation.
11. `$XDG_CONFIG_HOME` / `$XDG_DATA_HOME` support.
12. `--watch` mode.
13. Fish/PowerShell alias support.
