# 🌐 Argo 全家桶全景概览

## 从一家餐厅说起

想象你开了一家连锁餐厅，业务越来越复杂：

- **菜谱管理**：所有分店必须严格按照总部菜谱做菜，菜谱改了所有分店同步更新 → 这就是 **Argo CD**（Git 里存菜谱，自动同步到所有集群）
- **后厨流水线**：洗菜 → 切菜 → 炒菜 → 装盘，每道工序有依赖、有并行 → 这就是 **Argo Workflows**（把多步骤编排成 DAG 执行）
- **新菜试吃**：新菜品不会直接上全部餐桌，先让 10% 的顾客试吃，反馈好了再全推 → 这就是 **Argo Rollouts**（金丝雀 / 蓝绿渐进发布）
- **自动叫号**：有客人按门铃 → 自动出票 → 后厨开始做菜 → 通知服务员上菜 → 这就是 **Argo Events**（事件驱动触发）

把这四套系统串起来，就是一个完整的"从代码推送到安全上线"的自动化体系。

## 四个工具各自的定位

```text
你的代码仓库（GitHub/GitLab）
        │
        │ ① Push 事件
        ▼
┌─────────────────┐
│   Argo Events   │  监听事件源（Git push、Webhook、MQ、Cron...）
│   "门铃系统"    │  收到事件后触发动作
└────────┬────────┘
         │ ② 触发构建
         ▼
┌─────────────────┐
│  Argo Workflows │  CI Pipeline：构建镜像、跑测试、推到仓库
│  "流水线工厂"   │  按 DAG 编排多步骤任务
└────────┬────────┘
         │ ③ 新镜像 tag 写入 Git
         ▼
┌─────────────────┐
│    Argo CD      │  检测到 Git 变化，自动同步到集群
│   "装修队长"    │  保证集群状态 = Git 定义的状态
└────────┬────────┘
         │ ④ 更新 Rollout 对象
         ▼
┌─────────────────┐
│  Argo Rollouts  │  渐进发布：5% → 20% → 50% → 100%
│  "临床试验"     │  每一步都可以用指标分析自动回滚
└─────────────────┘
```

## 每个工具一页纸讲清楚

### Argo CD —— "Git 是唯一真相"

**核心思想**：你不应该 `kubectl apply` 到生产环境，而应该把所有 YAML 存到 Git 里，由 Argo CD 负责把 Git 的内容"同步"到集群。

```text
传统方式：
  开发者 → kubectl apply → 集群
  问题：谁改的？改了什么？怎么回滚？不知道！

GitOps 方式（Argo CD）：
  开发者 → git push → Git 仓库 ←── Argo CD（每 3 分钟对比）──→ 集群
  好处：所有变更有审计、有 PR、有回滚、有 diff
```

**核心 CRD**：`Application`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/k8s-manifests.git
    targetRevision: main
    path: apps/my-app        # Git 里 YAML 文件的路径
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:               # 自动同步（检测到 Git 变化就 apply）
      prune: true            # Git 里删了的资源，集群也删
      selfHeal: true         # 有人手动改了集群，自动改回来
```

**关键词**：Sync（同步）、Health（健康）、Diff（差异）、Prune（修剪）

---

### Argo Workflows —— "多步骤 DAG 编排"

**核心思想**：把"一组按依赖关系执行的容器任务"描述成 YAML，由控制器自动按 DAG 拉起 Pod 执行。

> 这部分已有 [完整专题](../10-argo-workflow/README.md)，这里只给核心速览。

**核心 CRD**：`Workflow`、`WorkflowTemplate`、`CronWorkflow`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: ci-pipeline-
spec:
  entrypoint: build-and-test
  templates:
    - name: build-and-test
      dag:
        tasks:
          - name: checkout
            template: git-clone
          - name: unit-test
            dependencies: [checkout]
            template: run-tests
          - name: build-image
            dependencies: [checkout]
            template: docker-build
          - name: push-image
            dependencies: [unit-test, build-image]
            template: docker-push
```

**关键词**：DAG、Steps、Template、Parameters、Artifacts

---

### Argo Rollouts —— "发新版不再心惊肉跳"

