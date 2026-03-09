# 🔄 Informer 机制详解

## 什么是 Informer？

**Informer 是 client-go 中最核心的组件之一**，它是一个智能的"资源监听器"，能够：

1. **监控 Kubernetes 资源的变化**（Pod 创建了、Service 删除了等）
2. **在本地维护一份资源的缓存副本**（不用每次都去问 API Server）
3. **当资源发生变化时，通知你的代码做出响应**

### 生活化比喻 🏠

想象你是一个快递站的管理员：

- **没有 Informer 的情况**：每次想知道有没有新快递，你都要打电话问总部。一天打100次电话，总部烦死了，你也累死了。

- **有 Informer 的情况**：总部给你装了一个实时显示屏（本地缓存），快递状态实时同步。有新快递到了，显示屏自动更新并响铃通知你（事件回调）。你想查快递？直接看屏幕，不用打电话！

---

## 为什么需要 Informer？

假设你要写一个程序，监控集群中所有 Pod 的状态变化。

### ❌ 方案一：定时轮询 API（最笨的方法）

```go
for {
    pods, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
    // 处理 pods...
    time.Sleep(5 * time.Second)
}
```

**问题**：
- 每 5 秒请求一次，API Server 压力巨大
- 如果有 1000 个 Pod，每次都要传输大量数据
- 无法实时感知变化（最多延迟 5 秒）

### ❌ 方案二：直接使用 Watch API（好一点，但还不够）

```go
watcher, _ := clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
for event := range watcher.ResultChan() {
    // 处理事件...
}
```

**问题**：
- Watch 连接可能断开，需要自己处理重连
- 重连后需要重新 List 所有资源，逻辑复杂
- 没有本地缓存，想查某个 Pod 还是要请求 API
- 多个地方监控同一资源，会创建多个 Watch 连接

### ✅ 方案三：使用 Informer（最佳方案）

Informer 帮你解决了上面所有问题：

| 问题 | Informer 的解决方案 |
|------|---------------------|
| Watch 断开重连 | Reflector 自动处理重连和续传 |
| 重连后数据同步 | 使用 ResourceVersion 增量同步 |
| 频繁查询 API | Indexer 本地缓存，查询不走网络 |
| 重复 Watch 连接 | SharedInformer 多处共享一个连接 |

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Informer 架构                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                        Informer                              │    │
│  │                                                               │    │
│  │  ┌───────────────┐      ┌───────────────┐                   │    │
│  │  │   Reflector   │─────>│  Delta FIFO   │                   │    │
│  │  │ (List & Watch)│      │   Queue       │                   │    │
│  │  └───────────────┘      └───────┬───────┘                   │    │
│  │                                 │                            │    │
│  │                                 ▼                            │    │
│  │  ┌───────────────┐      ┌───────────────┐                   │    │
│  │  │    Indexer    │<─────│   Processor   │                   │    │
│  │  │   (本地缓存)   │      │  (事件分发)    │                   │    │
│  │  └───────┬───────┘      └───────┬───────┘                   │    │
│  │          │                      │                            │    │
│  │          ▼                      ▼                            │    │
│  │  ┌───────────────┐      ┌───────────────┐                   │    │
│  │  │    Lister     │      │  EventHandler │                   │    │
│  │  │  (快速查询)    │      │  (回调函数)    │                   │    │
│  │  └───────────────┘      └───────────────┘                   │    │
│  │                                                               │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 核心组件详解

理解 Informer，需要先理解它的 6 个核心组件。我们用**快递站**的比喻来解释：

### 1️⃣ Reflector（反射器）- "快递收发员"

**职责**：负责与 API Server 通信，获取资源数据。

**工作流程**：
1. 启动时先执行 **List**：一次性获取所有现有资源（"把仓库里现有的快递都登记一遍"）
2. 然后执行 **Watch**：持续监听后续变化（"盯着门口，有新快递进来就记录"）

```go
// Reflector 内部大概是这样工作的（伪代码）
func (r *Reflector) Run() {
    // 1. 先 List 获取全量数据
    list, _ := r.listerWatcher.List(options)
    
    // 2. 然后 Watch 增量变化
    watcher, _ := r.listerWatcher.Watch(options)
    for event := range watcher.ResultChan() {
        // 把事件放入 DeltaFIFO 队列
        r.store.Add(event.Object)
    }
}
```

**关键特性**：
- 自动处理 Watch 断开重连
- 使用 ResourceVersion 实现增量同步，不会丢数据

---

### 2️⃣ DeltaFIFO（增量先进先出队列）- "待处理快递架"

