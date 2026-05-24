# ⚖️ controller-runtime 与原生 client-go 对比

> 写控制器有两条主流路线：直接用 client-go 自己拼装，或者用 controller-runtime（Kubebuilder 默认栈）。这一篇拆开两者的区别，讲清楚它们解决的同一个问题、不同的抽象、以及怎么选。

## 1. 它们是什么

| 路线 | 内容 |
|------|------|
| **client-go** | Kubernetes 官方提供的 Go 客户端库。包含 REST client、Informer、Workqueue、leader election 等基础原语 |
| **controller-runtime** | Kubernetes SIG 维护的高层框架，**底层就是 client-go**。封装了 Manager、Cache、Client、Builder、Reconciler 接口 |
| **Kubebuilder / operator-sdk** | 用 controller-runtime 的脚手架工具，帮你生成项目骨架 |

简化关系：

```text
        Kubebuilder（脚手架）
              ▼
        controller-runtime（框架）
              ▼
           client-go（基础库）
              ▼
            Kubernetes API
```

## 2. 同一个东西的两种写法

### 2.1 client-go 风格

```go
factory := informers.NewSharedInformerFactory(client, 30*time.Minute)
podInformer := factory.Core().V1().Pods()

queue := workqueue.NewNamedRateLimitingQueue(...)

podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc:    func(obj interface{}) { ... enqueue ... },
    UpdateFunc: func(o, n interface{}) { ... enqueue ... },
    DeleteFunc: func(obj interface{}) { ... enqueue ... },
})

factory.Start(stopCh)
factory.WaitForCacheSync(stopCh)

for i := 0; i < workers; i++ {
    go wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *Controller) reconcile(key string) error {
    ns, name, _ := cache.SplitMetaNamespaceKey(key)
    pod, err := c.podLister.Pods(ns).Get(name)
    ...
}
```

### 2.2 controller-runtime 风格

```go
mgr, _ := ctrl.NewManager(cfg, ctrl.Options{
    Scheme: scheme,
    LeaderElection: true,
    LeaderElectionID: "myapp.controller",
})

if err := ctrl.NewControllerManagedBy(mgr).
    For(&myv1.MyApp{}).
    Owns(&appsv1.Deployment{}).
    Complete(&MyAppReconciler{Client: mgr.GetClient()}); err != nil {
    panic(err)
}

mgr.Start(ctrl.SetupSignalHandler())

func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyApp{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    // ... 业务 ...
    return ctrl.Result{}, nil
}
```

同样一个控制器，controller-runtime 大约**少 60% 代码**。

## 3. controller-runtime 帮你做了哪些事

| 关注点 | client-go 你要自己写 | controller-runtime 帮你做了 |
|--------|---------------------|---------------------------|
| Informer 的创建 / 注册 / sync 等待 | 手动 | Manager 内部统一管 |
| Workqueue + RateLimiter | 手动 | Builder 默认配好 |
| 事件回调 → 入队 | 手动写 EventHandler | Builder 自动写 |
| Owns 对象（子资源变化触发父对象 reconcile） | 手动维护 OwnerReference + 索引 | `Owns()` 一行 |
| Watches 任意对象映射成 reconcile 请求 | 自己写 mapper | `Watches()` + handler.EnqueueRequestsFromMapFunc |
| Leader Election | 引入 leaderelection 包 | `LeaderElection: true` |
| 健康检查 / 指标 / pprof | 自己起 HTTP | Manager 内置 |
| 多控制器统一 lifecycle | 自己 channel 协调 | Manager.Start 一把 |
| Predicate（事件过滤） | EventHandler 内 if/else | `WithEventFilter` |
| Cache（带索引、按 namespace 过滤） | 拼 Indexer | `mgr.GetCache()` |

简单说：controller-runtime 把"老套路"全做了默认实现。**Kubebuilder 进一步把目录结构、CRD 生成、RBAC 生成都自动化**。

## 4. 哪些场景用 client-go 更合适

虽然 controller-runtime 是大多数情况的首选，但下面场景还会直接用 client-go：

1. **写非常薄的工具** —— 比如一个小脚本，只 watch 一种资源做一件事，不需要 manager 的复杂度
2. **写 K8s 内置控制器** —— `kube-controller-manager` 里所有控制器都是直接 client-go（保持核心库依赖最少）
3. **超严格的性能/内存需求** —— 你想自定义 Informer 的 transform 函数，去除大字段以省内存（cr 也支持，但不如直接 client-go 直观）
4. **需要直接操作 Reflector/DeltaFIFO** —— 比较罕见，但写一些诊断工具会用
5. **学习目的** —— 看懂 client-go 才能看懂 controller-runtime 内部、内置控制器源码

