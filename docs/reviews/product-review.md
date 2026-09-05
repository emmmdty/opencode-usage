# Product Review: token-usage

**Reviewer:** Subagent F (Product Reviewer)
**Date:** 2026-08-26
**Environment:** Linux, standard terminal

---

## Scenario 1: First-time user (`--help`)

**Output:**
```
Query OpenCode Go plan usage across multiple accounts, view available models and quota information.

Usage:
  token-usage [flags]
  token-usage [command]

Aliases:
  token-usage, tu

Available Commands:
  account     Manage OpenCode Go accounts
  alias       Manage shell aliases
  completion  Generate the autocompletion script for the specified shell
  current     Show current opencode configuration
  doctor      Diagnose configuration and connectivity issues
  help        Help about any command
  models      List available models
  quota       View quota usage
  update      Check for new releases on GitHub
  version     Show version information

Flags:
  -n, --account string   specify account
  -h, --help             help for token-usage
  -j, --json             JSON output
      --no-color         disable color output
  -o, --output string    output to file
  -v, --version          version for token-usage

Use "token-usage [command] --help" for more information about a command.
```

**Assessment:**
- **Better than previous?** Yes. The one-liner summary at the top clearly explains what the tool does. The `tu` alias is discoverable. The subcommand list is well-organized and scannable.
- **Still confusing?** Minor: "Manage shell aliases" is opaque — a user wouldn't know what aliases without trying it. The `current` command doesn't say "of opencode" in the description. But these are nitpicks.
- **What would keep a user using it?** The help is concise enough that a user can get started in one glance. The `account add` flow is discoverable from the help tree.

## Scenario 2: Multi-account overview (default command)

**Output:**
```
  OpenCode Go  refreshed 16:19:20

  ACCOUNT               5H                  Weekly              Monthly
  ───────────────────────────────────────────────────────────────────────────────────
    emmmmdty@gmail.com    ░░░░   0% 4h59m     ░░░░  24% 4d15h     ██░░  62% 21d18h
    emmmtjk@gmail.com     ░░░░   0% 2h53m     ░░░░   1% 4d15h     ██░░  61% 18d22h
    jktong2026@163.com    ░░░░   4% 3h37m     ░░░░  23% 4d15h     ██░░  58% 26d7h

  3 accounts  3 warning
  Best available: jktong2026@163.com
  Next reset: emmmtjk@gmail.com · 2h53m

  ● healthy  ▲ warning  ● critical  → active
```

**Assessment:**
- **Better than previous?** Significantly. The summary line ("3 accounts 3 warning") and the "Best available" callout are excellent — they answer the user's primary question ("which account should I use right now?") without requiring any analysis. The "Next reset" line tells you when things change. The legend at the bottom explains the symbols.
- **Still confusing?** "5H" as a column header is non-obvious — most users won't know it means "5-hour rolling window." A tooltip or expansion would help. The progress bars use `█` and `░` which are not universally visible in all terminals, but the percentage text is always present so it's fine.
- **What would keep a user using it?** This is the money view. You run it, you know exactly which account to use and when your quota resets. The "Best available" line is the killer feature for multi-account users.

## Scenario 3: Error handling (`account list` with no config, `doctor`)

**`account list` (error case):**
```
Error: failed to read account name: EOF
Error: failed to read account name: EOF
exit status 1
```

Wait — I ran `account add` which requires interactive input. Let me note what I observed:
- `account add` without a name argument shows: `Account name: Error: failed to read account name: EOF` — the error is printed twice (once from the prompt reader, once from the CLI wrapper), which is noisy. The message "failed to read account name: EOF" is technically correct but not user-friendly. A better message: "Account name is required. Usage: token-usage account add <name>"
- `doctor` output is excellent:
  ```
  [OK] Config file          /home/tjk/.config/token-usage/config.yaml
  [OK] Accounts             3 configured
  [!!] Keyring              using encrypted file fallback
  [OK] Network              opencode.ai reachable
  [OK] OpenCode auth        auth.json found

  All checks passed.
  ```
  Clear, scannable, with status indicators. The `[!!]` warning for the keyring fallback is actionable — a user knows it's not ideal but not broken.

**Assessment:**
- **Better than previous?** The `doctor` command is a major improvement — it gives users a self-service diagnostic tool. The `account add` error handling needs work (double-printing errors, EOF message is confusing).
- **Still confusing?** The double error output for `account add` is a UX issue. Users who accidentally run `account add` without a name get two error lines that look like a bug.
- **What would keep a user using it?** `doctor` gives confidence that the tool is working correctly. It's the kind of command users run once during setup and then forget — which is exactly right.

## Scenario 4: JSON scripting (`--json`)

**Output structure:**
```json
{
  "version": "1",
  "accounts": [
    {
      "name": "emmmmdty@gmail.com",
      "quota": {
        "rolling": { "status": "ok", "percent": 0, "resetsAt": "2026-08-26T13:19:22.624Z" },
        "weekly":  { "status": "ok", "percent": 24, "resetsAt": "2026-08-31T00:00:00.624Z" },
        "monthly": { "status": "ok", "percent": 62, "resetsAt": "2026-09-17T02:48:27.624Z" }
      }
    }
  ]
}
```

