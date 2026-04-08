# OpenClaw 从零详解：开源个人 AI 智能体框架完全指南

## 一、OpenClaw 是什么？

OpenClaw 是一个**开源的个人 AI 智能体（Agent）框架**，你可以在自己的设备上运行它。它不是一个传统的聊天机器人，而是一个**自主的 AI 代理**——能理解你的意图、编写代码来执行任务、跨平台与你交互、并在你不说话的时候主动帮你做事。

- **GitHub 星标**：330,000+
- **npm 包名**：`@anthropic/openclaw`
- **许可证**：开源
- **创始人**：Peter Steinberger（2025 年 11 月创建，2026 年 2 月加入 OpenAI，项目转交 OpenClaw Foundation）

### 1.1 项目历史

| 时间 | 事件 |
|------|------|
| 2025 年 11 月 | Peter Steinberger 发布 **Clawdbot**，一个"AI 自己写代码"的实验性助手 |
| 2026 年 1 月 | 更名为 **Moltbot**，社区爆发增长，产生数百个插件 |
| 2026 年 2 月 | 因商标问题正式更名为 **OpenClaw** |
| 2026 年 2 月 15 日 | 创始人加入 OpenAI，项目所有权转移至 OpenClaw Foundation，确保永远开源 |
| 2026 年 3 月 | 最新版本 v2026.3.2，新增 PDF 分析工具、Ollama 嵌入支持等 |

### 1.2 核心理念："LLM 即开发者"

传统聊天机器人是"硬编码"的——如果你让 ChatGPT 查日历，需要 OpenAI 工程师预先编写 Google Calendar 连接代码。如果代码不存在或损坏，AI 就无能为力。

OpenClaw 颠覆了这个模式。它使用 **AgentSkills（智能体技能）** 教 LLM 如何编写小型脚本（通常是 Node.js 或 Python）来与外部世界交互。

当你说"查看我的日历"时，OpenClaw 会：

1. **理解**你的意图
2. **编写脚本**调用 Google Calendar API
3. **在本地执行**该脚本
4. **读取输出**并告诉你结果

这意味着 OpenClaw 能做任何你能用代码做的事，无需等待开发者构建功能。

---

## 二、系统架构概览

OpenClaw 的架构采用清晰的**关注点分离**设计，分为五层：

```
┌───────────────────────────────────────────────────────────┐
│                    用户（你）                               │
│          WhatsApp / Telegram / Slack / Discord / Email      │
└─────────────────────────┬─────────────────────────────────┘
                          │
┌─────────────────────────▼─────────────────────────────────┐
│              Layer 1: Gateway（网关）                       │
│         持久化 Node.js 进程，始终在线运行                     │
│            模型无关，路由所有消息                              │
└─────────────────────────┬─────────────────────────────────┘
                          │
┌─────────────────────────▼─────────────────────────────────┐
│          Layer 2: Channel Adapters（通道适配器）             │
│    将 WhatsApp/Telegram/Slack 等不同 API 规范化为统一格式      │
└─────────────────────────┬─────────────────────────────────┘
                          │
┌─────────────────────────▼─────────────────────────────────┐
│          Layer 3: Agent Runtime（智能体运行时）              │
│       构建 Prompt → 调用 LLM → 执行工具 → ReAct 循环        │
└─────────────────────────┬─────────────────────────────────┘
                          │
┌─────────────────────────▼─────────────────────────────────┐
│          Layer 4: Skills Platform（技能平台）                │
│     Shell 执行 / 浏览器控制 / API 调用 / ClawHub 扩展       │
└─────────────────────────┬─────────────────────────────────┘
                          │
┌─────────────────────────▼─────────────────────────────────┐
│     Layer 5: Memory + Heartbeat（记忆 + 心跳引擎）          │
│         Markdown 文件持久化记忆 / Cron 调度主动任务           │
└───────────────────────────────────────────────────────────┘
```

---

## 三、五层架构组件详解

### 3.1 Layer 1: Gateway（网关）

Gateway 是 OpenClaw 最核心的架构洞察：**它是一个网关，不是一个应用程序。**

