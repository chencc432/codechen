# 🔗 全家桶联动实战：完整 GitOps 流水线

## 1. 目标：从 git push 到安全上线全自动

这一篇把前面学的四个工具串起来，实现一条**完整的 GitOps CI/CD Pipeline**：

```text
开发者 git push
    │
    ▼
Argo Events 监听 GitHub Webhook
    │
    ▼
Argo Workflows 跑 CI Pipeline（构建 + 测试 + 推镜像）
    │
    ▼
Workflow 最后一步：更新 Git 仓库中的镜像 tag
    │
    ▼
Argo CD 检测到 Git 变化，自动 Sync
    │
    ▼
Argo Rollouts 渐进发布（金丝雀 10% → 50% → 100%）
    │
    ▼
AnalysisTemplate 自动验证指标
    │
    ├── 通过 → 全量发布完成 ✅
    └── 失败 → 自动回滚 ❌
```

**整条链路，人只做了一件事：`git push`。**

## 2. 架构全景图

```text
┌─────────────────────────────────────────────────────────────────┐
│                        Git 仓库（双仓库模式）                     │
│                                                                 │
│  ┌──────────────────┐         ┌─────────────────────────────┐   │
│  │ app-source-repo  │         │ app-manifests-repo           │   │
│  │ (业务代码)        │         │ (K8s YAML / Helm Chart)     │   │
│  │ • src/           │         │ • overlays/dev/              │   │
│  │ • Dockerfile     │         │ • overlays/prod/             │   │
│  │ • tests/         │         │   └── kustomization.yaml     │   │
│  └────────┬─────────┘         │       (images.newTag: xxx)   │   │
│           │                   └──────────────┬──────────────┘   │
│           │ push                             │ Argo CD 监控     │
└───────────┼──────────────────────────────────┼──────────────────┘
            │                                  │
            ▼                                  ▼
┌───────────────────┐              ┌────────────────────┐
│   Argo Events     │              │     Argo CD        │
│   (GitHub Webhook)│              │  (GitOps Sync)     │
└─────────┬─────────┘              └──────────┬─────────┘
          │ trigger                            │ sync
          ▼                                    ▼
┌───────────────────┐              ┌────────────────────┐
│  Argo Workflows   │──update──▶   │   Argo Rollouts    │
│  (Build + Test    │  image tag   │   (Canary Deploy)  │
│   + Push Image)   │  in Git      │                    │
└───────────────────┘              └────────────────────┘
```

### 为什么要双仓库

| 模式 | 说明 | 适用 |
|------|------|------|
| **单仓库** | 代码和 K8s YAML 放一起 | 小项目、简单场景 |
| **双仓库** | 代码一个仓库，部署配置一个仓库 | 生产推荐 |

双仓库的好处：
- 代码变更和部署配置变更的权限分离
- CI（构建）和 CD（部署）解耦
- 避免 CI 的 commit 触发 CD 的循环

## 3. 完整实战：一步步搭建

### Step 1：准备 Git 仓库

**业务代码仓库** (`app-source-repo`)：

```text
my-app/
├── src/
│   └── main.go
├── Dockerfile
├── go.mod
└── tests/
    └── main_test.go
```

**部署配置仓库** (`app-manifests-repo`)：

```text
manifests/
├── base/
│   ├── rollout.yaml         # Argo Rollout（替代 Deployment）
│   ├── service.yaml
│   ├── service-canary.yaml
│   ├── ingress.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

### Step 2：部署配置示例

**base/rollout.yaml**：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  replicas: 5
  selector:
    matchLabels:
      app: my-app
  strategy:
    canary:
      stableService: my-app-stable
      canaryService: my-app-canary
      trafficRouting:
        nginx:
          stableIngress: my-app-ingress
      steps:
        - setWeight: 10
        - pause: {duration: 3m}
        - analysis:
            templates:
              - templateName: success-rate
            args:
              - name: service-name
                value: my-app-canary
        - setWeight: 50
        - pause: {duration: 5m}
        - analysis:
            templates:
              - templateName: success-rate
            args:
              - name: service-name
                value: my-app-canary
        - setWeight: 100
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: app
          image: your-registry/my-app:placeholder   # 会被 Kustomize 覆盖
          ports:
            - containerPort: 8080
```

