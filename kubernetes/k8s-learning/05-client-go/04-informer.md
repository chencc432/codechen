# 🔄 Informer 机制详解

## 为什么需要 Informer？

直接使用 Watch API 有以下问题：
- 每次重连需要重新列出所有资源
- 多个组件 Watch 同一资源会创建多个连接
- 没有本地缓存，每次查询都要请求 API

Informer 通过缓存和事件机制解决了这些问题。

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

## 基本用法

### 创建单资源 Informer

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
    // ... 创建 clientset ...
    
    // 创建 Informer 工厂
    // 第二个参数是 resync 周期，0 表示不定期重新同步
    factory := informers.NewSharedInformerFactory(clientset, time.Minute*30)
    
    // 获取 Pod Informer
    podInformer := factory.Core().V1().Pods()
    
    // 添加事件处理器
    podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            pod := obj.(*corev1.Pod)
            fmt.Printf("[新增] Pod: %s/%s\n", pod.Namespace, pod.Name)
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            pod := newObj.(*corev1.Pod)
            fmt.Printf("[更新] Pod: %s/%s\n", pod.Namespace, pod.Name)
        },
        DeleteFunc: func(obj interface{}) {
            pod := obj.(*corev1.Pod)
            fmt.Printf("[删除] Pod: %s/%s\n", pod.Namespace, pod.Name)
        },
    })
    
    // 启动 Informer
    stopCh := make(chan struct{})
    defer close(stopCh)
    
    factory.Start(stopCh)
    
    // 等待缓存同步
    factory.WaitForCacheSync(stopCh)
    
    fmt.Println("缓存同步完成，开始监听...")
    
    // 阻塞主程序
    <-stopCh
}
```

### 使用 Lister 查询缓存

```go
func listPodsFromCache(factory informers.SharedInformerFactory) {
    // 获取 Lister
    podLister := factory.Core().V1().Pods().Lister()
    
    // 列出所有 Pod
    pods, err := podLister.List(labels.Everything())
    if err != nil {
        fmt.Printf("列出 Pod 失败: %v\n", err)
        return
    }
    
    for _, pod := range pods {
        fmt.Printf("Pod: %s/%s\n", pod.Namespace, pod.Name)
    }
    
    // 按命名空间列出
    namespacePods, err := podLister.Pods("default").List(labels.Everything())
    if err != nil {
        fmt.Printf("列出 default 命名空间 Pod 失败: %v\n", err)
        return
    }
    
    // 获取单个 Pod
    pod, err := podLister.Pods("default").Get("nginx")
    if err != nil {
        fmt.Printf("获取 Pod 失败: %v\n", err)
        return
    }
    fmt.Printf("获取到 Pod: %s\n", pod.Name)
}
```

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

Informer 通常与 WorkQueue 配合使用，实现控制器模式。

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

type Controller struct {
    informer  cache.SharedIndexInformer
    workqueue workqueue.RateLimitingInterface
}

func NewController(clientset *kubernetes.Clientset) *Controller {
    factory := informers.NewSharedInformerFactory(clientset, time.Minute*30)
    informer := factory.Core().V1().Pods().Informer()
    
    // 创建限速工作队列
    queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
    
    // 添加事件处理器
    informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            key, err := cache.MetaNamespaceKeyFunc(obj)
            if err == nil {
                queue.Add(key)
            }
        },
        UpdateFunc: func(old, new interface{}) {
            key, err := cache.MetaNamespaceKeyFunc(new)
            if err == nil {
                queue.Add(key)
            }
        },
        DeleteFunc: func(obj interface{}) {
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

func (c *Controller) Run(stopCh <-chan struct{}) {
    defer utilruntime.HandleCrash()
    defer c.workqueue.ShutDown()
    
    // 启动 Informer
    go c.informer.Run(stopCh)
    
    // 等待缓存同步
    if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
        fmt.Println("缓存同步失败")
        return
    }
    
    fmt.Println("缓存同步完成，启动 worker")
    
    // 启动 worker
    go wait.Until(c.runWorker, time.Second, stopCh)
    
    <-stopCh
    fmt.Println("控制器停止")
}

func (c *Controller) runWorker() {
    for c.processNextItem() {
    }
}

func (c *Controller) processNextItem() bool {
    // 从队列获取任务
    key, quit := c.workqueue.Get()
    if quit {
        return false
    }
    defer c.workqueue.Done(key)
    
    // 处理任务
    err := c.syncHandler(key.(string))
    if err == nil {
        // 处理成功，清除重试计数
        c.workqueue.Forget(key)
        return true
    }
    
    // 处理失败，重新入队
    fmt.Printf("处理 %s 失败: %v，重新入队\n", key, err)
    c.workqueue.AddRateLimited(key)
    
    return true
}

func (c *Controller) syncHandler(key string) error {
    // 解析 namespace/name
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return err
    }
    
    // 从缓存获取对象
    obj, exists, err := c.informer.GetIndexer().GetByKey(key)
    if err != nil {
        return err
    }
    
    if !exists {
        fmt.Printf("Pod %s/%s 已删除\n", namespace, name)
        return nil
    }
    
    pod := obj.(*corev1.Pod)
    fmt.Printf("处理 Pod: %s/%s, 状态: %s\n", namespace, name, pod.Status.Phase)
    
    // 这里实现你的业务逻辑
    
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

Indexer 提供了对缓存数据的索引能力。

```go
// 自定义索引器
const NodeNameIndex = "spec.nodeName"

func nodeNameIndexFunc(obj interface{}) ([]string, error) {
    pod, ok := obj.(*corev1.Pod)
    if !ok {
        return nil, fmt.Errorf("not a pod")
    }
    return []string{pod.Spec.NodeName}, nil
}

func setupWithIndex(clientset *kubernetes.Clientset) {
    factory := informers.NewSharedInformerFactory(clientset, 0)
    podInformer := factory.Core().V1().Pods().Informer()
    
    // 添加自定义索引
    podInformer.AddIndexers(cache.Indexers{
        NodeNameIndex: nodeNameIndexFunc,
    })
    
    // ... 启动 informer ...
    
    // 使用索引查询
    indexer := podInformer.GetIndexer()
    pods, err := indexer.ByIndex(NodeNameIndex, "node-1")
    if err != nil {
        fmt.Printf("查询失败: %v\n", err)
        return
    }
    
    for _, obj := range pods {
        pod := obj.(*corev1.Pod)
        fmt.Printf("Node node-1 上的 Pod: %s\n", pod.Name)
    }
}
```

## 最佳实践

1. **使用 SharedInformerFactory**：复用 Informer，减少 API 请求
2. **等待缓存同步**：确保 `WaitForCacheSync` 返回后再处理事件
3. **使用 WorkQueue**：避免在事件处理器中执行耗时操作
4. **处理 DeletedFinalStateUnknown**：删除事件可能是 `DeletedFinalStateUnknown` 类型
5. **实现重试机制**：使用 RateLimitingQueue 处理临时失败

## 下一步

- [实战项目：自定义控制器](./05-controller-demo.md)