**核心思想**：替代原生 Deployment 的 RollingUpdate 策略，提供金丝雀（Canary）和蓝绿（Blue-Green）两种渐进发布策略，每一步都能用 Prometheus/Datadog 等指标自动判断是否继续。

```text
传统 Deployment 滚动更新：
  旧 Pod 逐个被新 Pod 替换，没有"观察期"
  一旦新版本有 bug，所有用户立刻受影响

Argo Rollouts 金丝雀发布：
  Step 1: 5% 流量给新版 → 等 5 分钟 → 检查错误率
  Step 2: 20% 流量给新版 → 等 10 分钟 → 检查 P99 延迟
  Step 3: 50% 流量 → 确认无异常
  Step 4: 100% 完成发布
  任何一步指标异常 → 自动回滚到旧版本
```

**核心 CRD**：`Rollout`、`AnalysisTemplate`、`AnalysisRun`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  replicas: 10
  strategy:
    canary:
      steps:
        - setWeight: 10          # 先放 10% 流量
        - pause: {duration: 5m}  # 观察 5 分钟
        - setWeight: 30
        - pause: {duration: 10m}
        - setWeight: 60
        - pause: {duration: 10m}
        - setWeight: 100         # 全量
      canaryMetadata:
        labels:
          role: canary
  selector:
    matchLabels:
      app: my-app
  template:
    # ... 和 Deployment 的 Pod template 一模一样
```

**关键词**：Canary、Blue-Green、AnalysisTemplate、setWeight、promote、abort

---

### Argo Events —— "万物皆可触发"

**核心思想**：监听各种事件源（Git push、Webhook、消息队列、定时、S3 文件上传...），根据条件触发动作（创建 Workflow、创建 K8s 资源、调用 HTTP...）

```text
┌─────────────┐      ┌──────────┐      ┌─────────────┐
│ EventSource │ ──→  │  Sensor  │ ──→  │   Trigger   │
│  "耳朵"     │      │  "大脑"  │      │   "手脚"    │
│ 监听事件    │      │ 过滤判断 │      │  执行动作   │
└─────────────┘      └──────────┘      └─────────────┘
```

**三个核心 CRD**：

1. **EventSource**（事件源）：监听 GitHub Webhook、Kafka、NATS、S3、Cron...
2. **Sensor**（传感器）：收到事件后，根据过滤条件决定是否触发
3. **Trigger**（触发器）：真正的动作——创建 Workflow、创建 Pod、调用 API...

```yaml
# EventSource：监听 GitHub Push 事件
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: github-push
spec:
  github:
    my-repo:
      repositories:
        - owner: your-org
          names: [my-app]
      webhook:
        endpoint: /push
        port: "12000"
      events: [push]
---
# Sensor：收到 push 事件 → 触发构建 Workflow
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: ci-trigger
spec:
  dependencies:
    - name: push-dep
      eventSourceName: github-push
      eventName: my-repo
  triggers:
    - template:
        name: build-workflow
        argoWorkflow:
          operation: submit
          source:
            resource:
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: ci-build-
              spec:
                entrypoint: build
                # ... workflow spec
```

**关键词**：EventSource、Sensor、Trigger、EventBus、依赖过滤

## 四个工具的关系——一张图说清

```text
┌────────────────────────────────────────────────────────────────────┐
│                        完整 GitOps 流水线                           │
│                                                                    │
│  ┌──────┐  push   ┌───────────┐  trigger   ┌──────────────┐       │
│  │ 开发者├────────▶│Argo Events├───────────▶│Argo Workflows│       │
│  └──────┘         └───────────┘            └──────┬───────┘       │
│                                                    │ 构建完成      │
│                                                    │ 更新 Git 镜像 tag
│                                                    ▼               │
│  ┌──────────┐  sync    ┌────────┐  管理    ┌────────────┐         │
│  │  集群    │◀─────────│Argo CD │◀─────────│ Git 仓库   │         │
│  │          │          └────────┘          └────────────┘         │
│  │          │                                                     │
│  │  ┌──────────────┐                                              │
│  │  │Argo Rollouts │  金丝雀/蓝绿                                  │
│  │  │  渐进发布    │  分析指标 → 继续/回滚                          │
│  │  └──────────────┘                                              │
│  └────────────────────────────────────────────────────────────────┘
```

## 我该用哪个？选型决策树

```text
你的需求是什么？
│
├─ "我想让 Git 成为唯一部署入口，自动同步到集群"
│   └─→ 用 Argo CD
│
├─ "我想在 K8s 上跑多步骤任务（CI、训练、ETL）"
│   └─→ 用 Argo Workflows
│       └─ 已有专题详解 → ../10-argo-workflow/
│
├─ "我想安全地发版本，先小流量验证再全推"
│   └─→ 用 Argo Rollouts
│
├─ "我想让 Git push / MQ 消息自动触发 Pipeline"
│   └─→ 用 Argo Events
│
└─ "我全都要！完整的 GitOps + CI/CD + 渐进发布"
    └─→ 四个一起用（看第 07 篇联动实战）
