# Claude Code 使用文档

> 本文档为 Claude Code（Anthropic 官方推出的 AI 编码助手命令行工具）的完整中文使用指南。

## 📚 文档索引

| 文档 | 内容 | 适合读者 |
| --- | --- | --- |
| [01-introduction.md](./01-introduction.md) | Claude Code 是什么、能做什么、与 Cursor/Copilot 的对比 | 新手入门 |
| [02-installation.md](./02-installation.md) | 安装、登录、环境配置（含 Windows/macOS/Linux） | 首次安装者 |
| [03-quickstart.md](./03-quickstart.md) | 10 分钟快速上手：第一个项目实战 | 快速体验 |
| [04-core-features.md](./04-core-features.md) | 核心功能详解：斜杠命令、上下文管理、工具调用、MCP | 日常使用者 |
| [05-best-practices.md](./05-best-practices.md) | 最佳实践与效率建议（CLAUDE.md、任务拆分、Prompt 技巧） | 所有开发者 |
| [06-use-cases.md](./06-use-cases.md) | 真实使用案例：重构、测试、调试、代码审查等 7 个场景 | 进阶用户 |
| [07-advanced.md](./07-advanced.md) | 高级特性：Hooks、Subagents、CI/CD 集成、自动化 | 团队负责人/平台工程师 |
| [08-troubleshooting.md](./08-troubleshooting.md) | 常见问题、错误排查、性能调优 | 遇到问题时查阅 |

## 🚀 快速开始

```bash
# 安装（需要 Node.js 18+）
npm install -g @anthropic-ai/claude-code

# 进入你的项目目录并启动
cd your-project
claude

# 第一次会要求登录 Anthropic 账号
```

然后直接用自然语言对话即可，例如：

```text
帮我看看这个项目的结构，并修复 README 中的拼写错误
```

## 🎯 本文档适合谁？

- 想快速上手 AI 编码助手的 **个人开发者**
- 希望将 Claude Code 引入团队工作流的 **团队技术负责人**
- 需要把 Claude Code 集成到 CI/CD 或内部平台的 **平台工程师**
- 已经在用 Cursor/Copilot，想了解 Claude Code 差异的 **AI 工具探索者**

## 📌 版本说明

- 本文档基于 Claude Code **2025Q4 ~ 2026Q2** 版本编写。
- Claude Code 迭代较快，如遇命令/参数差异请以 `claude --help` 和官方文档为准：https://docs.claude.com/en/docs/claude-code

## 🙋 反馈与贡献

- 发现错误/想补充案例：欢迎直接修改对应 `.md` 文件。
- 建议优先读：`03-quickstart` → `05-best-practices` → `06-use-cases`。
