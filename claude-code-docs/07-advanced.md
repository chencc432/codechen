# 07. 高级特性与团队集成

本章面向想把 Claude Code **规模化应用于团队/组织**的同学。

---

## 1. Hooks 事件钩子深度用法

Hooks 让 Claude Code 在关键时刻调用你自己的脚本，实现"可观察"和"可控制"。

### 支持的事件

| 事件 | 触发时机 |
| --- | --- |
| `PreToolUse` | 工具执行前 |
| `PostToolUse` | 工具执行后 |
| `UserPromptSubmit` | 用户提交输入时 |
| `Stop` | 任务结束时 |
| `SessionStart` | 会话开始 |

### 案例：提交前必须跑 lint

`.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash(git commit:*)",
        "hooks": [
          {
            "type": "command",
            "command": "pnpm lint && pnpm test --bail",
            "blocking": true
          }
        ]
      }
    ]
  }
}
```

`blocking: true` 表示脚本非 0 退出会阻止 Claude 继续执行命令。

### 案例：写文件后自动 format

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          { "type": "command", "command": "pnpm prettier --write \"$FILE_PATH\"" }
        ]
      }
    ]
  }
}
```

### 案例：审计日志

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "echo \"[$(date -Iseconds)] $USER :: $COMMAND\" >> ~/.claude/audit.log"
          }
        ]
      }
    ]
  }
}
```

---

## 2. Subagents 团队化

Subagents 本质是"带专属 prompt + 工具集 + 独立上下文"的小 Claude。典型用法：

### 2.1 专家型 Agent 库

`.claude/agents/` 下放多个专家：

- `code-reviewer.md` — Code Review 专家
- `security-auditor.md` — 安全审计
- `perf-analyzer.md` — 性能分析
- `sql-optimizer.md` — SQL 优化
- `migration-planner.md` — 迁移规划

每个都是一个 markdown，顶部是 frontmatter，下方是 system prompt。

### 2.2 Subagent 模板

```markdown
---
name: sql-optimizer
description: 分析慢 SQL 并提优化建议
tools: [Read, Grep, Bash]
model: opus  # 复杂任务可指定更强模型
---

你是资深 DBA，擅长 PostgreSQL 优化。

工作方式：
1. 用户给出慢查询 + EXPLAIN 结果
2. 你先分析执行计划
3. 给出 3 种可选优化：索引/重写/schema 变更
4. 说明每种的预期收益和副作用

输出格式：markdown，分"诊断、方案、风险"三节。
```

### 2.3 在主会话里调用

```text
用 sql-optimizer 分析这条慢查询：
===
SELECT ... （贴 SQL）
-- EXPLAIN:
（贴执行计划）
===
```

主 Claude 会把任务派发到子 agent，子 agent 独立工作，完成后返回结果。

### 2.4 团队共享

把 `.claude/agents/` **提交到 Git**，所有成员共用。这样团队的"最佳实践 prompt"就固化成了工具。

---

## 3. 规模化最佳实践

### 3.1 团队配置结构

```
<repo>/
├── CLAUDE.md                      # 项目说明（提交）
├── .claude/
│   ├── settings.json              # 团队共享配置（提交）
│   ├── settings.local.json        # 个人覆盖（.gitignore）
│   ├── commands/                  # 团队斜杠命令（提交）
│   │   ├── review.md
│   │   ├── write-test.md
│   │   └── deploy-check.md
│   ├── agents/                    # 团队子智能体（提交）
│   │   ├── security-auditor.md
│   │   └── sql-optimizer.md
│   └── hooks/                     # Hook 脚本（提交）
│       ├── pre-commit-check.sh
│       └── audit-log.sh
└── .gitignore                     # 别忘了把 settings.local.json 加进去
```

### 3.2 分层约定

- **根目录 CLAUDE.md** → 通用规则
- **子目录 CLAUDE.md** → 子系统规则（monorepo 场景）
- **`.claude/commands/`** → 高频操作模板化
- **`.claude/agents/`** → 专家角色
- **Hooks** → 硬性约束（lint、审计）

### 3.3 治理建议

- **代码评审** `.claude/` 的改动和代码改动等同对待。
- **定期回顾** 哪些 command/agent 没人用，可以删。
- **建立内部 wiki**：列出哪个场景用哪个 agent/command。
- **跟踪 ROI**：让每个成员每月写一次"Claude Code 帮我省了多少时间"。

---

## 4. 与 CI/CD 集成

### 4.1 典型集成点

| 阶段 | 用途 |
| --- | --- |
| **PR 开启** | AI Code Review、风险评估 |
| **Push 前** | 本地 Hook 做预检 |
| **构建失败** | 自动分析错误、建议修复 |
| **部署后** | 生成 Release Notes |
| **监控告警** | 分析日志 / 堆栈 |

### 4.2 GitLab CI 示例

