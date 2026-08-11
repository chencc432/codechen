# 实战项目：自定义控制器

## 什么是控制器？

在开始写代码之前，让我们先理解什么是"控制器"。

### 生活中的控制器 🌡️

想象一下你家的**空调恒温器**：

1. **目标状态**：你设定温度为 26°C（这是你期望的状态）
2. **当前状态**：房间现在是 30°C（这是实际状态）
3. **控制逻辑**：恒温器发现"目标 26°C ≠ 实际 30°C"，于是开启制冷
4. **持续调节**：温度下降到 26°C 后，停止制冷；如果又升高了，再次制冷

这个**不断检测、不断调整、让实际状态趋近于期望状态**的过程，就是控制器的核心思想。

### Kubernetes 中的控制器

Kubernetes 的控制器也是同样的原理：

```
┌─────────────────────────────────────────────────────────────────┐
│                    控制器循环 (Control Loop)                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│    ┌──────────────┐                      ┌──────────────┐       │
│    │   期望状态   │                      │   实际状态   │       │
│    │ (Desired)    │                      │ (Current)    │       │
│    │              │                      │              │       │
│    │ Deployment:  │                      │ 集群中实际   │       │
│    │ replicas: 3  │        比较          │ 有 2 个 Pod  │       │
│    └──────┬───────┘    ◄─────────────►   └──────┬───────┘       │
│           │                                      │               │
│           └───────────────┬──────────────────────┘               │
│                           │                                      │
│                           ▼                                      │
│                  ┌────────────────┐                             │
│                  │ 发现差异！      │                             │
│                  │ 需要再创建 1 个 │                             │
│                  │ Pod            │                             │
│                  └────────┬───────┘                             │
│                           │                                      │
│                           ▼                                      │
│                  ┌────────────────┐                             │
│                  │   执行操作     │                             │
│                  │ 创建新的 Pod   │                             │
│                  └────────────────┘                             │
│                                                                   │
│    这个循环会一直运行，确保实际状态 = 期望状态                    │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Kubernetes 内置的控制器

Kubernetes 自带了很多控制器，它们都运行在 `kube-controller-manager` 中：

| 控制器 | 职责 |
|--------|------|
| **Deployment Controller** | 确保 Pod 副本数与 Deployment 定义一致 |
| **ReplicaSet Controller** | 维护 Pod 副本数量 |
| **Node Controller** | 监控节点状态，处理节点故障 |
| **Service Controller** | 管理云厂商的负载均衡器 |
| **Endpoint Controller** | 填充 Endpoints 对象（Service 和 Pod 的映射） |
| **Namespace Controller** | 处理命名空间删除时的资源清理 |

**你也可以写自己的控制器！** 这就是本章要教你的。

---

## 项目目标

我们要创建一个简单但完整的控制器：**Pod 注解器**

**功能**：监控集群中所有 Pod，当有新 Pod 创建时，自动给它添加一个注解（annotation）。

**为什么选这个例子？**
- 足够简单，适合入门
- 包含控制器的所有核心组件
- 展示完整的 Watch → Queue → Handle 流程

### 先记住两个设计原则

1. **电平触发（level-triggered）**：事件只是“请再对一次账”的信号。丢事件不可怕，怕的是 Reconcile 不幂等。
2. **幂等**：对同一 key 处理 1 次和 100 次，终态一致。本例靠“已有注解则跳过”实现。

因此 Handler 里不要写“只在 ADDED 时干什么”的边沿逻辑；一律入队，在 `syncHandler` 里读**当前**状态再决定。

---

## 控制器模式

一个标准的 Kubernetes 控制器由以下部分组成：

```
┌─────────────────────────────────────────────────────────────────────┐
│                      控制器模式                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   ┌───────────────┐                                                 │
│   │   Informer    │  ← 监听 API Server，获取资源变化事件            │
│   │ (监听资源变化) │                                                 │
│   └───────┬───────┘                                                 │
│           │ 事件 (Add/Update/Delete)                                │
│           ▼                                                          │
│   ┌───────────────┐                                                 │
│   │   WorkQueue   │  ← 缓冲队列，解耦事件接收和处理                 │
│   │  (任务队列)    │    支持去重、限速、重试                         │
│   └───────┬───────┘                                                 │
│           │ key (namespace/name)                                    │
│           ▼                                                          │
│   ┌───────────────┐       ┌───────────────┐                        │
│   │    Worker     │──────>│  SyncHandler  │  ← 你的业务逻辑！       │
│   │ (消费任务)     │       │ (业务逻辑)     │    比较期望 vs 实际    │
│   └───────────────┘       └───────┬───────┘    然后执行操作         │
│                                   │                                  │
│                                   ▼                                  │
│                           ┌───────────────┐                        │
│                           │ Kubernetes API│  ← 创建/更新/删除资源   │
│                           │  (更新资源)    │                        │
│                           └───────────────┘                        │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### 各组件职责说明

