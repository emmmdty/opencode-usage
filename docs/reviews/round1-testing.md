# Round 1 Review: Testing & Adversarial Edge Cases

**Reviewer**: Reviewer C — Testing & Adversarial Edge Cases
**Date**: 2026-08-26

---

### [T-001] Client retry logic completely untested
- **Severity**: BLOCKER
- **File(s)**: `internal/client/opencode.go`
- **Description**: `doRequest` implements 3-retry with exponential backoff for HTTP 429/5xx and network errors. Zero tests exercise this path. If the backoff logic, retry count, or status code filtering breaks, no test will catch it.
- **Test Gap**: Test retry on 429, retry on 5xx, no-retry on 401/403, max retries exceeded, network error retries.
- **Suggested Test**:
```go
func TestDoRequestRetriesOn500(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts <= 2 {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"usage":{}}`))
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    body, err := client.doRequest("/usage")
    if err != nil {
        t.Fatalf("expected success after retries, got: %v", err)
    }
    if attempts != 3 {
        t.Errorf("expected 3 attempts, got %d", attempts)
    }
    if len(body) == 0 {
        t.Error("expected non-empty body")
    }
}

func TestDoRequestDoesNotRetryOn401(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        w.WriteHeader(http.StatusUnauthorized)
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    _, err := client.doRequest("/usage")
    if err == nil {
        t.Fatal("expected error for 401")
    }
    if attempts != 1 {
        t.Errorf("expected 1 attempt (no retry on 401), got %d", attempts)
    }
}

func TestDoRequestRetriesOn429(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts <= 2 {
            w.WriteHeader(http.StatusTooManyRequests)
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"usage":{}}`))
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    _, err := client.doRequest("/usage")
    if err != nil {
        t.Fatalf("expected success after retries, got: %v", err)
    }
    if attempts != 3 {
        t.Errorf("expected 3 attempts, got %d", attempts)
    }
}

func TestDoRequestMaxRetriesExceeded(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        w.WriteHeader(http.StatusInternalServerError)
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    _, err := client.doRequest("/usage")
    if err == nil {
        t.Fatal("expected error after max retries")
    }
    if attempts != 4 { // initial + 3 retries
        t.Errorf("expected 4 attempts, got %d", attempts)
    }
}
```

---

### [T-002] Client malformed/empty response body not tested
- **Severity**: MAJOR
- **File(s)**: `internal/client/opencode.go`
- **Description**: `GetUsage` and `GetModels` unmarshal JSON response. If the server returns malformed JSON, empty body, or an unexpected structure, the error path is untested.
- **Test Gap**: Test with invalid JSON, empty body, missing expected fields, wrong JSON structure.
- **Suggested Test**:
```go
func TestGetUsageMalformedJSON(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{invalid json`))
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    _, err := client.GetUsage()
    if err == nil {
        t.Fatal("expected error for malformed JSON")
    }
}

func TestGetUsageEmptyBody(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte{})
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    _, err := client.GetUsage()
    if err == nil {
        t.Fatal("expected error for empty body")
    }
}

func TestGetModelsEmptyData(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
    }))
    defer server.Close()

    client := NewClient("key", server.URL)
    models, err := client.GetModels()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(models) != 0 {
        t.Errorf("expected 0 models, got %d", len(models))
    }
}
```

---

### [T-003] Config migration loses UseMasterPassword field
- **Severity**: BLOCKER
- **File(s)**: `internal/config/config.go`
- **Description**: `migrateConfig` copies Accounts, ColorThresholds, and MaxConcurrentRequests from old config but **never copies `UseMasterPassword`**. If a user has `use_master_password: true` in an older config version, it will be silently dropped after migration. No test covers `migrateConfig` at all.
- **Test Gap**: Test `migrateConfig` preserves all fields, especially `UseMasterPassword`. Test migration from version "0" to "1".
- **Suggested Test**:
```go
func TestMigrateConfigPreservesUseMasterPassword(t *testing.T) {
    old := &Config{
        Version: "0",
        Accounts: map[string]Account{
            "work": {Name: "work", KeyID: "abc"},
        },
        MaxConcurrentRequests: 3,
    }
    enabled := true
    old.UseMasterPassword = &enabled

    migrated := migrateConfig(old)

    if migrated.UseMasterPassword == nil {
        t.Fatal("UseMasterPassword was dropped during migration")
    }
    if *migrated.UseMasterPassword != true {
        t.Errorf("UseMasterPassword should be true, got false")
    }
    if migrated.MaxConcurrentRequests != 3 {
        t.Errorf("MaxConcurrentRequests should be 3, got %d", migrated.MaxConcurrentRequests)
    }
    if _, exists := migrated.Accounts["work"]; !exists {
        t.Error("Account 'work' was lost during migration")
    }
}

