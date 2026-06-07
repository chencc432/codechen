# 🏗️ Argo CD 进阶：多集群、App of Apps、ApplicationSet、Secrets

## 1. 管理大量应用的挑战

当你只有 1-2 个应用时，手动创建 Application 没问题。但真实场景是：

- 20 个微服务，每个都要一个 Application
- 3 个环境（dev / staging / prod），每个微服务在每个环境都有一份
- 5 个集群，有些应用要部署到多个集群

这就是 20 × 3 × N = 几十上百个 Application。手动管理？不可能。

Argo CD 提供了三种模式来解决这个问题：

| 模式 | 适合场景 | 复杂度 |
|------|---------|--------|
| **App of Apps** | 用一个 Application 管理其他 Application 的 YAML | 中等 |
| **ApplicationSet** | 用模板 + 生成器批量生成 Application | 最灵活 |
| **Project 隔离** | 多团队权限划分 | 权限层面 |

## 2. App of Apps 模式

### 核心思想

"Application 本身也是 YAML，那为什么不把它们也存到 Git 里，用另一个 Application 来管理？"

```text
Git 仓库结构：
├── apps/                         ← "根 Application" 指向这个目录
│   ├── service-a.yaml           ← Application CR for service-a
│   ├── service-b.yaml           ← Application CR for service-b
│   ├── service-c.yaml           ← Application CR for service-c
│   └── redis.yaml               ← Application CR for redis
├── manifests/
│   ├── service-a/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── service-b/
│   │   └── ...
│   └── redis/
│       └── ...
```

### 实现

先定义各个子 Application（存在 Git 的 `apps/` 目录里）：

```yaml
# apps/service-a.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: service-a
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/manifests.git
    path: manifests/service-a
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: service-a
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

再定义一个"根 Application"来管理它们：

```yaml
# root-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: root-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/manifests.git
    path: apps                    # 指向存放 Application YAML 的目录
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd             # Application CR 本身在 argocd 命名空间
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**效果**：

1. Sync root-app → 创建 `apps/` 下定义的所有 Application CR
2. 每个 Application CR 再各自 sync 自己的应用
3. 新增微服务？在 `apps/` 目录加一个 YAML 文件，push 即可

## 3. ApplicationSet —— 模板化批量生成

### 核心思想

"如果 20 个 Application 长得差不多，只是名字、路径、命名空间不同，能不能用模板批量生成？"

ApplicationSet 就是干这个的——用 **Generator（生成器）** 产生参数列表，用 **Template（模板）** 渲染出多个 Application。

### 示例：一套代码部署到多个环境

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: my-app-set
  namespace: argocd
spec:
  generators:
    - list:
        elements:
          - env: dev
            cluster: https://kubernetes.default.svc
            namespace: my-app-dev
          - env: staging
            cluster: https://kubernetes.default.svc
            namespace: my-app-staging
          - env: prod
            cluster: https://prod-cluster-api:6443
            namespace: my-app-prod
  template:
    metadata:
      name: 'my-app-{{env}}'          # 生成 my-app-dev, my-app-staging, my-app-prod
    spec:
      project: default
      source:
        repoURL: https://github.com/your-org/manifests.git
        path: 'overlays/{{env}}'       # 每个环境用不同的 Kustomize overlay
        targetRevision: main
      destination:
        server: '{{cluster}}'
        namespace: '{{namespace}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
```

这一个 ApplicationSet 会自动生成 3 个 Application。

### 常用 Generator 类型

| Generator | 作用 | 典型场景 |
|-----------|------|---------|
| **list** | 手动列举参数 | 环境列表（dev/staging/prod） |
| **git** (directory) | 扫描 Git 目录结构自动生成 | monorepo 里每个子目录对应一个应用 |
| **git** (file) | 读 Git 里的 JSON/YAML 文件作为参数 | 配置文件驱动 |
| **cluster** | 已注册的集群作为参数 | 同一个应用部署到所有集群 |
| **merge** | 多个 generator 结果合并 | 复杂矩阵组合 |
| **matrix** | 两个 generator 做笛卡尔积 | 环境 × 应用的全排列 |

### Git Directory Generator 示例

```text
Git 仓库结构：
├── apps/
│   ├── service-a/
│   │   └── kustomization.yaml
│   ├── service-b/
│   │   └── kustomization.yaml
│   └── service-c/
│       └── kustomization.yaml
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: all-apps
  namespace: argocd