| 组件 | 职责 | 类比 |
|------|------|------|
| **Informer** | 监听资源变化，维护本地缓存 | 订阅消息的客户端 |
| **WorkQueue** | 存储待处理的任务，支持去重和重试 | 待办事项清单 |
| **Worker** | 不断从队列取任务并处理 | 干活的工人 |
| **SyncHandler** | 核心业务逻辑（你要实现的） | 具体的工作内容 |
| **Lister** | 从本地缓存快速查询资源 | 本地数据库查询 |
| **Clientset** | 调用 API 执行写操作 | API 调用客户端 |

### 数据流详解

```
1. 用户创建 Pod
       │
       ▼
2. API Server 存储 Pod 到 etcd
       │
       ▼
3. Informer 通过 Watch 接收到 Add 事件
       │
       ▼
4. EventHandler.AddFunc 被调用
       │
       ▼
5. 生成 key (如 "default/nginx")，放入 WorkQueue
       │
       ▼
6. Worker 从 WorkQueue 取出 "default/nginx"
       │
       ▼
7. SyncHandler 执行业务逻辑：
   - 从 Lister 获取 Pod 当前状态
   - 判断是否需要处理
   - 通过 Clientset 更新 Pod
       │
       ▼
8. 处理完成！如果失败则重新入队重试
```

## 完整代码

### 项目结构

```
pod-annotator/
├── go.mod          # Go 模块定义，声明依赖
├── go.sum          # 依赖版本锁定文件
├── main.go         # 程序入口：初始化、启动控制器
└── controller/
    └── controller.go   # 控制器核心逻辑
```

### 依赖说明

| 依赖包 | 作用 |
|--------|------|
| `k8s.io/api` | Kubernetes 资源类型定义（Pod、Deployment 等） |
| `k8s.io/apimachinery` | 通用工具：元数据、错误处理、标签选择器等 |
| `k8s.io/client-go` | 核心！Clientset、Informer、WorkQueue 都在这里 |
| `k8s.io/klog/v2` | Kubernetes 风格的日志库 |

### go.mod

```go
module pod-annotator

go 1.21

require (
    k8s.io/api v0.29.0
    k8s.io/apimachinery v0.29.0
    k8s.io/client-go v0.29.0
    k8s.io/klog/v2 v2.110.1
)
```

### controller/controller.go

这是控制器的核心代码，我会逐段详细解释。

