# 🎲 Argo Rollouts：金丝雀与蓝绿部署

## 1. 为什么需要 Argo Rollouts

### 原生 Deployment 滚动更新的问题

Kubernetes 原生 `Deployment` 的 `RollingUpdate` 策略确实能"滚动"更新 Pod，但它有几个致命缺陷：

```text
原生 RollingUpdate 的过程：
  旧 Pod: ●●●●●●●●●●  (10 个)
  更新中: ●●●●●●●○○○  (逐个替换)
  更新中: ●●●●○○○○○○
  完成:   ○○○○○○○○○○  (全是新 Pod)

问题：
  ❌ 没有"观察期"——新 Pod Ready 后立刻接流量
  ❌ 没有流量权重控制——不能"先给 5% 用户试试"
  ❌ 回滚靠手动——发现问题需要人去 rollback
  ❌ 没有指标验证——不会自动检查错误率、延迟
```

### Argo Rollouts 的方式

```text
金丝雀发布：
  旧 Pod: ●●●●●●●●●●  (10 个，承担 100% 流量)

  Step 1:  ●●●●●●●●●● + ○  (新 Pod 起 1 个，承担 10% 流量)
           等 5 分钟，自动检查错误率 < 1%  ✅ 通过

  Step 2:  ●●●●●●●●●● + ○○○  (新 Pod 3 个，承担 30% 流量)
           等 10 分钟，自动检查 P99 延迟 < 500ms  ✅ 通过

  Step 3:  ●●●●● + ○○○○○  (50/50)
           等 10 分钟  ✅ 通过

  Step 4:  ○○○○○○○○○○  (全量切换)
           发布完成 🎉

  任何一步失败？ → 自动回滚到旧版本，用户几乎无感
```

## 2. 一句话定义

> Argo Rollouts 是 Kubernetes `Deployment` 的增强替代品——提供金丝雀（Canary）和蓝绿（Blue-Green）两种渐进发布策略，每一步都可以用 Prometheus 等指标自动判断是继续还是回滚。

## 3. 金丝雀 vs 蓝绿 vs 滚动更新

| 策略 | 原理 | 优势 | 劣势 |
|------|------|------|------|
| **滚动更新** | 逐个替换 Pod | 简单，零额外资源 | 没有流量控制，回滚慢 |
| **金丝雀** | 小流量验证，逐步放大 | 风险最低，可细粒度控制 | 需要流量管理（Istio/Nginx） |
| **蓝绿** | 新旧两套完整环境，一键切 | 切换快，回滚快 | 双倍资源 |

```text
金丝雀（Canary）：
  ┌─────────┐  90%   ┌─────────┐
  │ Service ├───────▶│ 旧版本  │ ← stable
  │         │        └─────────┘
  │         │  10%   ┌─────────┐
  │         ├───────▶│ 新版本  │ ← canary
  └─────────┘        └─────────┘
  逐步从 10% → 30% → 50% → 100%

蓝绿（Blue-Green）：
  ┌─────────┐  100%  ┌─────────┐
  │ Active  ├───────▶│ 蓝(旧)  │ ← 当前提供服务
  │ Service │        └─────────┘
  └─────────┘        ┌─────────┐
  ┌─────────┐  100%  │ 绿(新)  │ ← 预热验证中
  │ Preview ├───────▶│         │
  │ Service │        └─────────┘
  └─────────┘
  验证通过后，Active 一键切到绿
```

## 4. 安装 Argo Rollouts

```bash
# 安装 Controller
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# 验证
kubectl get pods -n argo-rollouts
# argo-rollouts-xxx   Running

# 安装 kubectl plugin（可选，但推荐）
# Linux/macOS
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x kubectl-argo-rollouts-linux-amd64
sudo mv kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts

# 验证
kubectl argo rollouts version
```

安装 Dashboard（可视化看发布过程）：

```bash
kubectl argo rollouts dashboard
# 打开 http://localhost:3100
```

## 5. 第一个金丝雀发布

### 从 Deployment 迁移到 Rollout

Rollout 的 spec 和 Deployment **几乎一模一样**，只需要：

1. 把 `kind: Deployment` 改成 `kind: Rollout`
2. 把 `spec.strategy.rollingUpdate` 改成 `spec.strategy.canary`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout                             # ← 从 Deployment 改过来
metadata:
  name: my-app
spec:
  replicas: 10
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      app: my-app
  strategy:
    canary:                               # ← 金丝雀策略
      steps:
        - setWeight: 10                   # Step 1: 10% 流量给新版
        - pause: {duration: 5m}           # 等 5 分钟
        - setWeight: 30                   # Step 2: 30%
        - pause: {duration: 10m}          # 等 10 分钟
        - setWeight: 60                   # Step 3: 60%
        - pause: {duration: 10m}
        - setWeight: 100                  # Step 4: 全量
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: app
          image: my-app:v1.0.0
          ports:
            - containerPort: 8080
```

### 触发发布

```bash
# 方式一：直接修改镜像
kubectl argo rollouts set image my-app app=my-app:v2.0.0