**overlays/prod/kustomization.yaml**：

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: production
resources:
  - ../../base
images:
  - name: your-registry/my-app
    newTag: "abc123def"              # ← CI 完成后会自动更新这里
```

### Step 3：Argo Events —— 监听 Git Push

```yaml
# EventSource：监听业务代码仓库的 Push
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: app-source-push
  namespace: argo-events
spec:
  service:
    ports:
      - port: 12000
        targetPort: 12000
  github:
    app-push:
      repositories:
        - owner: your-org
          names: [my-app]
      webhook:
        endpoint: /push
        port: "12000"
      events: [push]
      apiToken:
        name: github-token
        key: token
      webhookSecret:
        name: github-webhook-secret
        key: secret
---
# Sensor：Push 到 main 分支 → 触发 CI Workflow
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: ci-sensor
  namespace: argo-events
spec:
  template:
    serviceAccountName: argo-events-sa
  dependencies:
    - name: app-push
      eventSourceName: app-source-push
      eventName: app-push
      filters:
        data:
          - path: body.ref
            type: string
            value: ["refs/heads/main"]
  triggers:
    - template:
        name: ci-pipeline
        argoWorkflow:
          operation: submit
          source:
            resource:
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: ci-pipeline-
                namespace: argo
              spec:
                serviceAccountName: ci-workflow-sa
                entrypoint: ci
                arguments:
                  parameters:
                    - name: repo-url
                    - name: commit-sha
                    - name: branch
                templates:
                  - name: ci
                    # 详见 Step 4
          parameters:
            - src:
                dependencyName: app-push
                dataKey: body.repository.clone_url
              dest: spec.arguments.parameters.0.value
            - src:
                dependencyName: app-push
                dataKey: body.after
              dest: spec.arguments.parameters.1.value
            - src:
                dependencyName: app-push
                dataKey: body.ref
              dest: spec.arguments.parameters.2.value
```

### Step 4：Argo Workflows —— CI Pipeline

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: ci-pipeline
  namespace: argo
spec:
  entrypoint: ci
  arguments:
    parameters:
      - name: repo-url
      - name: commit-sha
      - name: branch
  templates:
    - name: ci
      dag:
        tasks:
          - name: clone
            template: git-clone
            arguments:
              parameters:
                - name: repo-url
                  value: "{{workflow.parameters.repo-url}}"
                - name: commit-sha
                  value: "{{workflow.parameters.commit-sha}}"

          - name: test
            dependencies: [clone]
            template: run-tests

          - name: build-push
            dependencies: [test]
            template: build-and-push-image
            arguments:
              parameters:
                - name: commit-sha
                  value: "{{workflow.parameters.commit-sha}}"

          - name: update-manifests
            dependencies: [build-push]
            template: update-git-manifests
            arguments:
              parameters:
                - name: commit-sha
                  value: "{{workflow.parameters.commit-sha}}"

    # 模板：克隆代码
    - name: git-clone
      inputs:
        parameters:
          - name: repo-url
          - name: commit-sha
      container:
        image: alpine/git:2.40.1
        command: [sh, -c]
        args:
          - |
            git clone {{inputs.parameters.repo-url}} /workspace
            cd /workspace
            git checkout {{inputs.parameters.commit-sha}}
        volumeMounts:
          - name: workspace
            mountPath: /workspace

    # 模板：跑测试
    - name: run-tests
      container:
        image: golang:1.21
        command: [sh, -c]
        args:
          - |
            cd /workspace
            go test ./...
        volumeMounts:
          - name: workspace
            mountPath: /workspace

    # 模板：构建并推送镜像
    - name: build-and-push-image
      inputs:
        parameters:
          - name: commit-sha
      container:
        image: gcr.io/kaniko-project/executor:latest
        args:
          - --dockerfile=/workspace/Dockerfile
          - --context=/workspace
          - --destination=your-registry/my-app:{{inputs.parameters.commit-sha}}
        volumeMounts:
          - name: workspace
            mountPath: /workspace
          - name: docker-config
            mountPath: /kaniko/.docker

    # 模板：更新部署仓库的镜像 tag（核心！连接 CI 和 CD 的桥梁）
    - name: update-git-manifests
      inputs:
        parameters:
          - name: commit-sha
      container:
        image: alpine/git:2.40.1
        command: [sh, -c]
        args:
          - |
            # 克隆部署配置仓库
            git clone https://${GITHUB_TOKEN}@github.com/your-org/app-manifests.git /manifests
            cd /manifests

            # 更新 kustomization.yaml 中的镜像 tag
            cd overlays/prod
            kustomize edit set image your-registry/my-app:{{inputs.parameters.commit-sha}}

            # 提交并推送
            git config user.email "ci-bot@your-org.com"
            git config user.name "CI Bot"
            git add .
            git commit -m "chore: update my-app image to {{inputs.parameters.commit-sha}}"
            git push origin main
        env:
          - name: GITHUB_TOKEN
            valueFrom:
              secretKeyRef:
                name: github-token
                key: token

  volumes:
    - name: workspace
      emptyDir: {}
    - name: docker-config
      secret:
        secretName: docker-registry-creds
```