**职责**：存储待处理的资源变更事件。

**为什么叫 "Delta"？**
- Delta 意思是"增量/变化"
- 这个队列存的不是资源本身，而是资源的**变化事件**

**事件类型**：
```go
const (
    Added    DeltaType = "Added"    // 新增
    Updated  DeltaType = "Updated"  // 更新
    Deleted  DeltaType = "Deleted"  // 删除
    Replaced DeltaType = "Replaced" // 替换（重新 List 时）
    Sync     DeltaType = "Sync"     // 定期同步
)
```

**为什么叫 "FIFO"？**
- First In First Out，先进先出
- 保证事件按发生顺序处理（快递按到达顺序处理）

---

### 3️⃣ Indexer（索引器）- "快递货架 + 索引本"

**职责**：本地缓存 + 索引能力。

**存储功能**：
- 在内存中维护所有资源的副本
- 查询时直接读本地，不走网络

**索引功能**：
- 默认按 `namespace/name` 索引
- 可以自定义索引（比如按 NodeName 索引所有 Pod）

```go
// 默认索引：通过 namespace/name 快速找到资源
pod, _ := indexer.GetByKey("default/nginx")

// 自定义索引：找出某个 Node 上的所有 Pod
pods, _ := indexer.ByIndex("nodeName", "node-1")
```

---

### 4️⃣ Lister（列表器）- "快递查询台"

**职责**：提供类型安全的查询接口。

**和 Indexer 的区别**：
- Indexer 返回的是 `interface{}`，需要类型断言
- Lister 返回的是具体类型（如 `*corev1.Pod`），更方便使用

```go
// 使用 Indexer（需要类型断言）
obj, _, _ := indexer.GetByKey("default/nginx")
pod := obj.(*corev1.Pod)  // 需要手动转换

// 使用 Lister（直接返回正确类型）
pod, _ := podLister.Pods("default").Get("nginx")
// pod 已经是 *corev1.Pod 类型，直接用
```

---

### 5️⃣ Processor（处理器）- "快递分拣中心"

**职责**：将 DeltaFIFO 中的事件分发给所有注册的 EventHandler。

**工作流程**：
1. 从 DeltaFIFO 取出事件
2. 更新 Indexer（更新本地缓存）
3. 调用所有注册的 EventHandler（通知业务代码）

---

### 6️⃣ EventHandler（事件处理器）- "你的业务代码"

**职责**：定义资源变化时要执行的操作。

```go
// 三种事件回调
AddFunc    // 资源新增时调用
UpdateFunc // 资源更新时调用
DeleteFunc // 资源删除时调用
```

---

## 完整工作流程

让我们串起来看 Informer 的完整工作流程：

```
                          API Server
                              │
                              │ 1. List + Watch
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Reflector                              │
│  • 启动时 List 获取全量数据                                    │
│  • 然后 Watch 持续监听变化                                     │
│  • 自动处理断开重连                                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 2. 事件入队
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      DeltaFIFO Queue                         │
│  [Add nginx] → [Update nginx] → [Delete redis] → ...         │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 3. 出队处理
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Processor                              │
│                          │                                    │
│           ┌──────────────┴──────────────┐                    │
│           ▼                              ▼                    │
│   4. 更新 Indexer                  5. 调用 EventHandler       │
│   (更新本地缓存)                    (通知业务代码)             │
└─────────────────────────────────────────────────────────────┘
                              │
           ┌──────────────────┴──────────────────┐
           ▼                                      ▼
┌──────────────────────┐              ┌──────────────────────┐
│       Indexer        │              │    EventHandler      │
│  ┌────────────────┐  │              │                      │
│  │ default/nginx  │  │              │  AddFunc() {...}     │
│  │ default/redis  │  │              │  UpdateFunc() {...}  │
│  │ kube-system/x  │  │              │  DeleteFunc() {...}  │
│  └────────────────┘  │              │                      │
│                      │              │   你的业务逻辑 🎯     │
│   Lister 从这里查询   │              │                      │
└──────────────────────┘              └──────────────────────┘
```

---

## 基本用法

### 创建单资源 Informer

下面是一个最简单的例子：监控集群中所有 Pod 的变化。

