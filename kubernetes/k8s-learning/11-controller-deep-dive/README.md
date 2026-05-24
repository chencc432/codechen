# 🎛️ Controller 深度专题

> 这个专题在 [client-go 入门](../05-client-go/README.md) 和 [自定义资源专题](../07-custom-resources/README.md) 之上更深入一层。
> 目标：让你不仅会"写一个 Reconcile"，还能讲清楚 Informer 内部数据结构、Workqueue 限速、Leader Election、Finalizer 时序、controller-runtime 与原生 client-go 的取舍，以及生产环境的常见坑。

## 这个专题适合谁

- 写过简单 Operator / Controller，但对内部机制还是雾的
- 用 controller-runtime 写 Reconcile，但出了问题不知道是不是缓存/事件丢失
- 想能看懂 client-go / kubernetes 内置控制器源码
- 做平台扩展、做 Operator、做调度器、做自定义控制平面

## 你会学到什么

| 层面 | 关键问题 |
|------|----------|
| **模式** | 控制器为什么是声明式？为什么必须幂等？为什么不能"靠记忆"？ |
| **机制** | Informer 是什么？Reflector / DeltaFIFO / Indexer / SharedInformer 各自做什么？ |
| **可靠** | Workqueue 怎么去重、限速、重试？Leader Election 是怎么选的？ |
| **设计** | Reconcile 怎么写才不踩坑？Finalizer 怎么用？status 怎么设计？ |
| **工程** | controller-runtime vs 原生 client-go 怎么选？怎么测、怎么 debug、怎么发布？ |

## 学习路线

读这个专题前推荐你已经看过：

- [client-go 入门](../05-client-go/01-introduction.md)
- [Informer 机制详解](../05-client-go/04-informer.md)
- [自定义控制器实战项目](../05-client-go/05-controller-demo.md)
- [控制器、Reconcile 与 Operator 模式](../07-custom-resources/02-controller-and-operator.md)

然后按下面顺序读：

1. [控制器模式与声明式 API 的本质](./01-controller-pattern.md)
2. [Informer 内部机制详解](./02-informer-internals.md)
3. [WorkQueue 与限速器](./03-workqueue.md)
4. [Reconcile 函数的设计要点](./04-reconcile-design.md)
5. [Leader Election 与 Finalizer](./05-leader-election-and-finalizers.md)
6. [Status 子资源与 Conditions 设计](./06-status-and-conditions.md)
7. [controller-runtime 与原生 client-go 对比](./07-controller-runtime-vs-clientgo.md)
8. [最佳实践与常见坑](./08-best-practices-and-pitfalls.md)

## 文档列表

| 序号 | 标题 | 关键词 |
|------|------|--------|
| 01 | [控制器模式](./01-controller-pattern.md) | 声明式、幂等、Level-Triggered |
| 02 | [Informer 内部](./02-informer-internals.md) | Reflector、DeltaFIFO、Indexer、SharedInformer |
| 03 | [WorkQueue](./03-workqueue.md) | RateLimiter、指数退避、去重 |
| 04 | [Reconcile 设计](./04-reconcile-design.md) | 幂等、错误处理、子资源、Result |
| 05 | [Leader Election + Finalizer](./05-leader-election-and-finalizers.md) | Lease、续约、删除时序 |
| 06 | [Status + Conditions](./06-status-and-conditions.md) | status 子资源、Condition 标准 |
| 07 | [controller-runtime vs client-go](./07-controller-runtime-vs-clientgo.md) | manager、cache、Builder、何时用哪个 |
| 08 | [最佳实践与坑](./08-best-practices-and-pitfalls.md) | 缓存陈旧、热循环、内存泄漏、版本冲突 |

## 阅读建议

控制器是一个"看代码会快很多"的话题。读每篇时建议：

1. 文档里给的伪代码 / 真实片段，至少照着 IDE 里查一遍 client-go 源码
2. 自己写过的 Operator，回头对照本专题的概念再看一遍 Reconcile
3. 重要的概念尝试用一句话给同事讲清楚——讲不清楚就还没真懂

## 配套资源

- Kubernetes 源码：[github.com/kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)
- client-go 源码：[github.com/kubernetes/client-go](https://github.com/kubernetes/client-go)
- controller-runtime：[github.com/kubernetes-sigs/controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- Kubebuilder Book：[book.kubebuilder.io](https://book.kubebuilder.io/)
