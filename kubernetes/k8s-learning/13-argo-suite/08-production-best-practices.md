# 🛡️ 生产最佳实践与避坑指南

## 1. Argo CD 生产最佳实践

### 1.1 高可用部署

```yaml
# 使用 HA 安装清单
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/ha/install.yaml
```

HA 模式下：
- `argocd-application-controller`：StatefulSet，多副本 + Leader Election
- `argocd-server`：多副本 + HPA
- `argocd-repo-server`：多副本 + HPA
- Redis：Sentinel 或 Redis Cluster

### 1.2 性能调优

当管理的 Application 数量超过 50 个时，需要调参：

```yaml
# argocd-cmd-params-cm ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cmd-params-cm
  namespace: argocd
data:
  # 控制器并发 reconcile 数
  controller.status.processors: "50"          # 默认 20
  controller.operation.processors: "25"       # 默认 10

  # repo-server 并发数
  reposerver.parallelism.limit: "20"          # 默认 0（无限制）

  # Git 轮询间隔（减少 API 压力）
  timeout.reconciliation: "180s"              # 默认 180s，大仓库可以调大
```

### 1.3 仓库管理最佳实践

| 实践 | 说明 |
|------|------|
| 单一仓库放所有 manifests | 便于管理，但大团队会有合并冲突 |
| 每个团队一个 manifests 仓库 | 权限隔离好，但仓库多 |
| 使用 Kustomize overlays | 一套 base 多环境复用 |
| 用 targetRevision 区分环境 | dev 跟 main，prod 跟 release 分支 |
| Application YAML 也存 Git | 自己管自己（App of Apps） |

### 1.4 安全配置

```yaml
# 1. 禁用 admin 账号（配置 SSO 后）
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
data:
  admin.enabled: "false"

# 2. 使用 AppProject 限制权限
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: production
spec:
  sourceRepos:
    - 'https://github.com/your-org/manifests.git'
  destinations:
    - namespace: 'prod-*'
      server: 'https://prod-cluster-api:6443'
  # 禁止删除命名空间
  namespaceResourceBlacklist:
    - group: ''
      kind: Namespace
  # 禁止修改 ClusterRole
  clusterResourceBlacklist:
    - group: 'rbac.authorization.k8s.io'
      kind: ClusterRole

# 3. 资源级别的 Sync Window（维护窗口）
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: production
spec:
  syncWindows:
    - kind: allow
      schedule: '0 9-18 * * 1-5'       # 只允许工作日白天同步
      duration: 9h
      applications: ['*']
    - kind: deny
      schedule: '0 0 * * 0'            # 周日完全禁止
      duration: 24h
      applications: ['*']
```

### 1.5 监控 Argo CD

关键指标（暴露给 Prometheus）：

```yaml
# argocd-metrics Service 默认暴露 /metrics
# 关键 Metrics：
# argocd_app_info                    → 应用状态概览
# argocd_app_sync_total              → 同步次数
# argocd_app_reconcile_count         → reconcile 计数
# argocd_app_health_status           → 健康状态分布
# argocd_git_request_total           → Git 请求统计
# argocd_git_request_duration_seconds → Git 拉取耗时
```

告警规则示例：

```yaml
groups:
  - name: argocd
    rules:
      - alert: ArgoCDAppOutOfSync
        expr: argocd_app_info{sync_status="OutOfSync"} == 1
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "Application {{ $labels.name }} OutOfSync 超过 30 分钟"

      - alert: ArgoCDAppDegraded
        expr: argocd_app_info{health_status="Degraded"} == 1
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Application {{ $labels.name }} 健康状态 Degraded"
```

## 2. Argo Workflows 生产最佳实践

### 2.1 资源限制

