# 🎯 Argo CD 核心概念与原理

## 1. 一句话定义

> Argo CD 是一个 Kubernetes 原生的 GitOps 持续交付工具——它持续监控 Git 仓库中的应用定义，自动将集群状态同步成 Git 中描述的状态。

如果用更通俗的话说：

- **Git 仓库是"设计图"**
- **Kubernetes 集群是"房间"**
- **Argo CD 是"装修队长"**——每隔 3 分钟对比一次设计图和房间现状，发现不一致就改房间

## 2. 为什么需要 Argo CD

### 没有 Argo CD 时（传统部署）

```text
开发者 A: kubectl apply -f deploy.yaml      # 直接改了生产环境
开发者 B: helm upgrade my-app ./chart       # 也改了
运维 C:   kubectl scale deploy/app --replicas=5  # 手动调了

一周后出事了...
  ❓ 谁改的？ → 不知道
  ❓ 改了什么？ → 查不清
  ❓ 怎么回滚？ → 手动找上个版本的 YAML
  ❓ 现在集群状态是什么？ → 和 Git 里的对不上
```

### 有了 Argo CD 后（GitOps）

```text
开发者 A: 提 PR 修改 deploy.yaml → 代码审查 → 合并到 main
Argo CD:  检测到 main 分支变了 → 自动 sync 到集群

一周后出事了...
  ✅ 谁改的？ → Git log 看得清清楚楚
  ✅ 改了什么？ → PR 里有 diff
  ✅ 怎么回滚？ → git revert 那次 commit → Argo CD 自动同步回去
  ✅ 现在集群状态是什么？ → 等于 Git 的 main 分支里定义的状态
```

**核心原则**：任何人、任何情况下，都不应该直接 `kubectl apply` 到生产环境。所有变更必须走 Git。

## 3. GitOps 的三个核心原则

| 原则 | 含义 | Argo CD 如何实现 |
|------|------|-----------------|
| **声明式** | 系统的期望状态用声明式描述 | Git 里的 YAML / Helm / Kustomize |
| **版本化 + 不可变** | 所有变更通过 Git 提交，有完整历史 | Git 天然做到 |
| **自动拉取** | 自动将系统状态向期望状态收敛 | Argo CD 每 3 分钟 pull + reconcile |

注意第三点：传统 CI/CD 是 **push**（CI 构建完主动 push 到集群），GitOps 是 **pull**（Argo CD 主动 pull Git 并对比同步）。

```text
Push 模式（传统 CI/CD）：
  CI Server --push--> 集群
  问题：CI 需要集群的高权限凭据；CI 挂了就部署不了

Pull 模式（GitOps / Argo CD）：
  Argo CD <--pull-- Git 仓库
  Argo CD --sync--> 自己所在的集群（或远端集群）
  优势：不用把集群凭据暴露给外部；Argo CD 在集群内有天然权限
```

## 4. 核心概念详解

### 4.1 Application（应用）

Argo CD 最核心的 CRD。一个 Application 描述了：

- **Source**：从哪个 Git 仓库、哪个路径、哪个分支拿配置
- **Destination**：部署到哪个集群、哪个 Namespace
- **SyncPolicy**：手动同步还是自动同步

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: guestbook
  namespace: argocd           # Application 本身必须在 argocd 命名空间
spec:
  project: default

  # 从哪里拿 YAML
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD      # 分支/tag/commit
    path: guestbook           # 仓库里的子目录

  # 部署到哪里
  destination:
    server: https://kubernetes.default.svc   # 本集群
    namespace: guestbook

  # 同步策略
  syncPolicy:
    automated:
      prune: true             # Git 里删了的资源，集群也删
      selfHeal: true          # 有人手动改了集群状态，自动恢复
    syncOptions:
      - CreateNamespace=true  # namespace 不存在时自动创建
