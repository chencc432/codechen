# 🔁 WorkQueue 与限速器

> Informer 把事件转成 key 推到 WorkQueue，Worker 从队列里取 key 调 Reconcile。WorkQueue 是控制器"重试 + 去重 + 限速"的关键。

## 1. 三种 WorkQueue

client-go 提供三种递进的 WorkQueue 实现：

| 类型 | 接口 | 关键能力 |
|------|------|----------|
| `Interface` | `Add / Get / Done` | 基础队列，去重 |
| `DelayingInterface` | + `AddAfter` | 延迟入队 |
| `RateLimitingInterface` | + `AddRateLimited / NumRequeues / Forget` | 限速 + 退避 |

控制器用得最多的是 `RateLimitingInterface`：

```go
queue := workqueue.NewNamedRateLimitingQueue(
    workqueue.DefaultControllerRateLimiter(),
    "my-controller",
)
```

## 2. 三个核心特性

### 2.1 去重

加同一个 key 多次，最终只在队列里有一条：

```go
queue.Add("ns/name")
queue.Add("ns/name")     // 第二次进队会被吞掉，只保留一条
```

实现：内部用 `dirty set` + `processing set` 两个集合：

```text
                  Add(k)
                    │
                    ▼
         ┌────── dirty ─────┐
         │  k 已在 → 丢弃   │
         │  k 不在 → 加入   │
         └──────────────────┘

         Get() 拉出一个：
              移到 processing
              dirty 移除

         Done(k) 时：
              processing 移除
              如果在 processing 期间又 Add 过：再次入 dirty
```

这个机制保证：

- 同一 key 永远只有一个 worker 在处理它（**单一处理者**）
- 如果在处理期间又来事件，会自动安排下一次处理，不会丢

### 2.2 延迟入队

`AddAfter(key, duration)` 把 key 排到一个时间堆里，到点再入队。用法：

```go
queue.AddAfter("ns/name", 30*time.Second)
```

典型场景：

- 资源还在创建中，30 秒后再看
- 外部系统就绪检查，10 秒后重试

### 2.3 限速 + 退避

`AddRateLimited(key)` 不立即入队，而是按 RateLimiter 的策略决定 delay：

```go
queue.AddRateLimited("ns/name")
```

策略来自 `workqueue.DefaultControllerRateLimiter()`，这是几个 Limiter 的组合（取最大值）：

| Limiter | 行为 |
|---------|------|
| `ItemExponentialFailureRateLimiter` | **每个 key** 单独的指数退避，初始 5ms，每次 ×2，最大 1000s |
| `BucketRateLimiter`（token bucket） | **全局**令牌桶：默认 10 QPS，突发 100 |

两者取 **较大的延迟**作为入队延迟。结果：

- 单 key 反复失败 → 退避越来越长
- 全局并发瞬间起飞 → 被令牌桶削峰

### 2.4 NumRequeues 与 Forget

```go
err := reconcile(key)
if err != nil {
    if queue.NumRequeues(key) < 5 {
        queue.AddRateLimited(key)    // 限速重试
    } else {
        queue.Forget(key)            // 放弃，重置计数
        utilruntime.HandleError(err) // 上报指标
    }
    queue.Done(key)
    return
}
queue.Forget(key)    // 成功后忘记重试历史
queue.Done(key)
```

`Forget(key)` 是**重置该 key 的失败计数**——下一次失败重新从初始延迟开始退避。**不调 Forget 不会 panic，但会让你的退避永远从上次的位置接着算**。

## 3. 标准 worker 循环（背下来）

几乎所有 client-go 控制器的 worker 都是这个套路：

```go
func (c *Controller) runWorker(ctx context.Context) {
    for c.processNextItem(ctx) {
    }
}

func (c *Controller) processNextItem(ctx context.Context) bool {
    key, quit := c.queue.Get()
    if quit {
        return false
    }
    defer c.queue.Done(key)

    err := c.reconcile(ctx, key.(string))
    c.handleErr(err, key)
    return true
}

func (c *Controller) handleErr(err error, key interface{}) {
    if err == nil {
        c.queue.Forget(key)
        return
    }
    if c.queue.NumRequeues(key) < maxRetries {
        klog.Warningf("retry %s: %v", key, err)
        c.queue.AddRateLimited(key)
        return
    }
    klog.Errorf("give up %s: %v", key, err)
    c.queue.Forget(key)
    utilruntime.HandleError(err)
}
```

四件事：`Get → defer Done → reconcile → handleErr`。把这段背下来你写 80% 的控制器都不会跑偏。