```go
package main

import (
    "fmt"
    "time"
    
    corev1 "k8s.io/api/core/v1"
    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
)

func main() {
    // ... 创建 clientset（参考前面章节）...
    
    // ============================================
    // 第一步：创建 Informer 工厂
    // ============================================
    // SharedInformerFactory 是一个"工厂"，可以创建各种资源的 Informer
    // 
    // 参数说明：
    //   - clientset: 用于与 API Server 通信
    //   - time.Minute*30: resync 周期，每 30 分钟重新同步一次
    //                     设为 0 表示不定期重新同步
    //
    // 什么是 resync？
    //   为了防止本地缓存和 API Server 数据不一致，
    //   Informer 会定期触发 UpdateFunc（即使资源没变化）
    //   让你的代码有机会"修正"状态
    factory := informers.NewSharedInformerFactory(clientset, time.Minute*30)
    
    // ============================================
    // 第二步：获取特定资源的 Informer
    // ============================================
    // factory.Core().V1().Pods() 的意思是：
    //   Core API 组 → V1 版本 → Pods 资源
    // 
    // 类似的还有：
    //   factory.Apps().V1().Deployments()      // Deployment
    //   factory.Core().V1().Services()         // Service
    //   factory.Core().V1().ConfigMaps()       // ConfigMap
    //   factory.Networking().V1().Ingresses()  // Ingress
    podInformer := factory.Core().V1().Pods()
    
    // ============================================
    // 第三步：注册事件处理器（最重要的一步！）
    // ============================================
    // 这里定义了当 Pod 发生变化时，你的代码要做什么
    //
    // 注意：podInformer.Informer() 返回底层的 SharedIndexInformer
    //       podInformer 本身是带类型的包装器
    podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        // 当有新 Pod 创建时调用
        AddFunc: func(obj interface{}) {
            // obj 是 interface{} 类型，需要转换成具体类型
            pod := obj.(*corev1.Pod)
            fmt.Printf("[新增] Pod: %s/%s\n", pod.Namespace, pod.Name)
        },
        
        // 当 Pod 被更新时调用
        // 注意：有两个参数，oldObj 是更新前的，newObj 是更新后的
        UpdateFunc: func(oldObj, newObj interface{}) {
            oldPod := oldObj.(*corev1.Pod)
            newPod := newObj.(*corev1.Pod)
            
            // 小技巧：可以比较新旧对象，只处理真正关心的变化
            if oldPod.Status.Phase != newPod.Status.Phase {
                fmt.Printf("[状态变化] Pod: %s/%s, %s → %s\n", 
                    newPod.Namespace, newPod.Name,
                    oldPod.Status.Phase, newPod.Status.Phase)
            }
        },
        
        // 当 Pod 被删除时调用
        DeleteFunc: func(obj interface{}) {
            pod := obj.(*corev1.Pod)
            fmt.Printf("[删除] Pod: %s/%s\n", pod.Namespace, pod.Name)
        },
    })
    
    // ============================================
    // 第四步：启动 Informer
    // ============================================
    // stopCh 是停止信号，关闭这个 channel 会停止 Informer
    stopCh := make(chan struct{})
    defer close(stopCh)  // 程序退出时自动关闭
    
    // Start 会启动所有通过这个 factory 创建的 Informer
    // 它会在后台 goroutine 中运行，不会阻塞
    factory.Start(stopCh)
    
    // ============================================
    // 第五步：等待缓存同步（非常重要！）
    // ============================================
    // 为什么要等待？
    //   Informer 启动后会先 List 所有资源填充缓存
    //   在缓存填充完成之前，Lister 查询可能返回不完整的数据
    //   所以必须等待缓存同步完成后，再开始处理业务逻辑
    factory.WaitForCacheSync(stopCh)
    
    fmt.Println("✅ 缓存同步完成，开始监听...")
    
    // 阻塞主程序，让 Informer 持续运行
    // 直到 stopCh 被关闭（比如收到 Ctrl+C）
    <-stopCh
}
```

### 运行效果示例

```bash
✅ 缓存同步完成，开始监听...
[新增] Pod: default/nginx-7d9fc7b4c5-abc12
[状态变化] Pod: default/nginx-7d9fc7b4c5-abc12, Pending → Running
[新增] Pod: default/redis-5b4f6c8d9e-xyz34
[删除] Pod: default/old-pod-to-delete
```

### 使用 Lister 查询缓存

**Lister 是 Informer 的查询接口**，它从本地缓存读取数据，不会发起网络请求。