# 方式二：kubectl apply 修改后的 YAML（镜像改了就会触发）
kubectl apply -f rollout.yaml

# 方式三：通过 Argo CD 同步（最推荐的 GitOps 方式）
```

### 观察发布过程

```bash
# 实时 watch 发布进度
kubectl argo rollouts get rollout my-app --watch

# 输出类似：
# Name:            my-app
# Namespace:       default
# Status:          ॥ Paused
# Strategy:        Canary
#   Step:          2/8
#   SetWeight:     10
#   ActualWeight:  10
# Images:
#   my-app:v1.0.0 (stable)
#   my-app:v2.0.0 (canary)
# Replicas:
#   Desired:       10
#   Current:       11
#   Updated:       1
#   Ready:         11
#   Available:     11
```

### 手动操作

```bash
# 跳过当前的 pause，继续下一步
kubectl argo rollouts promote my-app

# 一键全量（跳过所有剩余步骤）
kubectl argo rollouts promote my-app --full

# 手动回滚（中止当前发布，回到旧版本）
kubectl argo rollouts abort my-app

# 重试（abort 后重新开始）
kubectl argo rollouts retry rollout my-app
```

## 6. 流量管理（配合 Ingress/Service Mesh）

默认情况下 Argo Rollouts 用 **副本数比例** 来近似流量比例（10 个 Pod 里 1 个新版 ≈ 10% 流量）。

如果需要**精确的流量比例控制**，需要配合流量管理组件：

| 方案 | 流量精度 | 复杂度 |
|------|---------|--------|
| 副本比例（默认） | 粗略 | 无需额外组件 |
| **Nginx Ingress** | 精确到百分比 | 需要 Nginx Ingress Controller |
| **Istio** | 精确 + 高级路由 | 需要 Istio Service Mesh |
| **AWS ALB** | 精确 | AWS 环境 |
| **Traefik** | 精确 | 需要 Traefik |

### 配合 Nginx Ingress 的精确金丝雀

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  strategy:
    canary:
      canaryService: my-app-canary        # 金丝雀流量走这个 Service
      stableService: my-app-stable        # 稳定流量走这个 Service
      trafficRouting:
        nginx:
          stableIngress: my-app-ingress   # 已有的 Ingress 名称
          additionalIngressAnnotations:
            canary-by-header: X-Canary    # 也可以按 header 路由
      steps:
        - setWeight: 10
        - pause: {duration: 5m}
        - setWeight: 50
        - pause: {duration: 10m}
        - setWeight: 100
```

需要的额外资源：

```yaml
# 稳定版 Service
apiVersion: v1
kind: Service
metadata:
  name: my-app-stable
spec:
  selector:
    app: my-app
  ports:
    - port: 80
      targetPort: 8080
---
# 金丝雀 Service
apiVersion: v1
kind: Service
metadata:
  name: my-app-canary
spec:
  selector:
    app: my-app
  ports:
    - port: 80
      targetPort: 8080
---
# Ingress
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: my-app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-app-stable
                port:
                  number: 80
```

Argo Rollouts 会自动创建一个额外的 Canary Ingress，带上 `nginx.ingress.kubernetes.io/canary-weight: "10"` 等注解来控制流量比例。

## 7. AnalysisTemplate —— 自动化指标验证

金丝雀发布最强大的能力是**自动验证**——发布过程中自动查 Prometheus 等指标，不达标就自动回滚。

### 定义 AnalysisTemplate

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  args:
    - name: service-name          # 接收参数
  metrics:
    - name: success-rate
      interval: 1m                # 每分钟查一次
      count: 5                    # 总共查 5 次
      successCondition: result[0] >= 0.99   # 成功率 >= 99% 算通过
      failureLimit: 2             # 允许失败 2 次，第 3 次失败即回滚
      provider:
        prometheus:
          address: http://prometheus.monitoring:9090
          query: |
            sum(rate(http_requests_total{service="{{args.service-name}}",status=~"2.."}[5m]))
            /
            sum(rate(http_requests_total{service="{{args.service-name}}"}[5m]))
```

### 在 Rollout 中引用

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  strategy:
    canary:
      steps:
        - setWeight: 10
        - pause: {duration: 2m}
        - analysis:                        # ← 在这一步执行分析
            templates:
              - templateName: success-rate
            args:
              - name: service-name
                value: my-app
        - setWeight: 50
        - pause: {duration: 5m}
        - analysis:
            templates:
              - templateName: success-rate
            args:
              - name: service-name
                value: my-app
        - setWeight: 100
```

**效果**：

1. 先放 10% 流量
2. 等 2 分钟
3. 开始查 Prometheus 指标：成功率是否 >= 99%？
4. 如果通过 → 继续放到 50%
5. 如果失败 → 自动 abort + 回滚到旧版本

### 支持的 Metrics Provider

