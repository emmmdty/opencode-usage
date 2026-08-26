# opencode-usage CLI工具设计文档

## 概述

opencode-usage是一个Go CLI工具，用于快速查询多个OpenCode账号下Go计划的使用情况、可用模型和配额信息。

## 版本规划

- **MVP版本（v0.1）**：核心功能 - 账号管理（添加/列表/删除）和配额查询
- **v0.2**：模型列表查询、当前配置显示、JSON输出
- **v0.3**：高级功能 - 导入导出、配额预警、缓存优化

**发布时间线**：
- v0.1：2周开发 + 1周测试 = 3周
- v0.2：1周开发 + 3天测试 = 1.5周
- v0.3：2周开发 + 1周测试 = 3周

**配置文件版本管理**：
- 配置文件包含`version`字段，当前为`"1"`
- 版本升级时自动迁移旧配置文件
- 备份旧配置文件到`config.yaml.bak`

## 技术栈

- **语言**：Go 1.22+
- **TUI框架**：bubbletea
- **安全存储**：系统密钥环（Keychain/Secret Service/Credential Manager）
- **构建工具**：Go modules
- **CI/CD**：GitHub Actions + GoReleaser

## 依赖管理

| 依赖 | 版本 | 用途 |
|------|------|------|
| bubbletea | v0.25+ | TUI框架 |
| bubbles | v0.20+ | TUI组件库 |
| lipgloss | v0.9+ | 样式和布局 |
| 99designs/keyring | v1.2+ | 跨平台密钥存储（统一API） |
| cobra | v1.8+ | 命令行解析 |
| viper | v1.18+ | 配置管理 |

## 项目结构

```
opencode-usage/
├── cmd/
│   └── opencode-usage/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── credential.go      # 凭证管理（密钥环/文件）
│   │   └── validator.go       # API Key验证
│   ├── client/
│   │   └── opencode.go        # OpenCode Go API客户端
│   ├── config/
│   │   └── config.go          # 配置文件管理
│   ├── tui/
│   │   ├── account.go         # 账号管理交互界面
│   │   ├── quota.go           # 配额显示界面
│   │   └── components.go      # 复用TUI组件
│   └── models/
│       └── usage.go           # 数据模型定义
├── pkg/
│   └── alias/
│       └── alias.go           # 命令别名管理
├── go.mod
└── go.sum
```

## 命令设计

**主命令**：`opencode-usage`（别名：`ou`）

**别名安装**：运行`opencode-usage alias install`自动添加shell别名到`~/.bashrc`或`~/.zshrc`

**别名卸载**：运行`opencode-usage alias uninstall`自动移除shell别名

**别名冲突检测**：安装前检测`ou`是否已存在别名，如冲突则提示用户确认或手动添加

**子命令结构**：

| 命令 | 别名 | 功能 | 示例 |
|------|------|------|------|
| `account add` | `aa` | 交互式添加账号 | `ou aa` |
| `account list` | `al` | 查看所有账号（脱敏） | `ou al` |
| `account remove` | `ar` | 删除账号 | `ou ar work` |
| `quota` | `q` | 查看所有账号配额 | `ou q` |
| `quota --account <name>` | `q -n <name>` | 查看特定账号配额 | `ou q -n work` |
| `models` | `m` | 查看可用模型列表 | `ou m` |
| `current` | `cc` | 显示当前opencode配置的账号 | `ou cc` |
| `help` | `h` | 显示帮助信息 | `ou h` |

**全局标志**：
- `--json` / `-j`：JSON格式输出
- `--account` / `-n`：指定账号（避免与别名冲突）
- `--output` / `-o`：输出到文件
- `--no-color`：禁用颜色输出

## 交互式界面设计

### 添加账号流程

```
$ ou aa

┌─────────────────────────────────────────┐
│         添加 OpenCode Go 账号            │
└─────────────────────────────────────────┘

账号名称: work
API Key: ••••••••••••••••••••••••••••••••
验证中... ✓

账号 "work" 已成功保存！
```

### 查看账号列表

```
$ ou al

┌─────────┬────────────────┬────────────┬──────────────┐
│ 名称     │ API Key        │ 状态       │ 上次验证       │
├─────────┼────────────────┼────────────┼──────────────┤
│ work    │ sk-...abc123   │ ✅ 有效    │ 2分钟前        │
│ personal│ sk-...xyz789   │ ✅ 有效    │ 5分钟前        │
└─────────┴────────────────┴────────────┴──────────────┘
```

### 查看配额

