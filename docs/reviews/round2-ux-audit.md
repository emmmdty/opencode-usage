# Round 2 — Fresh UX Audit

**Reviewer:** J (Fresh Eyes)
**Date:** 2026-08-26
**Scope:** CLI/TUI output quality, visual correctness, no-color handling, grammar

---

## Methodology

Built binary, read `internal/tui/quota.go`, `internal/tui/theme.go`, `internal/tui/quota_test.go`, and all `internal/cmd/*.go` files. Ran `go test ./...`. Executed the binary with `--no-color` and varied terminal widths. Wrote targeted edge-case tests for cell overflow and separator width.

---

## Findings

### F-01: Cell content overflows `colWidth` when reset time is long (Severity: Medium)

**Location:** `internal/tui/quota.go:213-218` (`formatQuotaCell`)

The `%-*s` format pads the cell to `colWidth` but does **not** truncate. When cell content (bar + pct + reset) exceeds `colWidth`, the rendered string is wider than intended, causing data rows to extend past the terminal edge.

**Analysis of `formatQuotaCell` output width vs `colWidth`:**

| Terminal width | Name length | `colWidth` | `barWidth` | Cell width (long reset) | Overflow per cell |
|---|---|---|---|---|---|
| 60 | 20 | 8 | 0 | 11–13 | +3 to +5 |
| 70 | 20 | 12 | 0 | 13 | +1 |
| 80 | 20 | 15 | 3 | 16 | +1 |
| 80 | 4 | 20 | 8 | 21 | +1 |

With 3 columns, worst case at width=60 with long names: **+9 chars total overflow** per row. At width=80: +3 chars (marginal).

**Verdict:** BUG — the cell should either truncate content or `colWidth` minimum should account for the minimum cell width (9 chars for bar=0, pct=4, reset=4).

**Fix suggestion:** Enforce `colWidth = max(colWidth, 13)` or truncate `reset` text when `barWidth == 0`.

---

### F-02: `formatCompact` uses ASCII arrow `->` while `formatTable` uses Unicode `→` (Severity: Low)

**Locations:** `quota.go:136` vs `quota.go:189`

Compact mode: `theme.Active.Render("-> ")`
Table mode: `theme.Active.Render("→ ")`

Both mark the current account, but use different arrow styles. This creates a visual inconsistency when the terminal resizes between modes (e.g., during a tmux split).

**Verdict:** INCONSISTENCY — minor, but should use the same symbol for consistency.

---

### F-03: `formatCompact` labels "Best:" vs `formatTable` labels "Best available:" (Severity: Low)

**Locations:** `quota.go:111` vs `quota.go:164`

Table mode: `"Best available: "`
Compact mode: `"Best: "`

Different labels for the same information. The compact mode truncates the label, which is reasonable for narrow terminals, but the inconsistency may confuse users who switch between modes.

**Verdict:** INCONSISTENCY — cosmetic, not a functional bug.

---

### F-04: `go-runewidth` treats `─` (U+2500) as double-width; separator test was a false positive (Severity: Info)

The `go-runewidth` library classifies Box Drawing characters as East Asian Width "Ambiguous" and returns width 2. In actual terminals, `─` renders as a single-width character. The separator line **is** properly constrained to terminal width.

**Verdict:** NOT A BUG — the code is correct; `go-runewidth` is overly conservative for box-drawing characters.

---

### F-05: Separator line is properly constrained to terminal width (Severity: Pass)

**Location:** `internal/tui/quota.go:84-88`

```go
sepLen := nameWidth + 3*colWidth + 10
if sepLen > usable {
    sepLen = usable
}
```

The separator length is clamped to `usable` (= `width - 2`). With the `"  "` prefix (2 chars), total display width ≤ `width`. Verified at widths 60, 70, and 80.

**Verdict:** PASS

---

### F-06: Compact mode includes summary and active account (Severity: Pass)

**Location:** `internal/tui/quota.go:127-171` (`formatCompact`)

Compact mode output includes:
- Per-account usage lines (lines 131–150)
- Summary line via `computeSummary` (line 152–153)
- Active account (lines 155–159)
- Best account (lines 162–165)
- Next reset (lines 166–169)

**Verdict:** PASS

---

### F-07: `computeSummary` correctly distinguishes errors from critical (Severity: Pass)

