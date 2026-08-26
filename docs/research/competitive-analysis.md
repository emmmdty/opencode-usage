# Competitive Analysis: OpenCode Go Usage/Quota Ecosystem

**Date:** 2026-08-26  
**Researcher:** Subagent A (Competitive Research)

---

## Executive Summary

The OpenCode Go usage/quota ecosystem has **significant fragmentation** across 5 distinct product categories. The current project (opencode-usage) occupies the **CLI-only quota viewer** niche. The most successful projects (slkiser/opencode-quota with 916 stars, PhilippPolterauer/opencode-quotas with 60 stars) are **OpenCode plugins** that display quota inside the TUI. The most architecturally sophisticated projects (Rishabh-Bajpai/opencode-go-multi-auth, x0c/subswap) are **multi-account routers/proxies**.

**Key finding:** No project successfully combines "lightweight Go CLI" with "multi-account quota awareness." This is opencode-usage's differentiation opportunity.

---

## Project Taxonomy

| Category | Projects | Core Function | Stars |
|----------|----------|---------------|-------|
| **Quota Viewer (Plugin)** | slkiser/opencode-quota, PhilippPolterauer/opencode-quotas, whosydd/opencode-quota, itzmegas/opencode-quota, yehezkielgunawan/opencode-quota-tracker | Display quota in OpenCode TUI/sidebar | 916, 60, ~5, ~5, ~2 |
| **Multi-Account Router (Proxy)** | Rishabh-Bajpai/opencode-go-multi-auth, David-Nahorniak/opencode-go-multi-auth, dhaalves/opencode-swap, xnqycs/OpenCodeGo_Pool | Proxy routing across multiple API keys | 8, 0, 7, 38 |
| **Multi-Account Switcher (CLI)** | masrurimz/opencode-go-multi-auth, x0c/subswap | CLI-based account switching/rotation | 12, 2 |
| **Usage Analytics (CLI+Web)** | gaboe/opencode-usage | Token usage tracking from SQLite DB | 20 |
| **Quota Dashboard (TUI)** | ZeroClue/llm-quota-tracker, opgginc/opencode-bar | Multi-provider terminal/menu bar dashboard | ~5, ~10 |
| **opencode-usage (current)** | emmmdty/opencode-usage | Go CLI for multi-account quota queries | — |

---

## Detailed Project Analysis

### 1. slkiser/opencode-quota ⭐ 916

**Product Positioning:** The de facto standard OpenCode quota plugin. Supports 20+ providers across American and Chinese markets.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | TypeScript OpenCode plugin + CLI. Reads from local `opencode.db`, remote APIs, and dashboard scraping |
| **UX Highlights** | Sidebar panel, TUI toast, compact status line, slash commands (`/quota`, `/tokens_today`). Zero context window pollution (commands inject ignored inline messages) |
| **Technical Highlights** | Provider auto-detection, custom provider framework (`provider add`), JSON/OTel output, pricing from models.dev, deterministic inline output shared by TUI/Desktop |
| **Security** | OAuth flows, local token storage, no external data |
| **Strengths** | Massive provider coverage, plugin ecosystem maturity, 724 commits, active maintenance |
| **Weaknesses** | Requires OpenCode running (can't check quota standalone). Node.js dependency. No multi-account rotation — just visibility |
| **What NOT to copy** | Provider sprawl — 20+ providers means high maintenance burden and complexity |

**Takeaway for opencode-usage:** The "zero context window pollution" pattern is excellent — injecting ignored inline messages rather than consuming model context. Consider if opencode-usage ever integrates with OpenCode TUI.

---

### 2. PhilippPolterauer/opencode-quotas ⭐ 60

**Product Positioning:** "The ultimate usage dashboard plugin" with smart predictions.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | TypeScript plugin with CLI fallback. Supports Antigravity, Codex, GitHub Copilot |
| **UX Highlights** | ANSI progress bars in chat footer, ETTL (Estimated Time To Limit) using linear regression, model-aware filtering, configurable bar styles |
| **Technical Highlights** | Linear regression on usage history for prediction, pattern-based aggregation, spike detection, JSON schema validation |
| **Strengths** | Prediction (ETTL) is unique and genuinely useful. Aggregation strategies (most_critical, max, min, mean, median) are sophisticated |
| **Weaknesses** | Limited provider support (3 providers). Plugin-only — no standalone CLI |

**Takeaway for opencode-usage:** The ETTL prediction pattern (linear regression on usage history) is a strong differentiator. opencode-usage could add this without complexity — just track quota snapshots over time in SQLite.

---

### 3. x0c/subswap ⭐ 2

**Product Positioning:** Multi-provider account switcher (Claude, Codex, ChatGPT, Kimi, Cursor, OpenCode Go).

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | Rust CLI with provider crates. Daemon mode (`subswapd`) for auto-swap. Atomic rollback-able swap with snapshots |
| **UX Highlights** | `subswap` one-liner for status + auto-swap. Account-isolated environments (`run`, `shell`, `env`). Settle-grace after manual swap |
| **Technical Highlights** | Provider trait architecture, file-backed credentials (0600), keyring migration, quota cache with stale fallback, network-independent manual swap |
| **Design Invariants** | 1) `swap` never depends on quota lookups. 2) Secrets out of registry. 3) Swap is atomic/rollback-able. 4) Adding a provider = adding a crate |
| **OpenCode Go Support** | M7 milestone: slot-only `auth.json` swap, 5h/weekly/monthly quota, auto-swap and isolated run |
| **Strengths** | Rust = single binary, zero runtime deps. Provider architecture is clean. Manual swap works offline |
| **Weaknesses** | 2 stars = low adoption. Daemon adds complexity. No TUI/dashboard — pure CLI |