```

**一个 Application = 一个部署单元**。你可以为每个微服务创建一个 Application。

### 4.2 Sync（同步）

Sync 是 Argo CD 最核心的动作——把集群状态从"当前"驱动到 Git 定义的"期望"。

Sync 有两种模式：

| 模式 | 说明 | 使用场景 |
|------|------|---------|
| **Manual Sync** | 必须手动点"Sync"按钮或 `argocd app sync` | 生产环境需要审批 |
| **Auto Sync** | 检测到 Git 变化自动 sync | 开发/测试环境 |

Sync 的过程：

```text
① Argo CD 拉取 Git 仓库最新内容
② 渲染 YAML（如果是 Helm/Kustomize，先 template/build）
③ 对比渲染结果与集群当前状态（diff）
④ 如果有差异：kubectl apply 那些变化的资源
⑤ 如果 prune=true：删除 Git 里已经没有但集群还存在的资源
⑥ 等待所有资源变 Healthy
```

### 4.3 Health（健康状态）

Argo CD 对每个管理的资源都有健康评估：

| 状态 | 含义 | 举例 |
|------|------|------|
| **Healthy** ✅ | 资源处于正常状态 | Deployment 所有副本都 Ready |
| **Progressing** 🔄 | 资源正在变化中 | Deployment 正在滚动更新 |
| **Degraded** ⚠️ | 资源有问题 | Pod 处于 CrashLoopBackOff |
| **Suspended** ⏸️ | 资源被暂停 | CronJob 被 suspend |
| **Missing** ❌ | 资源在 Git 里有定义但集群里不存在 | 首次部署前 |
| **Unknown** ❓ | 无法判断 | 自定义 CRD 没有 Health Check 规则 |

### 4.4 Sync Status（同步状态）

| 状态 | 含义 |
|------|------|
| **Synced** | 集群状态 = Git 定义的状态 |
| **OutOfSync** | 集群状态 ≠ Git 定义的状态（需要 sync） |

**OutOfSync 的三种常见原因**：

1. Git 里 YAML 改了，但还没 sync 到集群
2. 有人手动 kubectl 改了集群（如果没开 selfHeal）
3. Argo CD 检测到新的 commit

### 4.5 Project（项目）

用于多租户隔离——限制 Application 能访问哪些 Git 仓库、能部署到哪些集群/命名空间、能部署哪些资源类型。

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: team-backend
  namespace: argocd
spec:
  description: "后端团队的项目"
  sourceRepos:
    - 'https://github.com/your-org/backend-*'    # 只能用后端仓库
  destinations:
    - namespace: 'backend-*'                      # 只能部署到 backend- 开头的 ns
      server: https://kubernetes.default.svc
  clusterResourceWhitelist:
    - group: ''
      kind: Namespace                             # 允许创建 Namespace
  namespaceResourceBlacklist:
    - group: ''
      kind: ResourceQuota                         # 不允许动 ResourceQuota
```

### 4.6 Repository（仓库）

Argo CD 需要知道怎么连接你的 Git 仓库。支持：

- HTTPS + 用户名/密码或 Token
- SSH Key
- GitHub App

```bash
# 添加私有仓库
argocd repo add https://github.com/your-org/manifests.git \
  --username admin \
  --password $GITHUB_TOKEN
```

## 5. Argo CD 的架构

```text
┌─────────────────────────────────────────────────────────────┐
│                     argocd 命名空间                           │
│                                                             │
│  ┌─────────────┐   ┌──────────────┐   ┌────────────────┐   │
│  │ argocd-     │   │ argocd-      │   │ argocd-        │   │
│  │ server      │   │ repo-server  │   │ application-   │   │
│  │             │   │              │   │ controller     │   │
│  │ • UI        │   │ • 克隆 Git   │   │                │   │
│  │ • REST API  │   │ • 渲染 YAML  │   │ • Watch App CR │   │
│  │ • gRPC API  │   │ • Helm/Kust  │   │ • Diff 对比    │   │
│  │ • SSO 登录  │   │   模板渲染   │   │ • 执行 Sync    │   │
│  └─────────────┘   └──────────────┘   │ • Health Check │   │
│                                        └────────────────┘   │
│  ┌─────────────┐   ┌──────────────┐                         │
│  │ argocd-dex  │   │ Redis        │                         │
│  │ (SSO 代理)  │   │ (缓存)       │                         │
│  └─────────────┘   └──────────────┘                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         ↕ watch                ↕ clone/pull
    ┌──────────┐           ┌──────────────┐
    │ K8s API  │           │  Git 仓库    │
    │ (集群)   │           │  (GitHub等)  │
    └──────────┘           └──────────────┘
```

**各组件职责**：

| 组件 | 干什么 | 类比 |
|------|--------|------|
| **argocd-server** | 提供 UI + API，处理用户请求 | 售楼处前台 |
| **repo-server** | 克隆 Git、渲染 Helm/Kustomize，生成最终 YAML | 图纸翻译员 |
| **application-controller** | Watch Application CR，对比集群与 Git 差异，执行同步 | 装修队长 |
| **Redis** | 缓存 Git 仓库信息和 App 状态 | 记事本 |
| **Dex** | 处理 SSO 登录（OIDC / LDAP / GitHub OAuth） | 门禁刷卡系统 |

**核心流程**：

```text
① application-controller 每 3 分钟（可配）轮询：
   "我管理的所有 Application，Git 里有没有新 commit？"

② 如果有新 commit：
   → 告诉 repo-server："帮我克隆这个仓库的这个分支，渲染成最终 YAML"
   → repo-server 返回渲染好的 manifests

③ application-controller 拿到 manifests 后：
   → 用 kubectl get 拿到集群当前的资源
   → 做 diff 对比

④ 如果有差异（OutOfSync）：
   → 如果 syncPolicy.automated = true：自动 apply
   → 如果手动模式：标记为 OutOfSync，等用户点 Sync

⑤ apply 之后：
   → 持续检查每个资源的 Health 状态
   → 直到所有资源 Healthy 或超时
```

