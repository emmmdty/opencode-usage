# Delivery Report — token-usage v0.2.0 UX Refresh

## 1. What Changed

### Core UX Improvements
- **Default `tu` command now shows quota dashboard** — bare `token-usage` immediately shows all accounts, quota, progress bars, best available account, and next reset time
- **Progress bars** — visual `████░░░░` bars alongside percentages for instant scannability
- **Best available account recommendation** — "Best available: personal" callout answers the #1 multi-account question
- **Next reset countdown** — "Next reset: work · 4h30m" tells users when relief comes
- **Current account marker** — active opencode account is identified and displayed
- **Account summary** — "3 accounts  2 healthy  1 warning" overview at a glance
- **Legend** — explains symbols: ● healthy  ▲ warning  ● critical  → active

### Correctness Fixes
- **CJK width calculation** — replaced `len()` (byte count) with `runewidth.StringWidth()` for all column width calculations
- **80-column terminal support** — responsive table sizing fits within 80 columns
- **NO_COLOR compliance** — respects `NO_COLOR` env var and auto-detects non-TTY (pipes/redirects)
- **Token masking** — current config token now fully masked (was leaking suffix)

### New Commands
- **`tu doctor`** — diagnostics: checks config, keyring, network, accounts, opencode auth

### Code Quality
- **Unified design system** — `internal/tui/theme.go` with consistent color tokens
- **Removed duplicate aliases** — eliminated redundant root-level alias commands
- **English error messages** — consistent English across all user-facing text
- **TUI rendering tests** — 13 tests covering CJK, progress bars, error rows, summaries
- **JSON schema versioning** — output wrapped in `{"version": "1", "accounts": [...]}`

### Files Modified
| File | Changes |
|------|---------|
| `internal/tui/quota.go` | Complete rewrite: design system, progress bars, responsive width, CJK support, error styling |
| `internal/tui/theme.go` | New: unified theme with NO_COLOR support |
| `internal/cmd/root.go` | Default to quota, NO_COLOR, isatty detection |
| `internal/cmd/quota.go` | Refactored shared logic, current account resolution, JSON versioning |
| `internal/cmd/account.go` | Fixed alignment, removed duplicate aliases, English text |
| `internal/cmd/doctor.go` | New: diagnostics command |
| `internal/cmd/current.go` | Fixed token masking |
| `internal/cmd/alias.go` | English text |
| `internal/cmd/models.go` | Fixed empty state |

### Files Added
| File | Purpose |
|------|---------|
| `internal/tui/theme.go` | Design system |
| `internal/tui/quota_test.go` | 13 TUI rendering tests |
| `internal/cmd/doctor.go` | Diagnostics command |
| `docs/research/competitive-analysis.md` | Competitive analysis |
| `docs/research/ux-audit.md` | UX audit |
| `docs/research/product-review.md` | Product review |
| `docs/plans/ux-product-refresh.md` | Design document |
| `docs/reviews/visual-qa.md` | Visual QA report |
| `docs/reviews/engineering-review.md` | Engineering review |
| `docs/reviews/product-review.md` | Product QA report |

---

## 2. Competitive Research

| Project | Idea Borrowed | Implementation |
|---------|---------------|----------------|
| xnqycs/OpenCodeGo_Pool | Account health overview | Adapted as summary line + best-available |
| x0c/subswap | Cache with stale fallback | Deferred to v0.3 (not implemented yet) |
| masrurimz/opencode-go-multi-auth | Current account marker | Implemented via auth.json matching |
| slkiser/opencode-quota | Progress bars | Implemented with Unicode block chars |
| ZeroClue/llm-quota-tracker | Daily allowance metric | Deferred to v0.3 |
| Rishabh-Bajpai/opencode-go-multi-auth | "React, don't predict" | Followed — only report actual API responses |

**Rejected:** Web dashboards, plugin architecture, proxy daemons, multi-provider support, Docker, background daemons.

---

## 3. UX Before → After

| Aspect | Before | After |
|--------|--------|-------|
| Default `tu` | Shows cobra help | Shows quota dashboard with all accounts |
| Quota display | `"35% (剩余8h12m)"` | `████░░░░ 35% 4h30m` + best available + next reset |
| CJK names | Broken alignment | Correct alignment via runewidth |
| Error rows | `"错误: API error: HTTP 401"` | `✗ API error: HTTP 401` (red, distinct) |
| Pipe output | ANSI escape codes | Clean text (auto NO_COLOR) |
| Current account | Unknown | Shown with `→` marker |
| Best account | Manual analysis needed | "Best available: personal" |
| Reset time | `"剩余8h12m"` | `4h30m` |
| Diagnostics | None | `tu doctor` checks everything |
| Account list | Chinese text, misaligned | English, aligned, relative timestamps |
| Help text | Chinese | English |
| JSON output | Raw `[]accountResult` | Versioned `{"version":"1","accounts":[...]}` |

