# 🔬 Informer 内部机制详解

> 这一篇拆开 client-go Informer 的内部组件。如果你能讲清楚 Reflector / DeltaFIFO / Indexer / SharedInformer 之间的关系，你对控制器的理解就上了一个台阶。

## 1. 大图先过一遍

```text
              ┌──────────────────────────────────────┐
              │                API Server            │
              └────────────┬─────────────────────────┘
                           │ List + Watch
                           ▼
                ┌────────────────────┐
                │     Reflector      │   把 API server 的事件搬到本地
                └─────────┬──────────┘
                          │ Add/Update/Delete to queue
                          ▼
                ┌────────────────────┐
                │     DeltaFIFO      │   FIFO 队列 + 同 key 增量合并
                └─────────┬──────────┘
                          │ Pop
                          ▼
                ┌────────────────────┐
                │   Process loop     │   informer 的主循环
                └─────────┬──────────┘
                          │ 更新 Indexer + 触发回调
                          ▼
       ┌──────────────────┴──────────────────┐
       │                                     │
       ▼                                     ▼
┌────────────┐                     ┌─────────────────┐
│  Indexer   │                     │  EventHandlers  │
│ (本地缓存)  │  ←── Get/List ────  │ (你的回调)       │
└────────────┘                     └─────────────────┘
                                              │
                                              ▼
                                       ┌──────────────┐
                                       │  Workqueue   │
                                       └──────┬───────┘
                                              │
                                              ▼
                                       ┌──────────────┐
                                       │ Reconcile()  │
                                       └──────────────┘
```

记住这条数据流：**API Server → Reflector → DeltaFIFO → Indexer + Handlers → Workqueue → Reconcile**。

## 2. Reflector

### 2.1 它在干什么

`Reflector` 把"任意 K8s 资源"的 List + Watch 抽象成一段循环：

```text
forever:
    items, rv = ListAndWatch()
        - List 一次（首次或断线后）
        - 拿到 resourceVersion
        - 用 rv 启动 Watch
        - 每个事件转换成 Delta，丢到 DeltaFIFO
    if 出错：
        Sleep(backoff)
        continue
```

### 2.2 怎么处理"事件丢失"

API server 的 watch 是有"窗口"的，如果客户端断线太久，rv 已经过期，watch 会返回 `410 Gone`。这时 Reflector：

1. 整个 List 一次（重新拿到一致快照）
2. 用 List 出来的对象**重置整个 DeltaFIFO**（发 `Replaced` 事件）
3. 拿新 rv，再起一次 Watch

这就是为什么 **Informer 启动时永远先 List，再 Watch**——保证启动那一刻状态一致。

### 2.3 重要属性

- 一个 Reflector **只 reflect 一种资源**（Pod / Deployment / 自定义 CR 都各自一个）
- 它不知道下游是谁，只往 DeltaFIFO 塞数据
- ListWatch 用的是 `meta.RESTClient`，所以走 generic informer 时也能 reflect 任何 GVR

## 3. DeltaFIFO

### 3.1 它和普通队列的区别

普通 FIFO 是 `[event1, event2, event3, ...]`。**DeltaFIFO 的元素是按 key 聚合的增量列表**：

```text
key="pod-a" → [Add, Update, Update]
key="pod-b" → [Add]
```

每个 key 对应一个 `Deltas` 数组，依次记录这个 key 的事件序列。Pop 出来的是**整个 key 的 deltas**，不是单个事件。

### 3.2 为什么要这么设计

直接打平队列会有几个问题：

- 同一个对象短时间内来 5 次 Update，下游会被打 5 次
- 在 informer 处理慢的时候，事件越积越多浪费

DeltaFIFO 在入队时会做"压缩"：

- 同 key 的 Update + Update → 合并为一个 Update（只保留最新对象）
- 同 key 的 Add 后跟 Delete → 合并为 Sync 序列处理（看实现）

但**它保证不丢"对象删除"这种关键信号**——Delete 不会被吞掉。

### 3.3 Sync 与 Replace 两个特殊事件类型

| 类型 | 来源 | 含义 |
|------|------|------|
| `Added` | Watch | 新对象出现 |
| `Updated` | Watch | 对象变化 |
| `Deleted` | Watch | 对象被删 |
| `Replaced` | List | 一次完整 list 后的对象（v1.21+ 用 Replaced 替代 Sync） |
| `Sync` | 周期性 resync | 即便没变化，也定时把缓存里的对象再喂给 handler |

**Resync** 是 Informer 的一个安全网：你可以传 `resyncPeriod=30m`，每 30 分钟 informer 就会把所有对象当成"事件"再触发一次 handler。这能让你的 reconcile 周期性地"再算一遍"，对账兜底。

> 注意：Resync **不是从 API server 重新 List**，而是从本地 Indexer 把所有对象再喂一遍。所以 resync 不会增加 API server 压力。

## 4. Indexer / Store

### 4.1 它是 Informer 的本地缓存

Indexer 是个线程安全的 KV 存储，加了"二级索引"。Informer 处理 Delta 的时候：