传统 AI 工具（如 ChatGPT）是独立应用——有自己的 UI、存储和服务器，你需要去找它们。OpenClaw 不同，它坐在你已有的工具和 AI 模型之间，充当一个**智能桥梁**。

**关键特性：**

- **持久化 Node.js 服务**：长时间运行的进程，永不停止。不像无服务器函数那样用完即消失
- **始终在线**：维护状态，能监控条件，能按计划唤醒自己，能**主动给你发消息**
- **模型无关**：不关心底层 AI 是 GPT-5、Claude、Gemini 还是本地 Llama 3。在配置文件中切换模型即可升级
- **默认端口**：18789
- **绑定地址**：安全配置下应绑定 `127.0.0.1`（而非 `0.0.0.0`）

**关键配置（`openclaw.json`）：**

```json
{
  "gateway": {
    "bind": "lan",
    "port": 18789,
    "auth": {
      "mode": "token",
      "token": "YOUR_256_BIT_HEX_TOKEN"
    },
    "controlUI": false,
    "discovery": {
      "mdns": { "mode": "off" }
    }
  }
}
```

### 3.2 Layer 2: Channel Adapters（通道适配器）

每个与 OpenClaw 交互的消息平台都通过一个 **Channel Adapter** 连接。它们是薄的翻译层，将各种平台的不同 API 规范化为 OpenClaw 内部统一的消息格式。

**支持的平台：**

| 平台 | 特性支持 |
|------|---------|
| **WhatsApp** | 个人任务，Meta Business 集成 |
| **Telegram** | 个人任务，内联键盘，回调查询，流式输出 |
| **Slack** | 工作协作，斜杠命令，线程回复，文件上传，Block Kit |
| **Discord** | 团队沟通，机器人命令 |
| **Signal** | 隐私优先通信 |
| **iMessage** | Apple 生态集成 |
| **Email** | 异步任务处理 |

**工作流程：**

1. 你在 Telegram 发送消息
2. Telegram API 通过 Webhook 转发到 OpenClaw
3. Telegram Channel Adapter 解析 JSON 负载，提取文本、发送者 ID、时间戳
4. 转换为 OpenClaw **统一内部消息对象**
5. 从此刻起，核心网关不知道消息来自哪个平台

**跨平台上下文保持：** 你可以在 Slack 上开始一个任务，然后在 WhatsApp 上完成它——智能体在所有平台间维持上下文。

### 3.3 Layer 3: Agent Runtime（智能体运行时）

Agent Runtime 是智能所在。它负责构建 Prompt、调用 LLM 并解释响应。

**Prompt 构建过程（每次交互都会发生）：**

```
┌─────────────────────────────────────────────┐
│              Prompt 组装                     │
├─────────────────────────────────────────────┤
│  1. SOUL.md → 系统 Prompt（身份、价值观）     │
│  2. 对话历史 → 最近 10-20 轮交互             │
│  3. 持久化记忆 → 来自 PROFILE.md 等文件      │
│  4. 工具定义 → TOOLS.md + SKILLS.md          │
│  5. 当前消息 → 用户刚发送的内容               │
└─────────────────────────────────────────────┘
                     │
                     ▼
              ┌─────────────┐
              │   LLM 推理   │
              └──────┬──────┘
                     │
            ┌────────┴────────┐
            ▼                 ▼
      纯文本回复         工具调用请求
     （直接返回）      （进入 ReAct 循环）
```

**ReAct 循环（Reason + Act）：**

这是智能体行为的核心。模型不仅仅是回复——它会**规划、执行、观察结果、再调整**。一个复杂任务可能涉及数十个循环：

```
推理 → 调用工具 → 获取结果 → 再推理 → 调用工具 → ... → 最终文本响应
```

例如：搜索网页 → 读取文件 → 编写代码 → 运行测试 → 读取输出 → 修复 bug → 提交结果。

**运行时安全机制：**
- 参数验证：检查工具调用参数是否匹配 schema
- 超时控制：防止失控循环
- 工具调用日志：用于调试

### 3.4 Layer 4: Skills Platform（技能平台）

Skills 是智能体的"手"和"眼"。没有 Skills，智能体只能生成文本。有了 Skills，它可以执行 Shell 命令、控制浏览器、读写文件、调用外部 API、发送邮件等。

