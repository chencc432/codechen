 # 04. 核心功能详解

## 1. 斜杠命令（Slash Commands）

在交互界面输入 `/` 开头的指令会被 Claude Code **本地解析**，不消耗对话 token。

### 内置命令

| 命令 | 作用 | 常用场景 |
| --- | --- | --- |
| `/help` | 列出所有命令 | 查询 |
| `/init` | 生成 CLAUDE.md | 新项目首次使用 |
| `/clear` | 清空当前会话上下文 | 切换任务时 |
| `/compact` | 压缩历史（保留要点） | 长对话省 token |
| `/model` | 切换模型 | 复杂任务切 Opus |
| `/cost` | 查看本次消耗 | 控预算 |
| `/review` | 审查未提交改动 | 提交前 |
| `/config` | 打开配置 | 改权限/模型 |
| `/mcp` | 管理 MCP Server | 接入外部工具 |
| `/agents` | 管理 subagent | 定义子智能体 |
| `/hooks` | 管理 Hooks | 事件自动化 |
| `/resume` | 恢复上次会话 | 继续昨天的活 |

### 自定义斜杠命令

在项目或全局目录创建 `.claude/commands/<name>.md`，内容就是提示词模板。

示例：`.claude/commands/pr-review.md`

```markdown
请对当前分支相对于 main 的所有改动做一次严格的 code review：
1. 先用 `git diff main...HEAD` 看差异
2. 按照以下维度检查：
   - 可读性与命名
   - 边界条件与异常处理
   - 测试是否充分
   - 有无引入安全问题（注入、XSS、密钥泄漏）
3. 输出格式：
   - 🔴 必须修复
   - 🟡 建议优化
   - 🟢 做得不错
```

之后在会话里输入 `/pr-review` 即可一键触发。**团队级命令提交进 Git，所有成员共享**。

## 2. 上下文管理（Context）

Claude Code 的上下文由几部分组成：

1. **系统提示**（Anthropic 内置，不可见）
2. **CLAUDE.md**（项目说明，每次启动加载）
3. **历史对话**
4. **被读取的文件内容**

### 管理技巧

- **`/clear`**：任务切换时清空，避免上一个任务的代码"污染"下一个任务。
- **`/compact`**：让 Claude 自己总结历史，替换详细对话，节省 token（建议每 30~50 轮用一次）。
- **`@` 引用文件**：在提示里用 `@src/utils.js` 可以精确地把某个文件塞进上下文。
- **`--continue` 恢复会话**：`claude --continue` 恢复上次退出时的上下文。

### CLAUDE.md 的分层

- **全局**：`~/.claude/CLAUDE.md`（你的个人偏好）
- **项目**：`<repo>/CLAUDE.md`（提交 Git，团队共享）
- **子目录**：`<repo>/backend/CLAUDE.md`（只在子目录生效，用于 monorepo）

三层会叠加。例如你可以在全局写 "回复用中文"，在项目里写 "用 TypeScript 严格模式"。

## 3. 工具调用（Tools）

Claude Code 通过一组标准工具来与你的环境交互：

| 工具 | 作用 |
| --- | --- |
| **Read** | 读文件 |
| **Write** | 写/新建文件 |
| **Edit** | 编辑文件（精确替换） |
| **Bash** | 执行 shell 命令 |
| **Glob** | 按文件名模式查找 |
| **Grep** | 按内容搜索 |
| **WebFetch** | 抓取网页 |
| **WebSearch** | 联网搜索 |
| **Task** | 派发给子智能体 |

### 权限模型

每个工具都可以在配置里精细控制：

```json
{
  "permissions": {
    "allow": [
      "Read(**)",
      "Edit(src/**)",
      "Bash(npm test:*)",
      "Bash(git diff:*)",
      "Bash(git status:*)"
    ],
    "ask": [
      "Bash(git push:*)",
      "Write(migrations/**)"
    ],
    "deny": [
      "Bash(rm -rf:*)",
      "Bash(curl:*|sh)",
      "Edit(.env*)",
      "Read(.env*)"
    ]
  }
}
```

三个级别：
- **allow**：静默执行
- **ask**：每次询问（默认）
- **deny**：直接拒绝，即使你同意也不执行

**强烈推荐**：把 `.env`、密钥、`node_modules`、`.git` 加入 deny 或限制性读取。

## 4. MCP（Model Context Protocol）

MCP 是 Anthropic 主导的开放协议，让 Claude 能接入**外部工具和数据源**。把它当作"AI 的 USB 接口"。

### 常用 MCP Server

| 名称 | 作用 |
| --- | --- |
| `filesystem` | 更强的文件访问 |
| `github` | 管理 Issue/PR |
| `postgres` / `sqlite` | 查询数据库 |
| `slack` | 发消息/读频道 |
| `puppeteer` / `playwright` | 控制浏览器 |
| `sentry` | 读取错误监控 |
| `notion` | 读写 Notion 页面 |

### 配置示例

在 `.claude/settings.json`：

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres",
               "postgresql://localhost/mydb"]
    }
  }
}
```

配置后在会话里就可以说："查一下数据库 `users` 表结构" 或 "列出仓库 `facebook/react` 的近 5 个 open issue"。

## 5. Subagents（子智能体）

一个主 Claude 可以派发任务给"专用子智能体"，常用于：
- **搜索型任务**（多轮 grep/read）
- **独立上下文**（不污染主会话）
- **专家模式**（代码审查专家、安全审计专家……）

### 定义方式

`.claude/agents/security-reviewer.md`：

```markdown
---
name: security-reviewer
description: 专门做安全审计的子智能体
tools: [Read, Grep, Bash]
---

你是一名资深安全工程师，擅长：
- 识别 SQL 注入、XSS、SSRF
- 检查密钥硬编码
- 审计依赖漏洞

工作方式：
1. 先全局 grep 风险模式
2. 逐个可疑点读源码确认
3. 输出分级报告（高/中/低风险）
```

在主对话里调用：

```text
用 security-reviewer 对 src/api/ 做一次安全审计。
```

## 6. Hooks（事件钩子）

在特定事件触发时自动执行脚本，比如：

- 每次 Write 后自动跑 `prettier`
- 每次 Bash 前检查白名单
- 会话结束时推送日志到你的日志系统

### 配置示例

`.claude/settings.json`：

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          { "type": "command", "command": "npx prettier --write $FILE_PATH" }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "./scripts/audit.sh \"$COMMAND\"" }
        ]
      }
    ]
  }
}
```

## 7. 非交互式 / 脚本模式

Claude Code 可以像 Unix 工具一样用于脚本和 CI。

### 单次执行

```bash
claude -p "帮我统计 src 下所有 TODO 注释，输出成 markdown 表格"
```

`-p` 表示 "print once" 模式：执行完就退出，输出到 stdout。

### 管道组合

```bash
git log --since="1 week ago" --pretty=format:"%s" \
  | claude -p "根据这些 commit 写一份周报"
```

### 在 CI 中使用

```yaml
# .github/workflows/ai-review.yml
- name: Claude review
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    npm install -g @anthropic-ai/claude-code
    git diff origin/main...HEAD | claude -p "对这些改动做 code review，输出 markdown" > review.md
```

## 下一步

👉 [05-best-practices.md](./05-best-practices.md) 最佳实践与使用建议