**Assessment:**
- **Better than previous?** Yes. The JSON is clean, schema-versioned (`"version": "1"`), and has consistent structure across all accounts. The `status` field inside each quota window is useful for programmatic filtering.
- **Still confusing?** The `version` field being a string `"1"` rather than an integer is unconventional but harmless. The JSON doesn't include the "best available" summary that the TUI shows — a scripting user would need to compute it themselves. No `error` field in the account objects (error accounts are shown differently in TUI but the JSON shape doesn't include an `error` key I could see).
- **What would keep a user using it?** This is scriptable. You can pipe it to `jq`, write cron jobs, or build dashboards. The consistent schema means you can rely on it.

## Scenario 5: First-time setup (`account --help`)

**Output:**
```
Manage OpenCode Go accounts

Usage:
  token-usage account [command]

Aliases:
  account, a

Available Commands:
  add         Add a new account
  export      Export account configuration
  import      Import account configuration
  list        List all accounts
  remove      Remove an account

Flags:
  -h, --help   help for account

Global Flags:
  -n, --account string   specify account
  -j, --json             JSON output
      --no-color         disable color output
  -o, --output string    output to file

Use "token-usage account [command] --help" for more information about a command.
```

And `account add --help`:
```
Add a new account

Usage:
  token-usage account add [flags]

Aliases:
  add, aa

Flags:
  -h, --help   help for add

Global Flags:
  -n, --account string   specify account
  -j, --json             JSON output
      --no-color         disable color output
  -o, --output string    output to file
```

**Assessment:**
- **Better than previous?** The flow is discoverable: `--help` → `account --help` → `account add`. The `account add` command uses interactive prompts, which is the right UX for a CLI tool that doesn't want to store keys on the command line (which would leak into shell history).
- **Still confusing?** The `account add --help` doesn't explain what information is needed (API key? email? both?). A user has to just run it and hope for the best. There's no mention of where accounts are stored or what format they use.
- **What would keep a user using it?** The alias `a` for `account` and `aa` for `account add` are excellent for power users. The `import`/`export` commands are a nice touch for users managing multiple machines.

## Scenario 6: Narrow terminal (60 columns)

**Width detection:** The tool uses `term.GetSize()` to detect terminal width at init time, falling back to 80 columns. It has a `formatCompact` mode that activates when width < 60.

**Simulated compact output (from code):**
```
  OpenCode Go

  emmmmdty@gmail.com
    5H: 0%  W: 24%  M: 62%
  emmmtjk@gmail.com
    5H: 0%  W: 1%  M: 61%
  jktong2026@163.com
    5H: 4%  W: 24%  M: 58%
```

**Assessment:**
- **Better than previous?** Yes. The responsive width handling means the tool degrades gracefully on narrow terminals or when piped. The compact mode drops the progress bars and keeps the essential data.
- **Still confusing?** The compact mode drops the "Best available" summary and the "Next reset" info — the two most valuable pieces of information for multi-account users. The `5H` abbreviation becomes even more cryptic in compact mode without the column header context.
- **What would keep a user using it?** The fact that it works at all at 60 columns is a win. Most CLIs either overflow or wrap badly.

---

## Summary Verdict

### Is this meaningfully better than the previous version?

**Yes, across the board.** The key improvements:
1. **"Best available" callout** — answers the user's primary question instantly
2. **`doctor` command** — self-service diagnostics eliminate support burden
3. **Responsive width** — graceful degradation for narrow terminals and pipes
4. **Clean JSON output** — versioned, consistent schema for scripting
5. **Help text quality** — concise, accurate, with useful aliases (`tu`, `a`, `aa`)

### What is still confusing?

1. **`5H` column header** — non-obvious abbreviation for the 5-hour rolling window. Consider "5H Roll" or a tooltip
2. **`account add` error handling** — double error output and unhelpful EOF message when run non-interactively
3. **Compact mode loses context** — drops "Best available" and "Next reset" which are the most valuable summary lines
4. **`account add --help`** doesn't explain what inputs are needed before starting the interactive flow

### What would make a user keep using this tool?

1. **It solves a real problem** — multi-account quota management is genuinely useful, and this tool does it in one glance
2. **The default view is the right view** — running `token-usage` with no arguments gives you everything you need
3. **It stays out of your way** — no config files to edit, no auth tokens to manage manually (the tool discovers them from opencode's existing config)
4. **It scales** — the JSON output and shell aliases mean it fits into power-user workflows without friction

**Bottom line:** This is a well-designed CLI tool that gets the "one command, all answers" philosophy right. The few rough edges (5H abbreviation, compact mode info loss, account add error messages) are minor and fixable. The core UX — run it, know which account to use — is excellent.
