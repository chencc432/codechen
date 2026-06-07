# 🚀 Argo 全家桶专题

> 这是一个面向"想在 Kubernetes 上搭建完整 GitOps + CI/CD + 渐进发布 + 事件驱动"体系的同学的 Argo 全家桶系统讲解。
> 目标不是让你把四个工具都装上，而是让你理解它们各自解决什么问题、怎么配合、在什么场景下选什么。

## 什么是 Argo 全家桶

Argo 项目是 CNCF 毕业项目，由四个核心工具组成：

```text
┌─────────────────────────────────────────────────────────────┐
│                    Argo 全家桶                                │
├──────────────┬──────────────┬──────────────┬────────────────┤
│  Argo CD     │ Argo Workflow│ Argo Rollouts│ Argo Events    │
│  GitOps 部署 │  工作流引擎  │  渐进发布    │  事件驱动      │
│              │              │              │                │
│  "代码即环境"│ "编排任务"   │ "安全上线"   │ "自动触发"     │
└──────────────┴──────────────┴──────────────┴────────────────┘
```

用一句话概括每个工具的定位：

| 工具 | 一句话 | 生活比喻 |
|------|--------|----------|
| **Argo CD** | 把 Git 仓库当作唯一真相源，自动把集群状态同步成 Git 里定义的样子 | 装修公司照着设计图验收房间——图纸变了就改房间 |
| **Argo Workflows** | 把多步骤任务编排成 DAG，在 K8s 上跑完 | 流水线工厂——每道工序自动接力 |
| **Argo Rollouts** | 用金丝雀 / 蓝绿策略安全地发新版本 | 新药临床试验——先给一小批人试，没问题再全推 |
| **Argo Events** | 监听各种事件源，触发 Workflow / K8s 资源创建 | 门铃系统——有人按铃就自动开门、开灯、通知你 |

## 这个专题适合谁

- 已经熟悉 Kubernetes 基础（Deployment、Service、CRD）
- 有以下任意一种诉求：
  - 想搭建 GitOps 体系，不想再手动 `kubectl apply`
  - 想实现金丝雀发布 / 蓝绿部署，不想全量一把梭
  - 想把 CI/CD Pipeline 跑在 K8s 上
  - 想用事件驱动的方式触发各种自动化任务
  - 想了解 Argo 四件套怎么协同工作
- 如果你只关注 Argo Workflow 的详细使用，推荐先看 [Argo Workflow 专题](../10-argo-workflow/README.md)

## 你会学到什么

1. **全局视野**：四个工具各自的定位、边界、最佳搭配方式
2. **Argo CD 完整掌握**：从安装到多集群、App of Apps、Secrets 管理
3. **Argo Rollouts 实战**：金丝雀、蓝绿、Analysis 自动回滚
4. **Argo Events 实战**：EventSource、Sensor、Trigger 三件套
5. **全家桶联动**：Git Push → Argo Events 触发 → Argo Workflow 构建 → Argo CD 同步 → Argo Rollouts 渐进发布

## 学习路线

### 路线 1：先建立全局认知

1. [Argo 全家桶全景](./01-argo-ecosystem-overview.md)
2. [Argo CD 核心概念与原理](./02-argocd-core-concepts.md)

### 路线 2：立即上手 Argo CD

1. [Argo CD 安装与快速上手](./03-argocd-installation-and-quickstart.md)
2. [Argo CD 进阶：多集群、App of Apps、Secrets](./04-argocd-advanced.md)

### 路线 3：安全上线（渐进发布）

1. [Argo Rollouts：金丝雀与蓝绿部署](./05-argo-rollouts-core.md)

### 路线 4：事件驱动自动化

1. [Argo Events：事件监听与自动触发](./06-argo-events-core.md)

### 路线 5：全家桶联动（完整 GitOps 流水线）

1. [全家桶联动实战](./07-argo-suite-integration.md)
2. [生产最佳实践与避坑](./08-production-best-practices.md)

## 文档列表

| 序号 | 标题 | 主要内容 |
|------|------|----------|
| 01 | [全家桶全景](./01-argo-ecosystem-overview.md) | 四个工具定位、关系、选型决策树、典型架构图 |
| 02 | [Argo CD 核心概念](./02-argocd-core-concepts.md) | Application、Sync、Health、Repo Server、GitOps 原理 |
| 03 | [Argo CD 安装与上手](./03-argocd-installation-and-quickstart.md) | 30 分钟装好、部署第一个 App、UI/CLI 使用 |
| 04 | [Argo CD 进阶](./04-argocd-advanced.md) | 多集群、App of Apps、ApplicationSet、Secrets、SSO |
| 05 | [Argo Rollouts](./05-argo-rollouts-core.md) | 金丝雀、蓝绿、AnalysisTemplate、自动回滚 |
| 06 | [Argo Events](./06-argo-events-core.md) | EventSource、Sensor、Trigger、EventBus |
| 07 | [全家桶联动](./07-argo-suite-integration.md) | 完整 GitOps Pipeline：push→build→deploy→canary |
| 08 | [生产最佳实践](./08-production-best-practices.md) | 权限、多租户、监控、灾备、常见坑 |

## 推荐配套阅读

- [Argo Workflow 专题](../10-argo-workflow/README.md)（Workflow 的 Spec、DAG、参数制品等细节）
- [自定义资源专题](../07-custom-resources/README.md)（理解 Argo 是怎么用 CRD + Controller 实现的）
- [Kubernetes 网络模型](../04-advanced/01-networking.md)（理解 Argo CD 的多集群网络打通）

## 阅读建议

1. **先读全景**：用 20 分钟建立四个工具的全局心智模型
2. **按需深入**：大多数人用不到全部四个工具，根据业务诉求选择性深入
3. **动手优先**：每个工具都给了"最小可运行"的例子，先跑通再读原理
4. **联动最后看**：等单个工具都能独立使用后，再看它们怎么串起来