func TestMigrateConfigPreservesColorThresholds(t *testing.T) {
    old := &Config{
        Version: "0",
        ColorThresholds: struct {
            Warning int `yaml:"warning"`
            Danger  int `yaml:"danger"`
        }{Warning: 30, Danger: 90},
    }

    migrated := migrateConfig(old)

    if migrated.ColorThresholds.Warning != 30 {
        t.Errorf("Warning threshold should be 30, got %d", migrated.ColorThresholds.Warning)
    }
    if migrated.ColorThresholds.Danger != 90 {
        t.Errorf("Danger threshold should be 90, got %d", migrated.ColorThresholds.Danger)
    }
}
```

---

### [T-004] Config corrupt YAML not tested
- **Severity**: MAJOR
- **File(s)**: `internal/config/config.go`
- **Description**: If the config file contains invalid YAML, `yaml.Unmarshal` will return an error. This path is never tested. Also, partial YAML (valid syntax but missing fields) is untested.
- **Test Gap**: Test with invalid YAML, empty file, partial YAML, binary garbage.
- **Suggested Test**:
```go
func TestLoadOrCreateConfigCorruptYAML(t *testing.T) {
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")
    os.WriteFile(configPath, []byte(`{{{not yaml`), 0600)

    _, err := LoadOrCreateConfig(configPath)
    if err == nil {
        t.Fatal("expected error for corrupt YAML")
    }
}

