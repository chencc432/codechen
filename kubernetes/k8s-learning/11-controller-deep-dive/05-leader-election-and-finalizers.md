# 👑 Leader Election 与 Finalizer

> 这一篇讲两个看起来不相关、但都是控制器"扛生产"绕不开的话题：**Leader Election（高可用）** 和 **Finalizer（安全删除）**。

# 第一部分：Leader Election

## 1. 为什么需要 Leader Election

控制器是"对状态做决策"的进程。如果你部署 3 副本：

- 3 个进程都 watch 到同一个 wf 创建
- 3 个进程都跑 reconcile
- 3 个进程都尝试创建子资源 → 冲突 + 重复 + 状态来回抖

**单一处理者**是控制器的不变量。Leader Election 用来在多副本中选出唯一的"现任 leader"，只有 leader 真正干活，其它副本待命随时接替。

## 2. 实现原理（Lease 模式）

K8s 自带 `coordination.k8s.io/v1.Lease` 资源，专门用来做选主。

```text
副本 A: 我是 Leader（持有 Lease，holder=A）
副本 B: 等待，定期探测 Lease 过期没
副本 C: 等待

Leader 每 X 秒续约一次（Renew）
副本 A 挂了 → 续约停止 → Lease 过期 → B/C 抢着续约 → 新的 Leader 上位
```

三个关键参数（client-go 的 LeaderElectionConfig）：

| 参数 | 含义 | 典型值 |
|------|------|--------|
| `LeaseDuration` | Lease 的有效期；超过这个时长未续约就视为过期 | 15s |
| `RenewDeadline` | 当前 Leader 必须在此时间内成功续约，否则放弃身份 | 10s |
| `RetryPeriod` | 非 Leader 重试争抢的间隔；Leader 续约失败后的重试间隔 | 2s |

**约束：`LeaseDuration > RenewDeadline > RetryPeriod`**。例如 15/10/2 是常见组合。

## 3. controller-runtime 里怎么开

```go
mgr, err := ctrl.NewManager(cfg, ctrl.Options{
    LeaderElection:          true,
    LeaderElectionID:        "myapp.controller.example.com",
    LeaderElectionNamespace: "myapp-system",
    LeaderElectionResourceLock: "leases",
    // 可选：调整时长
    LeaseDuration: ptrDuration(15 * time.Second),
    RenewDeadline: ptrDuration(10 * time.Second),
    RetryPeriod:   ptrDuration(2 * time.Second),
})
```

`LeaderElectionID` 通常用 controller 的全限定域名，避免和别的控制器抢同一个 Lease。

部署时**多副本 + 同一 Deployment** 即可，Lease 会自动协调。

## 4. Leader 切换会发生什么

切换瞬间会有一段"无主时间"（取决于 LeaseDuration），新 Leader 接手后：

- 重建 Informer 缓存（List + Watch）
- WaitForCacheSync 完成后才开始处理
- workqueue 是新 Leader 自己的，不会"继承"旧 Leader 的待办

副作用：

- **短暂处理停顿**（一般 10s 以内）
- 已经在跑一半的外部调用如果 Leader 挂了，可能没回来更新 status——下一次 reconcile 会重新做一遍，因此 reconcile 必须幂等
- 切换抖动会出现"短时双 Leader"风险，主要靠 etcd 的 Lease 一致性兜底；业务侧靠 reconcile 幂等

## 5. Leader Election 的常见坑

| 坑 | 现象 | 解决 |
|----|------|------|
| 多副本但忘了开 LeaderElection | 多个进程同时 reconcile，资源被反复创建 | 必开 |
| 网络抖动导致频繁切主 | reconcile 中断、status 抖动 | 调高 LeaseDuration（但接管也变慢） |
| Lease 没权限 | controller 启动直接 panic | RBAC 给 `coordination.k8s.io/leases` 的 `get/list/watch/create/update/patch` |
| 不同环境用同一个 LeaseID | 测试环境意外抢生产 Lease | 用 namespace 隔离或不同 LeaseID |

## 6. 不只是"防止双跑"

Leader Election 还有一个更实用的副效果：**优雅升级**。
滚动更新时新副本起来 → 老 Leader Pod 收到 SIGTERM → 主动放弃 Lease → 新副本立即接管 → 几乎零中断。

为了让这条路径顺畅，记得：

- 给 controller Pod 设置 `terminationGracePeriodSeconds` 足够大（30s+）
- 用 controller-runtime 的 `mgr.Start(ctx)` 模式，让 ctx 取消时能正常释放 Lease

---

# 第二部分：Finalizer

## 7. 为什么需要 Finalizer

K8s 的删除是**异步**的：

```text
用户 kubectl delete obj
  → API server 把 deletionTimestamp 设上
  → controller 看到 deletionTimestamp，做清理
  → controller 移除 finalizer
  → API server 真正物理删除
```

如果没有 Finalizer，API server 会立即删除对象，控制器**根本来不及清理外部资源**（云盘、外部 API 创建的资源、依赖项的 owner reference 顺序）。

Finalizer 的作用：**把对象的"删除"动作拦下来，等清理完才放行**。

## 8. Finalizer 的工作流程

