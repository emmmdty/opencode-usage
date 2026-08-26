# UX/Product Refresh Plan — opencode-usage v0.2.0

## Current State

| Area | Status | Issues |
|------|--------|--------|
| **Product** | Basic quota viewer | Root command does nothing; no dashboard; no current-account marker |
| **CLI** | Working commands | Duplicate alias paths; Chinese-only text; no `--watch` |
| **TUI** | Basic table | No progress bars; CJK width broken; no loading state; no responsive width |
| **Visual** | ANSI256 colors | No NO_COLOR; no isatty; no color legend; errors not distinct |
| **Engineering** | Solid foundation | Config loaded twice; no TUI tests; no doctor; JSON schema unversioned |
| **Security** | Good | Keyring + encrypted fallback already solid |

## Competitive Lessons

### Borrow the idea
- **Auto-discovery from auth.json** (opencode-swap) — zero-config first run
- **Daily allowance metric** (llm-quota-tracker) — "safe to use X%/day"
- **"React, don't predict"** (opencode-go-multi-auth) — report actual API responses only

### Adapt the idea
- **Progress bars** (all quota viewers) — adapt for CLI context, not plugin
- **Cache with stale fallback** (subswap) — file-based, not daemon
- **Doctor command** (subswap) — lightweight diagnostic, not full health system

### Reject the idea
- Web dashboards, plugin architecture, proxy daemons, multi-provider support
- Docker, background daemons, complex historical analytics

## Implementation Plan

### P0 — This Release

1. **Default `ou` → quota dashboard**
   - Refactor quota logic into shared function
   - Root command runs quota overview when no subcommand given

2. **Fix CJK width calculation**
   - Replace `len()` with `runewidth.StringWidth()` in all TUI code
   - Add `runewidth` dependency

3. **NO_COLOR / non-TTY support**
   - Check `NO_COLOR` env var
   - Auto-detect non-TTY via `golang.org/x/term`
   - Disable colors for pipes/redirects

4. **Current account marker**
   - Read `~/.local/share/opencode/auth.json` to find active key
   - Match against config accounts by key ID
   - Display `*` or `→` marker in quota table

### P1 — This Release (if time permits)

5. **Progress bars** — `████░░░░` visual bars alongside percentages
6. **Error styling** — Red `✗` prefix for error rows
7. **Loading indicator** — "Fetching N accounts..." during concurrent fetch
8. **Responsive width** — Detect terminal width, adapt layout
9. **Remove duplicate aliases** — Clean up root-level alias commands
10. **Fix account list alignment** — Use runewidth for all columns
11. **JSON schema versioning** — Wrap output in `{"version": "1", "accounts": [...]}`
12. **Doctor command** — `ou doctor` checks config, keyring, network, API

### NOT in this release
- Watch mode, cache, account switching, health status, best-account recommendation
- These are v0.3.0 features

## Architecture Changes

### New dependency
- `github.com/mattn/go-runewidth` — CJK-aware string width calculation

### File changes
- `internal/tui/quota.go` — Major rewrite (design system, progress bars, responsive width)
- `internal/tui/theme.go` — New: unified design tokens
- `internal/cmd/root.go` — Default to quota; NO_COLOR; isatty
- `internal/cmd/quota.go` — Refactor shared logic; loading indicator
- `internal/cmd/account.go` — Fix alignment; remove duplicate aliases
- `internal/cmd/doctor.go` — New: diagnostics command
- `internal/cmd/models.go` — Fix empty state
- `internal/client/opencode.go` — Add current-account resolution
- `go.mod` — Add runewidth dependency

## Success Criteria

1. `ou` shows a useful quota dashboard (not help)
2. CJK account names display correctly in all tables
3. `NO_COLOR=1 ou q` produces clean output
4. `ou q | cat` produces clean output (no ANSI)
5. Active account is clearly marked
6. Progress bars are scannable at a glance
7. Error rows are visually distinct
8. All existing tests still pass
9. New tests for TUI rendering, width calculation, NO_COLOR
