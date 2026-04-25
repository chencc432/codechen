# 02. 安装与初始化

## 系统要求

- **Node.js** 18 或以上（建议 20 LTS）
- **操作系统**：macOS / Linux / Windows（推荐 WSL2 或原生 Windows 均可）
- **Anthropic 账号**（Pro/Team/Enterprise 订阅）或 **API Key**
- 稳定的网络（能访问 `api.anthropic.com`）

## 安装方式

### 方式一：npm 全局安装（推荐）

```bash
npm install -g @anthropic-ai/claude-code
```

安装完成后验证：

```bash
claude --version
```

### 方式二：使用 Homebrew（macOS）

```bash
brew install anthropic/tap/claude-code
```

### 方式三：Windows PowerShell

```powershell
# 先确保已安装 Node.js（https://nodejs.org/）
npm install -g @anthropic-ai/claude-code

# 验证
claude --version
```

> ⚠️ **Windows 提示**：如果 PowerShell 报 "脚本已禁用"，执行一次：
> ```powershell
> Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
> ```

## 首次登录

在任意目录下执行：

```bash
claude
```

会出现一个交互式登录流程，支持：

1. **浏览器登录（推荐）**：自动打开浏览器完成 OAuth。
2. **API Key 模式**：粘贴从 [console.anthropic.com](https://console.anthropic.com/) 获取的 Key。

登录信息默认保存在：

- macOS/Linux：`~/.config/claude-code/` 或系统 keychain
- Windows：`%APPDATA%\claude-code\`

## 环境变量（可选）

| 变量 | 作用 |
| --- | --- |
| `ANTHROPIC_API_KEY` | 使用 API Key 登录（跳过 OAuth） |
| `ANTHROPIC_BASE_URL` | 自定义 API 网关（企业私有部署） |
| `CLAUDE_CODE_MODEL` | 默认模型，如 `claude-sonnet-4-5` |
| `HTTPS_PROXY` / `HTTP_PROXY` | 使用代理访问 API |
| `CLAUDE_CODE_DISABLE_TELEMETRY` | 设为 `1` 关闭匿名遥测 |

**PowerShell 设置示例：**
```powershell
$env:ANTHROPIC_API_KEY = "sk-ant-..."
$env:HTTPS_PROXY = "http://127.0.0.1:7890"
claude
```

**bash/zsh 设置示例：**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export HTTPS_PROXY="http://127.0.0.1:7890"
claude
```

## 配置文件

Claude Code 有三级配置：

1. **全局配置**（所有项目）：`~/.claude/settings.json`
2. **项目配置**（当前仓库）：`<repo>/.claude/settings.json`
3. **本地配置**（不提交 Git）：`<repo>/.claude/settings.local.json`

常见配置字段示例：

```json
{
  "model": "claude-sonnet-4-5",
  "permissions": {
    "allow": ["Bash(git status:*)", "Bash(npm test:*)"],
    "deny": ["Bash(rm -rf:*)"]
  },
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "./"]
    }
  }
}
```

## 在项目中初始化

进入项目根目录后，建议做两件事：

### 1. 生成 `CLAUDE.md`

`CLAUDE.md` 是 Claude Code 每次启动时自动读取的"项目说明书"。可以让 Claude 自己生成：

```bash
cd your-project
claude
> /init
```

它会扫描项目，生成一份描述：技术栈、目录结构、常用命令、约定。**强烈建议每个项目都加一份**，这是提升效率的最关键动作。

### 2. 配置权限允许列表

首次运行一些命令时，Claude 会询问是否允许，可以勾选"always allow"，会写入 `.claude/settings.local.json`。

## 升级与卸载

**升级：**
```bash
npm update -g @anthropic-ai/claude-code
# 或
claude update
```

**卸载：**
```bash
npm uninstall -g @anthropic-ai/claude-code
rm -rf ~/.claude   # 清理配置（可选）
```

## 下一步

👉 [03-quickstart.md](./03-quickstart.md) 10 分钟快速上手
