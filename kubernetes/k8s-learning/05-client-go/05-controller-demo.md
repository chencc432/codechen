# 🎮 实战项目：自定义控制器

## 项目目标

创建一个简单的控制器，监控 Pod 并在 Pod 创建时自动添加一个注解。

## 控制器模式

```
┌─────────────────────────────────────────────────────────────────────┐
│                      控制器模式                                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   ┌───────────────┐                                                 │
│   │   Informer    │                                                 │
│   │ (监听资源变化) │                                                 │
│   └───────┬───────┘                                                 │
│           │ 事件                                                     │
│           ▼                                                          │
│   ┌───────────────┐                                                 │
│   │   WorkQueue   │                                                 │
│   │  (任务队列)    │                                                 │
│   └───────┬───────┘                                                 │
│           │ key                                                      │
│           ▼                                                          │
│   ┌───────────────┐       ┌───────────────┐                        │
│   │    Worker     │──────>│  SyncHandler  │                        │
│   │ (消费任务)     │       │ (业务逻辑)     │                        │
│   └───────────────┘       └───────┬───────┘                        │
│                                   │                                  │
│                                   ▼                                  │
│                           ┌───────────────┐                        │
│                           │ Kubernetes API│                        │
│                           │  (更新资源)    │                        │
│                           └───────────────┘                        │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## 完整代码

### 项目结构

```
pod-annotator/
├── go.mod
├── go.sum
├── main.go
└── controller/
    └── controller.go
```

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

```go
package controller

import (
    "context"
    "fmt"
    "time"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    coreinformers "k8s.io/client-go/informers/core/v1"
    "k8s.io/client-go/kubernetes"
    corelisters "k8s.io/client-go/listers/core/v1"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/util/workqueue"
    "k8s.io/klog/v2"
)

const (
    // 注解键
    AnnotationKey = "pod-annotator.example.com/processed"
    // 控制器名称
    ControllerName = "pod-annotator"
)

// Controller 是 Pod 注解控制器
type Controller struct {
    // clientset 用于与 Kubernetes API 交互
    clientset kubernetes.Interface

    // podLister 用于从缓存读取 Pod
    podLister corelisters.PodLister
    
    // podsSynced 表示 Pod Informer 缓存是否同步完成
    podsSynced cache.InformerSynced

    // workqueue 是限速工作队列
    workqueue workqueue.RateLimitingInterface
}

// NewController 创建新的控制器
func NewController(
    clientset kubernetes.Interface,
    podInformer coreinformers.PodInformer,
) *Controller {
    
    controller := &Controller{
        clientset:  clientset,
        podLister:  podInformer.Lister(),
        podsSynced: podInformer.Informer().HasSynced,
        workqueue:  workqueue.NewNamedRateLimitingQueue(
            workqueue.DefaultControllerRateLimiter(),
            ControllerName,
        ),
    }

    klog.Info("设置事件处理器")
    
    // 添加事件处理器
    podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: controller.enqueuePod,
        UpdateFunc: func(old, new interface{}) {
            controller.enqueuePod(new)
        },
    })

    return controller
}

// enqueuePod 将 Pod 加入工作队列
func (c *Controller) enqueuePod(obj interface{}) {
    var key string
    var err error
    
    if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
        utilruntime.HandleError(err)
        return
    }
    
    c.workqueue.Add(key)
}

// Run 启动控制器
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
    defer utilruntime.HandleCrash()
    defer c.workqueue.ShutDown()

    klog.Info("启动 Pod 注解控制器")

    // 等待缓存同步
    klog.Info("等待 Informer 缓存同步...")
    if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
        return fmt.Errorf("缓存同步失败")
    }

    klog.Info("缓存同步完成，启动 workers")

    // 启动多个 worker
    for i := 0; i < workers; i++ {
        go wait.Until(c.runWorker, time.Second, stopCh)
    }

    klog.Info("Workers 已启动")
    <-stopCh
    klog.Info("关闭 workers")

    return nil
}

// runWorker 运行单个 worker
func (c *Controller) runWorker() {
    for c.processNextWorkItem() {
    }
}