```
$ ou q

┌─────────────────────────────────────────────────────────────┐
│                    OpenCode Go 配额概览                      │
├─────────┬─────────────────┬─────────────────┬───────────────┤
│ 账号     │ 5小时滚动        │ 每周             │ 每月          │
├─────────┼─────────────────┼─────────────────┼───────────────┤
│ work    │ 35% (剩余8h12m)  │ 12% (剩余5d6h)  │ 8% (剩余23d)   │
│ personal│ 67% (剩余2h45m)  │ 45% (剩余3d12h) │ 22% (剩余18d)  │
└─────────┴─────────────────┴─────────────────┴───────────────┘
```

## 安全存储设计

**存储方案**：
- **Linux**：使用Secret Service API（gnome-keyring/kwallet）
- **macOS**：使用Keychain
- **Windows**：使用Windows Credential Manager
- **降级方案**：当密钥环不可用时（如headless服务器、SSH环境），使用加密配置文件

**密钥环不可用时的降级方案**：
1. 自动检测密钥环是否可用
2. 如不可用，提示用户选择：
   - 交互式输入主密码（从stdin读取）
   - 从环境变量`OPENCODE_USAGE_MASTER_PASSWORD`读取
   - 生成随机主密码并保存到文件
3. 使用AES-256-GCM加密API Key，存储到`~/.config/opencode-usage/secrets.enc`

**存储结构**：
- **系统密钥环**：存储完整的API Key（加密存储）
- **配置文件**：只存储账号元数据（名称、创建时间、最后验证时间）

**数据结构**：
```yaml
# ~/.config/opencode-usage/config.yaml
accounts:
  work:
    key_id: "abc123"  # 仅用于显示，存储Key的后6位
    created_at: "2026-08-24T10:00:00Z"
    last_verified: "2026-08-24T10:05:00Z"
  personal:
    key_id: "xyz789"
    created_at: "2026-08-24T11:00:00Z"
    last_verified: "2026-08-24T11:05:00Z"
```

**安全措施**：
1. API Key存储在系统密钥环中，使用操作系统级别的加密保护
2. 配置文件只存储Key的显示ID（后6位），不存储完整Key
3. 显示时只显示Key ID（如`sk-...abc123`）
4. 验证API Key时使用OpenCode Go的`/zen/go/v1/usage`端点
5. 配置文件权限设置为600（仅所有者可读写）
6. 降级方案使用AES-256-GCM加密，密钥通过PBKDF2派生

## API集成设计

**OpenCode Go API端点**：
- 基础URL：`https://opencode.ai/zen/go/v1`
- 认证：`Authorization: Bearer <API_KEY>`
- 自定义端点：支持通过环境变量`OPENCODE_USAGE_BASE_URL`覆盖

**核心API调用**：
```go
// 获取配额使用情况
GET /usage
Response: {
  "usage": {
    "rolling": { "status": "ok", "percent": 35, "resetsAt": "2026-08-24T18:00:00Z" },
    "weekly": { "status": "ok", "percent": 12, "resetsAt": "2026-08-30T00:00:00Z" },
    "monthly": { "status": "ok", "percent": 8, "resetsAt": "2026-09-24T00:00:00Z" }
  }
}

// 获取可用模型列表
GET /models
Response: {
  "data": [
    { "id": "mimo-v2.5", "name": "MiMo-V2.5", "pricing": { "input": 0.14, "output": 0.28 } },
    { "id": "deepseek-v4-flash", "name": "DeepSeek V4 Flash", "pricing": { "input": 0.22, "output": 0.66 } }
  ]
}
```

**错误处理**：
- 401 Unauthorized：API Key无效，提示用户检查Key
- 403 Forbidden：无Go计划订阅，提示用户订阅Go计划
- 429 Too Many Requests：请求过于频繁，实现指数退避重试
- 5xx服务器错误：显示友好错误信息，建议稍后重试
- 网络错误：超时（默认10秒）或连接失败，显示网络诊断信息

**错误代码枚举**：
| 错误代码 | HTTP状态码 | 含义 | 用户提示 |
|----------|------------|------|----------|
| `invalid_api_key` | 401 | API Key无效 | "请检查您的API Key" |
| `no_go_subscription` | 403 | 无Go计划订阅 | "请订阅OpenCode Go计划" |
| `rate_limited` | 429 | 请求过于频繁 | "请求过于频繁，请稍后重试" |
| `server_error` | 5xx | 服务器错误 | "服务器暂时不可用，请稍后重试" |
| `network_error` | - | 网络连接失败 | "网络连接失败，请检查网络" |
| `config_not_found` | - | 配置文件不存在 | "配置文件不存在，正在创建..." |
| `keyring_unavailable` | - | 密钥环不可用 | "密钥环不可用，使用加密文件" |