```go
package controller

import (
    "context"
    "fmt"
    "time"

    corev1 "k8s.io/api/core/v1"                    // Pod 类型定义
    "k8s.io/apimachinery/pkg/api/errors"          // 错误处理（如 NotFound）
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // 元数据类型
    utilruntime "k8s.io/apimachinery/pkg/util/runtime" // 运行时工具
    "k8s.io/apimachinery/pkg/util/wait"           // 等待/定时工具
    coreinformers "k8s.io/client-go/informers/core/v1" // Pod Informer
    "k8s.io/client-go/kubernetes"                 // Clientset
    corelisters "k8s.io/client-go/listers/core/v1" // Pod Lister
    "k8s.io/client-go/tools/cache"                // 缓存工具
    "k8s.io/client-go/util/retry"                 // 冲突重试
    "k8s.io/client-go/util/workqueue"             // 工作队列
    "k8s.io/klog/v2"                              // 日志
)

const (
    // 注解键：我们要添加的注解名称
    // 格式遵循 Kubernetes 惯例：<域名>/<键名>
    AnnotationKey = "pod-annotator.example.com/processed"
    
    // 控制器名称：用于日志和队列命名
    ControllerName = "pod-annotator"
)

// ============================================
// Controller 结构体定义
// ============================================
// 一个控制器通常需要以下组件：
// 1. Clientset - 用于调用 API（写操作）
// 2. Lister - 用于从缓存读取数据（读操作）
// 3. Synced 函数 - 判断缓存是否同步完成
// 4. WorkQueue - 任务队列

type Controller struct {
    // ----------------------------------------
    // clientset: 用于与 Kubernetes API 交互
    // ----------------------------------------
    // 为什么是 Interface 而不是 *Clientset？
    // 使用接口便于单元测试时 mock
    clientset kubernetes.Interface

    // ----------------------------------------
    // podLister: 从本地缓存读取 Pod
    // ----------------------------------------
    // 优点：速度快（内存读取），不给 API Server 压力
    // 注意：数据可能有微小延迟（通常毫秒级）
    podLister corelisters.PodLister
    
    // ----------------------------------------
    // podsSynced: 判断缓存是否同步完成的函数
    // ----------------------------------------
    // 返回 true 表示初始 List 已完成，缓存数据完整
    // 在缓存同步完成之前，不应该处理任何任务
    podsSynced cache.InformerSynced

    // ----------------------------------------
    // workqueue: 限速工作队列
    // ----------------------------------------
    // 特性：
    // - 去重：相同的 key 只保留一个
    // - 限速：失败重试时指数退避
    // - 有序：保证 Done() 之前不会重复获取同一个 key
    workqueue workqueue.RateLimitingInterface
}

// ============================================
// NewController: 创建控制器实例
// ============================================
// 这是控制器的"构造函数"
// 主要做两件事：
// 1. 初始化各个组件
// 2. 注册事件处理器

func NewController(
    clientset kubernetes.Interface,
    podInformer coreinformers.PodInformer,  // 由外部传入，便于共享
) *Controller {
    
    // 初始化控制器结构
    controller := &Controller{
        clientset:  clientset,
        
        // Lister() 返回一个类型安全的查询接口
        podLister:  podInformer.Lister(),
        
        // HasSynced 是一个函数，返回缓存是否同步完成
        podsSynced: podInformer.Informer().HasSynced,
        
        // 创建带名称的限速队列
        // NewNamedRateLimitingQueue 的参数：
        //   - RateLimiter: 限速策略（默认是指数退避）
        //   - name: 队列名称，用于 metrics 和日志
        workqueue:  workqueue.NewNamedRateLimitingQueue(
            workqueue.DefaultControllerRateLimiter(),
            ControllerName,
        ),
    }

    klog.Info("设置事件处理器")
    
    // ----------------------------------------
    // 注册事件处理器
    // ----------------------------------------
    // 当 Pod 发生变化时，Informer 会调用这些函数
    // 注意：我们只关心 Add 和 Update，不关心 Delete
    //       因为删除的 Pod 不需要添加注解
    podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        // Pod 创建时
        AddFunc: controller.enqueuePod,
        
        // Pod 更新时
        // 为什么 Update 也要处理？
        // 因为第一次处理可能失败，Pod 更新时可以重试
        UpdateFunc: func(old, new interface{}) {
            controller.enqueuePod(new)
        },
        
        // DeleteFunc: 我们不处理删除事件
    })

    return controller
}

// ============================================
// enqueuePod: 将 Pod 加入工作队列
// ============================================
// 这个函数被 EventHandler 调用
// 它只做一件事：生成 key 并入队

func (c *Controller) enqueuePod(obj interface{}) {
    var key string
    var err error
    
    // MetaNamespaceKeyFunc 生成 "namespace/name" 格式的 key
    // 例如：一个名为 nginx 的 Pod 在 default 命名空间
    //       key = "default/nginx"
    if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
        utilruntime.HandleError(err)
        return
    }
    
    // 将 key 加入队列
    // 如果队列中已存在相同的 key，不会重复添加（去重）
    c.workqueue.Add(key)
}

// ============================================
// Run: 启动控制器
// ============================================
// 参数：
//   - workers: 并行处理的 worker 数量
//   - stopCh: 停止信号，关闭这个 channel 会停止控制器

func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
    // 捕获 panic，避免整个程序崩溃
    defer utilruntime.HandleCrash()
    
    // 程序退出时关闭队列
    // 关闭后，workqueue.Get() 会返回 shutdown=true
    defer c.workqueue.ShutDown()

    klog.Info("🚀 启动 Pod 注解控制器")

    // ----------------------------------------
    // 第一步：等待缓存同步
    // ----------------------------------------
    // 为什么要等待？
    // Informer 启动后会先 List 所有 Pod 填充缓存
    // 在这期间，Lister 返回的数据不完整
    // 所以必须等待同步完成后，再开始处理任务
    klog.Info("⏳ 等待 Informer 缓存同步...")
    if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
        return fmt.Errorf("缓存同步失败")
    }

    klog.Info("✅ 缓存同步完成，启动 workers")

    // ----------------------------------------
    // 第二步：启动 worker
    // ----------------------------------------
    // 可以启动多个 worker 并行处理，提高吞吐量
    // wait.Until 会持续运行 runWorker，直到 stopCh 关闭
    // time.Second 是 runWorker 返回后等待的时间（正常情况下不会返回）
    for i := 0; i < workers; i++ {
        go wait.Until(c.runWorker, time.Second, stopCh)
    }

    klog.Infof("🔧 已启动 %d 个 workers", workers)
    
    // 阻塞，直到收到停止信号
    <-stopCh
    klog.Info("🛑 关闭 workers")

    return nil
}

// ============================================
// runWorker: 运行单个 worker
// ============================================
// 不断从队列取任务并处理，直到队列关闭

func (c *Controller) runWorker() {
    // 持续循环，直到 processNextWorkItem 返回 false
    for c.processNextWorkItem() {
    }
}

// ============================================
// processNextWorkItem: 处理队列中的下一个任务
// ============================================
// 返回值：
//   - true: 继续处理下一个任务
//   - false: 队列已关闭，退出循环

func (c *Controller) processNextWorkItem() bool {
    // ----------------------------------------
    // 步骤 1：从队列获取任务
    // ----------------------------------------
    // Get() 会阻塞，直到队列中有任务或队列关闭
    obj, shutdown := c.workqueue.Get()

    if shutdown {
        // 队列已关闭，退出
        return false
    }

    // ----------------------------------------
    // 步骤 2：处理任务（使用匿名函数便于 defer）
    // ----------------------------------------
    err := func(obj interface{}) error {
        // 无论成功失败，都要调用 Done()
        // 告诉队列这个任务的处理已结束
        // 否则这个 key 永远不会被重新处理
        defer c.workqueue.Done(obj)
        
        var key string
        var ok bool
        
        // 类型断言：确保是 string 类型
        if key, ok = obj.(string); !ok {
            // 无效的任务，直接丢弃
            // Forget() 清除重试计数
            c.workqueue.Forget(obj)
            utilruntime.HandleError(fmt.Errorf("期望 string 类型，但收到 %#v", obj))
            return nil
        }

        // ----------------------------------------
        // 步骤 3：执行核心业务逻辑
        // ----------------------------------------
        if err := c.syncHandler(key); err != nil {
            // 处理失败，重新入队等待重试
            // AddRateLimited 会根据重试次数计算延迟时间
            c.workqueue.AddRateLimited(key)
            return fmt.Errorf("同步 '%s' 失败: %s，重新入队", key, err.Error())
        }

        // 处理成功！
        // Forget() 清除这个 key 的重试计数
        // 下次如果这个 key 再入队，重试计数从 0 开始
        c.workqueue.Forget(obj)
        klog.Infof("✅ 成功同步 '%s'", key)
        return nil
    }(obj)

    if err != nil {
        utilruntime.HandleError(err)
        return true  // 继续处理下一个任务
    }

    return true
}

// ============================================
// syncHandler: 核心业务逻辑 🎯
// ============================================
// 这是你需要重点关注的函数！
// 控制器的"灵魂"就在这里
//
// 参数：key，格式为 "namespace/name"

func (c *Controller) syncHandler(key string) error {
    // ----------------------------------------
    // 步骤 1：解析 key
    // ----------------------------------------
    // "default/nginx" → namespace="default", name="nginx"
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        // key 格式错误，这是代码 bug，不需要重试
        utilruntime.HandleError(fmt.Errorf("无效的资源 key: %s", key))
        return nil  // 返回 nil 表示不需要重试
    }

    // ----------------------------------------
    // 步骤 2：从缓存获取 Pod 当前状态
    // ----------------------------------------
    // 注意：使用 Lister 从本地缓存读取，不会调用 API
    pod, err := c.podLister.Pods(namespace).Get(name)
    if err != nil {
        // Pod 不存在了（可能已被删除）
        if errors.IsNotFound(err) {
            // 这种情况很正常：
            // 1. Pod 创建 → 事件入队
            // 2. Pod 被删除
            // 3. Worker 处理时，Pod 已不存在
            utilruntime.HandleError(fmt.Errorf("Pod '%s' 在工作队列中，但已不存在", key))
            return nil  // 不需要重试
        }
        // 其他错误（如缓存损坏），需要重试
        return err
    }

    // ----------------------------------------
    // 步骤 3：业务逻辑判断
    // ----------------------------------------
    
    // 3.1 检查是否已处理过
    // 如果 Pod 已经有我们的注解，就不需要再处理了
    // 这是一种"幂等性"保证：多次处理结果相同
    if pod.Annotations != nil {
        if _, exists := pod.Annotations[AnnotationKey]; exists {
            // klog.V(4) 是 level 4 的日志，需要 -v=4 才能看到
            klog.V(4).Infof("Pod %s/%s 已处理，跳过", namespace, name)
            return nil
        }
    }

    // 3.2 跳过系统命名空间的 Pod
    // 不要修改系统组件，可能导致集群故障
    if namespace == "kube-system" {
        return nil
    }

    // ----------------------------------------
    // 步骤 4：执行操作
    // ----------------------------------------
    // 调用 API 为 Pod 添加注解
    return c.addAnnotation(pod)
}

// ============================================
// addAnnotation: 为 Pod 添加注解
// ============================================
// 这是实际修改 Pod 的函数

func (c *Controller) addAnnotation(pod *corev1.Pod) error {
    // Update 容易 409 Conflict：用 RetryOnConflict 在函数内重读再写
    // 队列级重试（AddRateLimited）仍然保留，用于其它瞬时错误
    return retry.RetryOnConflict(retry.DefaultRetry, func() error {
        // 每次重试都重新 Get，拿到最新 resourceVersion
        fresh, err := c.clientset.CoreV1().Pods(pod.Namespace).Get(
            context.TODO(), pod.Name, metav1.GetOptions{},
        )
        if err != nil {
            return err
        }
        if fresh.Annotations != nil {
            if _, ok := fresh.Annotations[AnnotationKey]; ok {
                return nil // 已处理，幂等返回
            }
        }

        podCopy := fresh.DeepCopy()
        if podCopy.Annotations == nil {
            podCopy.Annotations = map[string]string{}
        }
        podCopy.Annotations[AnnotationKey] = time.Now().Format(time.RFC3339)

        _, err = c.clientset.CoreV1().Pods(pod.Namespace).Update(
            context.TODO(),
            podCopy,
            metav1.UpdateOptions{},
        )
        return err
    })
}
```

