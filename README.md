# opencode-usage

OpenCode Go plan usage query tool — query usage, available models, and quota information across multiple accounts.

## Features

- **Multi-account management** — Add, remove, list, export, and import multiple OpenCode Go accounts
- **Quota monitoring** — View 5-hour rolling, weekly, and monthly usage across all accounts
- **Model listing** — See available models for your plan
- **Current config** — Display the active opencode configuration
- **Shell aliases** — Install/uninstall the `ou` shortcut
- **JSON output** — Machine-readable output with `--json`
- **Secure storage** — API keys stored in system keyring, with fallback to encrypted config
- **Concurrent queries** — Parallel quota fetching with configurable concurrency

## Installation

### Go install

```bash
go install github.com/emmmdty/opencode-usage/cmd/opencode-usage@latest
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

## Configuration

Config file: `~/.config/opencode-usage/config.yaml`

## Security

- API keys are stored in the system keyring (Linux: Secret Service, macOS: Keychain, Windows: Credential Manager)
- Config file only stores the last 6 characters of each key ID (`sk-...XXXXXX`)
- Falls back to encrypted config file if keyring is unavailable
- Master password option for encrypted config

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error |
| 3 | Authentication failure |
| 4 | Network error |
| 5 | Config file error |
| 6 | Config file not found |
| 7 | Keyring unavailable |

## License

MIT
