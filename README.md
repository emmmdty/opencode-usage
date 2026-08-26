# opencode-usage

OpenCode Go plan usage query tool — query usage, available models, and quota information across multiple accounts.

## Features

- **Multi-account management** — Add, remove, list, export, and import multiple OpenCode Go accounts
- **Quota monitoring** — View 5-hour rolling, weekly, and monthly usage across all accounts
- **Model listing** — See available models for your plan
- **Current config** — Display the active opencode configuration
- **Diagnostics** — Run `doctor` to check configuration and connectivity
- **Shell aliases** — Install/uninstall the `ou` shortcut
- **JSON output** — Machine-readable output with `--json`
- **Secure storage** — API keys stored in system keyring, with fallback to encrypted config
- **Concurrent queries** — Parallel quota fetching with configurable concurrency

## Installation

### Go install

```bash
go install github.com/emmmdty/opencode-usage/cmd/opencode-usage@latest
```

Requires Go 1.26.6+.

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
# Just run it — shows the quota dashboard for all accounts
opencode-usage

# Or use the alias
ou
```

## Usage

### Account management

```bash
# Add an account (interactive prompt)
opencode-usage account add

# List all accounts
opencode-usage account list

# Remove an account
opencode-usage account remove <name>

# Export accounts (names + key IDs only, no secrets)
opencode-usage account export

# Export to file
opencode-usage account export -o accounts.json

# Import accounts from file
opencode-usage account import accounts.json
```

### Quota

```bash
# View quota for all accounts
opencode-usage quota

# View quota for a specific account
opencode-usage quota -n work

# JSON output
opencode-usage quota --json

# Output to file
opencode-usage quota -o report.txt
```

### Models

```bash
# List available models
opencode-usage models
```

### Current config

```bash
# Show active opencode configuration
opencode-usage current
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
| `account` | `a` |
| `account add` | `aa` |
| `account list` | `al` |
| `account remove` | `ar` |
| `account export` | `ae` |
| `account import` | `ai` |
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
version: "1"
accounts:
  work:
    name: work
    key_id: "sk-...abc123"
    created_at: 2026-01-01T00:00:00Z
    last_verified: 2026-01-15T12:00:00Z
color_thresholds:
  warning: 50
  danger: 80
max_concurrent_requests: 5
use_master_password: false
```

### Configuration options

| Option | Default | Description |
|--------|---------|-------------|
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
| `OPENCODE_USAGE_BASE_URL` | Custom API base URL |

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
