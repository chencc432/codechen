# 08. 常见问题与排障

## 安装相关

### Q: `npm install -g @anthropic-ai/claude-code` 卡住或失败

**可能原因**：npm 源、网络、Node 版本。

**排查**：

```bash
# 1. 检查 Node 版本（需 >= 18）
node -v

# 2. 切换到官方源
npm config set registry https://registry.npmjs.org/

# 3. 清缓存重装
npm cache clean --force
npm install -g @anthropic-ai/claude-code

# 4. 国内可用淘宝源（但有时会有版本滞后）
npm config set registry https://registry.npmmirror.com
```

### Q: Windows 上 `claude` 不是命令

**排查**：

```powershell
# 看 npm 全局路径
npm config get prefix

# 把该路径加到 PATH，然后重开终端
```

或用 PowerShell 配置文件永久生效：

```powershell
$npmBin = npm config get prefix
[Environment]::SetEnvironmentVariable("Path", "$env:Path;$npmBin", "User")
```

---

## 登录与网络

### Q: 登录时浏览器不自动打开

复制终端提示的 URL 手动粘到浏览器。若仍不行：

```bash
# 使用 API Key 模式
export ANTHROPIC_API_KEY=sk-ant-...
claude
```

### Q: `Error: connect ETIMEDOUT api.anthropic.com`

多半是网络/代理问题。

```bash
# 设置代理
export HTTPS_PROXY=http://127.0.0.1:7890
export HTTP_PROXY=http://127.0.0.1:7890

# 或在公司内部代理
export HTTPS_PROXY=http://proxy.corp:3128
```

Windows PowerShell：

```powershell
$env:HTTPS_PROXY = "http://127.0.0.1:7890"
```

### Q: SSL 错误 `unable to verify the first certificate`

公司网络有 SSL 中间证书。可以：

```bash
# 导入公司 CA 证书
export NODE_EXTRA_CA_CERTS=/path/to/corp-ca.crt
```

**临时**（不建议）：

```bash
export NODE_TLS_REJECT_UNAUTHORIZED=0
```

---

## 使用中的问题

### Q: Claude 总是忘记我的约定

很可能是没有 CLAUDE.md，或里面没写清楚。检查：

1. `ls CLAUDE.md` 是否存在
2. 内容里是否明确列出约定（如"禁止用 any"）
3. 启动时终端是否提示 "Loaded CLAUDE.md"

### Q: 改错了一堆文件，怎么回滚？

```bash
# 查看改动
git status
git diff

# 全部回滚
git checkout .

# 如果有新建文件
git clean -fd

# 部分回滚
git checkout -- path/to/file
```

**预防**：每次让 Claude 动手前先 `git status`，保证工作区干净，出问题直接 `git checkout .` 就能回到起点。

### Q: Claude 卡住不回应 / 长时间没动作

```bash
# 1. Ctrl+C 打断当前任务
# 2. 如果还卡，Ctrl+D 退出
# 3. 再启动一次
claude --continue   # 恢复上次会话
```

### Q: Token 消耗太快，账单涨得吓人

查看消耗：

```text
/cost
```

优化：
- `/compact` 压缩上下文
- `/clear` 切任务时清空
- 换更小的模型（Haiku/Sonnet）
- 检查是不是把 `node_modules/` 或 `dist/` 读进去了（在 CLAUDE.md 或 `.claudeignore` 里排除）

### Q: 如何让 Claude 忽略某些文件？

创建 `.claudeignore`（语法同 `.gitignore`）：

```
node_modules/
dist/
build/
.next/
coverage/
*.log
.env*
*.key
*.pem
```

---

## 权限与安全

### Q: 每次都要确认命令很烦

在 `.claude/settings.local.json` 里加 allow 列表：

```json
{
  "permissions": {
    "allow": [
      "Bash(npm test:*)",
      "Bash(pnpm test:*)",
      "Bash(git status:*)",
      "Bash(git diff:*)",
      "Bash(git log:*)"
    ]
  }
}
```

**但不要 allow 所有 Bash**，特别是 `rm`、`mv`、`curl | sh` 这类。

### Q: 我不小心允许了危险命令怎么办

编辑 `settings.local.json` 删除对应条目，或：

```bash
claude
/config
# 在 UI 里管理权限
```

---

## 性能问题

### Q: 启动很慢

**原因**：CLAUDE.md 太大、或 MCP Server 很多在初始化。

**排查**：

```bash
claude --verbose
```

看启动日志是哪一步慢。优化：
- 拆分 CLAUDE.md（移到子目录）
- 禁用不用的 MCP

### Q: 回答很慢

- 检查网络延迟：`ping api.anthropic.com`
- 检查是否误选了 Opus（比 Sonnet 慢 3~5 倍）
- 上下文太长：`/compact`

---

## MCP 相关

### Q: `/mcp` 显示 server 启动失败

```bash
# 手动跑一下看真实报错
npx -y @modelcontextprotocol/server-github
```

常见问题：
- 缺环境变量（如 `GITHUB_TOKEN`）
- Node 版本太低
- 第一次下载卡住，先手动 `npx -y ...` 预热

### Q: MCP 工具调用不到

- 重启 claude
- 用 `/mcp` 检查 server 状态
- 确认提示里明确表达了需求，让 Claude 有动力去调用

---

## IDE 集成

### Q: 与 VSCode / Cursor 能协同工作吗？

可以。Claude Code 在终端跑，VSCode 打开代码。Claude 修改文件后 VSCode 会自动刷新（开启了自动保存时）。

建议：
- **Cursor 用户**：把 Claude Code 用于跨文件重构和脚本任务，Cursor 内联改单点代码。
- **VSCode 用户**：直接在 VSCode 集成终端运行 `claude`。

Claude Code 也有官方 IDE 插件（在 VSCode/JetBrains 商店搜索），可把 diff 在编辑器里可视化呈现。

---

## 遇到 Bug 怎么报告

1. 升级到最新版：`claude update`
2. 准备最小复现
3. 附上 `claude --version` 输出
4. 把 `~/.claude/logs/` 里最近的日志打包
5. 到 https://github.com/anthropics/claude-code 提 issue

---

## 更多资源

- 官方文档：https://docs.claude.com/en/docs/claude-code
- GitHub 仓库：https://github.com/anthropics/claude-code
- MCP 生态：https://github.com/modelcontextprotocol/servers
- 社区最佳实践集合：在 GitHub 搜 "awesome-claude-code"

---

## 返回目录

👉 [README.md](./README.md)