func TestLoadOrCreateConfigPartialYAML(t *testing.T) {
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")
    os.WriteFile(configPath, []byte(`version: "1"
accounts:
  work:
    key_id: "abc"
`), 0600)

    cfg, err := LoadOrCreateConfig(configPath)
    if err != nil {
        t.Fatalf("partial YAML should still load: %v", err)
    }
    if cfg.MaxConcurrentRequests != 5 {
        t.Errorf("expected default MaxConcurrentRequests=5, got %d", cfg.MaxConcurrentRequests)
    }
}
```

---

### [T-005] Encrypted store account name containing ":" breaks parsing
- **Severity**: MAJOR
- **File(s)**: `internal/auth/encrypted.go:189`
- **Description**: `storeEncrypted` and `getEncrypted` use `strings.SplitN(line, ":", 2)` to parse account:key pairs. If an account name contains ":", the key will be truncated on read. For example, account `prod:us` with key `sk-abc` would be stored as `prod:us:sk-abc`, but `SplitN("prod:us:sk-abc", ":", 2)` yields `["prod", "us:sk-abc"]`, so the account name becomes `prod` and key becomes `us:sk-abc`.
- **Test Gap**: Test with account names containing special characters (":", "\n", "=").
- **Suggested Test**:
```go
func TestStoreEncryptedAccountNameWithColon(t *testing.T) {
    if os.Getenv("TOKEN_USAGE_MASTER_PASSWORD") == "" {
        t.Skip("TOKEN_USAGE_MASTER_PASSWORD not set")
    }
    account := "prod:us"
    apiKey := "sk-test-colon-123456"

    path, _ := getEncryptedPath()
    os.Remove(path)
    defer os.Remove(path)

    if err := storeEncrypted(account, apiKey); err != nil {
        t.Fatalf("storeEncrypted failed: %v", err)
    }

    retrieved, err := getEncrypted(account)
    if err != nil {
        t.Fatalf("getEncrypted failed: %v", err)
    }
    if retrieved != apiKey {
        t.Errorf("expected %s, got %s (colon in account name breaks parsing)", apiKey, retrieved)
    }
}
```

---

### [T-006] Encrypted store concurrent access with global state
- **Severity**: MAJOR
- **File(s)**: `internal/auth/encrypted.go`
- **Description**: `storeEncrypted` and `deleteEncrypted` read-modify-write the same file without file locking. Concurrent goroutines calling these functions simultaneously (e.g., from parallel `account add` commands or the quota command's concurrent key retrieval) could cause data loss. The `passwordOnce` and `cachedMasterPassword` are also package-level globals that leak between tests.
- **Test Gap**: Test concurrent store/delete operations. Test that `passwordOnce` doesn't prevent password changes across test runs.
- **Suggested Test**:
```go
func TestEncryptedConcurrentAccess(t *testing.T) {
    if os.Getenv("TOKEN_USAGE_MASTER_PASSWORD") == "" {
        t.Skip("TOKEN_USAGE_MASTER_PASSWORD not set")
    }
    path, _ := getEncryptedPath()
    os.Remove(path)
    defer os.Remove(path)

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            name := fmt.Sprintf("account-%d", n)
            storeEncrypted(name, fmt.Sprintf("key-%d", n))
        }(i)
    }
    wg.Wait()

    // Verify no data corruption
    for i := 0; i < 10; i++ {
        name := fmt.Sprintf("account-%d", i)
        key, err := getEncrypted(name)
        if err != nil {
            t.Errorf("getEncrypted(%s) failed: %v", name, err)
        }
        if key != fmt.Sprintf("key-%d", i) {
            t.Errorf("expected key-%d, got %s", i, key)
        }
    }
}
```

---

### [T-007] Keyring test skipped in most CI environments
- **Severity**: MAJOR
- **File(s)**: `internal/auth/credential_test.go`
- **Description**: `TestKeyringOperations` skips entirely when keyring is unavailable AND `TOKEN_USAGE_MASTER_PASSWORD` isn't set. On most CI systems and containers, neither is available, so the only test for `StoreAPIKey`/`GetAPIKey`/`DeleteAPIKey` is never executed. This is the most critical storage path.
- **Test Gap**: Either mock the keyring interface or use `SetUseMasterPassword(false)` to force encrypted file path. Test the encrypted fallback path directly.
- **Suggested Test**:
```go
func TestCredentialStoreFallbackPath(t *testing.T) {
    // Force encrypted file path regardless of keyring availability
    enabled := false
    SetUseMasterPassword(&enabled)
    defer SetUseMasterPassword(nil)

    // Reset password cache for test isolation
    // Note: passwordOnce is a sync.Once and can't be reset.
    // This test would need to be in a subprocess or the code refactored.

    path, _ := getEncryptedPath()
    os.Remove(path)
    defer os.Remove(path)

    account := "test-fallback"
    apiKey := "sk-fallback-test"

    if err := StoreAPIKey("token-usage", account, apiKey); err != nil {
        t.Fatalf("StoreAPIKey failed: %v", err)
    }

    got, err := GetAPIKey("token-usage", account)
    if err != nil {
        t.Fatalf("GetAPIKey failed: %v", err)
    }
    if got != apiKey {
        t.Errorf("expected %s, got %s", apiKey, got)
    }
}
```

---

### [T-008] ExtractKeyID edge cases not tested
- **Severity**: MINOR
- **File(s**: `internal/auth/credential.go:83-88`
- **Description**: `ExtractKeyID` only tests keys longer than 6 chars and a 5-char key. Missing: empty string, exactly 6 chars (boundary), exactly 7 chars, string with only 1-2 chars.
- **Test Gap**: Test boundary conditions for ExtractKeyID.
- **Suggested Test**:
```go
func TestExtractKeyIDEdgeCases(t *testing.T) {
    tests := []struct {
        name, key, want string
    }{
        {"empty", "", ""},
        {"1 char", "a", "a"},
        {"6 chars exact", "123456", "123456"},
        {"7 chars", "1234567", "234567"},
        {"unicode", "sk-中文测试123", "试123"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ExtractKeyID(tt.key)
            if got != tt.want {
                t.Errorf("ExtractKeyID(%q) = %q, want %q", tt.key, got, tt.want)
            }
        })
    }
}
```

---

### [T-009] Validator env var fallback path not actually tested
- **Severity**: MAJOR
- **File(s)**: `internal/auth/validator_test.go:89-102`
- **Description**: `TestValidateAPIKeyDefaultBaseURL` calls `ValidateAPIKey("test-key", "")` and only checks that a response is returned. It doesn't set `TOKEN_USAGE_BASE_URL` env var, so it only tests the default URL fallback (which hits the real network and gets a connection error). The env var path is never tested.
- **Test Gap**: Set `TOKEN_USAGE_BASE_URL` env var to a test server and verify requests go there.
- **Suggested Test**:
```go
func TestValidateAPIKeyEnvBaseURL(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/usage" {
            t.Errorf("expected path /usage, got %s", r.URL.Path)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    os.Setenv("TOKEN_USAGE_BASE_URL", server.URL)
    defer os.Unsetenv("TOKEN_USAGE_BASE_URL")

    resp, err := ValidateAPIKey("test-key", "")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !resp.Valid {
        t.Error("expected valid=true when using env var base URL")
    }
}
```

---

### [T-010] Config permission error and read-only file not tested
- **Severity**: MINOR
- **File(s)**: `internal/config/config.go`
- **Description**: If the config file exists but is not readable (permission denied), or exists but the directory is not writable (for saveConfig), the error path is untested.
- **Test Gap**: Test with read-only config file, non-writable directory.
- **Suggested Test**:
```go
func TestLoadOrCreateConfigPermissionDenied(t *testing.T) {
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")
    os.WriteFile(configPath, []byte(`version: "1"`), 0000)

    _, err := LoadOrCreateConfig(configPath)
    if err == nil {
        t.Fatal("expected error for unreadable config file")
    }
}
```

---

### [T-011] No command-level tests exist
- **Severity**: BLOCKER
- **File(s)**: `internal/cmd/quota.go`, `internal/cmd/account.go`, `internal/cmd/models.go`, `internal/cmd/doctor.go`, `internal/cmd/alias.go`, `internal/cmd/current.go`
- **Description**: The entire `internal/cmd` package has zero test files. This package contains the most complex business logic: concurrent quota fetching with goroutines and channels, account CRUD operations, shell alias manipulation (file I/O with pattern matching), JSON output mode, file output mode. None of this is tested.
- **Test Gap**: Test at minimum: quota with mock HTTP server, account add/remove/list, JSON output format, file output, alias install/uninstall with temp files, current command with/without auth.json.
- **Suggested Test** (quota command):
```go
func TestRunQuotaOverviewJSON(t *testing.T) {
    // Setup temp config and auth files
    tmpHome := t.TempDir()
    os.Setenv("HOME", tmpHome)
    defer os.Unsetenv("HOME")

    // Create config with one account
    configDir := filepath.Join(tmpHome, ".config", "token-usage")
    os.MkdirAll(configDir, 0700)
    cfg := `version: "1"
accounts:
  test:
    name: test
    key_id: "abc123"
`
    os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(cfg), 0600)

    // Create mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "usage": map[string]interface{}{
                "rolling": map[string]interface{}{"percent": 50, "status": "ok"},
                "weekly":  map[string]interface{}{"percent": 30, "status": "ok"},
                "monthly": map[string]interface{}{"percent": 20, "status": "ok"},
            },
        })
    }))
    defer server.Close()

    // This would require refactoring runQuotaOverview to accept a base URL parameter
    // or mocking the client factory. Currently it's not testable.
    // This is itself a design issue worth noting.
}
```

---

### [T-012] Alias file manipulation not tested
- **Severity**: MAJOR
- **File(s)**: `internal/cmd/alias.go`
- **Description**: `aliasExists`, `removeAlias`, and the install/uninstall commands do file I/O on shell RC files. These could corrupt `.bashrc`/`.zshrc` if the pattern matching is wrong. No test exists for any of these functions.
- **Test Gap**: Test `aliasExists` with various file contents, `removeAlias` preserves surrounding lines, install doesn't duplicate aliases.
- **Suggested Test**:
```go
func TestAliasExists(t *testing.T) {
    tmpFile := filepath.Join(t.TempDir(), ".bashrc")
    content := "# some comment\nalias ll='ls -la'\n# token-usage alias\nalias tu='token-usage'\n"
    os.WriteFile(tmpFile, []byte(content), 0644)

    if !aliasExists(tmpFile, "tu") {
        t.Error("expected alias 'tu' to exist")
    }
    if aliasExists(tmpFile, "nonexistent") {
        t.Error("expected alias 'nonexistent' to not exist")
    }
}

