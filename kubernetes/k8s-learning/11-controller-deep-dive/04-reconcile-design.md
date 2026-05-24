# 🧩 Reconcile 函数的设计要点

> 写控制器最容易出问题的就是 Reconcile 函数。这一篇专门讲：怎么写出正确、幂等、可重试、性能合理、可读性好的 Reconcile。

## 1. Reconcile 的典型骨架

不论 client-go 风格还是 controller-runtime 风格，Reconcile 都遵循这个骨架：

```go
func (r *Reconciler) Reconcile(ctx context.Context, req Request) (Result, error) {
    // 1. Get 最新对象
    obj := &myv1.MyApp{}
    if err := r.client.Get(ctx, req.NamespacedName, obj); err != nil {
        if apierrors.IsNotFound(err) {
            return Result{}, nil           // 对象已删，无事可做
        }
        return Result{}, err               // 其他错误，重试
    }

    // 2. 处理删除（finalizer 路径）
    if !obj.DeletionTimestamp.IsZero() {
        return r.reconcileDelete(ctx, obj)
    }

    // 3. 确保 finalizer 存在
    if !controllerutil.ContainsFinalizer(obj, FinalizerName) {
        controllerutil.AddFinalizer(obj, FinalizerName)
        if err := r.client.Update(ctx, obj); err != nil { return Result{}, err }
        return Result{Requeue: true}, nil
    }

    // 4. 期望与现实对账
    desired := buildDesired(obj)
    actual, err := r.observe(ctx, obj)
    if err != nil { return Result{}, err }

    if needCreate(actual) {
        if err := r.client.Create(ctx, desired); err != nil { return Result{}, err }
    } else if drift(desired, actual) {
        if err := r.client.Patch(ctx, desired, ...); err != nil { return Result{}, err }
    }

    // 5. 写 status（子资源）
    if err := r.updateStatus(ctx, obj, actual); err != nil { return Result{}, err }

    return Result{RequeueAfter: 1 * time.Minute}, nil
}
```

下面分块讲清楚为什么这么写。

## 2. 处理 NotFound 是常态，不是错误

队列里的 key 在 reconcile 取出时，对象可能刚被删掉。**NotFound 必须当作"无事可做"返回，不要再当作错误重试**。

```go
if apierrors.IsNotFound(err) {
    return ctrl.Result{}, nil
}
```

否则你会看到一个已删除对象的 key 被无限重试 5 次后才 Forget，浪费、而且日志吵。

## 3. 永远从 Lister/缓存读，但写要走 API server

写控制器时一个不变的规律：

| 操作 | 方式 |
|------|------|
| **读对象** | 从 Lister / cache | （不打 API server） |
| **写对象** | 走 client（API server） | （cache 会通过 watch 自动更新） |

如果你写完之后**马上 Get 自己刚写的对象**，可能拿到的还是旧值（cache 还没同步）。这种"读自己刚写的"应当**避免**——下一次 reconcile 自然会拿到新值。

## 4. 幂等的几个落地手法

### 4.1 创建前 Get

```go
existing := &corev1.ConfigMap{}
err := r.client.Get(ctx, key, existing)
switch {
case apierrors.IsNotFound(err):
    return r.client.Create(ctx, desired)
case err != nil:
    return err
default:
    if !equal(existing, desired) {
        existing.Data = desired.Data
        return r.client.Update(ctx, existing)
    }
}
```

### 4.2 用 Server-Side Apply

更现代的方式：

```go
return r.client.Patch(ctx, desired, client.Apply, client.FieldOwner("my-controller"))
```

服务端会比较你声明的字段，自动 merge。多个控制器编辑同一资源时不会冲突。

### 4.3 用 controllerutil.CreateOrUpdate

```go
op, err := controllerutil.CreateOrUpdate(ctx, r.client, desired, func() error {
    desired.Data = renderData(obj)
    return nil
})
```

mutate 函数里只填字段，不要做副作用。

## 5. 错误处理三分类

把错误分成三类对待：

| 错误类型 | 例子 | 处理 |
|----------|------|------|
| **暂时性** | 网络抖动、API server 5xx、Conflict | 返回 err，让 workqueue 退避重试 |
| **永久性** | 对象本身 spec 不合法 | 写入 status.condition，**不要**反复重试 |
| **预期内** | NotFound（对象被删）、AlreadyExists（已创建） | 直接返回 nil |

写代码时一段非常常见的模式：

```go
if err := doSomething(); err != nil {
    if isPermanent(err) {
        setCondition(obj, "InvalidSpec", err.Error())
        _ = r.updateStatus(ctx, obj)
        return ctrl.Result{}, nil      // 不重试
    }
    return ctrl.Result{}, err         // 让 workqueue 退避
}
```

## 6. status 怎么写