| Provider | 查什么 |
|----------|--------|
| **Prometheus** | PromQL 查询 |
| **Datadog** | Datadog 指标 |
| **New Relic** | NRQL 查询 |
| **CloudWatch** | AWS 指标 |
| **Web** | 调用 HTTP API，检查返回值 |
| **Job** | 跑一个 K8s Job，检查 exit code |
| **Kayenta** | Netflix Kayenta ABA 测试 |

## 8. 蓝绿部署

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  replicas: 5
  strategy:
    blueGreen:
      activeService: my-app-active        # 当前对外服务的 Service
      previewService: my-app-preview      # 预览/验证用的 Service
      autoPromotionEnabled: false         # 不自动切换，等手动确认
      previewReplicaCount: 2              # 预览环境只起 2 个 Pod（省资源）
      scaleDownDelaySeconds: 30           # 切换后等 30s 再缩旧版本
      prePromotionAnalysis:               # 切换前自动验证
        templates:
          - templateName: success-rate
        args:
          - name: service-name
            value: my-app
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: app
          image: my-app:v2.0.0
          ports:
            - containerPort: 8080
```

蓝绿部署流程：

```text
① 当前状态：Active Service → 旧版本 (v1)

② 触发更新（改了镜像）：
   Active Service  → 旧版本 (v1)   ← 用户访问这里
   Preview Service → 新版本 (v2)   ← QA 在这里验证

③ prePromotionAnalysis 通过（或手动 promote）：
   Active Service  → 新版本 (v2)   ← 一键切换！
   旧版本 (v1)    → 30 秒后缩容删除
```

```bash
# 手动将蓝绿切换（promote）
kubectl argo rollouts promote my-app
```

## 9. 与 Argo CD 配合使用

Argo Rollouts 和 Argo CD 天然配合：

1. Argo CD 管理 Rollout 的 YAML（就像管理 Deployment 一样）
2. Git 里修改镜像 tag → Argo CD sync → Rollout 开始渐进发布
3. Argo CD 的 Health Check 内置了 Rollout 的支持，能显示发布进度

```text
Git 仓库修改镜像: v1 → v2
        │
        ▼
Argo CD 检测到变化 → sync → apply Rollout 新 spec
        │
        ▼
Argo Rollouts controller 接管：
  10% → 分析 → 30% → 分析 → 100%
        │
        ▼
Argo CD 显示 Health: Progressing → Healthy
```

在 Argo CD UI 中，你能看到 Rollout 资源的发布状态（替代 Deployment 的 Progressing）。

## 10. 常用命令速查

```bash
# 查看 Rollout 状态
kubectl argo rollouts get rollout <name> --watch

# 触发发布（改镜像）
kubectl argo rollouts set image <name> <container>=<image>:<tag>

# 继续发布（跳过 pause）
kubectl argo rollouts promote <name>

# 全量发布（跳过所有剩余步骤）
kubectl argo rollouts promote <name> --full

# 中止发布（回滚到 stable）
kubectl argo rollouts abort <name>

# 重试
kubectl argo rollouts retry rollout <name>

# 查看 AnalysisRun（指标验证结果）
kubectl get analysisrun

# 查看发布历史
kubectl argo rollouts history <name>

# 回退到某个版本
kubectl argo rollouts undo <name> --to-revision=2
```

## 11. 从 Deployment 迁移到 Rollout 的检查清单

| 步骤 | 动作 | 注意事项 |
|------|------|---------|
| 1 | 安装 Argo Rollouts Controller | 确保 CRD 注册成功 |
| 2 | 把 `kind: Deployment` 改为 `kind: Rollout` | apiVersion 也要改 |
| 3 | 把 `strategy.rollingUpdate` 改为 `strategy.canary` 或 `blueGreen` | |
| 4 | 添加 `steps`（金丝雀）或 `activeService`（蓝绿） | |
| 5 | 删除旧 Deployment，创建新 Rollout | 注意 selector 不变 |
| 6 | （可选）配置流量管理（Nginx/Istio） | 精确流量控制才需要 |
| 7 | （可选）配置 AnalysisTemplate | 自动验证才需要 |

> **注意**：Rollout 和 Deployment 不能同时管理相同 selector 的 Pod！迁移时需要删旧建新。参考[官方迁移指南](https://argoproj.github.io/argo-rollouts/migrating/)。

## 12. 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| Rollout 一直 Paused 不动 | 到了 pause 步骤，等你 promote | `kubectl argo rollouts promote` |
| AnalysisRun Failed 自动回滚了 | 指标不达标 | 查 AnalysisRun 的 measurements 看具体值 |
| Canary Pod 起不来 | 镜像/资源/配置问题 | 跟排查 Pod 一样：describe + logs |
| 流量没有按比例分配 | 没配 trafficRouting | 默认只是副本数近似，需要 Nginx/Istio |
| 和 HPA 冲突 | HPA 改了 replicas，Rollout 又改回来 | 使用 Rollout 内置的 HPA 支持 |

## 下一步

发布搞定后，再看事件驱动自动化：

→ [Argo Events：事件监听与自动触发](./06-argo-events-core.md)