func TestRemoveAliasPreservesOtherContent(t *testing.T) {
    tmpFile := filepath.Join(t.TempDir(), ".bashrc")
    content := "# some comment\nalias ll='ls -la'\n# token-usage alias\nalias tu='token-usage'\n# another alias\nalias gp='git push'\n"
    os.WriteFile(tmpFile, []byte(content), 0644)

    if err := removeAlias(tmpFile, "tu"); err != nil {
        t.Fatalf("removeAlias failed: %v", err)
    }

    result, _ := os.ReadFile(tmpFile)
    resultStr := string(result)

    if strings.Contains(resultStr, "alias tu=") {
        t.Error("alias 'tu' should have been removed")
    }
    if !strings.Contains(resultStr, "alias ll=") {
        t.Error("alias 'll' should be preserved")
    }
    if !strings.Contains(resultStr, "alias gp=") {
        t.Error("alias 'gp' should be preserved")
    }
    // The comment "# token-usage alias" should also be removed
    if strings.Contains(resultStr, "token-usage alias") {
        t.Error("associated comment should have been removed")
    }
}
```

---

### [T-013] TUI NO_COLOR mode not verified in output
- **Severity**: MINOR
- **File(s)**: `internal/tui/theme.go`, `internal/tui/quota_test.go`
- **Description**: `TestProgressBars` calls `DisableColor()` but doesn't verify that ANSI escape codes are absent from the output. A color-enabled theme would include `\033[` sequences. No test asserts the absence of ANSI codes.
- **Test Gap**: Call `DisableColor()`, render output, assert no ANSI escape sequences present.
- **Suggested Test**:
```go
func TestNoColorModeProducesNoANSI(t *testing.T) {
    DisableColor()
    defer func() { isColorEnabled = true }() // restore

    results := []AccountResult{
        {Name: "test", Usage: &models.Usage{
            Rolling: models.QuotaWindow{Percent: 90, ResetsAt: time.Now().Add(1 * time.Hour)},
            Weekly:  models.QuotaWindow{Percent: 80, ResetsAt: time.Now().Add(3 * time.Hour)},
            Monthly: models.QuotaWindow{Percent: 70, ResetsAt: time.Now().Add(5 * time.Hour)},
        }},
    }
    output := FormatQuotaOverview(results, DefaultQuotaStyle(), "")

    if strings.Contains(output, "\033[") {
        t.Errorf("output contains ANSI escape codes in NO_COLOR mode: %q", output)
    }
}
```

---

### [T-014] TUI narrow terminal compact mode switching not tested properly
- **Severity**: MINOR
- **File(s)**: `internal/tui/quota_test.go`
- **Description**: `TestFormatQuotaOverviewCompactWidth` sets width to 50 but the assertion only checks if the account name appears. It doesn't verify that compact format was actually used (no table headers, different layout).
- **Test Gap**: Verify compact output lacks table headers like "ACCOUNT", "5H", "Weekly", "Monthly".
- **Suggested Test**:
```go
func TestCompactModeDoesNotHaveTableHeaders(t *testing.T) {
    SetTerminalWidth(40)
    defer SetTerminalWidth(80)

    results := []AccountResult{
        {Name: "work", Usage: &models.Usage{
            Rolling: models.QuotaWindow{Percent: 35, ResetsAt: time.Now().Add(8 * time.Hour)},
            Weekly:  models.QuotaWindow{Percent: 12, ResetsAt: time.Now().Add(5 * time.Hour)},
            Monthly: models.QuotaWindow{Percent: 8, ResetsAt: time.Now().Add(23 * time.Hour)},
        }},
    }
    output := FormatQuotaOverview(results, DefaultQuotaStyle(), "")

    // Compact mode should NOT have table headers
    if strings.Contains(output, "ACCOUNT") {
        t.Error("compact mode should not contain table header 'ACCOUNT'")
    }
    if strings.Contains(output, "Weekly") {
        t.Error("compact mode should not contain table header 'Weekly'")
    }
    // But should still show the short form "W:" for weekly
    if !strings.Contains(output, "W:") {
        t.Error("compact mode should show 'W:' abbreviation")
    }
}
```

---

### [T-015] TestPadRight has weak assertions
- **Severity**: MINOR
- **File(s)**: `internal/tui/quota_test.go:134-159`
- **Description**: `TestPadRight` manually counts runes and checks if the result is shorter than input. It doesn't use the `runewidth` library to verify the result is the correct visual width. The test for `"测试"` expects width 6 but only checks `len(result) >= len(input)`, which is a byte-count check, not a visual width check.
- **Test Gap**: Use `runewidth.StringWidth()` in assertions to verify visual width matches expected.
- **Suggested Test**:
```go
func TestPadRightVisualWidth(t *testing.T) {
    tests := []struct {
        input string
        width int
    }{
        {"abc", 6},
        {"测试", 6},
        {"hello", 3},
        {"abc测试", 10},
    }
    for _, tt := range tests {
        result := padRight(tt.input, tt.width)
        rw := runewidth.StringWidth(result)
        if rw != tt.width && rw > len(tt.input) {
            t.Errorf("padRight(%q, %d): visual width = %d, want %d",
                tt.input, tt.width, rw, tt.width)
        }
    }
}
```

---

### [T-016] Integration test only tests --help
- **Severity**: MAJOR
- **File(s)**: `test/integration_test.go`
- **Description**: The integration test builds the binary and only runs `--help`. It doesn't test any actual functionality: quota, models, account, doctor, version, or JSON output. A regression in the main command flow would not be caught.
- **Test Gap**: Test at minimum: `version` subcommand, `--json` flag output format, `account list` with empty config, error handling for missing accounts.
- **Suggested Test**:
```go
func TestCLIVersion(t *testing.T) {
    binary := buildBinary(t)
    cmd := exec.Command(binary, "version")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("version command failed: %v", err)
    }
    if !strings.Contains(string(output), "token-usage") {
        t.Errorf("expected 'token-usage' in version output, got: %s", output)
    }
}