```text
对每个 delta：
  - Add/Update/Sync/Replaced → indexer.Add(obj)
  - Delete                    → indexer.Delete(obj)
然后再触发 handler 的 OnAdd/OnUpdate/OnDelete
```

所以 **handler 触发的时刻，Indexer 已经更新了**——你在 handler 里 Get 拿到的就是最新值。

### 4.2 二级索引

通过定义 `IndexFunc`，Indexer 能让你按非主键字段快速查询。例子：

```go
indexers := cache.Indexers{
    "byNode": func(obj interface{}) ([]string, error) {
        pod := obj.(*corev1.Pod)
        return []string{pod.Spec.NodeName}, nil
    },
}
```

之后可以这样查：`indexer.ByIndex("byNode", "node-1")`，O(1) 拿到这个节点上的所有 Pod。

控制器里常用的索引：按 OwnerReference、按 Label、按某个字段值。

### 4.3 Lister

Lister 是对 Indexer 的语义层封装：

```go
podLister := factory.Core().V1().Pods().Lister()
pods, err := podLister.Pods("ns").List(labels.Everything())
pod, err := podLister.Pods("ns").Get("name")
```

**永远从 Lister 读**，不要直接打 API server——这是控制器性能与一致性的基础。

## 5. SharedInformer / SharedInformerFactory

### 5.1 为什么要"共享"

如果一个进程里 5 个控制器都要 watch Pod，朴素做法是每个控制器一个 Reflector + Indexer，5 份。这浪费 API server 的 watch 连接、浪费内存。

**SharedInformer**：一个 Informer，多个 listener。

```text
                 ┌──────────────────┐
                 │ SharedInformer   │
                 │ (one Reflector,  │
                 │  one Indexer)    │
                 └────────┬─────────┘
                          │ broadcast
              ┌──────┬────┼────┬──────┐
              ▼      ▼    ▼    ▼      ▼
           Ctrl1  Ctrl2  Ctrl3  Ctrl4  Lister
           Handler Handler Handler Handler
```

### 5.2 SharedInformerFactory

`SharedInformerFactory` 是 SharedInformer 的工厂，按 `(GVR, namespace, resyncPeriod)` 缓存复用：

```go
factory := informers.NewSharedInformerFactory(client, 30*time.Minute)

podInformer := factory.Core().V1().Pods()
podLister   := podInformer.Lister()

podInformer.Informer().AddEventHandler(...)

factory.Start(stopCh)
factory.WaitForCacheSync(stopCh)
```

特点：

- 同一个 GVR 多次取，拿到的是同一个 Informer
- `Start` 启动所有已注册的 Informer
- `WaitForCacheSync` **必须调**——保证启动后 List 已完成，缓存可用

> 启动时不等 cache sync 就开始 reconcile，会导致 reconcile 里 Lister.Get 拿不到对象（明明集群有），是新手最常见的坑之一。

## 6. 自定义资源（CR）的 Informer

CRD 没有手写 client，怎么办？两种方式：

| 方式 | 何时用 |
|------|--------|
| **Typed Informer**：用 `code-generator` / `kubebuilder` 生成自己资源的 SharedInformerFactory | 自己写 Operator 时（推荐） |
| **Dynamic Informer**：用 `dynamicinformer.NewDynamicSharedInformerFactory` 直接传 GVR | 写通用工具、跨多种资源时 |

controller-runtime 的 `Manager` 内部其实就是 SharedInformerFactory + cache.Cache 的封装，但屏蔽了大部分细节，详见第 7 篇。

## 7. 一个易被忽略的细节：DeletedFinalStateUnknown

如果 watch 短暂断开期间一个对象被删了，重新 List 后这个对象不存在。Reflector 会把它包装成：

```go
cache.DeletedFinalStateUnknown{
    Key: "ns/name",
    Obj: <最后一次见到的对象>,
}
```

写 OnDelete 处理时**必须处理这种类型**：

```go
func onDelete(obj interface{}) {
    key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
    // 不要 obj.(*corev1.Pod) 直接断言，会 panic
    queue.Add(key)
}
```

`DeletionHandlingMetaNamespaceKeyFunc` 自动从 `DeletedFinalStateUnknown` 里取 key。

## 8. 真实代码地图（client-go 源码）

如果你想 IDE 跳过去看：

| 概念 | 文件 |
|------|------|
| Reflector | `tools/cache/reflector.go` |
| DeltaFIFO | `tools/cache/delta_fifo.go` |
| Indexer | `tools/cache/store.go`、`thread_safe_store.go` |
| SharedInformer | `tools/cache/shared_informer.go` |
| SharedInformerFactory | `informers/factory.go` |
| Lister | `listers/<group>/<version>/<resource>.go`（自动生成） |

## 9. 一句话记忆

> Reflector 搬数据，DeltaFIFO 攒数据，Indexer 存数据，SharedInformer 管分发，Lister 给你查——你只需要在回调里把 key 入队。

## 下一步

事件入队后，下一站是 Workqueue：

- [WorkQueue 与限速器](./03-workqueue.md)