```yaml
# 给 Workflow 加资源限制和超时
spec:
  activeDeadlineSeconds: 3600        # 整个 Workflow 最多跑 1 小时
  ttlStrategy:
    secondsAfterCompletion: 3600     # 完成后 1 小时自动清理
    secondsAfterSuccess: 1800        # 成功的 30 分钟后清理
    secondsAfterFailure: 86400       # 失败的保留 24 小时（方便排查）
  podGC:
    strategy: OnPodCompletion        # Pod 完成后自动删除（节约 etcd）
  templates:
    - name: my-step
      activeDeadlineSeconds: 600     # 单步最多 10 分钟
      retryStrategy:
        limit: 3                     # 最多重试 3 次
        retryPolicy: OnFailure
      container:
        resources:
          requests:
            cpu: "100m"
            memory: "256Mi"
          limits:
            cpu: "1"
            memory: "1Gi"
```

### 2.2 并发限制

```yaml
# 全局限制（controller 配置）
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: argo
data:
  # 最多同时跑 20 个 Workflow
  parallelism: "20"
  # 每个 Namespace 最多 5 个
  namespaceParallelism: "5"
```

```yaml
# 单个 Workflow 级别
spec:
  parallelism: 3                    # 这个 Workflow 最多同时跑 3 个 step
```

### 2.3 Artifact 归档

生产必须配置 Artifact Repository（S3/MinIO/OSS），不要用 emptyDir：

```yaml
# workflow-controller-configmap
data:
  artifactRepository: |
    archiveLogs: true              # 日志也存到 S3
    s3:
      bucket: argo-artifacts
      endpoint: s3.amazonaws.com
      accessKeySecret:
        name: argo-artifacts-s3
        key: accesskey
      secretKeySecret:
        name: argo-artifacts-s3
        key: secretkey
```

### 2.4 Workflow Archive（数据库归档）

Workflow CR 跑完后不能无限留在 etcd 里，需要归档到数据库：

```yaml
# workflow-controller-configmap
data:
  persistence: |
    archive: true
    postgresql:
      host: postgres.argo
      port: 5432
      database: argo
      tableName: argo_workflows
      userNameSecret:
        name: postgres-creds
        key: username
      passwordSecret:
        name: postgres-creds
        key: password
```

## 3. Argo Rollouts 生产最佳实践

### 3.1 AnalysisTemplate 设计原则

| 原则 | 说明 |
|------|------|
| **指标要有对照组** | 用 canary vs stable 的对比，而不是绝对值 |
| **给足观察时间** | interval 不要太短，避免噪声导致误判 |
| **设合理的 failureLimit** | 允许偶发失败，避免一次抖动就回滚 |
| **多维度指标** | 成功率 + 延迟 + 错误数，单一指标不够可靠 |

```yaml
# 推荐：对比型分析（canary 错误率 vs stable 错误率）
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: canary-vs-stable
spec:
  args:
    - name: canary-service
    - name: stable-service
  metrics:
    - name: error-rate-comparison
      interval: 2m
      count: 5
      failureLimit: 2
      # canary 的错误率不能比 stable 高出 1%
      successCondition: result[0] - result[1] < 0.01
      provider:
        prometheus:
          address: http://prometheus:9090
          query: |
            (
              sum(rate(http_requests_total{service="{{args.canary-service}}",status=~"5.."}[5m]))
              / sum(rate(http_requests_total{service="{{args.canary-service}}"}[5m]))
            ) - (
              sum(rate(http_requests_total{service="{{args.stable-service}}",status=~"5.."}[5m]))
              / sum(rate(http_requests_total{service="{{args.stable-service}}"}[5m]))
            )
```

### 3.2 回滚策略

```yaml
spec:
  strategy:
    canary:
      # 超时自动回滚
      progressDeadlineAbort: true
      progressDeadlineSeconds: 600    # 10 分钟内没完成就 abort

      # 分析失败的行为
      analysis:
        templates:
          - templateName: success-rate
        # 分析失败 → 自动 abort（默认行为）

      # abort 后的行为
      abortScaleDownDelaySeconds: 30  # abort 后 30s 缩容 canary pods
```

### 3.3 与 HPA 共存

Argo Rollouts 和 HPA 不能直接都管 replicas，需要用 Rollout 内置的 HPA 支持：

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-app-hpa
spec:
  scaleTargetRef:
    apiVersion: argoproj.io/v1alpha1
    kind: Rollout                     # 指向 Rollout 而不是 Deployment
    name: my-app
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

## 4. Argo Events 生产最佳实践

