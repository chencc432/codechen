# 🧭 控制器模式与声明式 API 的本质

> 想真正写好控制器，必须先把"为什么 Kubernetes 这样设计"想清楚。这一篇不写代码，专门讲 5 个底层概念。

## 1. 命令式 vs 声明式

| 模式 | 用户表达 | 系统职责 |
|------|----------|----------|
| **命令式** | "现在帮我做这件事" | 立刻执行 |
| **声明式** | "我希望最终是这样" | 持续把现实推到希望状态 |

`kubectl run` 是命令式入口，但底层一旦写到 etcd，**Kubernetes 全部都是声明式**：

```text
用户：spec.replicas = 3
系统：当前只有 2 个 → 自动再创一个 → 3 → OK
       当前有 4 个 → 自动删一个 → 3 → OK
       任何时候掉到 < 3 → 自动补
```

> 控制器存在的全部意义，就是让"现实状态（status）持续追上期望状态（spec）"。

## 2. 控制循环的三个不变量

任何一个控制器，不论是 Deployment Controller、CronJob Controller、还是你自己写的 MyAppController，都遵守 3 条不变量：

### 不变量 1：声明式 API

**用户写 spec，控制器写 status，谁都不写对方。**

如果你写的控制器**回头修改了 spec**，那它就不是控制器，是另一个用户。这会导致：

- 用户改一次 spec，控制器又改回来
- 多个控制器互相抢着改 spec → 死循环 / flapping

### 不变量 2：Level-Triggered（电平触发），不是 Edge-Triggered（边沿触发）

**控制器不靠"事件"决策，靠"当前状态"决策。**

| Edge-Triggered（错误的写法） | Level-Triggered（Kubernetes 风格） |
|----------------------------|------------------------------------|
| "我看到一个 Created 事件，那我就建 Pod" | "对象当前期望 3 副本，现实只有 2 个，那我就再建 1 个" |
| 漏一个事件就坏 | 漏多少事件都没关系，下次仍然能算对 |

这就是为什么 Reconcile **总是从最新对象重新算一遍**，不"记忆"上次干了什么。

### 不变量 3：幂等

**对同一对象 reconcile 100 次，效果应当和 reconcile 1 次相同。**

幂等不是"控制器永远不写资源"，而是：

- 创建前先 Get 看是否已存在
- 更新前先看是否需要更新（diff）
- 删除前判断是否已经被删

如果你写的 Reconcile 第一次 Create 一个 ConfigMap，第二次 Reconcile 又 Create 一遍报 `AlreadyExists` 然后整个失败，那它**不是幂等的**。

## 3. 为什么是 Watch + List，而不是 Poll

最朴素的设计是：每隔 30s 把所有 Pod list 一遍。这种方式：

- 实时性差
- 流量巨大（list 全量）
- API server 顶不住

Kubernetes 的解法是 **List + Watch + 本地缓存**：

```text
启动时:    List → 把全量对象拉下来塞缓存
之后:      Watch → 增量收事件 → 更新缓存
本地反应:  从缓存读，不打 API server
```

这就是 Informer 的工作原理（下一篇详细讲）。从控制器角度看：

- 控制器从**本地缓存**读对象，不直接 list/get API server
- API server 只承担 watch 通道压力
- 大量控制器都共享一份本地缓存（SharedInformer）

## 4. Spec / Status / Observation 三件事

理解控制器最简洁的心智模型：

```text
   ┌──── Spec   ────┐  期望（用户意图）
   │                │
   ▼                ▲
Reconcile()  ←─────────  Observation（看到的真实世界）
   │                │
   ▼                │
   └──── Status ────┘  现实（控制器报告）
```

- **Spec**：用户写的、控制器只读
- **Observation**：控制器看到的真实世界（实际 Pod 数、外部资源状态）
- **Status**：控制器写回去的"我现在能告诉你什么"

写 Reconcile 的时候，永远在做：

```text
diff = 比较 Spec 和 Observation
if diff:
    采取动作（创建/更新/删除底层资源）
update Status，反映 Observation
```

## 5. 为什么控制器是"乐观并发"的

K8s 所有对象有 `resourceVersion`。Update 时如果版本号不匹配，API server 会返回 `Conflict`：

```go
err := client.Update(ctx, pod)
if errors.IsConflict(err) {
    // 别人改过了，重新 get → 重算 → 再 update
}
```

控制器要做的是：

- 拿到最新对象
- 算出新值
- Update；冲突就重试

这种"乐观锁"模式是分布式系统里常见做法，比悲观锁简单且高吞吐。

> 当你看到 controller-runtime 自动重试 Conflict、看到自己 reconcile 偶尔报 "the object has been modified; please apply your changes to the latest version"，背后就是这个机制。

## 6. 一个控制器的完整生命周期（顶层视角）

```text
启动
 ├─ 注册 Informer（指定要 watch 哪些资源）
 ├─ 注册事件回调：把对象 key 推到 workqueue
 ├─ 启动 leader election（HA）
 ├─ 启动 N 个 worker goroutine：从 workqueue 取 key → Reconcile
 └─ 等信号

事件流转
 ├─ API Server 推送 watch 事件
 ├─ Informer 更新本地缓存 + 触发回调
 ├─ 回调把对象 key 入队
 └─ Worker 取 key → 从 indexer 读最新对象 → Reconcile

退出
 ├─ 优雅停止 informer / workqueue
 └─ 释放 leader lease
```

后面几篇会一块一块拆这个图。

## 7. 一些误解的修正

| 误解 | 正确认识 |
|------|----------|
| "控制器是一个事件处理器" | 不是。它是一个**状态收敛器**，事件只是"该看一下这个对象了"的提醒 |
| "Reconcile 失败了控制器就停了" | 不会。失败会被 workqueue 重新入队（带退避） |
| "Reconcile 必须很快" | 应当尽量快，但更重要的是**幂等、可重入、可重试** |
| "控制器要保存上次状态" | 不要。所有真相都在 spec/status，不要在内存里缓存"业务记忆" |
| "Watch 一定不丢事件" | 长时间断开会丢；所以**重启时要 List 一次重建状态**，这就是 Informer 干的 |

## 8. 心智模型一句话总结

> 控制器 = "永远拿最新的 spec 和现实，做幂等地推一步收敛"。
> 它不依赖事件本身，只依赖事件触发的"重新看一眼"。

带着这个模型，你读后面所有篇章都会顺畅很多。

## 下一步

接下来开始拆"控制器是怎么拿到最新对象的"——Informer：

- [Informer 内部机制详解](./02-informer-internals.md)