```yaml
ai-review:
  stage: review
  image: node:20
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
    - npm install -g @anthropic-ai/claude-code
    - |
      git fetch origin $CI_MERGE_REQUEST_TARGET_BRANCH_NAME
      git diff origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME...HEAD \
        | claude -p "对这些改动做严格 review，输出 markdown" \
        > review.md
    - |
      curl --request POST \
        --header "PRIVATE-TOKEN: $CI_API_TOKEN" \
        --form "body=$(cat review.md)" \
        "$CI_API_V4_URL/projects/$CI_PROJECT_ID/merge_requests/$CI_MERGE_REQUEST_IID/notes"
  variables:
    ANTHROPIC_API_KEY: $ANTHROPIC_API_KEY
```

### 4.3 生成 Release Notes

```bash
git log $(git describe --tags --abbrev=0)..HEAD --pretty=format:"%h %s" \
  | claude -p "按 Features / Fixes / Breaking Changes 分类生成 release notes。" \
  > RELEASE_NOTES.md
```

### 4.4 CI 失败自动诊断

```yaml
- name: Diagnose on failure
  if: failure()
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    npm install -g @anthropic-ai/claude-code
    tail -n 500 build.log \
      | claude -p "分析这段 CI 日志，定位失败原因并给出最可能的修复步骤。" \
      > diagnosis.md
    cat diagnosis.md
```

---

## 5. 成本控制

### 5.1 监控

```text
/cost
```

会显示本次会话消耗。进一步可以在 `settings.json` 里开启：

```json
{
  "costTracking": {
    "warnAt": 5.00,
    "hardLimit": 20.00,
    "logFile": "~/.claude/cost.log"
  }
}
```

### 5.2 省 token 十条

1. 写好 CLAUDE.md → 避免每次重复背景
2. 用 `@文件` 引用代替贴全文
3. 长会话用 `/compact`
4. 切换任务用 `/clear`
5. 简单任务用 Sonnet，不要默认 Opus
6. 批量脚本任务用 Haiku
7. 告诉它"回答不超过 X 字"
8. 让它"只输出改动片段"而不是整文件
9. CI 里用 `-p` 一次性模式而非交互
10. Cache 友好：同一项目多次会话，Anthropic 会自动缓存共有前缀（CLAUDE.md 等）

### 5.3 模型分层策略

```json
{
  "model": "claude-sonnet-4-5",
  "modelFallbacks": {
    "simpleTasks": "claude-haiku-4",
    "complexTasks": "claude-opus-4"
  }
}
```

(具体字段以官方版本为准)

---

## 6. 企业部署考量

### 6.1 代理与 VPN

通过公司代理：

```bash
export HTTPS_PROXY=http://proxy.company.com:3128
export NO_PROXY=localhost,127.0.0.1,.company.com
claude
```

### 6.2 私有网关

Anthropic 支持自建网关转发（Bedrock/Vertex），设置：

```bash
export ANTHROPIC_BASE_URL=https://claude-gw.company.com/v1
```

可以在网关层做：
- 统一鉴权（SSO 映射 API Key）
- 费用分摊
- 日志/合规审计
- 敏感词过滤 / DLP

### 6.3 数据合规

- Claude Code **默认不训练你的代码**
- 企业版可签 Zero Data Retention 协议
- 对高敏感项目：通过 `deny` 列表阻止读取密钥/配置
- 配合 Hooks 做 DLP 检查

### 6.4 多人协作规范模板

起草一份《团队 Claude Code 使用规范》，至少包含：

1. 允许使用的项目范围
2. 禁止放入 prompt 的信息（客户数据、PII、证书）
3. AI 生成代码的 review 要求（至少 1 人复核）
4. PR 描述里标注哪些是 AI 主力产出
5. 事故责任归属（永远是提交者）
6. 费用报销/分摊流程

---

## 7. 进阶场景示例

### 7.1 把 Claude Code 做成"值班助手"

结合 MCP 的 Slack / Sentry / PagerDuty，写一个 cron 脚本：

```bash
# 每 10 分钟检查一次 Sentry
*/10 * * * * cd /srv/oncall && claude -p "用 sentry MCP 拉最近 10 分钟新增错误，如果有未分配的，用 github MCP 创建 issue 并 @相关 owner。" --no-interactive
```

### 7.2 自动化代码迁移

```bash
# 批量把 JavaScript 文件转 TypeScript，每次 5 个
for batch in $(ls src/**/*.js | head -n 5); do
  claude -p "把 $batch 改为 .ts，保持原行为。如果类型不确定用 unknown 而不是 any。"
  pnpm build && pnpm test || git checkout -- .
done
```

### 7.3 内部知识库问答

把团队文档用 MCP filesystem 挂给 Claude：

```json
{
  "mcpServers": {
    "team-docs": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem",
               "/mnt/docs/engineering"]
    }
  }
}
```

然后就能问："我们的发布流程是什么？"，它会去团队文档检索并回答。

---

## 下一步

👉 [08-troubleshooting.md](./08-troubleshooting.md) 常见问题与排障