spec:
  generators:
    - git:
        repoURL: https://github.com/your-org/manifests.git
        revision: main
        directories:
          - path: 'apps/*'            # 自动扫描 apps/ 下的每个目录
  template:
    metadata:
      name: '{{path.basename}}'        # 目录名作为 Application 名
    spec:
      project: default
      source:
        repoURL: https://github.com/your-org/manifests.git
        path: '{{path}}'
        targetRevision: main
      destination:
        server: https://kubernetes.default.svc
        namespace: '{{path.basename}}'
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

**效果**：新增微服务时，只需在 `apps/` 下新建一个目录，push 后 ApplicationSet 自动发现并创建 Application。

### Matrix Generator 示例（应用 × 环境）

```yaml
spec:
  generators:
    - matrix:
        generators:
          # 第一维：应用列表
          - git:
              repoURL: https://github.com/your-org/manifests.git
              revision: main
              directories:
                - path: 'apps/*'
          # 第二维：环境列表
          - list:
              elements:
                - env: dev
                  cluster: https://kubernetes.default.svc
                - env: prod
                  cluster: https://prod-api:6443
  template:
    metadata:
      name: '{{path.basename}}-{{env}}'    # service-a-dev, service-a-prod, ...
    spec:
      source:
        path: 'apps/{{path.basename}}/overlays/{{env}}'
      destination:
        server: '{{cluster}}'
        namespace: '{{path.basename}}-{{env}}'
```

3 个应用 × 2 个环境 = 自动生成 6 个 Application。

## 4. 多集群管理实战

### 注册远端集群

```bash
# 前提：kubectl config 里有目标集群的 context
kubectl config get-contexts

# 注册到 Argo CD
argocd cluster add my-prod-context --name prod-cluster
```

Argo CD 会在目标集群创建一个 ServiceAccount + ClusterRole，用于后续 sync 操作。

### 查看已注册集群

```bash
argocd cluster list
# SERVER                           NAME           VERSION  STATUS
# https://kubernetes.default.svc   in-cluster     1.28     Successful
# https://prod-api.example.com     prod-cluster   1.28     Successful
```

### 典型多集群架构

```text
┌──────────────────────────────────────────────┐
│          管理集群（装 Argo CD）                │
│                                              │
│  ApplicationSet:                             │
│    apps × [dev-cluster, staging, prod]       │
│                                              │
│  ┌───────┐  ┌──────────┐  ┌────────────┐    │
│  │App-dev│  │App-staging│  │App-prod    │    │
│  └───┬───┘  └─────┬────┘  └─────┬──────┘    │
└──────┼─────────────┼─────────────┼───────────┘
       │             │             │
       ▼             ▼             ▼
  ┌─────────┐  ┌──────────┐  ┌──────────┐
  │Dev 集群 │  │Staging   │  │Prod 集群 │
  │         │  │集群      │  │          │
  └─────────┘  └──────────┘  └──────────┘
```

## 5. Secrets 管理

**最大痛点**：GitOps 要求所有配置存 Git，但密钥不能明文存 Git！

几种主流方案：

| 方案 | 原理 | 适用场景 |
|------|------|---------|
| **Sealed Secrets** | 用公钥加密后存 Git，集群内控制器用私钥解密 | 简单、不依赖外部系统 |
| **External Secrets Operator** | 从 Vault/AWS SM/GCP SM 拉取 Secret | 已有外部 Secret 管理 |
| **SOPS + Kustomize** | 用 SOPS 加密 YAML 字段，解密由 Argo CD 插件完成 | 中等复杂度 |
| **Vault + argocd-vault-plugin** | Argo CD 渲染时从 Vault 注入 | 企业级 Vault 用户 |

### 方案一：Sealed Secrets（最简单）

```bash
# 安装 Sealed Secrets Controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.0/controller.yaml

# 安装 kubeseal CLI
brew install kubeseal   # 或下载二进制
```

使用流程：