> 导入：`"k8s.io/client-go/util/retry"`。Lister 的对象只用于判断；真正写入前以 Get/重试循环里的对象为准，并始终 `DeepCopy`。

### main.go

这是程序的入口点，负责初始化和启动控制器。

```go
package main

import (
    "flag"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"
    "time"

    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
    "k8s.io/klog/v2"

    "pod-annotator/controller"
)

func main() {
    // ============================================
    // 步骤 1：初始化日志
    // ============================================
    // klog 是 Kubernetes 的标准日志库
    // InitFlags 注册命令行参数（如 -v 控制日志级别）
    klog.InitFlags(nil)
    flag.Parse()

    // ============================================
    // 步骤 2：获取 Kubernetes 配置
    // ============================================
    config, err := getConfig()
    if err != nil {
        klog.Fatalf("获取配置失败: %v", err)
    }

    // ============================================
    // 步骤 3：创建 Clientset
    // ============================================
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        klog.Fatalf("创建 clientset 失败: %v", err)
    }

    // ============================================
    // 步骤 4：创建 Informer 工厂
    // ============================================
    // time.Second*30 是 resync 周期
    // 即每 30 秒会触发一次全量同步（所有资源触发 UpdateFunc）
    informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)

    // ============================================
    // 步骤 5：创建控制器
    // ============================================
    // 传入 clientset 和 Pod Informer
    ctrl := controller.NewController(
        clientset,
        informerFactory.Core().V1().Pods(),
    )

    // ============================================
    // 步骤 6：设置优雅关闭
    // ============================================
    // stopCh 用于控制程序退出
    // 当收到 SIGINT/SIGTERM 时，会关闭 stopCh
    stopCh := setupSignalHandler()

    // ============================================
    // 步骤 7：启动 Informer
    // ============================================
    // Start 会启动所有注册的 Informer
    // 它们会在后台 goroutine 中运行
    informerFactory.Start(stopCh)

    // ============================================
    // 步骤 8：运行控制器
    // ============================================
    // 参数 2 表示启动 2 个 worker 并行处理
    // Run 会阻塞，直到 stopCh 关闭
    if err = ctrl.Run(2, stopCh); err != nil {
        klog.Fatalf("控制器运行失败: %v", err)
    }
}

// getConfig 获取 Kubernetes 配置
// 支持两种模式：
// 1. In-Cluster：控制器运行在 Pod 中，使用 ServiceAccount
// 2. Out-of-Cluster：本地开发，使用 kubeconfig 文件
func getConfig() (*rest.Config, error) {
    // 首先尝试 In-Cluster 配置
    // 如果程序运行在 Pod 中，这会成功
    config, err := rest.InClusterConfig()
    if err == nil {
        klog.Info("使用 In-Cluster 配置")
        return config, nil
    }

    // 回退到 kubeconfig 文件
    // 通常在 ~/.kube/config
    klog.Info("使用 kubeconfig 配置")
    var kubeconfig string
    if home := homedir.HomeDir(); home != "" {
        kubeconfig = filepath.Join(home, ".kube", "config")
    }
    
    return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// setupSignalHandler 设置信号处理
// 实现优雅关闭
func setupSignalHandler() <-chan struct{} {
    stopCh := make(chan struct{})
    
    // 创建信号 channel
    c := make(chan os.Signal, 2)
    // 监听 SIGINT（Ctrl+C）和 SIGTERM（kill）
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        // 第一次收到信号：优雅关闭
        <-c
        klog.Info("📥 收到停止信号，开始优雅关闭...")
        close(stopCh)  // 关闭 stopCh，触发控制器停止
        
        // 第二次收到信号：强制退出
        <-c
        klog.Info("📥 收到第二个停止信号，强制退出")
        os.Exit(1)
    }()
    
    return stopCh
}
```