**错误响应格式**：
```json
{
  "error": {
    "code": "invalid_api_key",
    "message": "The API key provided is invalid",
    "details": "Please check your API key and try again"
  }
}
```

**缓存策略**：
- 配额数据缓存5分钟，避免频繁API调用
- 模型列表缓存1小时，变化频率较低
- 缓存存储在内存中，程序退出后失效

**重试策略**：
- 网络错误：最多重试3次，指数退避（1s, 2s, 4s）
- 429错误：根据`Retry-After`头等待后重试
- 5xx错误：最多重试2次，间隔2秒

**并发限制**：
- 默认最大并发数：5（可通过配置文件调整）
- 并发数配置项：`max_concurrent_requests`

## 输出格式设计

### 人类可读格式（默认）

```
$ ou q

┌─────────────────────────────────────────────────────────────┐
│                    OpenCode Go 配额概览                      │
├─────────┬─────────────────┬─────────────────┬───────────────┤
│ 账号    │ 5小时滚动       │ 每周            │ 每月          │
├─────────┼─────────────────┼─────────────────┼───────────────┤
│ work    │ 35% (剩余8h12m) │ 12% (剩余5d6h)  │ 8% (剩余23d)  │
│ personal│ 67% (剩余2h45m) │ 45% (剩余3d12h) │ 22% (剩余18d) │
└─────────┴─────────────────┴─────────────────┴───────────────┘
```

### JSON格式

```json
{
  "accounts": [
    {
      "name": "work",
      "quota": {
        "rolling": { "percent": 35, "resetsAt": "2026-08-24T18:00:00Z" },
        "weekly": { "percent": 12, "resetsAt": "2026-08-30T00:00:00Z" },
        "monthly": { "percent": 8, "resetsAt": "2026-09-24T00:00:00Z" }
      }
    },
    {
      "name": "personal",
      "quota": {
        "rolling": { "percent": 67, "resetsAt": "2026-08-24T12:30:00Z" },
        "weekly": { "percent": 45, "resetsAt": "2026-08-28T12:00:00Z" },
        "monthly": { "percent": 22, "resetsAt": "2026-09-24T00:00:00Z" }
      }
    }
  ]
}
```

**颜色方案**：
- 绿色（<50%）：正常使用
- 黄色（50-80%）：接近限制
- 红色（>80%）：即将用尽
- 阈值可通过配置文件自定义

**输出到文件**：
- 使用`-o <filename>`标志将输出保存到文件
- 人类可读格式保存为纯文本
- JSON格式保存为.json文件

## 测试策略

**测试策略**：
1. **单元测试**：覆盖核心业务逻辑（API调用、数据解析、安全存储）
2. **集成测试**：测试完整工作流程（添加账号、查询配额）
3. **端到端测试**：模拟真实使用场景

**配置文件初始化**：
1. 首次运行时自动创建`~/.config/opencode-usage/config.yaml`
2. 创建目录并设置权限为700
3. 创建空配置文件并设置权限为600

**并发查询实现**：
- 使用goroutine并发查询多个账号
- 使用sync.WaitGroup等待所有查询完成
- 使用channel收集结果并按顺序输出

**日志设计**：
- 日志级别：DEBUG、INFO、WARN、ERROR
- 默认级别：INFO
- 日志输出：stderr（可通过`--log-file`标志输出到文件）
- 日志格式：`[时间] [级别] 消息`

**配置文件迁移策略**：
1. 读取配置文件时检查`version`字段
2. 如果版本过旧，自动备份到`config.yaml.bak`
3. 按照迁移规则升级配置文件格式
4. 写入新版本配置文件

**验收标准**：
1. **功能验收**：
   - [ ] 能够交互式添加账号
   - [ ] 能够查看所有账号列表
   - [ ] 能够删除账号
   - [ ] 能够查看配额信息
   - [ ] 能够查看可用模型
   - [ ] 能够显示当前opencode配置的账号
   - [ ] 支持JSON输出
   - [ ] 支持命令别名

2. **安全验收**：
   - [ ] API Key不以明文存储
   - [ ] 显示时只显示Key ID
   - [ ] 配置文件权限正确

3. **性能验收**：
   - [ ] 命令响应时间 < 2秒
   - [ ] 支持并发查询多个账号

4. **易用性验收**：
   - [ ] 帮助信息清晰
   - [ ] 错误信息友好
   - [ ] 交互式界面直观

## 退出码设计

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 一般错误 |
| 2 | 用法错误（参数错误） |
| 3 | 认证失败（API Key无效） |
| 4 | 网络错误 |
| 5 | 配置文件错误 |
| 6 | 配置文件不存在（首次运行） |
| 7 | 密钥环不可用 |
