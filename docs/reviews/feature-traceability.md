# Feature Traceability Matrix

**Date**: 2026-08-26
**Version**: 0.2.0

| Feature | Source | Implementation | Test | Docs | Runtime | Status |
|---------|--------|---------------|------|------|---------|--------|
| Multi-account quota | requirement | quota.go | tui/quota_test.go | README.md | verified | PASS |
| account add | requirement | account.go:25-96 | — | README.md | verified | PASS |
| account list | requirement | account.go:98-161 | — | README.md | verified | PASS |
| account remove | requirement | account.go:191-227 | — | README.md | verified | PASS |
| account import | requirement | account.go:296-378 | — | README.md | verified | PASS |
| account export | requirement | account.go:247-294 | — | README.md | verified | PASS |
| quota | requirement | quota.go | tui/quota_test.go | README.md | verified | PASS |
| quota --account | requirement | quota.go:54-59 | — | README.md | verified | PASS |
| quota --json | requirement | quota.go:141-143 | — | README.md | verified | PASS |
| models | requirement | models.go | client/opencode_test.go | README.md | verified | PASS |
| current | requirement | current.go:17-62 | — | README.md | verified | PASS |
| doctor | requirement | doctor.go:14-85 | — | README.md | verified | PASS |
| alias install | requirement | alias.go:17-64 | — | README.md | verified | PASS |
| alias uninstall | requirement | alias.go:67-97 | — | README.md | verified | PASS |
| version | requirement | root.go:58-64 | — | README.md | verified | PASS |
| update | requirement | root.go:66-75 | — | README.md | verified | PASS |
| NO_COLOR | UX audit | root.go:31, theme.go:17 | — | README.md | verified | PASS |
| --no-color flag | UX audit | root.go:81 | — | README.md | verified | PASS |
| --output / -o | UX audit | root.go:80 | — | README.md | verified | PASS |
| --json flag | requirement | root.go:78 | — | README.md | verified | PASS |
| config | requirement | config/config.go | config/config_test.go | README.md | verified | PASS |
| color_thresholds | UX audit | config/config.go:37-38 | — | README.md | verified | PASS |
| max_concurrent_requests | requirement | config/config.go:35 | — | README.md | verified | PASS |
| use_master_password | security | config/config.go:21 | — | README.md | verified | PASS |
| keyring storage | security | auth/credential.go | auth/credential_test.go | README.md | verified | PASS |
| encrypted fallback | security | auth/encrypted.go | auth/encrypted_test.go | README.md | verified | PASS |
| master password env var | security | auth/encrypted.go:55 | — | README.md | verified | PASS |
| Keyring password env var | security | auth/credential.go:22 | — | README.md | verified | PASS |
| Base URL env var | security | auth/validator.go:23 | — | README.md | verified | PASS |
| Exit codes | requirement | main.go:13 | — | README.md | verified | PASS |
| CJK width | UX audit | tui/quota.go | tui/quota_test.go | — | verified | PASS |
| Unicode support | UX audit | tui/quota.go | tui/quota_test.go | — | verified | PASS |
| Non-TTY detection | UX audit | root.go:35-38 | — | — | verified | PASS |
| Concurrent queries | requirement | quota.go:80-84 | — | README.md | verified | PASS |
| Partial failure | requirement | quota.go:93-118 | tui/quota_test.go | — | verified | PASS |
| Cross-platform builds | requirement | — | — | README.md | verified | PASS |
| Atomic file writes | security | auth/encrypted.go, config/config.go | — | — | verified | PASS |
| License file | legal | LICENSE | — | README.md | verified | PASS |