### 启动流程图

```
main() 执行流程：

    ┌─────────────────┐
    │  初始化日志     │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  获取 K8s 配置  │ ─── In-Cluster 或 kubeconfig
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  创建 Clientset │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  创建 Informer  │
    │  工厂           │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  创建控制器     │
    │  (注册事件处理) │
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  设置信号处理   │ ─── 监听 Ctrl+C / SIGTERM
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  启动 Informer  │ ─── 开始 List & Watch
    └────────┬────────┘
             │
             ▼
    ┌─────────────────┐
    │  运行控制器     │ ─── 等待缓存同步 → 启动 Worker
    │  (阻塞)         │
    └─────────────────┘
```

## 运行和测试

### 本地运行

```bash
# 编译
go build -o pod-annotator .

# 运行
./pod-annotator -v=2
```

### 创建测试 Pod

```bash
# 创建 Pod
kubectl run test-pod --image=nginx

# 查看注解
kubectl get pod test-pod -o jsonpath='{.metadata.annotations}'
```

### 部署到集群

```yaml
# deployment.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pod-annotator
  namespace: default

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-annotator
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch", "update", "patch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: pod-annotator
subjects:
- kind: ServiceAccount
  name: pod-annotator
  namespace: default
roleRef:
  kind: ClusterRole
  name: pod-annotator
  apiGroup: rbac.authorization.k8s.io

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pod-annotator
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pod-annotator
  template:
    metadata:
      labels:
        app: pod-annotator
    spec:
      serviceAccountName: pod-annotator
      containers:
      - name: controller
        image: pod-annotator:latest
        imagePullPolicy: IfNotPresent
```