```go
import (
    "k8s.io/apimachinery/pkg/labels"
)

func listPodsFromCache(factory informers.SharedInformerFactory) {
    // ============================================
    // 获取 Lister
    // ============================================
    // Lister 提供类型安全的查询接口
    // 所有查询都是从本地内存读取，速度极快
    podLister := factory.Core().V1().Pods().Lister()
    
    // ============================================
    // 场景1：列出所有 Pod（所有命名空间）
    // ============================================
    // labels.Everything() 表示不过滤标签，返回所有
    pods, err := podLister.List(labels.Everything())
    if err != nil {
        fmt.Printf("列出 Pod 失败: %v\n", err)
        return
    }
    
    fmt.Printf("集群中共有 %d 个 Pod\n", len(pods))
    for _, pod := range pods {
        fmt.Printf("  - %s/%s\n", pod.Namespace, pod.Name)
    }
    
    // ============================================
    // 场景2：列出特定命名空间的 Pod
    // ============================================
    // podLister.Pods("default") 返回一个命名空间级别的 Lister
    namespacePods, err := podLister.Pods("default").List(labels.Everything())
    if err != nil {
        fmt.Printf("列出 default 命名空间 Pod 失败: %v\n", err)
        return
    }
    fmt.Printf("default 命名空间有 %d 个 Pod\n", len(namespacePods))
    
    // ============================================
    // 场景3：根据标签过滤
    // ============================================
    // 只获取带有 app=nginx 标签的 Pod
    selector := labels.SelectorFromSet(labels.Set{"app": "nginx"})
    nginxPods, err := podLister.Pods("default").List(selector)
    if err != nil {
        fmt.Printf("查询失败: %v\n", err)
        return
    }
    fmt.Printf("找到 %d 个 nginx Pod\n", len(nginxPods))
    
    // ============================================
    // 场景4：获取单个 Pod
    // ============================================
    // Get 方法根据名称获取，如果不存在会返回 NotFound 错误
    pod, err := podLister.Pods("default").Get("nginx")
    if err != nil {
        // 判断是否是"不存在"错误
        if errors.IsNotFound(err) {
            fmt.Println("Pod nginx 不存在")
        } else {
            fmt.Printf("获取 Pod 失败: %v\n", err)
        }
        return
    }
    fmt.Printf("获取到 Pod: %s, 状态: %s\n", pod.Name, pod.Status.Phase)
}
```

### Lister vs 直接调 API 的对比

| 特性 | Lister（本地缓存） | clientset（API 调用） |
|------|-------------------|---------------------|
| 速度 | 微秒级（内存读取） | 毫秒级（网络请求） |
| API Server 压力 | 无 | 有 |
| 数据实时性 | 可能有微小延迟 | 实时 |
| 适用场景 | 高频查询、只读场景 | 需要最新数据、写操作 |

**最佳实践**：
- 读操作优先使用 Lister
- 写操作（Create/Update/Delete）必须使用 clientset

### 带命名空间过滤的 Informer

```go
// 只监听特定命名空间
factory := informers.NewSharedInformerFactoryWithOptions(
    clientset,
    time.Minute*30,
    informers.WithNamespace("production"),
)
```

### 带标签过滤的 Informer

```go
import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/informers"
)

// 只监听特定标签的资源
factory := informers.NewSharedInformerFactoryWithOptions(
    clientset,
    time.Minute*30,
    informers.WithTweakListOptions(func(options *metav1.ListOptions) {
        options.LabelSelector = "app=myapp"
    }),
)
```

## SharedInformerFactory

### 多资源 Informer

```go
package main

import (
    "time"
    
    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
)

func setupInformers(clientset *kubernetes.Clientset) {
    factory := informers.NewSharedInformerFactory(clientset, time.Minute*30)
    
    // Pod Informer
    podInformer := factory.Core().V1().Pods().Informer()
    podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    func(obj interface{}) { /* ... */ },
        UpdateFunc: func(old, new interface{}) { /* ... */ },
        DeleteFunc: func(obj interface{}) { /* ... */ },
    })
    
    // Deployment Informer
    deployInformer := factory.Apps().V1().Deployments().Informer()
    deployInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    func(obj interface{}) { /* ... */ },
        UpdateFunc: func(old, new interface{}) { /* ... */ },
        DeleteFunc: func(obj interface{}) { /* ... */ },
    })
    
    // Service Informer
    svcInformer := factory.Core().V1().Services().Informer()
    svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    func(obj interface{}) { /* ... */ },
        UpdateFunc: func(old, new interface{}) { /* ... */ },
        DeleteFunc: func(obj interface{}) { /* ... */ },
    })
    
    stopCh := make(chan struct{})
    
    // 启动所有 Informer
    factory.Start(stopCh)
    
    // 等待所有缓存同步
    factory.WaitForCacheSync(stopCh)
}
```

## 工作队列 (WorkQueue)