**内置 Skills（基础能力）：**

| Skill | 功能 |
|-------|------|
| `execute_shell` | 运行终端命令（沙箱中） |
| `read_file` / `write_file` | 文件系统读写 |
| `search_web` | 搜索引擎查询 |
| `http_request` | 调用任何 REST API |

**社区 Skills（通过 ClawHub 扩展市场）：**

| 类别 | 示例 |
|------|------|
| 浏览器自动化 | Playwright 控制 |
| 智能家居 | Home Assistant 集成 |
| 开发工具 | GitHub 操作 |
| 金融数据 | 行情 API |
| 通信 | SMS、邮件 |
| 日历 | Google Calendar 管理 |

**Skill 加载机制：**
- OpenClaw 启动时扫描 skills 目录
- 加载每个 Skill 的 manifest
- 注册工具到运行时
- 运行时在每次交互时将完整工具列表传递给 LLM

**安全沙箱：** 在 Docker 部署中，Skills 可以被隔离在容器内，限制文件和网络访问。

### 3.5 Layer 5: Memory + Heartbeat（记忆 + 心跳引擎）

#### 3.5.1 Persistent Memory（持久化记忆）

OpenClaw 最独特的设计之一——使用**纯 Markdown 文件**在本地磁盘存储记忆，而非数据库或云服务。

**为什么用 Markdown？**

| 特性 | 说明 |
|------|------|
| 人类可读 | 任何文本编辑器都能打开，直接看到智能体知道什么 |
| 人类可编辑 | 智能体学错了？直接改文件 |
| LLM 兼容 | LLM 对 Markdown 的理解优于 JSON 或数据库记录 |
| Git 友好 | 可以版本控制，追踪知识演化，回滚到历史状态 |

**记忆文件结构：**

```
memory/
├── profile.md          # 核心个人档案（姓名、时区、工作风格）
├── context.md          # 当前进行中的项目和情况
├── preferences.md      # 通信和工作偏好
├── contacts.md         # 联系人及关系上下文
├── goals.md            # 长期和短期目标
├── decisions.md        # 重要决策及其理由
└── skills_data/        # 各 Skill 存储的数据
    ├── calendar.md
    ├── health.md
    └── finance.md
```

**记忆学习机制：**

| 机制 | 说明 |
|------|------|
| 显式指令 | 你说"记住我 CEO 叫 Sarah Kim"，立即写入记忆 |
| 任务观察 | 执行任务时遇到的上下文（如日历中的常规会议）自动提取存储 |
| 对话推断 | 发现你总是要求简洁回答，自动记为偏好 |
| 心跳更新 | 心跳任务定期更新当日状态信息 |

#### 3.5.2 Heartbeat Engine（心跳引擎）

心跳引擎是将 OpenClaw 从**被动助手**转变为**主动代理**的关键组件。它是一个独立于消息的后台调度器，按可配置的间隔（默认 30-60 分钟）定时触发。

**工作原理：**
1. 心跳触发（如同用户发送了消息："处理你的 HEARTBEAT.md"）
2. 智能体读取 HEARTBEAT.md 文件
3. 逐项处理清单中的任务
4. 使用 Skills 执行必要操作
5. 通过消息通道向你报告结果

**心跳间隔参考：**

| 间隔 | 适用场景 | 月估算 API 费用 |
|------|---------|----------------|
| 5 分钟 | 关键基础设施监控 | $20-60 |
| 15 分钟 | 活跃交易监控、生产告警 | $8-25 |
| 30 分钟 | 通用业务监控（推荐默认） | $4-12 |
| 60 分钟 | 休闲个人使用，节省成本 | $2-6 |
| 3-4 小时 | 低频报告任务 | <$2 |

---

## 四、Workspace 文件系统详解（~/clawd/）

OpenClaw 的所有配置和记忆都存储在 `~/clawd/` 目录下（纯文本文件，拒绝黑盒）。

