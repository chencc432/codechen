# 📝 Status 子资源与 Conditions 设计

> 一个写得好的 status 能让运维直接 `kubectl describe` 就看清现状；写得不好的 status 让用户看到一堆字段也不知道哪个对。本篇讲清楚 status 子资源、Conditions 模型、以及怎么落地。

## 1. 为什么 status 要单独是子资源

K8s 把 spec 和 status 在 API 层做了"逻辑分离"：

| 子资源 | endpoint | 谁写 |
|--------|----------|------|
| 主资源 (`/apis/.../foos`) | spec / metadata | 用户 |
| status 子资源 (`/apis/.../foos/<n>/status`) | 仅 status | 控制器 |

带来的好处：

- 用户 `kubectl edit` 改 spec 不会触碰 status
- 控制器 `Status().Update()` 走子资源路径，不会触发"对 spec 的更新"事件，避免反弹回路
- 可独立设置 RBAC：用户只能 patch spec、只有 controller SA 能 patch status
- ResourceVersion 仍共享同一个对象（status 改也会变 RV）

## 2. CRD 怎么开启 status 子资源

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: myapps.example.com
spec:
  versions:
    - name: v1
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec: {...}
            status: {...}
      subresources:
        status: {}                 # ← 关键
        scale:                     # 可选
          specReplicasPath: .spec.replicas
          statusReplicasPath: .status.replicas
```

Kubebuilder / operator-sdk 默认就会加上 `subresources.status`。

## 3. status 字段设计的几个原则

### 3.1 描述"现实"，不要复述 spec

不要把 spec 字段拷一份到 status；status 应当反映**控制器观察到的世界**：

| 应当放 status | 不应放 status |
|--------------|--------------|
| 实际可用副本数 | 期望副本数（spec 已经有） |
| 正在使用的 image SHA | 用户写的 image tag |
| 关联资源的 UID/名字 | 用户写的 selector |
| 失败原因、上次错误 | 用户的注释 |

### 3.2 字段命名要面向用户

用户在 `kubectl describe` 看到的会是字段名 → 名字尽量像"答案"：

```yaml
status:
  observedGeneration: 7
  readyReplicas: 3
  phase: Ready
  conditions: [...]
  lastReconcileTime: "2026-05-24T10:00:00Z"
  externalResourceID: "vol-abc123"
```

### 3.3 包含 observedGeneration

`metadata.generation` 是 spec 每次修改后递增的版本号。controller 在 status 里写 `observedGeneration` 表示"我已经看到并处理到 generation N"：

```text
spec.generation        = 5    （用户最后一次改是版本 5）
status.observedGeneration = 4 （controller 还在处理版本 4）
```

用户用这个对比能立刻知道"控制器是不是赶上了我的最新改动"。

## 4. Conditions：表达"多个并行状态"的标准

只用一个 `phase: Ready/Failed/Pending` 字段表达不了复杂状态。Kubernetes 在很多内置资源（Pod、Deployment、Node）上都使用 **Conditions 模型**：

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: AllReplicasAvailable
      message: 3/3 replicas available
      lastTransitionTime: "2026-05-24T10:00:00Z"
      observedGeneration: 5
    - type: Progressing
      status: "False"
      reason: NewReplicaSetAvailable
      message: ReplicaSet "abc" has successfully progressed.
      lastTransitionTime: "2026-05-24T09:55:00Z"
```

每个 Condition 是一组：

| 字段 | 含义 |
|------|------|
| `type` | 状态维度（Ready / Available / Degraded / Synced ...） |
| `status` | True / False / Unknown |
| `reason` | 机器可读的简短原因（CamelCase） |
| `message` | 人类可读的详细描述 |
| `lastTransitionTime` | 这个状态最后变化的时间 |
| `observedGeneration` | 状态对应的 spec 版本 |

### 4.1 推荐用 K8s 标准类型

`k8s.io/apimachinery/pkg/apis/meta/v1` 里有 `metav1.Condition` 标准类型，配合 `meta.SetStatusCondition()` 就能一行写 condition：

```go
import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/meta"
)

meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
    Type:    "Ready",
    Status:  metav1.ConditionTrue,
    Reason:  "AllReplicasAvailable",
    Message: fmt.Sprintf("%d/%d replicas available", ready, desired),
    ObservedGeneration: obj.Generation,
})
```

`SetStatusCondition` 会自动处理：
- 已有同 type 的 condition：更新 message/reason，仅在 status 变化时更新 LastTransitionTime
- 没有：追加一条

### 4.2 常用 Condition Type 命名建议