---

## 4. New Commands / Behavior

| Change | User Impact |
|--------|-------------|
| `tu` (bare) | Now shows quota dashboard (was: help) |
| `tu doctor` | New: diagnostics command |
| `NO_COLOR=1 tu` | Colors disabled |
| `tu \| cat` | Clean output, no ANSI |
| `tu --json` | Versioned JSON schema |
| `tu account list` | Relative timestamps, English text, current account marker |
| `tu` (80-col) | Table fits within terminal width |

---

## 5. Architecture

### New package
- `internal/tui/theme.go` — Design tokens, NO_COLOR handling, terminal width detection

### Refactored
- `internal/tui/quota.go` — Separated rendering from business logic; responsive width; CJK-safe
- `internal/cmd/quota.go` — Shared `runQuotaOverview()` function used by both root and quota commands
- `internal/cmd/root.go` — Default command, NO_COLOR, isatty

### Dependencies
- Added: `github.com/mattn/go-runewidth` (CJK width calculation)
- Already present: `golang.org/x/term` (terminal detection)

---

## 6. Tests

```bash
$ go vet ./...
VET OK

$ go test ./...
ok  github.com/token-usage/internal/auth
ok  github.com/token-usage/internal/client
ok  github.com/token-usage/internal/config
ok  github.com/token-usage/internal/tui (13 tests)
ok  github.com/token-usage/test

$ go test -race ./...
ok  (all packages — no races)

$ go build ./cmd/token-usage/
BUILD OK

$ wc -L <(NO_COLOR=1 go run ./cmd/token-usage/)
80  # fits in 80 columns
```

---

## 7. Independent Reviews

| Reviewer | Verdict | Key Issues |
|----------|---------|------------|
| Visual QA (D) | PASS after fix | 80-col overflow fixed; all other commands clean |
| Engineering (E) | APPROVE after fix | Dead code removed, token masking fixed, no races |
| Product QA (F) | PASS | "Best available" and doctor command are the killer features |

---

## 8. Remaining Issues (Deferred to v0.3)

| Issue | Priority | Notes |
|-------|----------|-------|
| Watch/refresh mode | Medium | `tu --watch 30s` |
| Cache with stale fallback | Medium | File-based cache, 30s TTL |
| Account switching | Medium | `tu account use <name>` |
| Configurable thresholds | Low | CLI flags for warning/danger |
| Fish/PowerShell alias | Low | `alias install` only does bash/zsh |
| XDG support | Low | `$XDG_CONFIG_HOME` override |
| Account health badges | Low | HEALTHY/WARNING/EXHAUSTED status |

---

## 9. Suggested Roadmap

### v0.2.0 (current)
- Default dashboard
- Progress bars
- Best available
- NO_COLOR
- Doctor command
- CJK support
- 80-col support

### v0.3.0
- Cache with stale fallback
- Account switching (`tu account use <name>`)
- Watch mode (`tu --watch`)
- Configurable thresholds
- Shell completions (bash/zsh/fish)

### v1.0
- Historical usage tracking (SQLite)
- Auto-discover from auth.json
- Daily allowance metric
- Full test coverage

---

## 10. Git Summary

### Modified (9 files)
- `go.mod` — runewidth dependency
- `go.sum` — checksums
- `internal/cmd/account.go` — English, fixed alignment, removed duplicates
- `internal/cmd/alias.go` — English text
- `internal/cmd/current.go` — Fixed token masking
- `internal/cmd/models.go` — Fixed empty state
- `internal/cmd/quota.go` — Refactored shared logic, JSON versioning
- `internal/cmd/root.go` — Default command, NO_COLOR
- `internal/tui/quota.go` — Complete rewrite with design system

### Added (6 files)
- `internal/tui/theme.go` — Design tokens
- `internal/tui/quota_test.go` — 13 TUI tests
- `internal/cmd/doctor.go` — Diagnostics command
- `docs/research/` — 3 research files
- `docs/reviews/` — 3 review files
- `docs/plans/` — 1 design document

### Deleted (0 files)
- None

### No secrets, no temp files, no build artifacts in diff.