**Takeaway for opencode-usage:** The "manual swap never depends on quota lookups" invariant is critical for reliability. The provider crate architecture is overkill for opencode-usage's scope but the design principle of "escape hatch that always works" is worth adopting.

---

### 4. Rishabh-Bajpai/opencode-go-multi-auth ⭐ 8

**Product Positioning:** TypeScript plugin + proxy router pooling multiple OpenCode Go + Zen subscriptions.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | OpenCode plugin that auto-starts a local proxy (port 18905) + web dashboard (port 18904). React 19 + TanStack Query frontend |
| **UX Highlights** | 7-page web dashboard: Overview, Accounts, Strategy, Tokens, Logs, Models, Settings. Strategy explainer in UI. Copy-snippet for model drift detection |
| **Technical Highlights** | 3 routing strategies (Priority Failover, Round Robin, Weighted Cycle). Session stickiness. Circuit breaker. Stream-aware usage tracking. AES-256-GCM encrypted key storage. Push notifications via ntfy |
| **Design Philosophy** | "React, don't predict" — never estimates quota, only marks keys exhausted on actual 402/429 |
| **Strengths** | Professional-grade dashboard. Dual upstream (Go + Zen). Cache-preserving header passthrough |
| **Weaknesses** | Requires Node.js 22+. Two processes (plugin + proxy). Plugin auto-start means implicit background daemon. Educational-purpose-only disclaimer |

**Takeaway for opencode-usage:** The "React, don't predict" philosophy is excellent — opencode-usage should never estimate quota, only report actual API responses. The per-model rate card for cost estimation (with `~` prefix and tooltip) is a smart UX pattern.

---

### 5. David-Nahorniak/opencode-go-multi-auth (Fork) ⭐ 0

**Product Positioning:** Fork of Rishabh-Bajpai's project with Docker, Apprise notifications, proxy access token, and Go plan usage API.

| Dimension | Assessment |
|-----------|------------|
| **Key Additions** | Docker containerization, Apprise (100+ notification services), proxy access token (timing-safe compare), Go plan usage API (`GET /zen/go/v1/usage`) |
| **Go Plan Usage** | Calls upstream usage API with account's own Go API key. Returns rolling/weekly/monthly usage (percent + ISO reset timestamps). Detects invalid/revoked key and unactivated plan |
| **Strengths** | Go plan usage endpoint is the same API opencode-usage uses. Docker support enables headless deployment |
| **Weaknesses** | Mirror repository with 0 stars. Feature additions complicate the original |

**Takeaway for opencode-usage:** The Go plan usage API module (`opencode-go-usage.ts`) is a clean reference implementation. The API returns `keyInvalid`, `planNotActivated` states — opencode-usage should handle these same edge cases.

---

### 6. dhaalves/opencode-swap ⭐ 7