```
~/clawd/
├── SOUL.md             # 智能体的灵魂：人格、价值观、长期指令
├── AGENTS.md           # 工作空间配置：角色定义、上下文切换
├── HEARTBEAT.md        # 主动任务清单：心跳引擎的执行清单
├── TOOLS.md            # 工具清单：低级别原语工具列表
├── SKILLS.md           # 技能清单：高级别扩展技能列表
├── IDENTITY.md         # 智能体名称和 emoji
├── USER.md             # 用户身份和称呼偏好
├── MEMORY.md           # （可选）手动策展的长期记忆
├── CONTACTS/           # 联系人目录（每人一个 .md 文件）
├── memory/             # 动态上下文记忆目录
│   ├── YYYY-MM-DD.md   # 每日记忆日志
│   └── ...
├── skills/             # 工作空间级别的自定义技能
└── config.yaml         # 主配置文件（LLM 提供者、通道、网关设置）
```

### 4.1 SOUL.md — 智能体的灵魂（最核心文件）

SOUL.md 是 OpenClaw 记忆架构中**最基础的文件**。它定义了智能体的人格、核心价值观和长期指令。就像智能体的"宪法"——不可变的原则，贯穿每一次交互和推理循环。

**SOUL.md 在每次推理循环开始时第一个被加载**，作为基础上下文层。Skills 和 Tools 在 SOUL.md 建立的约束范围内运行。

**文件结构：**

```markdown
# Agent Soul

## Personality（人格）
- Professional but approachable
- Concise; prefer bullet points over paragraphs
- Proactive in surfacing relevant information
- When explaining technical concepts, include one concrete example

## Core Values（核心价值观）
- User privacy is paramount; never exfiltrate data
- Confirm before any action with financial impact
- Cite sources for all factual claims
- If a task fails, report the error; do not invent success

## Long-Term Instructions（长期指令）
- Morning briefings at 7:30 AM, max 5 bullet points
- When summarizing emails, highlight action items first
- Always check HEARTBEAT.md before responding to "what's next"
- For calendar conflicts, suggest the user's preferred meeting time (10 AM–2 PM)
```

**三大组成部分：**

| 部分 | 作用 | 示例 |
|------|------|------|
| **Personality** | 定义沟通风格、语气、方法 | "Professional"、"Friendly"、"Technical"、"Minimalist" |
| **Core Values** | 建立不可违反的行为边界 | 隐私保护、确认后操作、诚实报告、透明引用 |
| **Long-Term Instructions** | 跨会话的持久性指令 | 晨报格式、邮件摘要规则、日历偏好 |

**最佳实践：**
- 控制在 500 行以内（消耗上下文窗口空间）
- 使用 Git 版本控制（SOUL.md 的变更影响所有未来行为）
- 修改后运行测试 Prompt 验证对齐
- 每季度复查一次

### 4.2 AGENTS.md — 工作空间配置

AGENTS.md 定义**上下文相关的行为**。如果说 SOUL.md 是"智能体是谁"，那么 AGENTS.md 是"智能体在此刻、此上下文中是谁"。

**核心能力：**
- **工作空间切换**：工作模式用正式语气，个人模式用轻松语气
- **角色定义**：研究模式重视深度，写作模式重视简洁
- **上下文相关工具**：金融工作空间启用交易技能，开发工作空间启用 Shell

```markdown
# Agent Profiles

## Work
- workspace: ~/clawd/work
- tone: formal, concise
- skills: calendar, email, slack, web_search
- heartbeat: HEARTBEAT-work.md

## Personal
- workspace: ~/clawd/personal
- tone: friendly, casual
- skills: calendar, weather, reminders
- heartbeat: HEARTBEAT-personal.md

## Dev
- workspace: ~/clawd/projects
- tone: technical
- skills: shell, file_system, browser, github
- heartbeat: HEARTBEAT-dev.md
```

每个 Profile 隔离上下文：**工作记忆留在工作区，个人记忆留在个人区，不交叉污染。**

**多智能体场景：** 每个智能体有自己的 AGENTS.md 部分。Strategy Agent 只读数据不执行，Execution Agent 只执行不做策略，Review Agent 只审查不执行。共享记忆（如 GOALS.md）协调各智能体。

### 4.3 HEARTBEAT.md — 主动任务清单

智能体每次主动执行的所有操作都源于这个文件。它使用标准 Markdown 复选框语法。