func TestCLINoAccountsJSON(t *testing.T) {
    tmpHome := t.TempDir()
    binary := buildBinary(t)

    configDir := filepath.Join(tmpHome, ".config", "token-usage")
    os.MkdirAll(configDir, 0700)
    os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(`version: "1"\naccounts: {}`), 0600)

    cmd := exec.Command(binary, "--json")
    cmd.Env = append(os.Environ(), "HOME="+tmpHome)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("quota --json failed: %v", err)
    }

    var resp map[string]interface{}
    if err := json.Unmarshal(output, &resp); err != nil {
        t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
    }
    if resp["version"] != "1" {
        t.Errorf("expected version '1' in JSON, got %v", resp["version"])
    }
}
```

---

### [T-017] formatResetTime missing zero-minutes edge case
- **Severity**: NIT
- **File(s)**: `internal/tui/quota.go:253`
- **Description**: `formatResetTime` returns `"<1m"` when `minutes == 0` but duration is positive (i.e., seconds only). The test for "30 minutes" checks for "m" but doesn't verify the exact format. The `<1m` case is tested implicitly via "expired" but not explicitly for the zero-but-positive case.
- **Test Gap**: Test with a time 30 seconds in the future.
- **Suggested Test**:
```go
func TestFormatResetTimeSecondsOnly(t *testing.T) {
    result := formatResetTime(time.Now().Add(30 * time.Second))
    if result != "<1m" {
        t.Errorf("expected '<1m' for 30 seconds, got %q", result)
    }
}
```

---

### [T-018] computeSummary edge cases not tested
- **Severity**: MINOR
- **File(s)**: `internal/tui/quota.go:281-319`
- **Description**: `computeSummary` is tested with 3 accounts (1 healthy, 1 warning, 1 critical). Not tested: all healthy, all warnings, all errors, all critical (high percent + no errors), exactly at threshold boundaries (50%, 80%).
- **Test Gap**: Test boundary conditions for health classification.
- **Suggested Test**:
```go
func TestComputeSummaryAllHealthy(t *testing.T) {
    results := []AccountResult{
        {Name: "a", Usage: &models.Usage{
            Rolling: models.QuotaWindow{Percent: 49},
            Weekly:  models.QuotaWindow{Percent: 49},
            Monthly: models.QuotaWindow{Percent: 49},
        }},
    }
    summary := computeSummary(results)
    if !strings.Contains(summary, "1 healthy") {
        t.Errorf("expected '1 healthy', got: %s", summary)
    }
    if strings.Contains(summary, "warning") || strings.Contains(summary, "critical") {
        t.Errorf("should have no warnings or criticals, got: %s", summary)
    }
}