**Product Positioning:** "Zero-dependency" local proxy for automatic API key rotation on 429.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | Pure Node.js (zero npm deps). HTTP proxy with round-robin + 429 failover. `keys.json` persistence |
| **UX Highlights** | `oswap import` pulls keys from OpenCode's auth.json. `oswap install` backs up and patches opencode.json. `oswap status` shows live pool state. SSE byte-for-byte streaming |
| **Technical Highlights** | Cooldown honors `Retry-After`/`retry-after-ms`. 401/403 park key 1 hour. 5xx park 30s. All keys cooling = wait up to `--max-wait-ms` then 429. Retries only before first response byte |
| **Strengths** | Truly zero dependencies. Transparent proxy (model names pass through). Monitoring endpoints (`/oswap/health`, `/oswap/status`). Test suite with fake upstream |
| **Weaknesses** | Requires separate process (`oswap serve`). No web dashboard. Single machine only |

**Takeaway for opencode-usage:** The "retries only before first response byte" pattern is critical for SSE streaming correctness. The `oswap import` command that auto-discovers keys from OpenCode's auth.json is a great UX pattern — opencode-usage should offer similar auto-discovery.

---

### 7. masrurimz/opencode-go-multi-auth ⭐ 12

**Product Positioning:** TypeScript OpenCode plugin for multi-account rotation with per-process stickiness.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | Pure plugin (no proxy daemon). Per-process account stickiness (same account for entire process lifetime). Round-robin rotation across processes |
| **UX Highlights** | Install via `opencode plugin github:masrurimz/opencode-go-multi-auth --global`. Auth settings integration. CLI subcommands via OpenCode auth panel |
| **Technical Highlights** | Atomic JSON writes (tmp + rename). 0o600 permissions. Automatic `.bak.<timestamp>` backup on corruption |
| **Strengths** | No daemon needed. Plugin-native. Simple architecture. 23 tests |
| **Weaknesses** | Per-process stickiness = must restart OpenCode to rotate. No web dashboard |

**Takeaway for opencode-usage:** The atomic JSON write pattern (tmp + rename + backup) is a good data safety practice. The `opencode plugin` global install command is the cleanest installation UX in the ecosystem.

---

### 8. gaboe/opencode-usage ⭐ 20

**Product Positioning:** CLI tool for tracking OpenCode usage/costs from SQLite DB, with Commander web dashboard.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | TypeScript/Bun. Reads `opencode.db` SQLite. Commander web UI with quota status, account management, plugins |
| **UX Highlights** | Watch mode (`--watch`), date range filtering (`--since 7d`), monthly aggregation, provider breakdown. Web dashboard with dark mode, auto-refresh |
| **Technical Highlights** | Reads OpenCode session data directly. Model pricing for cost estimation. Plugin system for custom providers |
| **Strengths** | SQLite-based = no API calls needed for historical usage. Web dashboard is comprehensive |
| **Weaknesses** | Different purpose (historical usage analytics vs. live quota). No Go plan quota API integration |

**Takeaway for opencode-usage:** The SQLite-based historical usage tracking is complementary to opencode-usage's live quota queries. Consider whether opencode-usage should also read `opencode.db` for usage history.

---

### 9. ZeroClue/llm-quota-tracker

**Product Positioning:** Multi-provider terminal dashboard with auto-discovery.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | Python (uv/Rich). Auto-discovers tools via `which`, known paths, credential files. SQLite history for pace tracking |
| **UX Highlights** | Unified dashboard with progress bars, daily allowance calculation (remaining / days until reset), pace tracking with overspend warnings |
| **Technical Highlights** | Provider registry with probe functions. Discovery scanner. Budget + burst calculators. Rich tables |
| **Strengths** | Auto-discovery is magical. Daily allowance is a genuinely useful metric. Multi-provider |
| **Weaknesses** | Python runtime dependency. Limited provider support |

**Takeaway for opencode-usage:** The "daily allowance" metric (remaining quota / days until reset) is simple but powerful. opencode-usage could compute this for each window. Auto-discovery of credentials from OpenCode's auth.json is also worth adopting.

---

### 10. opgginc/opencode-bar

**Product Positioning:** macOS menu bar provider usage tracker.