### 为什么需要 WorkQueue？

你可能会问：EventHandler 里直接处理不就行了，为什么还要 WorkQueue？

**原因 1：EventHandler 不能阻塞太久**

```go
// ❌ 错误示范：在 EventHandler 中做耗时操作
AddFunc: func(obj interface{}) {
    pod := obj.(*corev1.Pod)
    
    // 假设这里要调用外部 API，需要 5 秒
    result, _ := callExternalAPI(pod)  // 阻塞 5 秒！
    
    // 问题：在这 5 秒内，其他事件都被阻塞了
    // 如果同时有 100 个 Pod 变化，就要等 500 秒！
}
```

**原因 2：需要重试机制**

```go
// ❌ 错误示范：EventHandler 中处理失败没法重试
AddFunc: func(obj interface{}) {
    pod := obj.(*corev1.Pod)
    err := doSomething(pod)
    if err != nil {
        // 失败了怎么办？没有重试机制！
        // 这个 Pod 的处理就永远丢失了
    }
}
```

**原因 3：需要去重和合并**

```go
// 场景：一个 Pod 在 1 秒内更新了 10 次
// 如果直接在 EventHandler 处理，会处理 10 次
// 但其实我们只关心最终状态，处理 1 次就够了
```

### WorkQueue 的解决方案

```
┌──────────────────────────────────────────────────────────────────┐
│                    EventHandler + WorkQueue 模式                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│   EventHandler                WorkQueue                 Worker    │
│   (事件接收)                  (任务队列)                (处理器)   │
│                                                                    │
│   ┌─────────┐                ┌─────────┐              ┌─────────┐ │
│   │ AddFunc │──key入队──────>│  Queue  │──取出key───>│  处理   │ │
│   └─────────┘                │         │              │  逻辑   │ │
│                              │ ns/name │              │         │ │
│   ┌─────────┐                │ ns/name │  失败重入队  │         │ │
│   │ Update  │──key入队──────>│ ns/name │<────────────│         │ │
│   └─────────┘                │   ...   │              │         │ │
│                              └─────────┘              └─────────┘ │
│   ┌─────────┐                    │                                │
│   │ Delete  │──key入队───────────┘                                │
│   └─────────┘                                                      │
│                                                                    │
│   特点：                                                           │
│   1. EventHandler 只做"入队"操作，不阻塞                          │
│   2. 相同的 key 会自动去重（一个 Pod 更新 10 次只入队 1 次）      │
│   3. Worker 处理失败可以重新入队重试                              │
│   4. 可以启动多个 Worker 并行处理                                 │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

### 完整示例代码

下面的代码实现了一个完整的控制器框架，这是 Kubernetes 控制器的标准模式。

```go
package main

import (
    "fmt"
    "time"
    
    corev1 "k8s.io/api/core/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/util/workqueue"
)

// Controller 控制器结构体
type Controller struct {
    informer  cache.SharedIndexInformer    // Informer，用于监听和缓存
    workqueue workqueue.RateLimitingInterface // 工作队列
}

// NewController 创建控制器
func NewController(clientset *kubernetes.Clientset) *Controller {
    factory := informers.NewSharedInformerFactory(clientset, time.Minute*30)
    informer := factory.Core().V1().Pods().Informer()
    
    // ============================================
    // 创建限速工作队列
    // ============================================
    // RateLimitingQueue 有三个特性：
    // 1. 普通队列功能：Add, Get, Done
    // 2. 去重功能：相同的 key 只会存在一个
    // 3. 限速重试：失败后会延迟重试，延迟时间指数增长
    //
    // DefaultControllerRateLimiter 的重试策略：
    //   第 1 次失败：等 5ms 后重试
    //   第 2 次失败：等 10ms 后重试
    //   第 3 次失败：等 20ms 后重试
    //   ...以此类推，最大等待 1000 秒
    queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
    
    // ============================================
    // 添加事件处理器
    // ============================================
    // 注意：这里只做"入队"操作，不做实际处理
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            // MetaNamespaceKeyFunc 生成 "namespace/name" 格式的 key
            // 例如："default/nginx"
            key, err := cache.MetaNamespaceKeyFunc(obj)
            if err == nil {
                queue.Add(key)  // 只入队，不处理
            }
        },
        UpdateFunc: func(old, new interface{}) {
            key, err := cache.MetaNamespaceKeyFunc(new)
            if err == nil {
                queue.Add(key)
            }
        },
        DeleteFunc: func(obj interface{}) {
            // DeletionHandlingMetaNamespaceKeyFunc 专门处理删除事件
            // 它能正确处理 DeletedFinalStateUnknown 类型（后面会解释）
            key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
            if err == nil {
                queue.Add(key)
            }
        },
    })
    
    return &Controller{
        informer:  informer,
        workqueue: queue,
    }
}