## 5. 哪些场景用 controller-runtime 更合适

99% 的"业务 Operator / 自定义控制器"场景：

- 写 CRD + Operator
- 多种资源协同（Owns、Watches）
- 需要 Webhook（cr 自带 webhook server）
- 需要 leader election + metrics + healthz（cr 一把搞定）
- 需要单测（cr 配套 envtest 体验最佳）

> 一句话：业务 Operator 走 controller-runtime；底层组件 / kube-* 系列走 client-go。

## 6. controller-runtime 几个常用能力点

### 6.1 For / Owns / Watches

```go
ctrl.NewControllerManagedBy(mgr).
    For(&myv1.MyApp{}).            // 主资源
    Owns(&appsv1.Deployment{}).    // 子资源（自动按 ownerRef 找父）
    Owns(&corev1.Service{}).
    Watches(                        // 自定义 watch 映射
        &corev1.ConfigMap{},
        handler.EnqueueRequestsFromMapFunc(r.findMyAppForConfigMap),
    ).
    Complete(r)
```

`Owns` 的本质：自动给子资源加 OwnerReference 索引，子资源变化时找到 owner 入队。

### 6.2 Predicate

只在 spec 变化时触发，避免 status 变化触发自己：

```go
ctrl.NewControllerManagedBy(mgr).
    For(&myv1.MyApp{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
    Complete(r)
```

### 6.3 Manager 提供的统一服务

```go
mgr.GetClient()        // 带 cache 的读 + 不带 cache 的写
mgr.GetAPIReader()     // 不走 cache，直接打 API server（特殊场景）
mgr.GetCache()         // 直接拿 cache
mgr.GetEventRecorderFor("my-controller")
mgr.AddHealthzCheck(...)
mgr.AddReadyzCheck(...)
mgr.GetWebhookServer()
```

### 6.4 Cache 的两个细节

- **默认按 controller 维度共享同一个 cache**：所有控制器读到的 client.Get 走的是同一份 informer 缓存
- **可以限制 cache 范围**（节省内存）：`Options.Cache.DefaultNamespaces` / `ByObject` 配置

```go
ctrl.Options{
    Cache: cache.Options{
        DefaultNamespaces: map[string]cache.Config{
            "my-namespace": {},
        },
    },
}
```

### 6.5 Webhook 一把梭

ValidatingWebhook / MutatingWebhook / Conversion webhook，cr 都自带 server + 路由。
Kubebuilder 用 `kubebuilder create webhook` 命令一键生成骨架。

## 7. 注意 controller-runtime 的"看不见的行为"

cr 的"自动化"是把双刃剑：

| 行为 | 注意点 |
|------|--------|
| `client.Get` 默认走 cache | 写完之后立刻 Get 可能拿到旧值；要立即一致用 `APIReader` |
| `Owns` 默认 Predicate | 子资源 status 变化也会触发父；如果不希望，用自定义 predicate |
| Reconcile 返回 err 默认走限速器 | 看不到的退避；调试时盯着 metrics |
| 默认所有 namespace 的 cache | 单 namespace 控制器记得限制，否则内存占用高 |
| Manager 是 singleton 模式 | 多控制器共享 cache、queue 限速器互不干扰 |

## 8. 学习路径建议

如果你时间有限，下面顺序最高效：

1. 先用 Kubebuilder 跑通一个完整 CRD + Reconciler（黑盒）
2. 通读 controller-runtime 主仓的 Examples，看 For/Owns/Watches 用法
3. 当遇到"为什么我的 reconcile 没被触发"或"为什么 cache 里没有这个对象"时，回头读 client-go 的 Informer / DeltaFIFO（前一篇）
4. 看一个内置控制器源码（推荐 ReplicaSet，不复杂），用 client-go 风格写过一次
5. 之后日常就用 controller-runtime，但出问题时能下沉到 client-go 层去理解

## 9. 一句话对比

| 维度 | client-go | controller-runtime |
|------|-----------|---------------------|
| 抽象层级 | 低，灵活 | 高，规范 |
| 代码量 | 多 | 少 |
| 学习曲线 | 陡 | 缓 |
| 适合 | 内置组件、研究 | 业务 Operator |
| 出问题排查 | 你看得到所有底层细节 | 需要懂 cache/manager 抽象 |

> client-go 让你"什么都自己做"，controller-runtime 让你"做该做的事"。生产首选 controller-runtime，但理解 client-go 是基本功。

## 下一步

最后一篇是上面所有内容的"避坑大全"：

- [最佳实践与常见坑](./08-best-practices-and-pitfalls.md)
