# 06. 真实使用案例

本章给出 7 个在实际工作中高频且收益明显的场景，每个都包括：**场景描述 → 完整 Prompt → 关键技巧 → 预期产出**。

## 目录

1. [为遗留项目补全单元测试](#案例-1-为遗留项目补全单元测试)
2. [定位并修复生产 bug](#案例-2-定位并修复生产-bug)
3. [跨文件重构：API 改名 / 模块拆分](#案例-3-跨文件重构)
4. [从 Issue 到 PR 的一站式开发](#案例-4-从-issue-到-pr-的一站式开发)
5. [代码审查与安全审计](#案例-5-代码审查与安全审计)
6. [新人入职：3 天读懂一个项目](#案例-6-新人入职读懂项目)
7. [批量脚本化任务（周报、统计、清理）](#案例-7-批量脚本化任务)

---

## 案例 1: 为遗留项目补全单元测试

### 场景
一个祖传 Express 项目，0 测试覆盖率，老板要求本周达到 60%。

### 思路
不要一次性"全部生成"。按**模块 → 函数 → 边界**三层推进，让 Claude 先选"投入产出比最高"的函数。

### Prompt 模板

```text
我需要给这个项目补单测，目标覆盖率 60%。请按以下方式推进：

【第 1 步：诊断】
1. 运行 npx jest --coverage（如果没有 jest 先装）
2. 读 src/ 的所有 .js 文件，列出所有导出函数
3. 按"重要性×易测性"排序，给我一个 Top 20 函数清单

【第 2 步：等我确认清单后】
每次只为清单中前 3 个函数写测试：
- 放在 __tests__/ 目录，文件名与源文件对应
- 每个函数至少覆盖：正常路径、边界值、异常输入
- 使用 jest，不要引入其他测试库
- 写完立即跑测试，失败的先修代码还是先改测试要问我

【约束】
- 不要修改任何现有源码
- 不要新增 npm 依赖（除了 jest 和 @types/jest）
- 每批测试跑通后暂停，等我说"继续"
```

### 关键技巧
- **Top 20 清单机制**：让 Claude 先做"信息整理"再做"生成"，避免写一堆没价值的测试。
- **每批 3 个函数**：方便 review，出问题也能快速回滚。
- **明确禁止改源码**：遗留代码的测试必须先"照旧状态"写，之后再讨论重构。

### 预期产出
- `__tests__/utils.test.js` 等测试文件
- 覆盖率报告，从 0% 提升到目标值
- 一份"暂时没测的函数及原因"清单

---

## 案例 2: 定位并修复生产 bug

### 场景
生产环境报错：`TypeError: Cannot read properties of undefined (reading 'id')`，堆栈指向 `src/services/order.ts:142`。用户只描述了"点击支付后白屏"。

### Prompt 模板

```text
线上报错如下：
===
TypeError: Cannot read properties of undefined (reading 'id')
    at OrderService.finalize (src/services/order.ts:142:23)
    at processPayment (src/handlers/pay.ts:58:12)
    at ...（省略）
用户操作：下单 → 点击支付 → 白屏
===

请按以下流程排查，不要急着改代码：

1. 读 src/services/order.ts 的 finalize 函数和 src/handlers/pay.ts 的 processPayment
2. 画一张数据流图（用 mermaid），说明从前端点击到 line 142 之间数据经过哪些地方
3. 给出 3 个最可能的根因假设，并说明如何验证（哪些 log、哪些测试）
4. 等我选定假设后，再提修复方案

特别注意：
- 支付相关代码高风险，任何修改都必须附带测试
- 不要改 src/infra/payment/ 下的文件
```

### 关键技巧
- **先根因后方案**：让 Claude 做"医生"不是"药房"。
- **Mermaid 数据流图**：可视化后人类 review 效率翻倍。
- **标出高风险目录**：CLAUDE.md 里已有规则时可省略。

### 推荐后续
修完后再问：
```text
这个 bug 类型（未校验返回值是否为 undefined）在项目里还有多少处？全局 grep 一下，列出可疑位置。
```
往往一个 bug 背后是一类隐患，顺手清掉。

---

## 案例 3: 跨文件重构

### 场景
想把一个叫 `userManager` 的全局单例改成 `UserService` 类，涉及 23 个文件。

### Prompt 模板

```text
重构需求：
- 旧: import { userManager } from '@/userManager'
- 新: 注入 UserService（NestJS DI）

请按以下步骤进行，每步结束等我确认：

【步骤 1：影响面分析】
grep 出所有引用 userManager 的位置，按文件分组列出，给我一个"改动预估清单"。

【步骤 2：建立新类，旧类暂不删除】
- 在 src/user/user.service.ts 新建 UserService
- 复制 userManager 的所有方法
- 在 UserModule 里注册为 provider
- 此时旧代码照常工作

【步骤 3：按文件批量迁移】
从改动最少的文件开始，每次迁移 3 个文件：
- 注入 UserService
- 替换调用
- 保证测试不变（暂不改测试）
- 跑 pnpm test 验证

【步骤 4：全部迁移完再删旧代码】
- 确认没有任何引用后，删除 src/userManager.ts
- 跑一次完整测试和构建

【步骤 5：更新测试】
把测试中对 userManager 的 mock 改为 UserService 的 mock。
```

### 关键技巧
- **新旧并存过渡期**：任何时刻都能跑起来，能提前合并 PR 分阶段上线。
- **从少到多**：先改最简单的文件，暴露方案问题。
- **测试放最后**：避免"方案一改，测试全红"的迷茫状态。

---

## 案例 4: 从 Issue 到 PR 的一站式开发

### 场景
在 GitHub 有个 issue：`#123 添加深色模式切换`。你想用 Claude 一把完成。

### 前置条件
配置好 GitHub MCP（见 04-core-features）。

### Prompt 模板

```text
处理 issue #123。按以下流程：

1. 用 github MCP 读取 issue #123 的正文和评论，总结需求要点
2. 提出 2 个实现方案（CSS 变量 vs Tailwind darkMode class），各列优劣
3. 等我选定后：
   a. 在 feature/123-dark-mode 分支工作
   b. 写实现
   c. 加单测 / 组件测试
   d. 更新 docs/ui.md
   e. 提交时 commit 信息遵循 Conventional Commits 并在正文引用 "Closes #123"
4. 不要 push，不要建 PR。全部完成后输出：
   - 改动文件清单
   - 建议的 PR 标题和正文（markdown）
   - 待 reviewer 关注的关键点
```

### 关键技巧
- **先需求后方案**：避免误解 issue。
- **不自动 push/建 PR**：保留人类最终审阅权。
- **给出 PR 描述模板**：你只需复制粘贴。

---

## 案例 5: 代码审查与安全审计

### 场景
PR 合并前想做一轮 AI 审查。

### 配合自定义命令

`.claude/commands/review.md`:

```markdown
对当前分支相对 main 的改动做严格 review：

1. 先 `git diff main...HEAD --stat` 看规模
2. 逐文件 diff，按以下维度检查：
   - 命名/可读性
   - 边界条件、异常处理
   - 并发/幂等问题
   - 性能（N+1 查询、重复计算）
   - 安全（注入、XSS、密钥、越权）
   - 测试充分性
3. 输出 markdown 报告：
   - 🔴 必须修（带文件:行号和建议代码）
   - 🟡 建议改
   - 🟢 做得好的地方
4. 最后给一个整体评分（通过 / 需修改 / 打回）
```

使用：

```text
/review
```

### 安全审计专用

```text
用 security-reviewer 子智能体对 src/api/ 做一次安全审计。
重点关注：
- 用户输入是否被校验
- SQL 是否参数化
- 认证/鉴权是否到位
- 敏感数据日志脱敏
```

---

## 案例 6: 新人入职读懂项目

### 场景
刚加入团队，面对 50 万行代码的项目，两周后要独立干活。

### Day 1：全局鸟瞰

```text
我是新人，请帮我快速理解这个项目。产出：

1. 一份架构总览（200 字 + 一张 mermaid 架构图）
2. 前 10 个最重要的文件清单，每个配一句话说明
3. 核心业务流程：用户下单的完整调用链（用 sequence diagram）
4. 技术债务和坑点清单（看 TODO/FIXME/HACK 注释、近期 bug 修复提交）

输出到 onboarding/overview.md
```

### Day 2：建立词汇表

```text
读 src/domain/ 和 README.md，提取项目里使用的业务术语（比如 "Order"、"Fulfillment"、"Voucher"），
写一份术语表到 onboarding/glossary.md，中英文对照 + 一句话解释。
```

### Day 3：学习路径

```text
根据架构总览，为我制定一份 5 天的代码阅读计划。每天指定：
- 要读的文件（按依赖关系排序）
- 读完能回答的 3 个问题
- 一个动手练习（比如：改一个变量名、加一行日志、跑一个单测）
```

### 关键技巧
- **把 AI 当"学长"**：它读源码比你快，但理解不一定准，务必配合人工追问。
- **产出落到文件里**：你以后可以回看，新人来了也能用。

---

## 案例 7: 批量脚本化任务

Claude Code 的 `-p` 模式让它可以像 Unix 工具一样嵌入脚本。

### 7.1 自动生成周报

```bash
git log --since="1 week ago" --author="$(git config user.email)" \
  --pretty=format:"%h %s" \
  | claude -p "根据这些 commit，写一份周报，分三部分：完成、进行中、下周计划。输出 markdown，不超过 300 字。"
```

### 7.2 批量补注释

```bash
for f in src/utils/*.ts; do
  claude -p "为 $f 中所有导出函数补 JSDoc 注释，直接修改文件，不要改代码逻辑。"
done
```

### 7.3 依赖升级风险评估

```bash
npm outdated --json | claude -p "分析这些过时依赖，按升级风险（低/中/高）分类，每个给出升级建议。"
```

### 7.4 日志分析

```bash
tail -n 1000 app.log | claude -p "分析日志中的错误模式，Top 5 的错误类型和出现次数，建议排查方向。"
```

### 7.5 每日 CI 代码审查机器人

`.github/workflows/ai-review.yml`:

```yaml
name: AI Review
on: [pull_request]
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-node@v4
        with: { node-version: 20 }
      - run: npm install -g @anthropic-ai/claude-code
      - name: Run review
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          git diff origin/${{ github.base_ref }}...HEAD \
            | claude -p "对这些 diff 做 code review，输出 markdown，只报 🔴 和 🟡 问题。" \
            > review.md
      - uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const body = fs.readFileSync('review.md', 'utf8');
            await github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `### 🤖 AI Review\n\n${body}`
            });
```

---

## 场景 x 角色 推荐矩阵

| 角色 | 最值得先用的场景 |
| --- | --- |
| 独立开发者 | 案例 1、2、3 |
| 团队 lead | 案例 4、5、7.5 |
| 新人 | 案例 6 |
| 运维/平台工程 | 案例 7.1 ~ 7.4 |
| 架构师 | 案例 3、5 + 子智能体 |

## 下一步

👉 [07-advanced.md](./07-advanced.md) 高级特性与团队集成
