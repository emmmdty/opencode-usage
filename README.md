# opencode-usage

Multi-provider AI coding tool usage monitor — query usage, available models, and quota information across OpenCode, Claude, Codex, and Volcengine.

## Features

- **Multi-provider support** — Monitor OpenCode, Claude, Codex, and Volcengine usage in one place
- **Multi-account management** — Add, remove, list, export, and import multiple accounts
- **Account switching** — Switch active account with interactive menu or direct selection
- **Quota monitoring** — View 5-hour rolling, weekly, and monthly usage across all providers
- **Model listing** — See available models for your plan
- **Current config** — Display the active configuration with provider details
- **Diagnostics** — Run `doctor` to check configuration and connectivity
- **Shell aliases** — Install/uninstall the `ou` shortcut
- **JSON output** — Machine-readable output with `--json`
- **Secure storage** — API keys stored in system keyring, with fallback to encrypted config
- **Concurrent queries** — Parallel quota fetching with configurable concurrency

## Supported Providers

| Provider | Auth Method | Quota Windows |
|----------|-------------|---------------|
| **OpenCode** | API Key | 5h / Weekly / Monthly |
| **Claude** | OAuth (auto-detect) | 5h / 7d |
| **Codex** | OAuth (auto-detect) | 5h / 7d |
| **Volcengine** | API Key | 5h / Weekly / Monthly |

## Installation

### Go install

```bash
go install github.com/emmmdty/opencode-usage/cmd/opencode-usage@latest
```

Requires Go 1.26.6+. If `~/go/bin` is not in your PATH, use:

```bash
GOBIN=~/.local/bin go install github.com/emmmdty/opencode-usage/cmd/opencode-usage@latest
```

### Download binary

Download the latest release from [GitHub Releases](https://github.com/emmmdty/opencode-usage/releases).

Available for Linux, macOS, and Windows (amd64/arm64).

### Build from source

```bash
git clone https://github.com/emmmdty/opencode-usage.git
cd opencode-usage
go build -o opencode-usage ./cmd/opencode-usage/
```

## Quick start

```bash
# Just run it — shows usage across all configured providers
opencode-usage providers

# Or use the alias
ou
```

## Usage

### Multi-provider view

```bash
# View usage across all providers
opencode-usage providers

# JSON output
opencode-usage providers --json
```

### Account management (OpenCode)

```bash
# Add an account (interactive prompt)
opencode-usage account add

# List all accounts (-> marks the current one)
opencode-usage account list

# Switch active account (interactive menu)
opencode-usage account switch

# Export accounts (names + key IDs only, no secrets)
opencode-usage account export

# Import accounts from file
opencode-usage account import accounts.json
```

### Quota (legacy OpenCode view)

```bash
# View quota for all accounts
opencode-usage quota

# View quota for a specific account
opencode-usage quota -n work

# JSON output
opencode-usage quota --json
```

### Models

```bash
# List available models
opencode-usage models
```

### Diagnostics

```bash
# Check configuration and connectivity
opencode-usage doctor
```

### Shell alias

```bash
# Install the 'ou' alias
opencode-usage alias install

# Uninstall the 'ou' alias
opencode-usage alias uninstall
```

### Version & updates

```bash
opencode-usage version
opencode-usage update
```

## Short aliases

| Command | Alias |
|---------|-------|
| `providers` | `p` |
| `account` | `a` |
| `account add` | `aa` |
| `account list` | `al` |
| `account remove` | `ar` |
| `account export` | `ae` |
| `account import` | `ai` |
| `account switch` | `sw` |
| `quota` | `q` |
| `models` | `m` |
| `current` | `cc` |
| `opencode-usage` | `ou` (shell alias) |

## Global flags

| Flag | Description |
|------|-------------|
| `-j, --json` | JSON output |
| `-n, --account` | Specify account |
| `-o, --output` | Output to file |
| `--no-color` | Disable color output |

## Configuration

Config file: `~/.config/opencode-usage/config.yaml`

```yaml
version: "2"
accounts:
  work:
    name: work
    key_id: "sk-...abc123"
    created_at: 2026-01-01T00:00:00Z
    last_verified: 2026-01-15T12:00:00Z
providers:
  claude:
    enabled: true
    creds_path: ~/.claude/.credentials.json
  codex:
    enabled: true
    auth_path: ~/.codex/auth.json
  volcengine:
    enabled: false
    api_key: "your-api-key"
color_thresholds:
  warning: 50
  danger: 80
max_concurrent_requests: 5
use_master_password: false
```

### Provider configuration

#### Claude

Claude credentials are auto-detected from `~/.claude/.credentials.json`. Run `claude` CLI first to authenticate.

```yaml
providers:
  claude:
    enabled: true
    creds_path: ~/.claude/.credentials.json
```

#### Codex

Codex credentials are auto-detected from `~/.codex/auth.json`. Run `codex` CLI first to authenticate.

```yaml
providers:
  codex:
    enabled: true
    auth_path: ~/.codex/auth.json
```

#### Volcengine

Requires API Key from [Volcengine Console](https://console.volcengine.com/).

```yaml
providers:
  volcengine:
    enabled: true
    api_key: "your-api-key"
```

### Configuration options

| Option | Default | Description |
|--------|---------|-------------|
| `providers.*.enabled` | true | Enable/disable provider |
| `providers.*.endpoint` | - | Custom API endpoint |
| `color_thresholds.warning` | 50 | Quota percentage to trigger warning color |
| `color_thresholds.danger` | 80 | Quota percentage to trigger danger color |
| `max_concurrent_requests` | 5 | Max parallel API requests |
| `use_master_password` | false | Use custom master password for encrypted storage |

## Environment variables

| Variable | Description |
|----------|-------------|
| `NO_COLOR` | Disable color output (see [no-color.org](https://no-color.org)) |
| `OPENCODE_USAGE_MASTER_PASSWORD` | Master password for encrypted storage |
| `OPENCODE_USAGE_KEYRING_PASSWORD` | Password for keyring file backend |

## Security

- API keys are stored in the system keyring (Linux: Secret Service, macOS: Keychain, Windows: Credential Manager)
- Config file only stores the last 6 characters of each key ID (`sk-...XXXXXX`)
- Falls back to AES-256-GCM encrypted config file if keyring is unavailable
- Master password option for encrypted config
- Config and secrets files use restrictive permissions (0600)

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (usage, auth, network, config, or keyring) |

## License

MIT
