# 🛡️ 生产实践与故障排查

> 这一篇汇总把 Argo Workflow 跑到生产之后会遇到的事：稳定性、性能、安全、值班排障。读完本篇，你应该能拍着胸脯把 Argo 接到业务里。

## 1. 上生产前的 10 个 checkbox

| # | 项 | 默认是否OK | 必做 |
|---|-----|-------------|------|
| 1 | controller 至少 2 副本 + leader election | ❌（quick-start 1 副本） | ✅ |
| 2 | TTL + PodGC 已配置 | ❌ | ✅ |
| 3 | 归档（archive）打开 + Postgres 备份 | ❌ | ✅ |
| 4 | parallelism / namespaceParallelism 限流 | ❌ | ✅ |
| 5 | 默认 SA 收紧；业务 wf 用专用 SA | ❌（默认 SA 高权限） | ✅ |
| 6 | activeDeadlineSeconds 强制兜底 | ❌ | ✅ |
| 7 | 制品仓库默认 + keyFormat 命名规范 | ❌ | ✅ |
| 8 | metrics 接 Prometheus + 告警规则 | ❌ | ✅ |
| 9 | UI 走 Ingress + RBAC + 真实证书 | ❌（自签） | ✅ |
| 10 | 镜像源走内网 / 镜像缓存 | ❌ | ✅ |

下面分维度展开。

## 2. 稳定性最佳实践

### 2.1 限制单个 wf 的资源使用

```yaml
spec:
  parallelism: 10                       # 同一 wf 内并行 Pod 数上限
  podGC:
    strategy: OnPodCompletion           # 跑完就删，避免堆积
  ttlStrategy:
    secondsAfterCompletion: 86400       # 跑完一天后自动清
    secondsAfterSuccess: 3600           # 成功的更短
    secondsAfterFailure: 604800         # 失败的留久点便于排查
  activeDeadlineSeconds: 7200
```

### 2.2 每一步必须有资源 request/limit

不写 request 容易把节点压垮，不写 limit 容易跑爆 OOM。**强制写**。

```yaml
container:
  resources:
    requests: {cpu: 500m, memory: 512Mi}
    limits:   {cpu: "2",  memory: 4Gi}
```

可以在 `workflowDefaults` 兜底，但还是建议业务方显式写。

### 2.3 控制器侧限流

`workflow-controller-configmap`：

```yaml
data:
  parallelism: "30"            # 全集群同时进入 operate 的 wf 上限
  namespaceParallelism: "10"   # 每个 namespace 上限
```

在 wf 上也可以显式声明：

```yaml
spec:
  podPriorityClassName: low-priority
```

### 2.4 默认 wf 兜底

```yaml
data:
  workflowDefaults: |
    spec:
      ttlStrategy: {secondsAfterCompletion: 86400}
      podGC: {strategy: OnWorkflowSuccess}
      activeDeadlineSeconds: 7200
      serviceAccountName: argo-workflow-sa
      podMetadata:
        labels:
          app.kubernetes.io/managed-by: argo
```

兜底是平台治理最便宜的手段，业务方什么都不用改也能拿到默认行为。

## 3. 性能调优

### 3.1 Workflow 太大（节点数过多）会拖累控制器

Argo 把 wf 整个状态机存在 `wf.status.nodes` 里，节点数过多（> 几千）会导致：

- etcd 单 key 体积过大（K8s 限制 1.5MB / object）
- controller 反复读写慢
- UI 渲染卡

**应对**：