### Step 5：Argo CD —— 自动同步部署

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app-prod
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/app-manifests.git
    path: overlays/prod
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

当 CI Workflow 的最后一步 push 了新的 commit 到 manifests 仓库，Argo CD 3 分钟内（或 Webhook 即时）检测到变化并 sync。

### Step 6：Argo Rollouts —— 渐进发布 + 自动验证

AnalysisTemplate（Step 2 中 Rollout 引用的）：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
  namespace: production
spec:
  args:
    - name: service-name
  metrics:
    - name: success-rate
      interval: 60s
      count: 5
      successCondition: result[0] >= 0.99
      failureLimit: 2
      provider:
        prometheus:
          address: http://prometheus.monitoring:9090
          query: |
            sum(rate(
              http_requests_total{service="{{args.service-name}}", status=~"2.."}[2m]
            )) /
            sum(rate(
              http_requests_total{service="{{args.service-name}}"}[2m]
            ))
```

## 4. 完整时间线

从开发者 push 代码到线上全量发布，整个过程：

```text
T+0s     开发者 git push 到 main 分支
T+1s     GitHub 发送 Webhook 到 Argo Events
T+2s     Argo Events Sensor 触发 CI Workflow
T+3s     Workflow 开始：克隆代码
T+30s    跑测试（假设 30 秒）
T+2min   构建 Docker 镜像 + 推送到仓库
T+3min   更新 manifests 仓库的镜像 tag
T+3min   Argo CD 收到 Webhook / 轮询检测到变化
T+3.5min Argo CD sync → 更新 Rollout 的 image
T+4min   Argo Rollouts 开始金丝雀发布：10% 流量
T+7min   自动分析（AnalysisRun）：检查成功率
T+8min   分析通过 → 50% 流量
T+13min  再次分析 → 通过
T+14min  100% 流量 → 发布完成 🎉

总耗时：约 14 分钟，全程无人干预
```

## 5. 关键设计决策

### 为什么 CI（Workflow）和 CD（Argo CD）之间用"更新 Git"而不是直接调 K8s API？

```text
方案 A（不推荐）：
  Workflow → 直接 kubectl apply 新镜像到集群
  问题：
    - 绕过了 GitOps 原则
    - Git 里的配置和集群不同步
    - 失去了审计和回滚能力

方案 B（推荐）：
  Workflow → 更新 Git 仓库中的镜像 tag → Argo CD 检测并 sync
  好处：
    - Git 永远是唯一真相源
    - 每次部署都有 commit 记录
    - 回滚只需 git revert