// Run 启动控制器
func (c *Controller) Run(stopCh <-chan struct{}) {
    // 捕获 panic，避免程序崩溃
    defer utilruntime.HandleCrash()
    // 程序退出时关闭队列
    defer c.workqueue.ShutDown()
    
    fmt.Println("🚀 启动控制器...")
    
    // 在后台 goroutine 中运行 Informer
    go c.informer.Run(stopCh)
    
    // 等待缓存同步完成
    // HasSynced 函数返回缓存是否已同步
    fmt.Println("⏳ 等待缓存同步...")
    if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
        fmt.Println("❌ 缓存同步失败")
        return
    }
    
    fmt.Println("✅ 缓存同步完成，启动 worker")
    
    // ============================================
    // 启动 worker（可以启动多个）
    // ============================================
    // wait.Until 会持续执行 runWorker，直到 stopCh 关闭
    // time.Second 是两次执行之间的间隔（如果 runWorker 返回了）
    //
    // 如果想启动多个 worker 并行处理：
    // for i := 0; i < 3; i++ {
    //     go wait.Until(c.runWorker, time.Second, stopCh)
    // }
    go wait.Until(c.runWorker, time.Second, stopCh)
    
    // 阻塞，直到收到停止信号
    <-stopCh
    fmt.Println("🛑 控制器停止")
}

// runWorker 持续从队列取任务并处理
func (c *Controller) runWorker() {
    // 不断循环处理队列中的任务
    for c.processNextItem() {
    }
}

// processNextItem 处理队列中的下一个任务
func (c *Controller) processNextItem() bool {
    // ============================================
    // 步骤 1：从队列获取任务
    // ============================================
    // Get() 会阻塞，直到队列中有任务
    // 返回值：key（任务标识）, quit（队列是否已关闭）
    key, quit := c.workqueue.Get()
    if quit {
        return false  // 队列关闭，退出循环
    }
    
    // ============================================
    // 步骤 2：标记任务完成（无论成功失败都要调用）
    // ============================================
    // Done() 告诉队列这个 key 的处理已结束
    // 如果不调用 Done()，这个 key 永远不会被重新处理
    defer c.workqueue.Done(key)
    
    // ============================================
    // 步骤 3：执行实际的业务逻辑
    // ============================================
    err := c.syncHandler(key.(string))
    
    if err == nil {
        // 处理成功！
        // Forget() 清除这个 key 的重试计数
        // 下次这个 key 再入队，重试计数从 0 开始
        c.workqueue.Forget(key)
        return true
    }
    
    // ============================================
    // 步骤 4：处理失败，重新入队重试
    // ============================================
    // AddRateLimited() 会根据重试次数计算延迟时间
    // 重试次数越多，等待时间越长（指数退避）
    fmt.Printf("⚠️ 处理 %s 失败: %v，将重新入队重试\n", key, err)
    c.workqueue.AddRateLimited(key)
    
    return true
}

// syncHandler 实际的业务处理逻辑
// 这是你需要实现的核心函数！
func (c *Controller) syncHandler(key string) error {
    // ============================================
    // 步骤 1：解析 key
    // ============================================
    // key 格式是 "namespace/name"，需要拆分
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return err
    }
    
    // ============================================
    // 步骤 2：从缓存获取对象
    // ============================================
    // 注意：这里是从 Indexer（本地缓存）获取，不是调 API
    obj, exists, err := c.informer.GetIndexer().GetByKey(key)
    if err != nil {
        return err
    }
    
    // 对象不存在，说明已被删除
    if !exists {
        fmt.Printf("🗑️ Pod %s/%s 已删除\n", namespace, name)
        // 这里可以做一些清理工作
        return nil
    }
    
    // ============================================
    // 步骤 3：执行业务逻辑
    // ============================================
    pod := obj.(*corev1.Pod)
    fmt.Printf("🔄 处理 Pod: %s/%s, 状态: %s\n", namespace, name, pod.Status.Phase)
    
    // 🎯 在这里实现你的业务逻辑！
    // 例如：
    // - 检查 Pod 是否符合某些条件
    // - 根据 Pod 状态创建其他资源
    // - 更新某些配置
    // - 发送通知
    
    return nil
}