```markdown
# HEARTBEAT TASKS
> Last updated: Mar 23, 2026
> Heartbeat interval: 30 minutes

## Always Run（每次心跳都执行）
- [ ] Check that primary website returns 200 OK. Alert via Telegram if not.
- [ ] Verify disk usage on main server is below 80%. Alert if above 85%.

## Morning Routine（工作日 7:30-8:30）
- [ ] Pull today's calendar events and send a formatted daily briefing to Telegram
- [ ] Summarize any high-priority emails received overnight

## Market Monitoring（工作日 9-16 点）
- [ ] Check Bitcoin price. Alert if it moves more than 3% since last heartbeat.

## Weekly Review（仅周一上午）
- [ ] Summarize completed tasks from the past 7 days

## One-Time Tasks（一次性任务）
- [ ] Research top 5 AI agent frameworks
- [x] Set up initial HEARTBEAT.md (completed)
```

**任务编写三原则：**

1. **指定触发条件**：不要写"监控网站"，要写"检查网站是否返回 HTTP 200，如果不是则立即通过 Telegram 告警"
2. **定义紧急程度**：服务器宕机凌晨 2 点也要通知，但竞品发新文章可以等到早报
3. **包含相关上下文**：写清楚具体参数（如"我的 BTC 持仓成本 $48,000，止损 $45,000"）

**高级模式：**
- **条件升级**：第三次连续 500ms+ 响应时间，从 Telegram 升级到电话告警
- **自修改心跳**：Sprint 期间自动添加每日站会摘要任务，非 Sprint 期间移除
- **智能体间任务**：向其他智能体的任务文件写入指令

### 4.4 TOOLS.md — 工具定义

列出**低级别原语工具**：execute_shell、search_web、read_file、send_email 等。每个条目包含名称、描述、参数和安全级别。

```markdown
# TOOLS.md
- execute_shell: Run command in sandbox. Use for scripts, not arbitrary code.
  Never run rm -rf. Prefer .sh scripts over inline commands.
- search_web: Query search engine. Use for current information.
- read_file: Read file contents. Use for documents, configs.
- write_file: Write to file. Use for outputs, logs.
- send_email: Send via configured provider. Always confirm with user first.
```

**关键设计：**
- 工具描述不只是列举能力，还是**指导智能体如何使用**的 Prompt 工程
- 如果一个工具不在 TOOLS.md 中，智能体就不知道它的存在
- 移除条目 = 立即失去该能力（无需改代码、无需重启）

### 4.5 SKILLS.md — 技能定义

列出**高级别扩展技能**——将多个工具打包的模块化包。

```markdown
# SKILLS.md
- calendar: Google Calendar integration.
  Actions: get_events, create_event, find_free_slots.
  Use for scheduling and availability checks.
- email: Gmail read/send.
  Actions: summarize_inbox, send_message, search_threads.
- slack: Slack integration.
  Actions: post_message, read_channel, create_channel.
- browser: Playwright-based web automation.
  Actions: navigate, click, extract_text, screenshot.
```

**Tools vs Skills 的关系：**

| 对比 | TOOLS.md | SKILLS.md |
|------|----------|-----------|
| 级别 | 低级原语 | 高级组合 |
| 来源 | 内置在运行时 | ClawHub 或本地安装 |
| 示例 | execute_shell、read_file | calendar、email、slack |
| 类比 | 操作系统 syscall | 应用程序 |

两者共同定义了智能体的**能力边界**——智能体在规划时会同时查阅两个文件决定使用什么工具。

### 4.6 其他工作空间文件

| 文件 | 用途 |
|------|------|
| `IDENTITY.md` | 智能体名称和图标 emoji |
| `USER.md` | 用户身份信息和称呼偏好 |
| `MEMORY.md` | 手动策展的长期记忆条目 |
| `memory/YYYY-MM-DD.md` | 每日自动记忆日志 |
| `CONTACTS/` | 联系人 CRM（每人一个 .md 文件） |
| `config.yaml` | 主配置（LLM 提供者、通道、网关设置） |

---

## 五、安装与配置

### 5.1 系统要求

- **Node.js**：20.19.0+
- **Docker**：推荐（用于隔离）
- **反向代理**：nginx / Caddy / Traefik（生产环境）
- **SSL 证书**：Let's Encrypt

