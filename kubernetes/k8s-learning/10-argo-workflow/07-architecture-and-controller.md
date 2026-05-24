# 🧠 Argo 架构与控制器原理

> 如果你只是用 Argo 写 Pipeline，前面 6 篇就够了。但如果你做平台、做集成、要排查"为什么 wf 卡在 Running 不动"、"为什么 pod 都跑完了 wf 还没结束"，就必须理解这一篇。

## 1. 高层架构图

```text
                    ┌─────────────────────────────┐
                    │           User              │
                    └────┬────────────┬───────────┘
              CLI/UI/REST│            │ kubectl apply (CR)
                         ▼            ▼
                  ┌─────────────┐  ┌────────────┐
                  │ argo-server │  │   etcd     │
                  │  (REST API) │  │            │
                  └────┬────────┘  └─────▲──────┘
                       │ create CR       │
                       ▼                 │
                  ┌─────────────────────────────┐
                  │       kube-apiserver        │
                  └────┬────────────────────────┘
                       │ watch
                       ▼
        ┌────────────────────────────────────────┐
        │        workflow-controller             │
        │  ┌──────────────────────────────────┐  │
        │  │  Informer (Workflow / Pod)       │  │
        │  ├──────────────────────────────────┤  │
        │  │  Operator (per-Workflow loop)    │  │
        │  ├──────────────────────────────────┤  │
        │  │  Pod Reconciler / GC / Metrics   │  │
        │  └──────────────────────────────────┘  │
        └─────┬───────────────────────┬──────────┘
              │ create/update Pod     │ persist status
              ▼                       ▼
        ┌────────────┐         ┌──────────────┐
        │ Step Pods  │ ──────▶ │ Object Store │
        │ (main+wait)│  upload │  (S3/MinIO)  │
        └────────────┘         └──────────────┘
```

记住三个核心组件：

| 组件 | 职责 |
|------|------|
| **argo-server** | REST API + UI；不直接干活，只是包装 K8s API |
| **workflow-controller** | 真正的"大脑"，watch CR、计算 DAG、创建 Pod、维护 status |
| **每一步的 Pod**（main + wait sidecar） | 真正干业务的地方 |

## 2. Pod 长什么样

每一步运行时的 Pod 至少有 2 个容器：

```text
┌──────────────────────────────────────────────┐
│ Pod (一步 = 一个 Pod)                         │
│                                              │
│  ┌──────────┐    ┌──────────┐                │
│  │  main    │    │  wait    │  (Argo 注入)   │
│  │ (你的)   │    │ sidecar  │                │
│  └──────────┘    └──────────┘                │
│        │              │                      │
│        │              ├─ 下载 input.artifact │
│        │              ├─ 监控 main 退出      │
│        │              ├─ 拷贝输出文件        │
│        │              └─ 上传 output.artifact│
│                                              │
│  共享卷：/argo（脚本）/argo-staging（artifact）│
└──────────────────────────────────────────────┘
```

老版本（< v3.4）还有一个 init 容器在 Pod 启动时下载文件；v3.4+ 大部分场景用 wait 一肩挑或 emissary executor。

### 2.1 Executor（执行器）

Argo 通过 "executor" 来跟主容器交互（拿日志、拿退出码、拿输出文件）。历史上有过 4 种：

| Executor | 状态 | 工作方式 |
|----------|------|----------|
| `docker` | 已废弃 | 通过宿主机 docker.sock |
| `kubelet` | 已废弃 | 通过 kubelet API |
| `k8sapi` | 已废弃 | 通过 K8s API exec |
| **`emissary`** | **当前默认** | wait 容器与 main 共享 volume，通过 `command` 重写实现 |

在 emissary 模式下，main 容器的 `command` 会被 Argo 重写成 emissary 二进制 + 你原来的 command，用来捕获 stdout/exit code。这是**为什么 main 容器的 image 必须有可执行 sh / 兼容 emissary 二进制**——纯 `FROM scratch` 镜像可能跑不起来，需要在 controller config 里加 `containerRuntimeExecutor` 或者镜像里手动加 emissary。

## 3. workflow-controller 内部循环

控制器的核心循环（伪代码）：

```text
forever:
    wf = workQueue.Get()                    # 从 workqueue 拿一个待处理的 wf
    wfOp = newOperator(wf)                  # 构造 operator
    wfOp.operate(ctx)                       # 主操作
        - 解析 spec.templates / entrypoint
        - 计算 DAG 当前节点状态
        - 找出"可以跑"的下一批节点
        - create Pod
        - 处理 Pod 状态变化（Pending/Running/Succeeded/Failed）
        - 更新 wf.status.nodes
        - 处理 retry / hook / onExit
    persist(wf)                             # 写回 K8s
    requeue(wf, after=...)                  # 视情况再排队
```

关键点：

- 每个 Workflow 在控制器里都对应一次 `operate()`，**整个 wf 的状态机都在内存里展开**
- 控制器**单写者**：通过 leader election 保证只有一个实例修改 wf.status
- Pod 状态变化也会 enqueue 关联的 wf 重新 operate

## 4. 状态机：节点 status 怎么走

每个节点（DAG 任务、step、template 调用都是节点）有 phase：

```text
Pending  -> Running -> Succeeded
                  \-> Failed
                  \-> Error
节点也可以是：Skipped / Omitted / Daemoned
```

Workflow 整体 phase：

```text
Pending  -> Running -> Succeeded
                  \-> Failed
                  \-> Error
```

排查时**永远先看 wf.status.nodes**，里面记录每个节点的 phase、message、children、template、boundaryID（属于哪个 dag/steps 的边界）。