func main() {
    // ... 创建 clientset ...
    
    controller := NewController(clientset)
    
    stopCh := make(chan struct{})
    controller.Run(stopCh)
}
```

## Indexer 索引

### 什么是索引？

想象一下，你有 10000 个 Pod 的缓存，现在要找出所有运行在 `node-1` 上的 Pod。

**没有索引**：遍历 10000 个 Pod，逐个检查 `pod.Spec.NodeName == "node-1"`。

**有索引**：直接根据 "node-1" 这个 key，瞬间找到所有对应的 Pod。

这就是数据库索引的原理，Informer 的 Indexer 也是一样的。

### 默认索引

Informer 默认有一个索引：**按 namespace 索引**。

```go
// 默认可以按 namespace 快速查询
pods, _ := indexer.ByIndex("namespace", "default")
// 快速返回 default 命名空间的所有 Pod
```

### 自定义索引

如果你经常需要"按 NodeName 查找 Pod"，可以添加自定义索引：

```go
// ============================================
// 第一步：定义索引名称和索引函数
// ============================================
const NodeNameIndex = "spec.nodeName"

// 索引函数：从对象中提取索引值
// 返回值是 []string，因为一个对象可以有多个索引值
func nodeNameIndexFunc(obj interface{}) ([]string, error) {
    pod, ok := obj.(*corev1.Pod)
    if !ok {
        return nil, fmt.Errorf("not a pod")
    }
    
    // 如果 Pod 还没调度（NodeName 为空），返回空
    if pod.Spec.NodeName == "" {
        return []string{}, nil
    }
    
    // 返回这个 Pod 的 NodeName 作为索引值
    return []string{pod.Spec.NodeName}, nil
}

func setupWithIndex(clientset *kubernetes.Clientset) {
    factory := informers.NewSharedInformerFactory(clientset, 0)
    podInformer := factory.Core().V1().Pods().Informer()
    
    // ============================================
    // 第二步：注册自定义索引
    // ============================================
    // 必须在 Informer 启动前注册！
    podInformer.AddIndexers(cache.Indexers{
        NodeNameIndex: nodeNameIndexFunc,
    })
    
    // 启动 informer...
    stopCh := make(chan struct{})
    go podInformer.Run(stopCh)
    cache.WaitForCacheSync(stopCh, podInformer.HasSynced)
    
    // ============================================
    // 第三步：使用索引查询
    // ============================================
    indexer := podInformer.GetIndexer()
    
    // 快速找出 node-1 上的所有 Pod
    pods, err := indexer.ByIndex(NodeNameIndex, "node-1")
    if err != nil {
        fmt.Printf("查询失败: %v\n", err)
        return
    }
    
    fmt.Printf("node-1 上有 %d 个 Pod:\n", len(pods))
    for _, obj := range pods {
        pod := obj.(*corev1.Pod)
        fmt.Printf("  - %s/%s\n", pod.Namespace, pod.Name)
    }
}
```

### 常用的自定义索引示例

```go
// 按 OwnerReference 索引（找某个 ReplicaSet 的所有 Pod）
func ownerIndexFunc(obj interface{}) ([]string, error) {
    pod := obj.(*corev1.Pod)
    var keys []string
    for _, ref := range pod.OwnerReferences {
        keys = append(keys, string(ref.UID))
    }
    return keys, nil
}

// 按 Label 索引（找某个 app 的所有 Pod）
func appLabelIndexFunc(obj interface{}) ([]string, error) {
    pod := obj.(*corev1.Pod)
    if app, ok := pod.Labels["app"]; ok {
        return []string{app}, nil
    }
    return []string{}, nil
}
```

## 最佳实践

### 1️⃣ 使用 SharedInformerFactory

**为什么？** 同一资源类型只需要一个 Watch 连接。

```go
// ✅ 正确：使用同一个 factory，共享连接
factory := informers.NewSharedInformerFactory(clientset, time.Minute*30)
podInformer1 := factory.Core().V1().Pods().Informer()
podInformer2 := factory.Core().V1().Pods().Informer()
// podInformer1 和 podInformer2 是同一个对象！

// ❌ 错误：创建多个 factory，浪费连接
factory1 := informers.NewSharedInformerFactory(clientset, time.Minute*30)
factory2 := informers.NewSharedInformerFactory(clientset, time.Minute*30)
// 这会创建两个独立的 Watch 连接
```

### 2️⃣ 必须等待缓存同步

**为什么？** 缓存未同步时，Lister 可能返回不完整的数据。

```go
// ✅ 正确
factory.Start(stopCh)
factory.WaitForCacheSync(stopCh)  // 阻塞，直到缓存同步完成
// 现在可以安全使用 Lister 了