### 5.2 安装步骤

```bash
# 1. 全局安装
npm install -g @anthropic/openclaw

# 2. 首次设置（交互式引导）
openclaw onboard

# 3. 启动网关
openclaw gateway
```

### 5.3 核心配置文件 `openclaw.json`

配置文件默认位于 `~/.openclaw/openclaw.json`：

```json
{
  "gateway": {
    "bind": "lan",
    "port": 18789,
    "auth": {
      "mode": "token",
      "token": "YOUR_256_BIT_HEX_TOKEN"
    },
    "controlUI": false,
    "discovery": {
      "mdns": { "mode": "off" }
    }
  },
  "sandbox": {
    "mode": "all",
    "scope": "agent"
  },
  "session": {
    "dmScope": "per-channel-peer"
  },
  "channels": {
    "telegram": {
      "enabled": true,
      "token": "YOUR_BOT_TOKEN",
      "dmPolicy": "pairing",
      "groups": {
        "*": { "requireMention": true }
      }
    }
  },
  "logging": {
    "level": "info",
    "destination": "/var/log/openclaw/agent.log",
    "format": "json"
  }
}
```

### 5.4 LLM 模型配置

OpenClaw 是**模型无关**的，支持多种 LLM：

| 提供者 | 模型示例 | 配置方式 |
|--------|---------|---------|
| Anthropic | Claude 3.5/4 | `openclaw models auth paste-token --provider anthropic` |
| OpenAI | GPT-4o/GPT-5 | `openclaw models auth paste-token --provider openai` |
| DeepSeek | DeepSeek Chat | `openclaw config set agents.defaults.model deepseek/deepseek-chat` |
| Ollama（本地） | Llama 3、Qwen | 无需 API Key，本地运行 |
| Google | Gemini Ultra | `openclaw models auth paste-token --provider google` |

---

## 六、CLI 命令完整参考

### 6.1 网关命令

```bash
openclaw gateway                  # 启动网关（前台）
openclaw gateway --force          # 强制启动（先杀死已有实例）
openclaw gateway run              # 后台运行
openclaw gateway restart          # 重启（应用配置变更）
openclaw gateway health           # 健康检查
openclaw gateway status           # 详细状态
openclaw gateway install          # 安装为系统服务（开机自启）
openclaw gateway --port 18790     # 自定义端口
```

### 6.2 配置命令

```bash
openclaw config get                              # 查看完整配置
openclaw config get gateway.auth.token           # 获取特定值
openclaw config set agents.defaults.model gpt-5  # 设置值
openclaw config reset                            # 重置为默认
```

### 6.3 通道命令

```bash
openclaw channels add telegram --bot-token YOUR_TOKEN  # 添加 Telegram
openclaw channels add whatsapp                         # 添加 WhatsApp
openclaw channels add discord --bot-token YOUR_TOKEN   # 添加 Discord
openclaw channels list                                 # 列出已连接通道
openclaw channels remove telegram                      # 移除通道
```

### 6.4 诊断命令

```bash
openclaw doctor                       # 完整健康检查
openclaw doctor --fix                 # 自动修复常见问题
openclaw doctor --generate-gateway-token  # 生成安全令牌
```

检查内容：Node.js 版本、网关连接性、配置有效性、提供者认证、通道状态。

### 6.5 技能与模型命令

```bash
openclaw skills list                  # 列出已安装技能
openclaw skills install <name>        # 从市场安装技能
openclaw skills create my-skill       # 创建技能模板
openclaw models list                  # 列出可用模型
openclaw models auth paste-token --provider anthropic  # 认证提供者
```

### 6.6 备份与系统命令

```bash
openclaw backup create                # 创建备份
openclaw backup list                  # 列出备份
openclaw backup restore <id>          # 恢复备份
openclaw dashboard                    # 打开 Web 控制台
openclaw --version                    # 查看版本
openclaw uninstall --dry-run          # 预览卸载内容
```

---

## 七、端到端数据流示例

当你在 Telegram 发送 "今天日历上有什么？" 时：