**Location:** `internal/tui/quota.go:297-343`

```go
if r.Error != "" {
    errors++
    continue  // ← skipped, not compared against thresholds
}
maxPercent := ...  // only computed for non-error results
if maxPercent >= style.DangerThreshold {
    criticals++
}
```

Errors (API failures, auth issues) are counted in a separate bucket and never compared against the danger threshold. Critical means "high usage but API succeeded"; error means "API failed". The summary correctly reports both independently.

**Test confirmation:** `TestComputeSummary` verifies "1 healthy  1 warning  1 error" with 3 accounts (one of each). My edge-case test confirmed "1 healthy  1 critical  1 error" separates all three categories.

**Verdict:** PASS

---

### F-08: `findNextReset` considers all three quota windows (Severity: Pass)

**Location:** `internal/tui/quota.go:368-389`

```go
for _, resetTime := range []time.Time{
    r.Usage.Rolling.ResetsAt,
    r.Usage.Weekly.ResetsAt,
    r.Usage.Monthly.ResetsAt,
} {
```

All three windows are iterated for each non-error account. The earliest reset across all accounts and all windows is selected. Verified that a weekly reset at 2h is chosen over a rolling reset at 10h and a monthly reset at 5d.

**Verdict:** PASS

---

### F-09: Grammar: "1 account" (singular) vs "2 accounts" (plural) (Severity: Pass)

**Location:** `internal/tui/quota.go:338-341`

```go
noun := "accounts"
if total == 1 {
    noun = "account"
}
```

Correctly handles singular/plural for the total count. Sub-counts ("1 healthy", "1 warning", etc.) are always numeric and don't need plural forms.

**Test confirmation:** `TestComputeSummarySingularPlural` verifies "1 account " (with trailing space) for single, "2 accounts" for multiple.

**Verdict:** PASS

---

### F-10: `--no-color` flag works for all commands including doctor (Severity: Pass)

**Location:** `internal/cmd/root.go:30-38`, `internal/cmd/doctor.go:99-104`

- `--no-color` is a `PersistentFlag` on `rootCmd`, available to all subcommands.
- `PersistentPreRun` calls `tui.DisableColor()` and `lipgloss.SetColorProfile(termenv.Ascii)` — affects the TUI theme.
- `doctor` uses its own `doctorTheme` struct with raw ANSI codes, independently checking `noColor` via `newDoctorTheme()`.
- Commands without styling (`models`, `current`, `account list`) use plain `fmt.Printf` and are unaffected by color settings.

**Verified output:**
```
$ token-usage doctor --no-color
  [OK] Config file          /home/tjk/.config/token-usage/config.yaml
  [OK] Accounts             3 configured
  [!!] Keyring              using encrypted file fallback
  [OK] Network              opencode.ai reachable
  [OK] OpenCode auth        auth.json found
```

Plain text, no ANSI escapes, correct bracket-style icons.

**Verdict:** PASS

---

### F-11: Tests all pass (Severity: Pass)

All 12 existing tests in `internal/tui/quota_test.go` pass. Additional edge-case tests for cell overflow and separator width were written and confirmed the findings above.

**Verdict:** PASS

---

## Summary

| # | Finding | Severity | Verdict |
|---|---------|----------|---------|
| F-01 | Cell content overflows `colWidth` at narrow widths | Medium | BUG |
| F-02 | Compact mode uses `->` vs table uses `→` | Low | INCONSISTENCY |
| F-03 | Compact mode labels "Best:" vs table "Best available:" | Low | INCONSISTENCY |
| F-04 | `go-runewidth` over-counts `─` as double-width | Info | False positive |
| F-05 | Separator line constrained to terminal width | — | PASS |
| F-06 | Compact mode includes summary and active account | — | PASS |
| F-07 | `computeSummary` distinguishes errors from critical | — | PASS |
| F-08 | `findNextReset` considers all three quota windows | — | PASS |
| F-09 | Grammar: singular/plural handled correctly | — | PASS |
| F-10 | `--no-color` works for all commands including doctor | — | PASS |
| F-11 | All tests pass | — | PASS |

**Recommended action:** Fix F-01 (cell overflow) — enforce a minimum `colWidth` of 13 or truncate reset text when `barWidth == 0`.