```

## 它们跟你已经在用的工具是什么关系

| 你在用的 | Argo 对应方案 | 两者关系 |
|---------|--------------|---------|
| Jenkins / GitLab CI | Argo Workflows | Argo 是云原生方案，每步是 Pod 而非 Jenkins slave |
| Flux CD | Argo CD | 都是 GitOps，Argo CD 有 UI + 多集群管理更强 |
| Spinnaker | Argo Rollouts + Argo CD | Spinnaker 太重，Argo 更轻量 K8s 原生 |
| Tekton | Argo Workflows | 定位类似，Argo 社区更大、DAG 更灵活 |
| Helm hooks | Argo CD + Hooks | Argo CD 直接支持 Helm Chart 作为 Source |
| K8s Deployment | Argo Rollouts | Rollout 是 Deployment 的增强版，多了金丝雀/蓝绿 |
| CronJob | Argo Workflows CronWorkflow | CronWorkflow 支持 DAG 多步骤定时 |
| 手动 kubectl apply | Argo CD | GitOps 替代手动部署 |

## 四个工具的架构共性

Argo 全家桶的四个工具有非常一致的架构套路：

| 共性 | 说明 |
|------|------|
| **全是 CRD + Controller** | 每个工具都定义自己的 CRD，有自己的控制器来 reconcile |
| **声明式** | 你描述"想要的状态"，控制器负责把"当前状态"驱动到"想要的状态" |
| **K8s 原生** | 装在 K8s 里，用 K8s RBAC，存在 etcd 里，kubectl 能管 |
| **有 UI + CLI** | 每个工具都提供 Web UI 和命令行工具 |
| **可独立也可组合** | 四个工具可以单独用，也可以自由组合 |

## 安装复杂度与学习曲线

| 工具 | 安装难度 | 学习曲线 | 最小使用时间 |
|------|---------|---------|------------|
| Argo CD | ⭐⭐ | ⭐⭐ | 1 小时能跑通第一个 App |
| Argo Workflows | ⭐⭐ | ⭐⭐⭐ | 30 分钟跑通 hello world |
| Argo Rollouts | ⭐ | ⭐⭐ | 30 分钟改一个 Deployment 成 Rollout |
| Argo Events | ⭐⭐⭐ | ⭐⭐⭐⭐ | 需要理解 EventSource+Sensor+Trigger 三层 |

## 你需要建立的核心心智模型

在进入后面每个工具的详解之前，请记住这 5 点：

1. **Argo 是 K8s 的扩展，不是替代品**：它们都跑在 K8s 上，用 CRD 定义、用 Controller 执行
2. **Git 是真相**（对 Argo CD 来说）：集群状态应该等于 Git 里写的状态
3. **每一步都是 Pod**（对 Workflows 和 Rollouts 来说）：Argo 不直接跑业务代码，它只是编排 Pod
4. **四个工具可以独立使用**：先用 Argo CD 就够了，不需要一次性全上
5. **Event → Workflow → CD → Rollout** 是最完整的链路，但大多数团队只用其中 2-3 个

## 下一步

接下来建议按这个顺序：

1. **先看 Argo CD**（大多数人最先需要的工具）：[Argo CD 核心概念与原理](./02-argocd-core-concepts.md)
2. 如果你对 Workflow 感兴趣，直接去 [Argo Workflow 专题](../10-argo-workflow/README.md)
3. 如果你已经会用 Argo CD 了，直接跳到 [Argo Rollouts](./05-argo-rollouts-core.md)