```bash
# 1. 先写一个普通 Secret（不提交到 Git！）
kubectl create secret generic db-creds \
  --from-literal=username=admin \
  --from-literal=password=SuperSecret123 \
  --dry-run=client -o yaml > secret.yaml

# 2. 用 kubeseal 加密
kubeseal --format yaml < secret.yaml > sealed-secret.yaml

# 3. sealed-secret.yaml 可以安全地存入 Git
cat sealed-secret.yaml
```

加密后的内容长这样：

```yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: db-creds
spec:
  encryptedData:
    username: AgBy3i4OJ... (加密的)
    password: AgCtr8hkD... (加密的)
```

把 `sealed-secret.yaml` 存到 Git，Argo CD sync 时 Sealed Secrets Controller 会自动解密成真正的 Secret。

### 方案二：External Secrets Operator

```yaml
# 安装后，定义 ExternalSecret 拉取 AWS Secrets Manager 里的密钥
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: db-creds
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: db-creds                     # 最终创建的 K8s Secret 名
  data:
    - secretKey: username
      remoteRef:
        key: prod/db-credentials       # AWS SM 里的 key
        property: username
    - secretKey: password
      remoteRef:
        key: prod/db-credentials
        property: password
```

## 6. SSO 配置（企业必备）

生产环境不能用 admin 密码，应该接 SSO：

```yaml
# argocd-cm ConfigMap 里配置 Dex
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
data:
  url: https://argocd.your-domain.com
  dex.config: |
    connectors:
      - type: github
        id: github
        name: GitHub
        config:
          clientID: $GITHUB_CLIENT_ID
          clientSecret: $GITHUB_CLIENT_SECRET
          orgs:
            - name: your-org
```

配合 RBAC 策略：

```yaml
# argocd-rbac-cm
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-rbac-cm
  namespace: argocd
data:
  policy.csv: |
    # 后端团队可以管理 backend-* 的应用
    p, role:backend, applications, *, backend-*/*, allow
    p, role:backend, repositories, get, *, allow

    # GitHub org 的 backend-team 组绑定到 backend role
    g, your-org:backend-team, role:backend
  policy.default: role:readonly    # 默认只读
```

## 7. Notification（通知）

Argo CD 支持 Sync 成功/失败时发通知：

```yaml
# 安装 argocd-notifications（Argo CD 2.6+ 已内置）
# 配置发送到 Slack
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  service.slack: |
    token: $slack-token
  trigger.on-sync-succeeded: |
    - send: [app-sync-succeeded]
      when: app.status.operationState.phase in ['Succeeded']
  template.app-sync-succeeded: |
    message: |
      ✅ {{.app.metadata.name}} sync 成功！
      Revision: {{.app.status.sync.revision}}
```

## 8. 生产环境推荐配置

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app-prod
  namespace: argocd
spec:
  project: production           # 用独立 Project 隔离
  source:
    repoURL: https://github.com/your-org/manifests.git
    path: overlays/prod
    targetRevision: release     # 生产只跟踪 release 分支
  destination:
    server: https://prod-cluster-api:6443
    namespace: my-app
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground   # 删除时等子资源先删完
      - ApplyOutOfSyncOnly=true             # 只 apply 有变化的资源（提升性能）
    retry:
      limit: 3                 # sync 失败重试 3 次
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 1m
  ignoreDifferences:           # 忽略某些字段的 diff（HPA 会改 replicas）
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas
```

## 9. App of Apps vs ApplicationSet 怎么选

| 维度 | App of Apps | ApplicationSet |
|------|------------|---------------|
| 灵活度 | 每个 App 可以完全不同 | 模板化，结构必须相似 |
| 自动发现 | 不支持（需要手动加文件） | 支持（Git directory generator） |
| 学习成本 | 低，就是嵌套 Application | 中等，需学 Generator 语法 |
| 适用规模 | < 30 个应用 | 30+ 个应用，或跨多集群/多环境 |
| 推荐场景 | 异构应用（有 Helm 有 Kustomize 有裸 YAML） | 同构应用批量管理 |

**实践建议**：小团队 5-10 个应用用 App of Apps 就够了；大团队 50+ 应用、多集群一定要上 ApplicationSet。

## 下一步

Argo CD 搞定后，下一个高频需求是"安全发布"：

→ [Argo Rollouts：金丝雀与蓝绿部署](./05-argo-rollouts-core.md)
