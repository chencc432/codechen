# 📚 client-go 入门

## 什么是 client-go？

`client-go` 是 Kubernetes 官方提供的 Go 语言客户端库，用于与 Kubernetes API Server 进行交互。

```
┌─────────────────────────────────────────────────────────────────────┐
│                      client-go 架构                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   你的 Go 程序                                                        │
│       │                                                               │
│       ▼                                                               │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │                     client-go                                │  │
│   │                                                               │  │
│   │  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │  │
│   │  │   Clientset   │  │   Informer    │  │    Lister     │   │  │
│   │  │  (类型安全)    │  │  (缓存+事件)   │  │  (本地查询)    │   │  │
│   │  └───────────────┘  └───────────────┘  └───────────────┘   │  │
│   │                                                               │  │
│   │  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │  │
│   │  │ RESTClient    │  │  WorkQueue    │  │   Discovery   │   │  │
│   │  │  (底层HTTP)    │  │  (任务队列)    │  │  (API发现)    │   │  │
│   │  └───────────────┘  └───────────────┘  └───────────────┘   │  │
│   │                                                               │  │
│   └─────────────────────────────────────────────────────────────┘  │
│                               │                                      │
│                               ▼                                      │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │                    Kubernetes API Server                     │  │
│   └─────────────────────────────────────────────────────────────┘  │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## 核心组件

| 组件 | 说明 | 用途 |
|------|------|------|
| **Clientset** | 类型安全的客户端集合 | CRUD 操作 |
| **Informer** | 带缓存的事件监听器 | 高效地监听资源变化 |
| **Lister** | 从本地缓存读取数据 | 避免频繁请求 API |
| **WorkQueue** | 任务队列 | 控制器开发 |
| **RESTClient** | 底层 REST 客户端 | 自定义请求 |
| **Discovery** | API 发现客户端 | 动态获取 API 信息 |

## 环境准备

### 创建 Go 项目

```bash
# 创建项目目录
mkdir my-k8s-client
cd my-k8s-client

# 初始化 Go 模块
go mod init my-k8s-client
```

### 安装依赖

```bash
# 安装 client-go
go get k8s.io/client-go@latest
go get k8s.io/apimachinery@latest

# 查看版本对应关系
# Kubernetes 1.28 -> client-go v0.28.x
# Kubernetes 1.29 -> client-go v0.29.x
# Kubernetes 1.30 -> client-go v0.30.x
```

### go.mod 示例

```go
module my-k8s-client

go 1.21

require (
    k8s.io/apimachinery v0.29.0
    k8s.io/client-go v0.29.0
)
```

## 客户端类型

### 1. Clientset（推荐）

类型安全的客户端，用于操作内置资源。

```go
import (
    "k8s.io/client-go/kubernetes"
)

// 使用 Clientset
clientset, err := kubernetes.NewForConfig(config)
if err != nil {
    panic(err)
}

// 操作 Pod
pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})

// 操作 Deployment
deploys, err := clientset.AppsV1().Deployments("default").List(ctx, metav1.ListOptions{})
```

### 2. Dynamic Client

用于操作任意资源（包括 CRD）。

```go
import (
    "k8s.io/client-go/dynamic"
)

// 使用 Dynamic Client
dynamicClient, err := dynamic.NewForConfig(config)
if err != nil {
    panic(err)
}

// 定义资源
gvr := schema.GroupVersionResource{
    Group:    "",
    Version:  "v1",
    Resource: "pods",
}

// 操作资源
pods, err := dynamicClient.Resource(gvr).Namespace("default").List(ctx, metav1.ListOptions{})
```

### 3. RESTClient

最底层的客户端，直接发送 HTTP 请求。

```go
import (
    "k8s.io/client-go/rest"
)

// 使用 RESTClient
restClient, err := rest.RESTClientFor(config)
if err != nil {
    panic(err)
}

// 发送请求
result := restClient.Get().
    Namespace("default").
    Resource("pods").
    Name("nginx").
    Do(ctx)
```

## 第一个程序：列出所有 Pod

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "path/filepath"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

func main() {
    // 解析 kubeconfig 路径
    var kubeconfig *string
    if home := homedir.HomeDir(); home != "" {
        kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "kubeconfig 文件路径")
    } else {
        kubeconfig = flag.String("kubeconfig", "", "kubeconfig 文件路径")
    }
    flag.Parse()

    // 构建配置
    config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
    if err != nil {
        panic(err)
    }

    // 创建 clientset
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    // 列出所有命名空间的 Pod
    pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        panic(err)
    }

    fmt.Printf("找到 %d 个 Pod:\n", len(pods.Items))
    for _, pod := range pods.Items {
        fmt.Printf("  - %s/%s (状态: %s)\n", pod.Namespace, pod.Name, pod.Status.Phase)
    }
}
```

### 运行程序

```bash
# 编译运行
go run main.go

# 指定 kubeconfig
go run main.go --kubeconfig=/path/to/kubeconfig
```

## In-Cluster 配置

在 Pod 内运行时，使用 In-Cluster 配置：

```go
import (
    "k8s.io/client-go/rest"
)

func getConfig() (*rest.Config, error) {
    // 尝试 In-Cluster 配置
    config, err := rest.InClusterConfig()
    if err != nil {
        // 回退到 kubeconfig
        kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
            return nil, err
        }
    }
    return config, nil
}
```

## 项目结构建议

```
my-k8s-client/
├── go.mod
├── go.sum
├── main.go
├── pkg/
│   ├── client/
│   │   └── client.go      # 客户端初始化
│   ├── handlers/
│   │   └── pods.go        # Pod 操作
│   └── informers/
│       └── informer.go    # Informer 相关
└── examples/
    ├── list_pods/
    ├── create_deploy/
    └── watch_events/
```

## 常见错误

### 1. 版本不匹配

```
cannot use xxx (type xxx) as type xxx
```

解决：确保 client-go 和 apimachinery 版本一致

```bash
go get k8s.io/client-go@v0.29.0
go get k8s.io/apimachinery@v0.29.0
go get k8s.io/api@v0.29.0
go mod tidy
```

### 2. 连接失败

```
Unable to connect to the server
```

解决：检查 kubeconfig 配置和集群连接

### 3. 权限不足

```
forbidden: User "xxx" cannot list resource "pods"
```

解决：配置正确的 RBAC 权限

## 下一步

- [客户端配置与连接](./02-client-setup.md)