### 4.1 EventBus 高可用

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstream:
    version: "2.9.0"
    replicas: 3
    persistence:
      storageClassName: ssd
      accessMode: ReadWriteOnce
      volumeSize: 20Gi
```

### 4.2 EventSource 可靠性

```yaml
spec:
  # 多副本（Active-Active）
  replicas: 2
  
  # 事件去重
  eventBusName: default
  
  webhook:
    my-event:
      port: "12000"
      endpoint: /events
      # 配置 TLS
      serverCertSecret:
        name: webhook-tls
        key: tls.crt
      serverKeySecret:
        name: webhook-tls
        key: tls.key
```

### 4.3 Sensor 幂等性

Trigger 执行的动作必须是幂等的——因为事件可能重复投递：

```yaml
# 好的做法：用 generateName，每次创建新 Workflow
metadata:
  generateName: build-

# 不好的做法：用固定 name，重复创建会报 AlreadyExists
metadata:
  name: build-workflow
```

## 5. 通用生产清单

### 5.1 RBAC 最小权限

| 组件 | 需要的最小权限 |
|------|--------------|
| Argo CD Controller | 目标命名空间的 admin（不需要 cluster-admin） |
| Argo Workflows SA | 创建 Pod 的权限 + Artifact 存储权限 |
| Argo Events SA | 创建 Workflow/Pod 的权限 |
| Argo Rollouts Controller | Rollout/ReplicaSet/Service 的管理权限 |

### 5.2 网络策略

```yaml
# 限制 Argo CD 只能访问指定的 Git 仓库和集群 API
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: argocd-repo-server-egress
  namespace: argocd
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: argocd-repo-server
  policyTypes:
    - Egress
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0     # 生产中应限制为 Git 服务器 IP
      ports:
        - port: 443
          protocol: TCP
        - port: 22
          protocol: TCP
```

### 5.3 备份与灾难恢复

```bash
# 备份 Argo CD 的所有 Application 定义
kubectl get applications -n argocd -o yaml > argocd-apps-backup.yaml

# 备份 AppProject
kubectl get appprojects -n argocd -o yaml > argocd-projects-backup.yaml