## 6. Source 支持的配置格式

Argo CD 不限于"裸 YAML"，它支持多种配置格式：

| 格式 | 说明 | 适用场景 |
|------|------|---------|
| **目录（plain YAML）** | 直接读路径下所有 .yaml/.json | 简单项目 |
| **Helm Chart** | 自动 `helm template` 渲染 | 复杂应用有 values 覆盖 |
| **Kustomize** | 自动 `kustomize build` | 多环境（dev/staging/prod）同一套 base |
| **Jsonnet** | 自动渲染 Jsonnet 文件 | 程序化生成 manifest |
| **自定义插件** | 配置 ConfigManagementPlugin | 特殊需求 |

### Helm 示例

```yaml
spec:
  source:
    repoURL: https://charts.bitnami.com/bitnami
    chart: redis                    # Helm Chart 名
    targetRevision: 17.3.14         # Chart 版本
    helm:
      releaseName: my-redis
      values: |
        replica:
          replicaCount: 3
        auth:
          enabled: true
```

### Kustomize 示例

```text
Git 仓库结构：
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
├── overlays/
│   ├── dev/
│   │   └── kustomization.yaml    # replicas: 1, image: :dev
│   ├── staging/
│   │   └── kustomization.yaml    # replicas: 2, image: :staging
│   └── prod/
│       └── kustomization.yaml    # replicas: 5, image: :v1.2.3
```

```yaml
# dev 环境的 Application
spec:
  source:
    repoURL: https://github.com/your-org/manifests.git
    path: overlays/dev            # 指向 dev overlay
    targetRevision: main
```

## 7. Sync Wave 与 Sync Hook

当你的应用有复杂依赖（比如先建 Namespace → 再建 Secret → 再建 Deployment），可以用 **Sync Wave** 控制顺序：

```yaml
# 这个资源在 wave 0（最先创建）
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"
---
# 这个资源在 wave 1（等 wave 0 的都 Healthy 了才开始）
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"
---
# 这个资源在 wave 2
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "2"
```

**Sync Hook** 可以在 Sync 的不同阶段执行任务：

| Hook | 时机 | 典型用途 |
|------|------|---------|
| `PreSync` | 同步前 | 数据库 Migration |
| `Sync` | 同步中 | 默认阶段 |
| `PostSync` | 同步后 | 发通知、跑集成测试 |
| `SyncFail` | 同步失败时 | 发告警 |

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
  annotations:
    argocd.argoproj.io/hook: PreSync           # 在 sync 前执行
    argocd.argoproj.io/hook-delete-policy: HookSucceeded  # 跑完删掉
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: my-app:v1.2.3
          command: ["./migrate", "--up"]
      restartPolicy: Never
```

## 8. Prune 与 Self-Heal

### Prune（修剪）

- 你从 Git 里删掉了某个 Service 的定义
- 如果 `prune: true`：Argo CD 下次 Sync 时会从集群里也删掉它
- 如果 `prune: false`：Argo CD 只会标记它为 "orphaned"，不删除

### Self-Heal（自愈）

- 有人手动 `kubectl scale deployment/app --replicas=1`
- 如果 `selfHeal: true`：Argo CD 检测到集群状态偏离 Git 定义，自动改回来
- 如果 `selfHeal: false`：Argo CD 会标记为 OutOfSync，但不自动修复

**生产建议**：

- 开发/测试环境：`automated + prune + selfHeal` 全开
- 生产环境：看团队成熟度——初期可以先手动 Sync，等流程跑顺了再开 automated

## 9. 多集群管理

Argo CD 天然支持管理多个集群：

```bash
# 注册一个远端集群（需要能访问该集群的 kubeconfig）
argocd cluster add my-prod-cluster --name prod

# 然后在 Application 里指定 destination
spec:
  destination:
    server: https://prod-cluster-api.example.com   # 远端集群的 API 地址
    namespace: production
```

```text
┌──────────────────┐
│  管理集群        │
│  (Argo CD 装这里)│
│                  │
│  Application A ──┼──→ 生产集群 A
│  Application B ──┼──→ 生产集群 B
│  Application C ──┼──→ 测试集群
└──────────────────┘
```

## 10. 你需要记住的核心心智模型

| 模型 | 说明 |
|------|------|
| **Git = 真相** | 集群里的状态应该等于 Git 里定义的状态 |
| **Application = 部署单元** | 每个 App 描述了"从哪拿 + 部署到哪" |
| **Sync = 对比 + 应用** | 拉 Git → 渲染 → Diff → Apply |
| **不要直接 kubectl** | 所有变更走 Git PR，Argo CD 负责 apply |
| **Controller 是核心** | application-controller 才是真正干活的组件 |

## 下一步

概念理解后，最有效的方式是立刻动手装一个：

→ [Argo CD 安装与快速上手](./03-argocd-installation-and-quickstart.md)