```bash
kubectl get wf <name> -o yaml | yq .status.nodes
```

## 5. 为什么 wf 会"卡住"

理解了上面的循环，常见的"卡住"现象就能对号入座：

| 现象 | 通常的原因 |
|------|------------|
| 节点 Pending 不变 | Pod 调度不到（节点资源/亲和性/SA 权限） |
| 节点 Running 但 Pod 已结束 | wait 容器没退出 / executor 收尾失败（看 wait 日志） |
| 整个 wf Running 但没有 Pod | controller 没收到事件 / leader 丢失 / informer cache 异常 |
| wf 一直未启动 | controller 挂了 / workqueue 堵了 / RBAC 没权限 watch wf |
| 输出参数为空 | emissary 没拿到文件 / path 写错 / 容器自己删除了 |

## 6. CRD 大全

Argo Workflow 体系里的 CRD：

| CRD | 作用 |
|-----|------|
| `Workflow` (`wf`) | 一次具体执行 |
| `WorkflowTemplate` (`wftmpl`) | 命名空间级模板 |
| `ClusterWorkflowTemplate` (`cwt`) | 集群级模板 |
| `CronWorkflow` (`cwf`) | 定时调度 |
| `WorkflowTaskSet` (`wfts`) | http template / agent 模式相关 |
| `WorkflowArtifactGCTask` | 制品 GC |
| `WorkflowEventBinding` | webhook 提交映射 |

UI 上看到的"Workflow"列表、"Cron Workflows"列表，对应的就是这些 CR。

## 7. 高可用部署要点

| 维度 | 推荐做法 |
|------|----------|
| controller | 至少 2 副本 + leader election（默认开） |
| argo-server | 至少 2 副本 + Service 暴露 |
| 数据库 | PostgreSQL 集群（用于归档） |
| 对象存储 | S3 / MinIO 集群 |
| etcd 压力 | 一定开 ttlStrategy + podGC + archive，避免历史 wf 把 etcd 撑爆 |
| 限流 | controller config 配 `parallelism`、`namespaceParallelism` |

### 7.1 归档（archive）

Argo 默认所有 wf 都存在 etcd 里。这不可持续。开启归档后：

- 历史 wf 异步写到 PostgreSQL
- 老 wf 从 etcd 删除（按 ttlStrategy）
- UI 还能从 PostgreSQL 读出来回看

```yaml
# workflow-controller-configmap
data:
  persistence: |
    archive: true
    archiveTTL: 30d
    postgresql:
      host: pg.svc
      port: 5432
      database: argo
      tableName: argo_archived_workflows
      userNameSecret: {name: argo-pg, key: username}
      passwordSecret: {name: argo-pg, key: password}
```

## 8. 控制器关键配置文件

`workflow-controller-configmap`（namespace: argo）是 Argo 行为的总闸：

```yaml
data:
  # 全局并行度（controller 同时处理的 wf 数）
  parallelism: "20"
  # 每个 ns 并行 wf 数
  namespaceParallelism: "5"
  # 默认全部 wf 套上的 spec
  workflowDefaults: |
    spec:
      ttlStrategy: {secondsAfterCompletion: 86400}
      podGC: {strategy: OnWorkflowSuccess}
  # executor
  containerRuntimeExecutor: emissary
  # metrics
  metricsConfig:
    enabled: true
    path: /metrics
    port: 9090
```

修改后 controller 会自动 reload（多数字段），少数需要重启 deploy。

## 9. 监控指标（生产必看）

Argo 暴露了 Prometheus 指标，几个最关键的：

| 指标 | 含义 | 用途 |
|------|------|------|
| `argo_workflows_count` | 不同 phase 的 wf 数 | 看 Failed 是否飙升 |
| `argo_workflows_queue_depth_count` | controller workqueue 长度 | 高代表处理跟不上 |
| `argo_workflows_workers_busy_count` | 忙碌 worker | 看是否打满 |
| `argo_pod_missing` | wf 期望但找不到的 Pod 数 | 不应长期 > 0 |
| `argo_workflows_error_count` | 控制器内部错误 | 升级/回滚的依据 |

**告警建议**：queue_depth_count 持续 > 100 / 持续 10min 内一直在涨；error_count 任何时刻 > 0 都看一眼。

## 10. 一个真实的卡住排查流程示范

> wf 一直 Running，但 UI 上看节点是 Succeeded，时间戳 30 分钟没动了。

**Step 1**：看控制器日志

```bash
kubectl logs -n argo deploy/workflow-controller | grep <wf-name>
```

如果日志里在反复重试某个错误（如 etcd 写超时、archive 失败），先解决基础设施。

**Step 2**：看 wf.status.phase 和 nodes

```bash
kubectl get wf <name> -o yaml
```

如果 phase 还是 Running，但所有节点都 Succeeded，可能是 `onExit` 没跑。看 onExit 模板配置和它对应的 Pod。

**Step 3**：看 leader

```bash
kubectl get lease -n argo
```

确认 controller 当前 leader 是谁，是否在频繁切换（leader 抖动会导致 wf 短暂"卡住"）。

**Step 4**：看 Pod / wait 容器日志

```bash
kubectl logs <pod> -c wait
```

通常 emissary / wait 报错会写在这里，比如 artifact 上传失败、退出码读不到。

90% 的"卡住"问题都能在这 4 步内找到。

## 下一步

最后一篇专门讲生产实战和排障：

- [生产实践与故障排查](./08-production-and-troubleshooting.md)