# 备份 Repository 和 Cluster Secrets（注意这些包含凭据！）
kubectl get secrets -n argocd -l argocd.argoproj.io/secret-type=repository -o yaml > repos-backup.yaml
kubectl get secrets -n argocd -l argocd.argoproj.io/secret-type=cluster -o yaml > clusters-backup.yaml
```

灾难恢复步骤：
1. 重新安装 Argo CD
2. 恢复 Secret（仓库和集群凭据）
3. 恢复 AppProject
4. 恢复 Application（会自动开始 sync）

### 5.4 升级策略

| 组件 | 升级建议 |
|------|---------|
| Argo CD | 先在测试集群升级；检查 breaking changes；HA 模式下可滚动升级 |
| Argo Workflows | Controller 升级不影响正在运行的 Workflow |
| Argo Rollouts | Controller 升级不影响正在进行的 Rollout |
| Argo Events | EventBus 升级需注意数据迁移 |

## 6. 常见踩坑汇总

### Argo CD 常见坑

| 坑 | 原因 | 解法 |
|-----|------|------|
| 资源一直 OutOfSync | 有自动变化的字段（如 HPA 改 replicas） | 用 `ignoreDifferences` 忽略 |
| sync 超慢 | repo-server 克隆大仓库 | 加 `--depth 1` 或拆仓库 |
| repo-server OOM | 仓库太大 / Helm Chart 渲染占内存 | 加内存限制 |
| Application 删不掉 | finalizer 卡住 | 手动删 finalizer 注解 |
| Helm 的 Release 和 Argo CD 冲突 | 不要 `helm install`，让 Argo CD 管 | 用 Argo CD 的 Helm Source |
| 循环 Sync（CI push 触发 CD，CD sync 又触发 CI） | Webhook 配置问题 | 过滤自动 commit（bot 用户） |

### Argo Workflows 常见坑

| 坑 | 原因 | 解法 |
|-----|------|------|
| Step Pod 一直 Pending | 资源不足 / PVC 未 bound | 检查 requests + PVC |
| Artifact 传递失败 | S3 配置错误 / 权限 | 看 wait container 日志 |
| Workflow 跑完不清理 | 没配 TTL / PodGC | 加 `ttlStrategy` 和 `podGC` |
| etcd 被打满 | 太多 Workflow CR 没清理 | 开 Archive + TTL |
| 并发 Workflow 互相抢资源 | 没限制并发 | 加 `parallelism` |

### Argo Rollouts 常见坑

| 坑 | 原因 | 解法 |
|-----|------|------|
| 流量比例不精确 | 没配 trafficRouting | 配 Nginx/Istio 精确控制 |
| AnalysisRun 永远失败 | Prometheus 查询语法错误 / 无数据 | 先手动跑 PromQL 验证 |
| Rollout 一直 Paused | 到了 `pause: {}` 步骤 | 手动 `promote`（这是预期行为） |
| 升级镜像没触发 Canary | 只改了 ConfigMap 没改 Pod template | Rollout 只在 template 变化时触发 |
| abort 后新版本 Pod 还在 | scaleDownDelay 配置过长 | 调 `abortScaleDownDelaySeconds` |

### Argo Events 常见坑

| 坑 | 原因 | 解法 |
|-----|------|------|
| Webhook 收不到事件 | 网络不可达 / TLS 问题 | 检查公网 Ingress + 证书 |
| 事件收到了但没触发 | Sensor filter 不匹配 | 看 Sensor Pod 日志 |
| 重复触发 | EventBus at-least-once 投递 | Trigger 确保幂等 |
| EventSource Pod 频繁重启 | 内存不够 / 事件积压 | 加资源 / 消费速率 |

## 7. 监控大盘建议

一个完整的 Argo 全家桶监控大盘应该包含：

```text
┌─────────────────────────────────────────────────────────────┐
│                     Argo Suite Dashboard                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [Argo CD]                                                  │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Synced  │  │OutOfSync │  │ Degraded │  │ Git Req  │    │
│  │   45    │  │    2     │  │    1     │  │  120/min │    │
│  └─────────┘  └──────────┘  └──────────┘  └──────────┘    │
│                                                             │
│  [Argo Workflows]                                           │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Running │  │ Succeeded│  │  Failed  │  │  Pending │    │
│  │    3    │  │   156    │  │    2     │  │    0     │    │
│  └─────────┘  └──────────┘  └──────────┘  └──────────┘    │
│                                                             │
│  [Argo Rollouts]                                            │
│  ┌─────────────────────┐  ┌──────────────────────────────┐  │
│  │ Active Rollouts: 2  │  │ my-app: 30% canary ████░░░░ │  │
│  │ Last abort: 3d ago  │  │ api-svc: 100% stable ██████ │  │
│  └─────────────────────┘  └──────────────────────────────┘  │
│                                                             │
│  [Argo Events]                                              │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐                   │
│  │Events/h │  │ Triggers │  │  Errors  │                   │
│  │   240   │  │   45     │  │    0     │                   │
│  └─────────┘  └──────────┘  └──────────┘                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 8. 版本兼容性

确保四个工具的版本互相兼容：

| Argo 组件 | 推荐版本线 | K8s 兼容 |
|-----------|-----------|---------|
| Argo CD | v2.9+ | K8s 1.25+ |
| Argo Workflows | v3.5+ | K8s 1.25+ |
| Argo Rollouts | v1.6+ | K8s 1.22+ |
| Argo Events | v1.8+ | K8s 1.23+ |

> 升级前务必阅读对应版本的 Release Notes，关注 Breaking Changes。

## 总结：生产核心原则

1. **最小权限**：每个组件只给必要的 RBAC
2. **高可用**：HA 部署 + 多副本 + 持久化
3. **可观测**：Metrics + Alerting + 日志
4. **渐进推进**：先 dev/staging 验证，再上 prod
5. **备份恢复**：定期备份 Application/Secret 等关键资源
6. **限制资源**：TTL、并发限制、超时，避免跑满集群
7. **安全第一**：SSO、NetworkPolicy、Sealed Secrets、审计日志