### 构建容器镜像

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o pod-annotator .

FROM alpine:3.18
COPY --from=builder /app/pod-annotator /pod-annotator
ENTRYPOINT ["/pod-annotator"]
```

```bash
# 构建镜像
docker build -t pod-annotator:latest .

# 如果使用 minikube
minikube image load pod-annotator:latest

# 部署
kubectl apply -f deployment.yaml
```

## 把示例补强到更接近生产

下面几项不必一次写进第一版，但上线前应有意识补齐。

### 1. UpdateFunc 跳过无意义更新

```go
UpdateFunc: func(oldObj, newObj interface{}) {
    oldPod := oldObj.(*corev1.Pod)
    newPod := newObj.(*corev1.Pod)
    if oldPod.ResourceVersion == newPod.ResourceVersion {
        return // resync 触发的“假更新”
    }
    controller.enqueuePod(newPod)
},
```

### 2. 打 Event

在 `addAnnotation` 成功/失败后：

```go
c.recorder.Event(pod, corev1.EventTypeNormal, "Annotated", "added processing annotation")
c.recorder.Eventf(pod, corev1.EventTypeWarning, "AnnotateFailed", "update failed: %v", err)
```

构造方式见 [常用机制与工具包](./07-common-mechanisms.md)。

### 3. 错误分类

| 错误 | 建议 |
|------|------|
| NotFound | 返回 nil，不必重试 |
| Conflict | `RetryOnConflict` 或返回 error 让队列限速重试 |
| Forbidden / Invalid | 打 Warning Event，Forget，避免死循环打爆 API |
| 超时 / 5xx | 返回 error，`AddRateLimited` |

### 4. 启动顺序 checklist

1. 造 `rest.Config`（QPS、UserAgent）
2. 造 Clientset
3. 造 SharedInformerFactory，**先** `AddEventHandler` / 注册索引
4. `factory.Start(stopCh)`
5. `WaitForCacheSync`
6. 启动 workers
7. 多副本时外包 Leader Election

### 5. 单测思路

用 `fake.NewSimpleClientset` 塞入 Pod，直接调 `syncHandler("default/p")`，断言注解存在。Informer 集成可用更重的 envtest；本模块机制说明见 [07](./07-common-mechanisms.md)，排障清单见 [08](./08-debugging-and-pitfalls.md)。

## 扩展建议


把本章控制器补强到更接近生产形态时，优先加这些（机制说明见后续章节）：

1. **EventRecorder**：`kubectl describe` 可见的 Normal/Warning 事件 — [常用机制](./07-common-mechanisms.md)
2. **Leader Election**：多副本只跑一个活跃循环 — 同上，深度见 [11-05](../11-controller-deep-dive/05-leader-election-and-finalizers.md)
3. **RetryOnConflict / DeepCopy**：写路径别污染 Informer 缓存 — [CRUD 写路径](./03-crud-operations.md)
4. **Metrics / healthz**：就绪探针与队列深度
5. **CRD + Operator**：把「注解 Pod」升级成领域对象 — 整专题在 [自定义资源](../07-custom-resources/README.md)

## 你已经串起来的能力

- Clientset 读写作
- Informer 缓存与事件
- WorkQueue 去重、限速、重试
- 标准控制循环（enqueue → syncHandler）

## 下一步

- [Discovery 与 Dynamic Client](./06-discovery-and-dynamic.md)
- [常用机制与工具包](./07-common-mechanisms.md)
- [排障与生产清单](./08-debugging-and-pitfalls.md)
- [控制器深度专题](../11-controller-deep-dive/README.md)
- [模块总览](./README.md)
