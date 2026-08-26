# Round 1 — Visual QA Review

**Reviewer**: D — CLI/TUI UX & Visual QA  
**Date**: 2026-08-26  
**Method**: Built binary, ran tests, generated visual output at 7 widths × 9 scenarios, measured line lengths programmatically

---

## Summary

The TUI rendering in `internal/tui/quota.go` produces clean, well-structured output at the default 80-column width. Colors, progress bars, and status indicators are visually clear. However, there is a **significant layout overflow bug** that breaks output at narrower terminal widths (60–70 cols), a **column alignment defect** caused by the Unicode arrow marker, and a minor grammar issue. The code is well-tested (all existing tests pass) but lacks test coverage for the overflow and alignment scenarios discovered here.

---

## Issues

### [VQ-01] Table content overflows terminal width at narrow terminals

- **Severity**: MAJOR
- **Terminal Width**: 60, 70 cols (also affects long account names at 80)
- **Command/Scenario**: `opencode-usage quota` with 3+ accounts on a 60-col terminal
- **Actual Output** (width=60, 3 accounts):
  ```
    ACCOUNT         5H               Weekly           Monthly        
    ─────────────────────────────────────────────────────────────────────
    → work            █░░  35% 7h59m   ░░░  12% 4d23h   ░░░   8% 22d23h
    personal        ██░  67% 1h59m   █░░  45% 2d23h   ░░░  22% 17d23h
  ```
  Separator is 71 visible chars, header is 67, data rows are 69–72. All exceed the 60-col terminal.
- **Expected Output**: All lines should be ≤60 chars wide
- **Problem**: `colWidth` is bumped to a minimum of 15 (`quota.go:64`) regardless of available space. When `usable = width - 2` (58 at width=60), the actual layout needs `nameWidth + 3×15 + 10 = 68` chars, exceeding the terminal by 10+ chars.
- **Suggested Fix**: Either (a) remove the `colWidth < 15` minimum, or (b) after bumping `colWidth`, re-check if total exceeds `usable` and scale down proportionally, or (c) switch to compact mode at a higher threshold (e.g., width < 75 for 3+ accounts).

### [VQ-02] Separator line length not constrained to terminal width

- **Severity**: MAJOR
- **Terminal Width**: 60, 70 cols (all scenarios with ≥2 accounts)
- **Command/Scenario**: Any `quota` output with table mode
- **Actual Output** (width=60, multi): separator is 71 chars (11 over limit)
- **Expected Output**: Separator should be ≤ `width - 2` chars
- **Problem**: `sepLen = nameWidth + 3×colWidth + 10` (`quota.go:81`) is computed from column layout, not constrained by terminal width. The `usable` variable is used only for column width calculation but not enforced on the separator.
- **Suggested Fix**: `sepLen = min(sepLen, usable)` or recompute after enforcing width constraints.

### [VQ-03] Active account column misalignment due to `→` marker width

- **Severity**: MAJOR
- **Terminal Width**: All widths (80 confirmed)
- **Command/Scenario**: `opencode-usage quota` with `--account` or default current account
- **Actual Output** (width=80, multi):
  ```
  → work            ██░░░░  35% 7h59m   ░░░░░░  12% 4d23h   ░░░░░░   8% 22d23h
    personal        ████░░  67% 1h59m   ██░░░░  45% 2d23h   █░░░░░  22% 17d23h
    side-project    ░░░░░░   8% 19h59m  ░░░░░░   5% 5d23h   ░░░░░░   2% 24d23h
  ```
  The first data column (`█`/`░` bars) starts at different positions for the active vs inactive rows.
- **Expected Output**: All data columns should start at the same horizontal position
- **Problem**: `→` (U+2192) has `runewidth` of 2 in some locales (ambiguous East Asian width), while the inactive marker `  ` has width 2. The code assumes `→ ` is 2 chars wide (`quota.go:264`: `w += 2`) but it may be 3. This shifts the active row's data by 1 char relative to inactive rows.
- **Suggested Fix**: Use a guaranteed-width marker like `▸` or `>>`, or compute marker width dynamically using `runewidth.RuneWidth('→')`. Alternatively, use ASCII `->` consistently in both modes.

### [VQ-04] Grammar: "1 accounts" should be "1 account"

- **Severity**: MINOR
- **Terminal Width**: All
- **Command/Scenario**: Single account configured (or all accounts fail, leaving 1)
- **Actual Output**: `1 accounts  1 healthy`
- **Expected Output**: `1 account  1 healthy`
- **Problem**: `computeSummary()` (`quota.go:318`) uses `fmt.Sprintf("%d accounts", total)` without singular/plural handling.
- **Suggested Fix**: Add a pluralization check: `if total == 1 { "1 account" } else { "%d accounts" }`