### 6.1 用 status 子资源

`Status().Update()` 走的是 `/status` 子资源，**不会改 spec、不会触发再次 watch（如果只 watch spec 变化）**：

```go
err := r.client.Status().Update(ctx, obj)
```

CRD 必须开启 `subresources.status: {}` 才能用。

### 6.2 不要在 spec watcher 里因为自己改 status 又触发 reconcile

如果 watch 的是整个对象（默认就是），改 status 会触发 ResourceVersion 变化 → cache 推送 → 又一次 reconcile。这通常没问题（reconcile 是幂等的），但要注意**不要让 status 的写入产生新的差异 → 又写 status → 死循环**。

实战经验：

- 同一字段每次写值要稳定（不要每次 reconcile 都 push 新时间戳到无关字段）
- 如果有"上次更新时间"，只在状态真变化时更新

### 6.3 优先用 Conditions 表达状态

详细见 [Status 与 Conditions 设计](./06-status-and-conditions.md)。

## 7. Result 返回值的语义复习

| 返回 | 何时用 |
|------|--------|
| `Result{}, nil` | 完成，按事件驱动等下一次 |
| `Result{Requeue: true}, nil` | 立即再来一次（用于"刚改了对象，下一轮要补做") |
| `Result{RequeueAfter: 30s}, nil` | 30s 后定期回看（用于"等外部资源就绪") |
| `Result{}, err` | 报错，按限速器退避 |

绝大多数业务用前两种 + `Result{}, err` 三种就够了。RequeueAfter 用于"目标资源是异步的，需要 poll" 场景，比如等 Pod Ready、等外部 API 返回 200。

## 8. 怎么在 Reconcile 里调用外部系统

很多 Operator 要调用云厂商 API、调用内部服务。原则：

1. **超时**：`context.WithTimeout(ctx, 30*time.Second)`，避免 reconcile 卡死整个 worker
2. **重试**：返回 err 让 workqueue 退避，**不要在 reconcile 内部 sleep + retry**
3. **写 status**：调用结果记到 condition，让用户在 `kubectl describe` 时看到
4. **幂等参数**：把"对象 UID"作为外部资源的 idempotency key，避免重复创建外部资源

## 9. 关于"读 spec 写 status" 的代码风格

为了避免引用混乱，建议：

```go
obj := &myv1.MyApp{}
r.Get(ctx, req.NamespacedName, obj)

original := obj.DeepCopy()    // 备份一份用于 patch base

// 业务处理：obj 上修改 status 字段

if !reflect.DeepEqual(obj.Status, original.Status) {
    if err := r.client.Status().Patch(ctx, obj, client.MergeFrom(original)); err != nil {
        return ctrl.Result{}, err
    }
}
```

`Patch` + `MergeFrom` 比 `Update` 更不容易冲突，推荐写 status 用这种方式。

## 10. 性能与并发关注点

| 关注点 | 建议 |
|--------|------|
| Reconcile 单次耗时 | 控制在 100ms 以内为佳；超过 1s 立刻找原因 |
| 单对象循环触发频率 | 监控；可观测；如果一秒内同一 key 多次入队，看是不是自己在 update 自己 |
| 内存 | 别在 controller 实例上保留对象引用（cache 已经有一份） |
| Goroutine | 默认 worker 数 1~10 够大多数场景；高频对象多的可调高 |
| API 调用 | 优先 Patch，少用 Update；多对象操作用 SSA |

## 11. Reconcile 写不下的事情：把它拆出去

如果一个 reconcile 函数超过 200 行，强烈建议拆：

```text
Reconcile()
├── reconcileFinalizer()
├── reconcileServiceAccount()
├── reconcileDeployment()
├── reconcileService()
└── reconcileStatus()
```

每个子函数遵循"对账某一类资源"的单一职责，外层 Reconcile 只串起来。这样后续加资源、加单测都很容易。

## 12. 单测怎么写

controller-runtime 提供 `envtest`：起一个真实的 etcd + apiserver 用于跑测试。

```go
testEnv := &envtest.Environment{
    CRDDirectoryPaths: []string{"config/crd/bases"},
}
cfg, _ := testEnv.Start()
```

写单测时关注：

- 提交 spec → 等条件 → 断言 status / 子资源
- 模拟错误（删 secret、改坏 spec），断言 condition 反映 reason
- 多次 reconcile 应当幂等（断言资源数量不变）

## 13. 一句话总结

> Reconcile 不是事件处理函数，而是"对账函数"：每次都从最新 spec 出发，看现实差什么就补什么；失败就让 workqueue 退避重试；状态都写到 status。

## 下一步

控制器要在生产高可用、要安全删除资源，离不开 Leader Election + Finalizer：

- [Leader Election 与 Finalizer](./05-leader-election-and-finalizers.md)