// processNextWorkItem 处理队列中的下一个任务
func (c *Controller) processNextWorkItem() bool {
    obj, shutdown := c.workqueue.Get()

    if shutdown {
        return false
    }

    // 处理完成后标记 Done
    err := func(obj interface{}) error {
        defer c.workqueue.Done(obj)
        
        var key string
        var ok bool
        
        if key, ok = obj.(string); !ok {
            // 无效的任务，直接丢弃
            c.workqueue.Forget(obj)
            utilruntime.HandleError(fmt.Errorf("期望 string 类型，但收到 %#v", obj))
            return nil
        }

        // 执行同步逻辑
        if err := c.syncHandler(key); err != nil {
            // 处理失败，重新入队
            c.workqueue.AddRateLimited(key)
            return fmt.Errorf("同步 '%s' 失败: %s，重新入队", key, err.Error())
        }

        // 处理成功，清除重试计数
        c.workqueue.Forget(obj)
        klog.Infof("成功同步 '%s'", key)
        return nil
    }(obj)

    if err != nil {
        utilruntime.HandleError(err)
        return true
    }

    return true
}

// syncHandler 是核心业务逻辑
func (c *Controller) syncHandler(key string) error {
    // 解析 namespace/name
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        utilruntime.HandleError(fmt.Errorf("无效的资源 key: %s", key))
        return nil
    }

    // 从缓存获取 Pod
    pod, err := c.podLister.Pods(namespace).Get(name)
    if err != nil {
        // Pod 已删除，忽略
        if errors.IsNotFound(err) {
            utilruntime.HandleError(fmt.Errorf("Pod '%s' 在工作队列中，但已不存在", key))
            return nil
        }
        return err
    }

    // 检查是否已处理
    if pod.Annotations != nil {
        if _, exists := pod.Annotations[AnnotationKey]; exists {
            klog.V(4).Infof("Pod %s/%s 已处理，跳过", namespace, name)
            return nil
        }
    }

    // 跳过系统 Pod
    if namespace == "kube-system" {
        return nil
    }

    // 添加注解
    return c.addAnnotation(pod)
}

// addAnnotation 为 Pod 添加注解
func (c *Controller) addAnnotation(pod *corev1.Pod) error {
    // 创建副本以避免修改缓存
    podCopy := pod.DeepCopy()
    
    if podCopy.Annotations == nil {
        podCopy.Annotations = make(map[string]string)
    }
    
    // 添加注解
    podCopy.Annotations[AnnotationKey] = time.Now().Format(time.RFC3339)

    // 更新 Pod
    _, err := c.clientset.CoreV1().Pods(pod.Namespace).Update(
        context.TODO(),
        podCopy,
        metav1.UpdateOptions{},
    )
    
    if err != nil {
        return fmt.Errorf("更新 Pod %s/%s 失败: %v", pod.Namespace, pod.Name, err)
    }
    
    klog.Infof("成功为 Pod %s/%s 添加注解", pod.Namespace, pod.Name)
    return nil
}
```

### main.go

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
    // 初始化 klog
    klog.InitFlags(nil)
    flag.Parse()

    // 获取配置
    config, err := getConfig()
    if err != nil {
        klog.Fatalf("获取配置失败: %v", err)
    }

    // 创建 clientset
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        klog.Fatalf("创建 clientset 失败: %v", err)
    }

    // 创建 Informer 工厂
    informerFactory := informers.NewSharedInformerFactory(clientset, time.Second*30)

    // 创建控制器
    ctrl := controller.NewController(
        clientset,
        informerFactory.Core().V1().Pods(),
    )

    // 设置信号处理
    stopCh := setupSignalHandler()

    // 启动 Informer
    informerFactory.Start(stopCh)

    // 运行控制器
    if err = ctrl.Run(2, stopCh); err != nil {
        klog.Fatalf("控制器运行失败: %v", err)
    }
}

func getConfig() (*rest.Config, error) {
    // 尝试 In-Cluster 配置
    config, err := rest.InClusterConfig()
    if err == nil {
        return config, nil
    }

    // 回退到 kubeconfig
    var kubeconfig string
    if home := homedir.HomeDir(); home != "" {
        kubeconfig = filepath.Join(home, ".kube", "config")
    }
    
    return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func setupSignalHandler() <-chan struct{} {
    stopCh := make(chan struct{})
    
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        klog.Info("收到停止信号")
        close(stopCh)
        <-c
        klog.Info("收到第二个停止信号，强制退出")
        os.Exit(1)
    }()
    
    return stopCh
}
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

## 扩展建议

1. **添加 Metrics**：暴露 Prometheus 指标
2. **Leader Election**：多副本时使用 Leader Election
3. **Webhook**：使用 Admission Webhook 实现更强大的控制
4. **自定义资源**：使用 CRD 扩展功能

## 恭喜！

你已经完成了 client-go 的学习！现在你可以：

- 使用 Clientset 进行 CRUD 操作
- 使用 Informer 高效监听资源变化
- 使用 WorkQueue 实现控制器模式
- 开发自己的 Kubernetes 控制器

## 下一步

返回 [课程主页](../README.md) 查看更多内容。