### [VQ-05] Compact mode uses `->` while table mode uses `→`

- **Severity**: NIT
- **Terminal Width**: <60 (compact), ≥60 (table)
- **Command/Scenario**: Switching between narrow and wide terminals
- **Actual Output**: Compact mode shows `-> work`, table mode shows `→ work`
- **Expected Output**: Consistent marker across modes
- **Problem**: `formatCompact()` (`quota.go:129`) uses `"-> "` while `formatQuotaRow()` (`quota.go:182`) uses `"→ "`
- **Suggested Fix**: Use the same marker in both code paths.

### [VQ-06] `truncateError` over-truncates exact-width strings

- **Severity**: NIT
- **Terminal Width**: All
- **Command/Scenario**: Error messages that exactly fill the maxLen
- **Actual Output**: `truncateError("测试测试", 4)` → `"测…"` (width 3, not 4)
- **Expected Output**: `"测试"` (width 4, fits exactly)
- **Problem**: The check `current+rw+1 > maxLen` (`quota.go:371`) reserves 1 extra char for the ellipsis even when the string fits exactly. The `+1` is unnecessary when `current+rw == maxLen`.
- **Suggested Fix**: Change to `current+rw > maxLen` and only append `"…"` when `current+rw > maxLen` (not `>=`).

### [VQ-07] Empty account name produces bare "Active:" with no name

- **Severity**: NIT
- **Terminal Width**: All
- **Command/Scenario**: Account with empty string name
- **Actual Output**: `Active: ` (trailing space, no name)
- **Expected Output**: Either skip the "Active" line, or show `"Active: (unnamed)"`
- **Problem**: `formatQuotaOverview()` renders `Active: ` + `theme.Active.Render(r.Name)` where `r.Name` is empty. The empty string produces valid but odd-looking output.
- **Suggested Fix**: Skip the "Active" footer line when name is empty, or add a guard.

### [VQ-08] Footer legend shows same symbol for healthy and critical in NO_COLOR mode

- **Severity**: MINOR
- **Terminal Width**: All (NO_COLOR mode only)
- **Command/Scenario**: `--no-color quota` or piped output
- **Actual Output**: `● healthy  ▲ warning  ● critical  → active`
- **Expected Output**: Different symbols, e.g., `✓ healthy  ▲ warning  ✗ critical  → active`
- **Problem**: In no-color mode, `●` (U+25CF) is used for both "healthy" (`theme.Success`) and "critical" (`theme.Danger`). Without color differentiation, the two statuses are visually identical.
- **Suggested Fix**: Use distinct symbols: `●` for healthy, `◆` or `✗` for critical, `▲` for warning (already distinct).

---

## What Works Well

- **Progress bars**: `█`/`░` block characters render clearly and proportionally at all widths. The bar width scales correctly from 3 chars (narrow) to 30+ chars (wide).
- **Color theme**: The green/yellow/red palette is intuitive and accessible. Status colors match user expectations.
- **Reset time format**: Relative durations (`7h59m`, `4d23h`) are compact and readable. The `expired` state is clear.
- **Compact mode fallback**: The `< 60` width threshold with simplified layout is a good UX decision.
- **CJK support**: `runewidth.StringWidth` handles Chinese characters correctly, with proper padding.
- **Emoji support**: Emoji account names (`🚀rocket`) align correctly with ASCII names.
- **Error display**: Error rows with `✗` prefix and truncated messages are clear and don't break layout.
- **Summary line**: The `"3 accounts  2 healthy  1 warning"` format is concise and informative.
- **NO_COLOR / non-TTY detection**: Both the `NO_COLOR` env var and pipe detection work correctly (`root.go:31-38`, `theme.go:17-25`).
- **All existing tests pass**: 12/12 tests in `internal/tui/quota_test.go` pass.

---

## Verdict

**REQUEST CHANGES**

The two MAJOR issues (table overflow at narrow widths and column misalignment) degrade the experience for users with smaller terminals or when the output is viewed in split-pane editors. The overflow issue is especially problematic because it affects the default behavior when accounts have long names (e.g., email addresses like `emmmmdty@gmail.com` already push the layout to the edge at width=80). The column misalignment is subtle but noticeable when comparing the active account row to others.

The MINOR and NIT issues are polish items that can be addressed in a follow-up.