```
Step 1  你在 Telegram 发送消息
   │
   ▼
Step 2  Telegram API 转发到你的 OpenClaw Webhook
   │
   ▼
Step 3  Telegram Channel Adapter 解析 webhook
        提取消息文本、发送者 ID、时间戳
        转换为 OpenClaw 统一内部消息对象
   │
   ▼
Step 4  Gateway 接收消息，路由到 Agent Runtime
   │
   ▼
Step 5  Agent Runtime 组装 Prompt：
        ├── 检索相关记忆（你的偏好、过去的日历查询）
        ├── 加载对话历史
        ├── 注入 SOUL.md 系统提示
        └── 添加工具定义（包括 Calendar Skill）
   │
   ▼
Step 6  LLM 推理：需要调用 get_calendar_events
        生成工具调用请求
   │
   ▼
Step 7  Runtime 执行 Calendar Skill 的 handler
        → 调用 Google Calendar API
        → 返回今日事件列表
   │
   ▼
Step 8  Runtime 将结果反馈给 LLM
   │
   ▼
Step 9  LLM 格式化自然语言响应：
        "你今天有 3 个事件：9 点站会、14 点客户电话、16 点团队同步"
   │
   ▼
Step 10 Telegram Adapter 将响应转为 Telegram 格式
        → 发送回 Telegram
   │
   ▼
Step 11 你在 Telegram 收到回复 ✓
```

**整个过程在几秒内完成。** 网关本身不增加显著开销，大部分时间花在 LLM 推理和 API 调用上。

---

## 八、安全注意事项

### 8.1 安全现状

2026 年 1 月，安全研究员 Maor Dayan 发现 **42,665 个暴露的 OpenClaw 实例**，其中 **93.4% 存在漏洞**。Cisco 发现 **26% 的智能体技能含有漏洞**。

Simon Willison 的"致命三角"解释了原因：同时满足以下三个条件的智能体天然危险：
1. 访问私人数据
2. 暴露于不可信内容
3. 可以对外通信

### 8.2 安全配置清单

| 检查项 | 措施 |
|--------|------|
| 网关绑定 | 绑定 `127.0.0.1`，而非 `0.0.0.0` |
| 令牌认证 | 启用 256-bit 令牌认证 |
| Control UI | 生产环境禁用 |
| mDNS 发现 | 禁用 |
| 反向代理 | 配置 nginx + SSL |
| 沙箱模式 | 启用所有智能体沙箱 |
| 出站过滤 | 配置 Squid 代理域名白名单 |
| 审计日志 | 启用 JSON 格式日志 |

### 8.3 Docker 部署（推荐）

```dockerfile
FROM node:22-bookworm-slim
RUN npm install -g @anthropic/openclaw
RUN useradd -m openclaw
USER openclaw
WORKDIR /app
COPY openclaw.json .
COPY --chown=openclaw:openclaw workspace ./workspace
EXPOSE 18789
CMD ["openclaw", "serve"]
```

---

## 九、生态系统：封装和发行版

由于 OpenClaw 是框架（类似 Linux），它可以被打包成多种形式：

| 发行版 | 描述 | 适合人群 |
|--------|------|---------|
| **ClawApp** | macOS/Windows 桌面应用 | 非技术用户 |
| **SimpleClaw** | 托管云服务 $5/月 | 想省心的用户 |
| **Spinup** | 托管云服务，面向日常使用 | 普通用户 |
| **WrapClaw** | 预装 50+ 插件的增强版 | 高级用户 |
| **Clawctl** | 企业级托管 $49/月 | 需要生产安全的企业 |
| **CrewClaw** | 可视化智能体生成器 $9/个 | 想快速创建智能体的用户 |
| **ClawSquire** | 桌面 GUI 管理工具 | 不喜欢 CLI 的用户 |

---

## 十、多智能体协作

OpenClaw 支持多个智能体协同工作。基于文件的记忆架构天然支持协调：

```
┌──────────────┐    共享 GOALS.md    ┌──────────────┐
│  Strategy    │◄──────────────────►│  Execution   │
│  Agent       │                    │  Agent       │
│  只读分析     │    共享 DECISIONS.md │  执行任务     │
│  不执行操作   │◄──────────────────►│  Shell+Slack │
└──────┬───────┘                    └──────┬───────┘
       │                                    │
       │         ┌──────────────┐          │
       └────────►│   Review     │◄─────────┘
                 │   Agent      │
                 │   审查结果    │
                 │   无执行权限  │
                 └──────────────┘
```