// ❌ 错误：启动后立即使用
factory.Start(stopCh)
pods, _ := podLister.List(labels.Everything())  // 可能返回空或不完整！
```

### 3️⃣ 在 EventHandler 中只做入队操作

**为什么？** EventHandler 是同步调用的，阻塞会影响其他事件的处理。

```go
// ✅ 正确：只入队
AddFunc: func(obj interface{}) {
    key, _ := cache.MetaNamespaceKeyFunc(obj)
    queue.Add(key)  // 立即返回
}

// ❌ 错误：在 EventHandler 中做耗时操作
AddFunc: func(obj interface{}) {
    pod := obj.(*corev1.Pod)
    http.Post("http://external-api", ...)  // 阻塞！
}
```

### 4️⃣ 正确处理 DeletedFinalStateUnknown

**什么是 DeletedFinalStateUnknown？**

当 Watch 连接断开再重连时，Informer 会重新 List。如果发现某个资源在断开期间被删除了，Informer 不知道删除前的最终状态，就会用 `DeletedFinalStateUnknown` 包装。

```go
DeleteFunc: func(obj interface{}) {
    // 尝试获取正常的对象
    pod, ok := obj.(*corev1.Pod)
    if !ok {
        // 可能是 DeletedFinalStateUnknown
        tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
        if !ok {
            // 真的不知道是什么类型
            return
        }
        // 从 tombstone 中获取原对象
        pod, ok = tombstone.Obj.(*corev1.Pod)
        if !ok {
            return
        }
    }
    
    // 现在可以安全使用 pod 了
    fmt.Printf("删除: %s/%s\n", pod.Namespace, pod.Name)
}
```

### 5️⃣ 使用限速队列处理临时失败

**为什么？** 网络抖动、API 限流等临时问题会导致失败，重试通常能成功。

```go
err := c.syncHandler(key.(string))
if err != nil {
    // AddRateLimited 会延迟重试，避免疯狂重试
    c.workqueue.AddRateLimited(key)
    
    // 可以设置最大重试次数
    if c.workqueue.NumRequeues(key) > 10 {
        // 重试太多次了，放弃
        c.workqueue.Forget(key)
        fmt.Printf("放弃处理 %s\n", key)
    }
}
```

---

## 常见问题 FAQ

### Q: resync 周期设多少合适？

**A:** 取决于你的业务需求：
- `0`：不定期 resync，依赖 Watch 事件
- `30秒~5分钟`：对实时性要求高的场景
- `30分钟~1小时`：一般场景
- 更长：对实时性要求不高的场景

### Q: Informer 会不会占用很多内存？

**A:** 会。如果集群有 10000 个 Pod，Informer 就会在内存中缓存 10000 个 Pod 对象。

解决方案：
- 使用命名空间过滤：`informers.WithNamespace("production")`
- 使用标签过滤：`informers.WithTweakListOptions(...)`
- 使用 Metadata Informer（只缓存元数据，不缓存 spec/status）

### Q: 为什么 UpdateFunc 会被频繁调用？

**A:** 两个原因：
1. **resync 机制**：定期触发 UpdateFunc（即使对象没变化）
2. **Status 更新**：Pod 的 status 经常变化

解决方案：比较 ResourceVersion，跳过不需要处理的更新。

```go
UpdateFunc: func(old, new interface{}) {
    oldPod := old.(*corev1.Pod)
    newPod := new.(*corev1.Pod)
    
    // 跳过 resync 触发的假更新
    if oldPod.ResourceVersion == newPod.ResourceVersion {
        return
    }
    
    // 真正的更新，入队处理
    key, _ := cache.MetaNamespaceKeyFunc(new)
    queue.Add(key)
}
```

---

## 总结

| 组件 | 职责 | 类比 |
|------|------|------|
| Reflector | 与 API Server 通信 | 快递收发员 |
| DeltaFIFO | 存储变更事件 | 待处理快递架 |
| Indexer | 本地缓存 + 索引 | 快递货架 + 索引本 |
| Lister | 类型安全的查询接口 | 快递查询台 |
| Processor | 分发事件给 Handler | 快递分拣中心 |
| EventHandler | 用户定义的回调 | 你的业务代码 |
| WorkQueue | 异步处理 + 重试 | 任务队列 |

掌握 Informer，你就掌握了 Kubernetes 控制器开发的核心！

---

## 下一步

- [实战项目：自定义控制器](./05-controller-demo.md)