func TestComputeSummaryBoundaryWarning(t *testing.T) {
    results := []AccountResult{
        {Name: "a", Usage: &models.Usage{
            Rolling: models.QuotaWindow{Percent: 50},
            Weekly:  models.QuotaWindow{Percent: 0},
            Monthly: models.QuotaWindow{Percent: 0},
        }},
    }
    summary := computeSummary(results)
    if !strings.Contains(summary, "1 warning") {
        t.Errorf("expected '1 warning' at 50%%, got: %s", summary)
    }
}

func TestComputeSummaryBoundaryCritical(t *testing.T) {
    results := []AccountResult{
        {Name: "a", Usage: &models.Usage{
            Rolling: models.QuotaWindow{Percent: 80},
            Weekly:  models.QuotaWindow{Percent: 0},
            Monthly: models.QuotaWindow{Percent: 0},
        }},
    }
    summary := computeSummary(results)
    if !strings.Contains(summary, "1 critical") {
        t.Errorf("expected '1 critical' at 80%%, got: %s", summary)
    }
}
```

---

### [T-019] findBestAccount all-errored case not tested
- **Severity**: MINOR
- **File(s)**: `internal/tui/quota.go:321-342`
- **Description**: `findBestAccount` skips errored accounts. If ALL accounts have errors, it returns `""`. This path is tested implicitly (the errored account in the test), but the explicit case of "no valid accounts at all" is not tested.
- **Test Gap**: Test with only errored accounts.
- **Suggested Test**:
```go
func TestFindBestAccountAllErrors(t *testing.T) {
    results := []AccountResult{
        {Name: "a", Error: "timeout"},
        {Name: "b", Error: "HTTP 401"},
    }
    best := findBestAccount(results)
    if best != "" {
        t.Errorf("expected empty string when all accounts errored, got %q", best)
    }
}
```

---

### [T-020] Concurrent quota fetching race condition risk untested
- **Severity**: MAJOR
- **File(s)**: `internal/cmd/quota.go:86-147`
- **Description**: `runQuotaOverview` spawns goroutines to fetch quota for multiple accounts concurrently using a semaphore. Results are collected via a channel and then sorted. The sort is on a slice built from channel receives (which are already complete by the sort call). However, the goroutines access `auth.GetAPIKey` which uses global keyring state, and `resolveCurrentAccount` reads the filesystem. If two commands run simultaneously, the config file could be read/written concurrently. No test exercises the concurrent path.
- **Test Gap**: Test with multiple accounts where some succeed and some fail. Test that output is deterministic regardless of goroutine scheduling.
- **Suggested Test**:
```go
func TestQuotaConcurrentResultsDeterministic(t *testing.T) {
    // Test that with multiple goroutines, the output is always sorted by name
    results := []accountResult{}
    names := []string{"zebra", "alpha", "middle"}
    for _, name := range names {
        results = append(results, accountResult{Name: name, Usage: &models.Usage{
            Rolling: models.QuotaWindow{Percent: 10},
        }})
    }
    sort.Slice(results, func(i, j int) bool {
        return results[i].Name < results[j].Name
    })
    if results[0].Name != "alpha" || results[1].Name != "middle" || results[2].Name != "zebra" {
        t.Errorf("results not sorted: %v", results)
    }
}
```

---

### [T-021] Unreachable code in doRequest
- **Severity**: NIT
- **File(s)**: `internal/client/opencode.go:75`
- **Description**: Line 75 (`return nil, fmt.Errorf("API error: max retries exceeded")`) is unreachable. The `for` loop has `return` statements in every branch (network error, OK, non-retryable error). The loop variable `attempt` goes from 0 to `maxRetries` (inclusive, 4 iterations), and every iteration returns. This dead code will never be executed.
- **Test Gap**: N/A (this is a code quality issue, not a test gap). Noted for awareness.
- **Suggested Test**: N/A

---

### [T-022] ValidateAPIKeyTestValidationResponseJSON tests useless behavior
- **Severity**: MINOR
- **File(s)**: `internal/auth/validator_test.go:104-128`
- **Description**: `TestValidationResponseJSON` tests JSON marshaling of `ValidationResponse`, but the struct has **no JSON tags**. The test passes because Go's default encoding uses PascalCase field names, but the test doesn't verify the actual JSON structure is useful for API consumers. It also doesn't test any actual validation behavior.
- **Test Gap**: Remove this test or add JSON tags to `ValidationResponse` and test the actual serialized format.
- **Suggested Test**: Either add JSON tags and verify the output format, or remove this test.

---

### [T-023] SetTerminalWidth with 0 or negative not guarded
- **Severity**: NIT
- **File(s)**: `internal/tui/theme.go:32-36`
- **Description**: `SetTerminalWidth` checks `w > 0`, which correctly rejects 0 and negative values. However, no test verifies this guard. If someone removes the guard, no test would fail.
- **Test Gap**: Test that `SetTerminalWidth(0)` and `SetTerminalWidth(-1)` don't change the width.
- **Suggested Test**:
```go
func TestSetTerminalWidthRejectsInvalid(t *testing.T) {
    original := GetTerminalWidth()
    SetTerminalWidth(0)
    if GetTerminalWidth() != original {
        t.Error("SetTerminalWidth(0) should not change width")
    }
    SetTerminalWidth(-1)
    if GetTerminalWidth() != original {
        t.Error("SetTerminalWidth(-1) should not change width")
    }
}
```

---

### [T-024] No test for renderBar width=0
- **Severity**: MINOR
- **File(s)**: `internal/tui/quota.go:204-223`
- **Description**: `renderBar` with `width=0` would produce `filled=0`, `empty=0`, resulting in an empty string. This edge case is untested.
- **Test Gap**: Test `renderBar(percent, 0, ...)` returns empty string.
- **Suggested Test**:
```go
func TestRenderBarZeroWidth(t *testing.T) {
    DisableColor()
    bar := renderBar(50, 0, DefaultQuotaStyle(), NewTheme())
    if bar != "" {
        t.Errorf("expected empty bar for width=0, got %q", bar)
    }
}
```

---

### [T-025] No test for formatResetTime exactly 24 hours
- **Severity**: NIT
- **File(s)**: `internal/tui/quota.go:237-257`
- **Description**: `formatResetTime` has special handling for days, hours, minutes. The boundary between "hours only" and "days + hours" (exactly 24h) is not tested. At exactly 24h, `days=1, hours=0` should produce "1d0h".
- **Test Gap**: Test with exactly 24 hours, exactly 1 hour, exactly 1 minute.
- **Suggested Test**:
```go
func TestFormatResetTimeExactBoundaries(t *testing.T) {
    tests := []struct {
        name     string
        input    time.Time
        expected string
    }{
        {"exactly 24h", time.Now().Add(24 * time.Hour), "1d0h"},
        {"exactly 1h", time.Now().Add(1 * time.Hour), "1h0m"},
        {"exactly 1m", time.Now().Add(1 * time.Minute), "1m"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := formatResetTime(tt.input)
            if got != tt.expected {
                t.Errorf("formatResetTime = %q, want %q", got, tt.expected)
            }
        })
    }
}
```

---

### [T-026] Global state pollution between tests
- **Severity**: MAJOR
- **File(s)**: `internal/tui/theme.go`, `internal/auth/encrypted.go`
- **Description**: Multiple packages use package-level mutable state: `isColorEnabled`, `terminalWidth` in tui; `cachedMasterPassword`, `passwordOnce`, `useMasterPassword` in auth. `passwordOnce` is a `sync.Once` that **cannot be reset** between tests, meaning once `doInitMasterPassword` runs in any test, all subsequent tests in the same process are affected. `TestProgressBars` calls `DisableColor()` and relies on the global state, which could affect other tests.
- **Test Gap**: Tests should use subprocess isolation (via `exec.Command("go", "test", "-run", ...)`) for tests that depend on `passwordOnce`, or the code should be refactored to accept dependencies via parameters.
- **Suggested Test**: Document the limitation and add a comment. For critical tests, use `TestMain` with subprocess isolation.

---

## Verdict

**REQUEST CHANGES**

The test suite has several critical gaps:

1. **Zero tests for the entire `internal/cmd` package** (BLOCKER) — this is where most business logic lives, including concurrent fetching, file I/O, and JSON output.
2. **Client retry logic completely untested** (BLOCKER) — 3-retry exponential backoff with status-code-aware decisions is a complex behavior with no tests.
3. **Config migration drops `UseMasterPassword`** (BLOCKER) — a real bug that no test catches.
4. **Encrypted storage account name parsing bug with ":"** (MAJOR) — a real bug in production code.
5. **Keyring and encrypted store tests skip in most environments** (MAJOR) — critical storage paths are untested on CI.
6. **No integration test beyond `--help`** (MAJOR) — integration coverage is effectively zero.

The existing tests that do exist are reasonable for what they cover, but the coverage is far too narrow. The TUI tests are the strongest module, with good edge case coverage for CJK, compact mode, and summary computation. However, even here, ANSI code absence in NO_COLOR mode is not verified.

Recommended priority:
1. Fix the `migrateConfig` bug (drops `UseMasterPassword`)
2. Fix the `:` parsing bug in encrypted store
3. Add client retry tests
4. Add `cmd` package tests (at minimum quota with mock server)
5. Add config migration tests
6. Improve integration test coverage