```

### Webhook vs 轮询

| 方式 | 延迟 | 可靠性 | 配置复杂度 |
|------|------|--------|-----------|
| Webhook | 1-2 秒 | 可能丢（网络问题） | 需要公网可达 |
| 轮询（默认） | 最多 3 分钟 | 高 | 零配置 |
| Webhook + 轮询兜底 | 1-2 秒 | 最高 | 中等 |

生产建议：Webhook 保速度，轮询做兜底。

### Image Updater vs CI 更新 Git

Argo CD 还有一个 [Image Updater](https://argocd-image-updater.readthedocs.io/) 组件，可以自动监控镜像仓库中新的 tag 并更新 Git：

```text
方案 1：CI Workflow 自己更新 Git（本文方式）
  优点：完全可控，CI 里能做任何逻辑
  缺点：需要 CI 有 Git 写权限

方案 2：Argo CD Image Updater
  优点：CI 不需要知道 manifests 仓库
  缺点：tag 命名需要规律，不够灵活
```

## 6. 通知集成

在关键节点发通知：

```yaml
# Workflow 完成后发 Slack 通知（用 Exit Handler）
spec:
  onExit: notify
  templates:
    - name: notify
      container:
        image: curlimages/curl:8.4.0
        command: [sh, -c]
        args:
          - |
            curl -X POST https://hooks.slack.com/services/xxx \
              -H 'Content-type: application/json' \
              -d '{"text": "CI Pipeline 完成 ✅\nCommit: {{workflow.parameters.commit-sha}}"}'
```

Argo Rollouts 的通知（用 Argo CD Notifications 或 Rollouts Notifications）：

```yaml
# 在 Rollout 上加注解
metadata:
  annotations:
    notifications.argoproj.io/subscribe.on-rollout-completed.slack: my-channel
    notifications.argoproj.io/subscribe.on-analysis-run-failed.slack: my-channel
```

## 7. 回滚场景

### 场景 1：金丝雀分析失败，自动回滚

```text
Argo Rollouts 自动处理：
  AnalysisRun Failed → Rollout 自动 abort → 流量切回 stable 版本
  无需人工干预
```

### 场景 2：发现线上 bug，需要手动回滚

```bash
# 方式 1（推荐）：Git revert
cd app-manifests
git revert HEAD
git push origin main
# Argo CD 自动 sync → Rollout 回到旧镜像

# 方式 2：Argo CD 回滚
argocd app history my-app-prod
argocd app rollback my-app-prod <revision-id>

# 方式 3：手动 abort 当前发布
kubectl argo rollouts abort my-app -n production
```

## 8. 完整依赖组件清单

| 组件 | 命名空间 | 作用 |
|------|---------|------|
| Argo Events (controller + EventBus) | argo-events | 事件监听和触发 |
| Argo Workflows (controller + server) | argo | CI Pipeline 执行 |
| Argo CD (server + controller + repo-server) | argocd | GitOps 同步 |
| Argo Rollouts (controller) | argo-rollouts | 渐进发布 |
| Nginx Ingress Controller | ingress-nginx | 流量管理 |
| Prometheus | monitoring | 指标采集（给 AnalysisTemplate 用） |
| 镜像仓库 | 外部 | 存储构建好的镜像 |
| GitHub / GitLab | 外部 | 代码和配置仓库 |

## 9. 最小化启动建议

不需要一步到位全上，建议分阶段：

```text
阶段 1（1 周）：只上 Argo CD
  - 把现有的 kubectl apply 替换成 GitOps
  - 先管 dev 环境，跑稳了再推 prod

阶段 2（2 周）：加上 Argo Rollouts
  - 把 Deployment 替换成 Rollout
  - 先用简单的 pause 手动 promote
  - 稳了再加 AnalysisTemplate

阶段 3（1 周）：加上 Argo Workflows
  - 替换现有 CI 系统（Jenkins/GitLab CI）
  - 或者跟现有 CI 共存（现有 CI 构建，Workflow 做后半段）

阶段 4（可选）：加上 Argo Events
  - 实现完全自动化触发
  - 替代手动点 CI 按钮
```

## 下一步

全链路搞定后，看生产环境的最佳实践和避坑指南：

→ [生产最佳实践与避坑](./08-production-best-practices.md)