- 大批量数据处理用 fan-out（withParam）展开成子任务，但避免单 wf 超过几百个节点
- 真要跑万级任务量，拆成多个 wf；或用 Argo 提供的 `WorkflowTaskSet` 模式（v3.5+）将每步视为独立资源
- 节点超大时考虑 [Pod Cleanup + Status Compression](https://argo-workflows.readthedocs.io/en/latest/scaling/) 等机制

### 3.2 高频提交（CronWorkflow 触发量大）

- 把 controller 的 worker 数调上去（启动参数 `--workflow-workers`，默认 32）
- archive 异步、podGC 异步，别让它们和主循环抢 CPU
- 监控 `argo_workflows_queue_depth_count`，看是不是积压

### 3.3 etcd 压力

- 必开 ttlStrategy + archive + podGC
- 单 wf 不要塞大段日志/JSON 进 parameter
- 单 ns 长期保留的 wf 数量控制在几百以内（archive 之后就清）

## 4. 安全实践

### 4.1 ServiceAccount 最小权限

默认 `argo` namespace 的 SA 给得很大方，**业务命名空间一定要单独设**：

```yaml
# argo-workflow-sa：业务 wf 内部 Pod 用
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-workflow-sa
  namespace: app-team

---
# 提交者用：argo-submit-sa（只允许 create wf、wftmpl）
```

通过 RoleBinding 精细授权：

```yaml
- apiGroups: [argoproj.io]
  resources: [workflows, workflowtaskresults, workflowtasksets]
  verbs: [create, get, list, watch, update, patch]
- apiGroups: [""]
  resources: [pods, pods/log]
  verbs: [get, list, watch]
```

### 4.2 不要让业务方拿到 controller 的 token

argo-server 的 SA 别和业务共享。UI 走 SSO（OIDC）授权后再决定能看哪些 namespace。

### 4.3 UI/REST 鉴权

argo-server 启动参数：

```bash
--auth-mode=sso              # 推荐用 SSO（OIDC）
--secure=true                # HTTPS
--access-control-allow-origin
```

避免 `auth-mode=server` 这种"只要进得来就是 admin"的模式。

### 4.4 镜像与依赖

- 限定镜像 registry（OPA / Kyverno 拦截）
- 跑用户提交的脚本（`script` template）时，注入更小权限的 SA
- 不要让 Workflow 默认能访问宿主机网络 / hostPath

## 5. 排障速查

### 5.1 wf 一直 Pending

```bash
kubectl describe wf <name> -n <ns>
kubectl logs -n argo deploy/workflow-controller --tail=200 | grep <name>
```

主因：

- 没找到 entrypoint 模板
- 引用的 wftmpl 不存在 / 跨 ns 没权限
- controller leader 没选出来 / 没起来

### 5.2 节点 Pending 不变

```bash
# 找到对应 Pod
kubectl get pods -n <ns> -l workflows.argoproj.io/workflow=<name>
kubectl describe pod <pod>
```

经典原因：

- Insufficient cpu/memory
- Image pull 失败（私有 registry secret 没挂）
- nodeSelector / taints 不匹配（GPU 池常见）
- SA 没有创建 Pod 权限（看 events）

### 5.3 节点 Succeeded 但 wf 没结束

- 看 onExit 是不是失败了 / 卡死了
- 看 controller log 是否在 retry / panic
- 看 wf.status.phase 与 message

### 5.4 Output / Artifact 拿不到

```bash
kubectl logs <pod> -c wait | tail -100
```

最常见：

- path 写错、目录不存在
- artifact 仓库凭证过期
- 容器自己 `exit 0` 之后又被杀（OOM / preempt）

### 5.5 UI 加载慢 / 列表打不开

- archive 表越来越大没建索引：在归档 PG 上建 `(namespace, name)` 等复合索引
- 单 ns 上 wf 数量过多（几千以上），UI 拉列表时间长：缩短 archiveTTL 或加分页过滤
- argo-server 副本数不够：扩到 3+

### 5.6 控制器 OOM / 频繁重启

- 单 wf 太大（节点 > 几千）：拆 wf
- workqueue 严重积压：扩 worker 或加 controller 副本（leader 模式还是只有一个干活，但备机存在能快速 failover）
- archive 写入卡住反压主循环：把 archive 异步化（默认就是异步，但 PG 慢会导致 channel 满）

## 6. 一份"上线前 yaml 评审 checklist"

代码 review 你的同事提的 wf 时，依次看：

```text
[ ] entrypoint 是否存在
[ ] 所有 dependencies 节点名是否拼写正确
[ ] 每个 container 是否写了 resources requests + limits
[ ] activeDeadlineSeconds 是否有上限
[ ] retryStrategy 是否有 limit（避免无限重试）
[ ] withParam 输入是否合法 JSON
[ ] artifact path 是否绝对路径，是否真的会被生成
[ ] serviceAccountName 是否最小权限
[ ] image 是否走内网 registry
[ ] 是否设置 ttlStrategy / podGC（如果没全局兜底）
[ ] 敏感信息是否走 envFrom secret，而不是写在 parameter 里
```

## 7. 一些"血的教训"

### 7.1 不要在 parameter 里塞大段日志

> 真实事故：某团队把 build log 全塞进 output parameter，单条 wf 占 etcd 4MB，controller 写超时雪崩。

文件 → artifact，永远不要 parameter。

### 7.2 不要 `concurrencyPolicy: Allow` 加上重并发任务

> 某 cron 每分钟跑一次，单次跑 30s，遇到外部依赖卡住后所有 wf 全堆积，半天打满 namespace 配额。

`Forbid` / `Replace` 是默认选择。

### 7.3 不要在退出处理器里依赖前面节点的输出

`onExit` 模板里 `{{tasks.X.outputs}}` 不可用，只能用 `{{workflow.*}}`。

### 7.4 镜像没装 sh / 兼容 emissary 会怪事不断

`FROM scratch` + 静态二进制的镜像跑 Argo 经常莫名报错。最简单的解法是镜像里加 `busybox` / `alpine`，或在 controller 配置里把 emissary 二进制注入到 initContainer 共享卷。

### 7.5 节点数爆炸的 fan-out

`withParam` 一次展开几千个子任务，会让 wf.status 极速膨胀。**一次 fan-out 控制在百级以内**，更大批量考虑切多个 wf。

## 8. 一份生产部署"开箱即用" controller config 模板

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-controller-configmap
  namespace: argo
data:
  parallelism: "30"
  namespaceParallelism: "10"
  containerRuntimeExecutor: emissary
  workflowDefaults: |
    spec:
      ttlStrategy:
        secondsAfterCompletion: 86400
        secondsAfterSuccess: 3600
        secondsAfterFailure: 604800
      podGC:
        strategy: OnWorkflowSuccess
      activeDeadlineSeconds: 7200
      serviceAccountName: argo-workflow-sa
      podMetadata:
        labels:
          app.kubernetes.io/managed-by: argo
  persistence: |
    archive: true
    archiveTTL: 30d
    postgresql:
      host: argo-pg.svc
      port: 5432
      database: argo
      tableName: argo_archived_workflows
      userNameSecret: {name: argo-pg, key: username}
      passwordSecret: {name: argo-pg, key: password}
  artifactRepository: |
    s3:
      endpoint: minio.argo.svc:9000
      bucket: argo-artifacts
      insecure: true
      keyFormat: "{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}"
      accessKeySecret: {name: minio-cred, key: accesskey}
      secretKeySecret: {name: minio-cred, key: secretkey}
  metricsConfig: |
    enabled: true
    path: /metrics
    port: 9090
```

把这份 config 套上、装上多副本 controller、装上多副本 argo-server、配好 SSO + Ingress，你就拿到了一套基本扛得住生产负载的 Argo Workflow。

## 总结

读完整个专题，你应该具备这些能力：

1. 看懂任意一份 Workflow yaml 的字段含义
2. 写出带 DAG、参数、制品、重试、超时的真实业务流水线
3. 用 WorkflowTemplate / CronWorkflow 把流水线复用、定时化
4. 描述清楚 argo-server / controller / Pod / 对象存储 之间的关系
5. 把 Argo 接到生产环境，配置好 RBAC、限流、归档、监控
6. 出问题时能 30 分钟内找到根因（卡住、Pending、artifact 失败等）

## 拓展阅读

- 官方文档：[argo-workflows.readthedocs.io](https://argo-workflows.readthedocs.io/)
- Examples 仓：[github.com/argoproj/argo-workflows/tree/main/examples](https://github.com/argoproj/argo-workflows/tree/main/examples)
- Argo Events（事件驱动触发）：[argoproj.github.io/events](https://argoproj.github.io/events/)
- Kubeflow Pipelines（基于 Argo 的 ML pipeline 上层 SDK）：[kubeflow.org/docs/components/pipelines/](https://www.kubeflow.org/docs/components/pipelines/)