## 4. controller-runtime 的封装

controller-runtime 的 `Reconciler.Reconcile` 返回 `(ctrl.Result, error)`：

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

| 返回值 | 等价于 |
|--------|--------|
| `Result{}, nil` | `Forget`，正常完成 |
| `Result{}, err` | `AddRateLimited`，按限速器退避 |
| `Result{Requeue: true}, nil` | 立即重新入队（不退避） |
| `Result{RequeueAfter: 30s}, nil` | 30 秒后再入队（不退避） |
| `Result{Requeue: true}, err` | `err != nil` 优先生效，按限速器退避 |

> 注意：`Requeue: true` 时仍然会被 `BucketRateLimiter` 削峰，不会无限狂转。

## 5. 限速器源码地图

| Limiter | 文件 |
|---------|------|
| ItemExponentialFailureRateLimiter | `util/workqueue/default_rate_limiters.go` |
| BucketRateLimiter | 同上，封装 `golang.org/x/time/rate.Limiter` |
| MaxOfRateLimiter | 同上，对多个 Limiter 取最大值 |

如果你需要自定义限速策略（比如 SLA 任务上限频率、按 namespace 限速），实现 `RateLimiter` 接口然后传给 `NewNamedRateLimitingQueue` 即可。

## 6. 为什么 Workqueue 是单消费者也能高吞吐

控制器经常一个进程开 N 个 worker（默认 2 ~ 10），并发处理 reconcile。但同一 key 永远只在一个 worker 上跑——其它 worker 拿不到那个 key。

这意味着：

- 不同对象的 reconcile **天然并行**
- 同一对象的 reconcile **天然串行**

这正是声明式控制器期待的"对单对象幂等顺序处理，对多对象并行扩展"。

## 7. 工程上常见的几个问题

### 7.1 reconcile 里又往队列加自己 → 热循环

```go
// 错误：每次失败都立刻无延迟再加
queue.Add(key)
return err
```

应该用 `AddRateLimited` 让退避起作用，否则一个错误的 key 会让 worker 100% CPU 在转圈。

### 7.2 处理 panic 后忘了 Done → 队列卡住

`Done` 必须在每次 `Get` 之后被调一次（成功失败都调），用 `defer queue.Done(key)`。如果忘了，那个 key 会留在 processing，下次再 Add 也只会进 dirty 不进新一轮处理 → 看起来"reconcile 卡住了"。

### 7.3 把对象本身入队，而不是 key

队列的元素必须是**字符串 key**（`namespace/name`）。原因：

- 入队后到取出之间对象可能变了，应当从 Indexer 重新 Get
- 把整个对象入队会浪费内存

入队规范：

```go
key, err := cache.MetaNamespaceKeyFunc(obj)
queue.Add(key)
```

### 7.4 跨 namespace key 解析

```go
ns, name, err := cache.SplitMetaNamespaceKey(key.(string))
```

注意空 namespace（cluster scope 资源），返回的 ns 是空字符串。

### 7.5 Forget 时机

成功了才 Forget，失败的 retry 路径也应该在"放弃重试"那个分支 Forget——否则该 key 的退避计数永远在涨，下次哪怕成功后再失败的初始 delay 也会很大。

## 8. 一段完整可读的最小控制器骨架

```go
type Controller struct {
    queue    workqueue.RateLimitingInterface
    informer cache.SharedIndexInformer
    lister   corelisters.PodLister
}

func (c *Controller) Run(ctx context.Context, workers int) {
    defer utilruntime.HandleCrash()
    defer c.queue.ShutDown()

    if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
        return
    }
    for i := 0; i < workers; i++ {
        go wait.UntilWithContext(ctx, c.runWorker, time.Second)
    }
    <-ctx.Done()
}

func (c *Controller) runWorker(ctx context.Context) {
    for c.processNextItem(ctx) {}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
    key, quit := c.queue.Get()
    if quit { return false }
    defer c.queue.Done(key)

    if err := c.reconcile(ctx, key.(string)); err != nil {
        if c.queue.NumRequeues(key) < 5 {
            c.queue.AddRateLimited(key); return true
        }
        utilruntime.HandleError(err)
    }
    c.queue.Forget(key)
    return true
}
```

这 30 行就是 client-go 风格控制器的全部"事件 → reconcile"骨架。

## 下一步

下一篇专门讲 reconcile 函数本身怎么写：

- [Reconcile 函数的设计要点](./04-reconcile-design.md)