| Dimension | Assessment |
|-----------|------------|
| **Architecture** | Swift/macOS native. Auto-detects all providers. Menu bar + submenus |
| **UX Highlights** | Color-coded progress (green → yellow → orange → red). Auth source labels. EOM prediction with weighted averages. Daily history graphs |
| **Technical Highlights** | Multi-source account discovery (OpenCode auth, VS Code, Keychain, browser cookies). Parallel fetching. Smart caching |
| **Strengths** | Native macOS = zero resource overhead. EOM prediction is sophisticated. Multi-source auth detection |
| **Weaknesses** | macOS only. Not cross-platform |

**Takeaway for opencode-usage:** The EOM (End Of Month) prediction using weighted averages is a strong pattern. The "auth source labels" showing where each credential was detected is excellent transparency UX.

---

## Cross-Project UX Patterns

### What Makes These Designs Good UX?

| Pattern | Where Used | Why It Works |
|---------|-----------|--------------|
| **Slash commands in TUI** | slkiser/opencode-quota, whosydd/opencode-quota | Zero friction — no context switch |
| **Sidebar panels** | slkiser/opencode-quota, itzmegas/opencode-quota | Persistent visibility without clutter |
| **Progress bars with color** | All quota viewers | Instant visual scan of status |
| **Reset countdown timers** | All projects | Actionable — tells you when relief comes |
| **JSON output for scripts** | opencode-usage, gaboe/opencode-usage, slkiser/opencode-quota | Composability with other tools |
| **Auto-discovery of credentials** | dhaalves/opencode-swap, ZeroClue/llm-quota-tracker, opgginc/opencode-bar | Zero-config first run |
| **"React, don't predict"** | Rishabh-Bajpai/opencode-go-multi-auth | Never show wrong quota data |
| **Escape hatch that always works** | x0c/subswap | Manual swap never depends on quota API |
| **Zero context window pollution** | slkiser/opencode-quota | Commands don't waste model tokens |
| **ETTL/prediction** | PhilippPolterauer/opencode-quotas, opgginc/opencode-bar | Proactive — warns before exhaustion |

---

## Which Designs Fit opencode-usage?

### ✅ Strong Fit (Adopt)

| Pattern | Source | Rationale |
|---------|--------|-----------|
| **Auto-discovery from auth.json** | dhaalves/opencode-swap, ZeroClue/llm-quota-tracker | opencode-usage should detect existing keys automatically |
| **Daily allowance metric** | ZeroClue/llm-quota-tracker | Simple, actionable: "You can use X per day for the rest of the window" |
| **"React, don't predict"** | Rishabh-Bajpai/opencode-go-multi-auth | Report actual API responses, never estimate |
| **Per-model rate card with `~` prefix** | Rishabh-Bajpai/opencode-go-multi-auth | Cost estimation with clear "estimated" indicator |
| **Atomic JSON writes (tmp + rename)** | masrurimz/opencode-go-multi-auth | Data safety for config file |
| **`oswap import` pattern** | dhaalves/opencode-swap | Pull keys from OpenCode's auth.json + account.json |
| **Exit code table** | opencode-usage (current) | Already implemented, keep it |
| **Keyring + encrypted fallback** | opencode-usage (current) | Already implemented, keep it |

### ⚠️ Partial Fit (Consider Carefully)

| Pattern | Source | Consideration |
|---------|--------|---------------|
| **SQLite usage history** | gaboe/opencode-usage, ZeroClue/llm-quota-tracker | Complementary data, but adds complexity. Consider as optional feature |
| **ETTL prediction** | PhilippPolterauer/opencode-quotas | Useful but requires historical data. Could add in v0.3 |
| **Watch mode** | gaboe/opencode-usage | Useful for monitoring, but opencode-usage's strength is "run, check, done" |
| **JSON output** | All CLI tools | Already implemented, keep it |

### ❌ Poor Fit (Do NOT Copy)

| Pattern | Source | Why It Breaks |
|---------|--------|---------------|
| **OpenCode plugin architecture** | slkiser/opencode-quota, all plugins | opencode-usage is a standalone CLI, not a plugin. Adding plugin API ties it to OpenCode's internals |
| **Web dashboard (React/Vite)** | Rishabh-Bajpai/opencode-go-multi-auth, gaboe/opencode-usage | Breaks "lightweight CLI" positioning. Requires Node.js |
| **Proxy daemon** | dhaalves/opencode-swap, Rishabh-Bajpai/opencode-go-multi-auth | Requires background process. opencode-usage should be fire-and-forget |
| **Multi-provider support (20+)** | slkiser/opencode-quota | opencode-usage is scoped to OpenCode Go only. Don't scope-creep |
| **Rust rewrite** | x0c/subswap | Go is the right choice for this project's scope. Single binary without Rust toolchain |
| **Docker containerization** | David-Nahorniak/opencode-go-multi-auth | opencode-usage is a local CLI tool, not a service |
| **Background daemon** | x0c/subswap, Rishabh-Bajpai/opencode-go-multi-auth | Breaks the "run and forget" mental model |