| Type | 含义 |
|------|------|
| `Ready` | 总状态，是否可用 |
| `Available` | 副本已就绪 |
| `Progressing` | 正在变化中（升级、滚动） |
| `Degraded` | 部分降级 |
| `Synced` / `Reconciled` | 控制器已同步到这个 generation |
| `<具体能力>Ready` | 比如 `DatabaseReady`、`StorageReady` |

> **Conditions 是加性的**：以后增加新 condition 不破坏老用户。但**重命名 condition type** 会破坏使用者，慎重。

## 5. status 写入要点

### 5.1 用 Status().Patch 而不是 Update

```go
patch := client.MergeFrom(original)
err := r.Client.Status().Patch(ctx, obj, patch)
```

Patch 比 Update 抗冲突好（其它 actor 改了别的字段不会让你的写失败）。

### 5.2 只在真的变化时写

```go
original := obj.DeepCopy()

// ... 修改 obj.Status ...

if reflect.DeepEqual(original.Status, obj.Status) {
    return nil    // 没变，不写
}
return r.Client.Status().Patch(ctx, obj, client.MergeFrom(original))
```

避免控制器自己 reconcile 自己（无变化也写一遍 → 触发自己的 watch → 又一次 reconcile）。

### 5.3 写 status 失败不要让整个 reconcile 反复重试

如果业务已经做完了，就因为 status 写失败而把一切重做，浪费且可能不幂等。把 status 写当作"尾声"，失败时重试 status 即可：

```go
if err := r.reconcileBusinessLogic(ctx, obj); err != nil {
    return ctrl.Result{}, err
}
return ctrl.Result{RequeueAfter: 1*time.Minute}, r.updateStatus(ctx, obj)
```

## 6. 一份"舒服"的 status 例子

```yaml
status:
  observedGeneration: 5
  phase: Ready
  externalDatabase:
    id: db-prod-9087
    endpoint: prod.rds.example.com:5432
  replicas:
    desired: 3
    ready: 3
    available: 3
  conditions:
    - type: Synced
      status: "True"
      reason: AsExpected
      message: "All resources match desired state"
      lastTransitionTime: "2026-05-24T10:00:00Z"
      observedGeneration: 5
    - type: Ready
      status: "True"
      reason: AllSubsystemsReady
      message: "Database, replicas, networking are all ready"
      lastTransitionTime: "2026-05-24T10:01:00Z"
      observedGeneration: 5
    - type: Degraded
      status: "False"
      reason: NoIssues
      message: ""
      lastTransitionTime: "2026-05-24T10:01:00Z"
      observedGeneration: 5
  lastReconcileTime: "2026-05-24T10:05:30Z"
```

`kubectl describe myapp` 时这种结构能清楚展现"我同步到哪个版本了，子系统各自如何，有没有问题"。

## 7. 配合 Printer Columns 让 kubectl get 更友好

CRD 里加一个 additionalPrinterColumns，让 `kubectl get` 直接展示 condition 摘要：

```yaml
additionalPrinterColumns:
  - name: Ready
    type: string
    jsonPath: .status.conditions[?(@.type=="Ready")].status
  - name: Phase
    type: string
    jsonPath: .status.phase
  - name: Age
    type: date
    jsonPath: .metadata.creationTimestamp
```

效果：

```text
NAME       READY   PHASE      AGE
myapp-1    True    Ready      3h
myapp-2    False   Degraded   1h
```

## 8. 常见问题

| 问题 | 解决 |
|------|------|
| status 写后又触发 reconcile | 改 watch 用 predicate 过滤 status 变化；或者只在内容变化时写 |
| status 字段太多用户看不懂 | 加 conditions + printer columns + events |
| 没人维护历史 condition | 旧 condition 不要删，状态翻成 False 即可，方便回看 |
| 多个并发 reconcile 互相覆盖 status | 用 Patch + MergeFrom；冲突让 workqueue 重试 |
| 异步任务的状态 | 在 status 记录"运行 ID"，下次 reconcile poll 它的状态 |

## 9. 用 Events 补充 status

status 描述当前状态，**Events 描述发生过什么**。两者是互补的，不要用 status 记历史。

```go
r.Recorder.Eventf(obj, corev1.EventTypeNormal,
    "Provisioned", "Database %s created", obj.Status.ExternalDatabase.ID)
```

`kubectl describe` 默认会展示最近的 events，让排障更顺。

## 10. 一句话总结

> spec 是输入，status 是输出，conditions 是"输出的多维表达"。
> 控制器写 status 要：observedGeneration 保新鲜，conditions 标准化，patch 防覆盖，无变化不写避免抖动。

## 下一步

了解了 reconcile + status，下一篇讲工程上常见的两条路线对比：

- [controller-runtime 与原生 client-go 对比](./07-controller-runtime-vs-clientgo.md)