**四种编排模式：**
1. **Hub-Spoke**：一个中心智能体分配任务给专业智能体
2. **Pipeline**：智能体按顺序处理，每个处理一个阶段
3. **Peer-to-Peer**：智能体直接相互通信
4. **Debate**：多个智能体辩论以获得更好的答案

---

## 十一、与其他框架对比

| 特性 | OpenClaw | CrewAI | LangChain | AutoGPT | n8n |
|------|---------|--------|-----------|---------|-----|
| 类型 | 个人 AI 代理 | 多智能体框架 | AI 应用框架 | 自主智能体 | 工作流自动化 |
| 核心语言 | Node.js | Python | Python | Python | Node.js |
| 记忆系统 | Markdown 文件 | 内存 | 向量数据库 | 内存 | 无 |
| 多平台消息 | 原生支持 20+ | 无 | 需自建 | 无 | 集成支持 |
| 主动行为 | 心跳引擎 | 无 | 无 | 循环执行 | Cron 触发 |
| 本地运行 | 原生支持 | 是 | 是 | 是 | 是 |
| 模型无关 | 是 | 部分 | 是 | 部分 | 部分 |

---

## 十二、实际应用场景

| 场景 | 具体实现 |
|------|---------|
| **晨间简报** | 每天 7:30 自动汇总日历、邮件、天气，发到 Telegram |
| **服务器监控** | 每 5 分钟检查 HTTP 200，宕机立即告警 |
| **邮件分拣** | 自动分类、高亮待办事项、草拟回复 |
| **内容创作** | 自动化博客管线：选题 → 写作 → 排版 → 发布 |
| **客户支持** | 自动分类工单、生成首次回复、升级复杂问题 |
| **潜在客户挖掘** | 监控 Reddit/Twitter、追踪竞争对手、构建销售管线 |
| **智能家居** | 自然语言控制 Home Assistant |
| **代码审查** | 自动 Review PR，提出改进建议 |

---

## 十三、快速开始指南

### 一步一步开始：

```bash
# 1. 安装
npm install -g @anthropic/openclaw

# 2. 首次配置（交互式）
openclaw onboard

# 3. 配置 LLM
openclaw models auth paste-token --provider anthropic

# 4. 连接 Telegram
openclaw channels add telegram --bot-token YOUR_TOKEN

# 5. 启动
openclaw gateway

# 6. 健康检查
openclaw doctor
```

然后在 Telegram 中找到你的机器人，开始聊天！

### SOUL.md 快速模板：

```markdown
# Agent Soul

## Personality
- 中文回复，专业但亲和
- 简洁，偏好要点列表
- 主动呈现相关信息

## Core Values
- 用户隐私至上，不泄露数据
- 有财务影响的操作需确认
- 不确定时坦诚说明
- 任务失败时如实报告

## Long-Term Instructions
- 晨报 7:30，最多 5 个要点
- 邮件摘要优先展示待办事项
- 日历冲突建议 10:00-14:00 时段
```

---

## 十四、常见问题

**Q: OpenClaw 完全免费吗？**
A: OpenClaw 框架本身开源免费。但你需要支付 LLM API 费用（如使用 GPT-5 或 Claude），除非使用 Ollama 运行本地模型。

**Q: 需要什么硬件？**
A: 最低 2GB RAM、1 核 CPU 即可运行（不含本地模型）。如果用 Ollama 跑本地模型，建议 16GB+ RAM 和 GPU。

**Q: 我的数据安全吗？**
A: 自托管模式下数据完全在本地。但务必遵循安全配置清单，尤其是网关绑定和令牌认证。

**Q: 支持中文吗？**
A: 取决于你选择的 LLM 模型。Claude 和 GPT-5 对中文支持良好。在 SOUL.md 中指定"中文回复"即可。

**Q: 和 ChatGPT 有什么区别？**
A: ChatGPT 是云端聊天应用，每次对话独立。OpenClaw 是本地代理框架，有持久化记忆、主动心跳、多平台集成、工具执行能力，并且你拥有所有数据。