---

## Engineering Practices Worth Adopting

### From slkiser/opencode-quota (916 ⭐)
- **Provider template** (`contributing/provider-template/`) for community contributions
- **CI/CD with GoReleaser** pattern (used by many projects)
- **`update --dry-run`** for safe self-updates
- **Biome/linting** for consistent code style

### From x0c/subswap
- **Design invariants document** — explicitly state load-bearing design decisions
- **Provider trait architecture** — clean interface for extending to new providers
- **Quota result cache with stale fallback** — always show something, refresh in background

### From dhaalves/opencode-swap
- **Zero dependencies** — pure Node.js runtime. opencode-usage's Go approach already achieves this
- **Monitoring endpoints** — `/health` and `/status` for observability
- **Test suite with fake upstream** — no network required for tests

### From masrurimz/opencode-go-multi-auth
- **Atomic JSON writes** with timestamped backups
- **23 tests across 4 modules** — good test coverage for a small project

---

## Market Gap Analysis

| Need | Current Solutions | Gap |
|------|------------------|-----|
| **Standalone quota CLI (no OpenCode dependency)** | opencode-usage (Go), dhaalves/opencode-swap (Node) | Only 2 projects. opencode-usage is the only Go option |
| **Multi-account quota viewer (not router)** | opencode-usage (Go), slkiser/opencode-quota (TS plugin) | opencode-usage is unique as standalone CLI |
| **Go binary, zero runtime deps** | opencode-usage only | Clear differentiation |
| **OpenCode Go-specific (not multi-provider)** | opencode-usage only | All other projects try to support everything |
| **Quota + cost estimation in CLI** | opencode-usage, gaboe/opencode-usage | Only 2 projects combine both |

---

## Recommendations for opencode-usage

### Positioning
**"The lightweight Go CLI for OpenCode Go quota and cost tracking across multiple accounts."**

Not a plugin. Not a router. Not a daemon. A CLI you run, check, and close.

### Short-term (v0.2)
1. **Auto-discover keys from auth.json** — `opencode-usage account import --auto`
2. **Daily allowance metric** — "35% used, 12% left, resets in 5d6h → safe to use 2.4%/day"
3. **Per-model cost estimation** with `~` prefix for estimates

### Medium-term (v0.3)
1. **SQLite usage history** — track quota snapshots for ETTL prediction
2. **Watch mode** — `opencode-usage quota --watch` with auto-refresh
3. **Shell completion** (bash/zsh/fish)

### Do NOT add
- Web dashboard
- Plugin architecture
- Multi-provider support
- Background daemon
- Docker support

---

## Appendix: Star Count Comparison

| Project | Stars | Forks | Language | Last Commit |
|---------|-------|-------|----------|-------------|
| slkiser/opencode-quota | 916 | 95 | TypeScript | Active (724 commits) |
| PhilippPolterauer/opencode-quotas | 60 | 9 | TypeScript | Active |
| gaboe/opencode-usage | 20 | 5 | TypeScript | Active |
| masrurimz/opencode-go-multi-auth | 12 | 3 | TypeScript | Active |
| Rishabh-Bajpai/opencode-go-multi-auth | 8 | 3 | TypeScript | Active |
| dhaalves/opencode-swap | 7 | 0 | TypeScript | Recent |
| xnqycs/OpenCodeGo_Pool | 38 | 7 | Go+React | Active |
| x0c/subswap | 2 | 1 | Rust | Active |
| David-Nahorniak/opencode-go-multi-auth | 0 | 0 | TypeScript | Mirror |

**Language breakdown:** TypeScript dominates (8 projects). Go is rare (2: OpenCodeGo_Pool, opencode-usage). Rust has 1 project (subswap).

**opencode-usage's Go advantage:** The ecosystem is TypeScript-heavy. A well-crafted Go CLI stands out as the "just works" option with zero runtime dependencies.