```text
对象创建
  └─ controller 在第一次 reconcile 时给对象加上 finalizer
     metadata.finalizers: ["myapp.example.com/cleanup"]

用户删除
  └─ API server 设置 deletionTimestamp（不立即删）
     对象进入 "Terminating" 状态

controller 看到 deletionTimestamp
  └─ 执行清理：
        - 删除外部资源
        - 通知下游
        - 等待依赖资源消失
  └─ 全部完成后：移除 finalizer

API server 看到 finalizers 为空 + deletionTimestamp 已设
  └─ 真正物理删除对象
```

## 9. 标准实现

```go
const FinalizerName = "myapp.example.com/cleanup"

func (r *Reconciler) Reconcile(ctx context.Context, req Request) (Result, error) {
    obj := &myv1.MyApp{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return Result{}, client.IgnoreNotFound(err)
    }

    // 删除路径
    if !obj.DeletionTimestamp.IsZero() {
        if !controllerutil.ContainsFinalizer(obj, FinalizerName) {
            return Result{}, nil    // 已经被处理过，等 API server 物理删
        }
        if err := r.cleanupExternalResources(ctx, obj); err != nil {
            return Result{}, err    // 清理失败，重试
        }
        controllerutil.RemoveFinalizer(obj, FinalizerName)
        if err := r.Update(ctx, obj); err != nil {
            return Result{}, err
        }
        return Result{}, nil
    }

    // 创建/更新路径：先确保 finalizer 在
    if !controllerutil.ContainsFinalizer(obj, FinalizerName) {
        controllerutil.AddFinalizer(obj, FinalizerName)
        if err := r.Update(ctx, obj); err != nil {
            return Result{}, err
        }
        return Result{Requeue: true}, nil   // 这一轮就到此为止，下一轮真正干业务
    }

    // 正常 reconcile 业务逻辑
    return r.reconcileNormal(ctx, obj)
}
```

记住三个动作的顺序：

```text
正常路径：     先加 Finalizer → 再干业务
删除路径：     先做清理       → 再移除 Finalizer
```

如果反过来，会有竞态：清理还没完，对象就被物理删除了。

## 10. Finalizer 命名规范

按 Kubernetes 约定：`<域名>/<动作或目的>`，例如：

- `kubernetes.io/pv-protection`
- `myapp.example.com/cleanup`
- `infra.acme.com/release-cloud-disk`

**带域名前缀**是必须的，否则 API server 会拒绝（防止与系统 finalizer 冲突）。

## 11. cleanupExternalResources 设计原则

清理函数必须满足：

1. **幂等**：可能反复执行
2. **不抛非预期错误**：把"已经不存在"当作成功；只有真正瞬时错误才返回 err 让重试
3. **不依赖被删对象的 spec**：spec 可能在删除前就变化了；用 status 里记录的真实位置（如外部资源 ID）来清理
4. **优先用 OwnerReference 让 K8s 自己清**：能交给 K8s 级联删除的，不要自己手撸

例子：

```go
func (r *Reconciler) cleanupExternalResources(ctx context.Context, obj *myv1.MyApp) error {
    if obj.Status.ExternalID == "" {
        return nil    // 还没创建过外部资源
    }
    if err := r.cloud.Delete(ctx, obj.Status.ExternalID); err != nil {
        if errors.Is(err, cloud.ErrNotFound) {
            return nil
        }
        return err
    }
    return nil
}
```

## 12. Finalizer 的常见坑

| 坑 | 现象 | 解决 |
|----|------|------|
| 卡在 Terminating 删不掉 | finalizer 一直没被移除 | 看 controller 日志为什么 cleanup 失败；最后兜底 `kubectl patch` 强制清空 finalizer |
| 控制器挂掉 → 永远删不掉 | 持有 finalizer 的 controller 不在了 | 部署多副本 + leader election |
| 清理函数写错，永远报错 | 对象永远卡 Terminating | 加超时/最大重试，超过就强制清并报警 |
| Finalizer 加了，没移除分支 | 每个对象删除都卡 | 删除路径必走 RemoveFinalizer + Update |
| 并发竞态 | Update 报 Conflict | 重新 Get → 修改 → Update；正常重试即可 |

## 13. 强制删除（不推荐但要会）

```bash
# 危险操作：清空 finalizer 强制删除
kubectl patch <kind> <name> -p '{"metadata":{"finalizers":null}}' --type=merge
```

这只在控制器彻底坏了无法恢复时使用。这样做的代价：**外部资源不会被清理，要手动收拾**。

## 14. 与 OwnerReference 的关系

| 机制 | 作用 |
|------|------|
| **OwnerReference** | 父对象删除时自动删子对象（K8s GC 来做） |
| **Finalizer** | 拦截"删除"动作，让控制器有机会做清理 |

最佳实践组合：

- 子资源（Deployment、Service、ConfigMap）用 OwnerReference 让 K8s 自动级联清
- 外部资源（云盘、外部 API、跨集群对象）用 Finalizer 自己清

## 15. 一句话总结

> Leader Election 解决"谁来干"，Finalizer 解决"删之前清干净再走"。
> 二者都要求 reconcile **幂等**——失败重试、切主重做、清理多调一次都不会出错。

## 下一步

讲完 Lifecycle，下一篇专门讲 status 怎么写：

- [Status 子资源与 Conditions 设计](./06-status-and-conditions.md)
