# 🔧 客户端配置与连接

## 配置方式

### 1. 从 kubeconfig 文件

```go
package main

import (
    "path/filepath"
    
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

func NewClientFromKubeconfig() (*kubernetes.Clientset, error) {
    // 获取 kubeconfig 路径
    kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
    
    // 构建配置
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        return nil, err
    }
    
    // 创建客户端
    return kubernetes.NewForConfig(config)
}
```

### 2. 从集群内部（In-Cluster）

```go
package main

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

func NewClientInCluster() (*kubernetes.Clientset, error) {
    // 自动读取 Pod 的 ServiceAccount Token
    config, err := rest.InClusterConfig()
    if err != nil {
        return nil, err
    }
    
    return kubernetes.NewForConfig(config)
}
```

### 3. 自动检测（推荐）

```go
package main

import (
    "path/filepath"
    
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

// NewClient 自动选择配置方式
func NewClient() (*kubernetes.Clientset, error) {
    var config *rest.Config
    var err error
    
    // 首先尝试 In-Cluster 配置
    config, err = rest.InClusterConfig()
    if err != nil {
        // 回退到 kubeconfig
        kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
            return nil, err
        }
    }
    
    return kubernetes.NewForConfig(config)
}
```

### 4. 手动配置

```go
package main

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

func NewClientManual() (*kubernetes.Clientset, error) {
    config := &rest.Config{
        Host:        "https://kubernetes.default.svc",
        BearerToken: "your-token",
        TLSClientConfig: rest.TLSClientConfig{
            CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
            // 或者跳过证书验证（不推荐生产使用）
            // Insecure: true,
        },
    }
    
    return kubernetes.NewForConfig(config)
}
```

## 配置选项

### rest.Config 常用字段

```go
config := &rest.Config{
    // 基本配置
    Host:        "https://api.k8s.example.com:6443",
    BearerToken: "token",
    Username:    "user",
    Password:    "password",
    
    // TLS 配置
    TLSClientConfig: rest.TLSClientConfig{
        Insecure: false,                    // 是否跳过证书验证
        CAFile:   "/path/to/ca.crt",        // CA 证书文件
        CAData:   []byte("..."),            // CA 证书内容
        CertFile: "/path/to/client.crt",    // 客户端证书
        KeyFile:  "/path/to/client.key",    // 客户端私钥
    },
    
    // 性能配置
    QPS:   100,   // 每秒请求数
    Burst: 200,   // 突发请求数
    
    // 超时配置
    Timeout: 30 * time.Second,
}
```

### 高 QPS 配置

```go
func NewHighQPSClient() (*kubernetes.Clientset, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
        return nil, err
    }
    
    // 提高 QPS 限制
    config.QPS = 100
    config.Burst = 200
    
    return kubernetes.NewForConfig(config)
}
```

## 多集群配置

### 从多个 kubeconfig 文件

```go
package main

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func NewClientFromMultipleConfigs(configFiles ...string) (*kubernetes.Clientset, error) {
    // 加载多个配置文件
    loadingRules := &clientcmd.ClientConfigLoadingRules{
        Precedence: configFiles,
    }
    
    configOverrides := &clientcmd.ConfigOverrides{}
    kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
    
    config, err := kubeConfig.ClientConfig()
    if err != nil {
        return nil, err
    }
    
    return kubernetes.NewForConfig(config)
}
```

### 切换 Context

```go
package main

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func NewClientWithContext(kubeconfig, context string) (*kubernetes.Clientset, error) {
    loadingRules := &clientcmd.ClientConfigLoadingRules{
        ExplicitPath: kubeconfig,
    }
    
    configOverrides := &clientcmd.ConfigOverrides{
        CurrentContext: context,
    }
    
    kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
    
    config, err := kubeConfig.ClientConfig()
    if err != nil {
        return nil, err
    }
    
    return kubernetes.NewForConfig(config)
}

// 使用
func main() {
    // 连接到 production context
    prodClient, _ := NewClientWithContext("/path/to/kubeconfig", "production-context")
    
    // 连接到 staging context
    stagingClient, _ := NewClientWithContext("/path/to/kubeconfig", "staging-context")
}
```

## 完整客户端封装

```go
package client

import (
    "path/filepath"
    "sync"
    
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
)

// K8sClient 封装 Kubernetes 客户端
type K8sClient struct {
    config        *rest.Config
    clientset     *kubernetes.Clientset
    dynamicClient dynamic.Interface
}

var (
    instance *K8sClient
    once     sync.Once
)

// GetClient 获取单例客户端
func GetClient() (*K8sClient, error) {
    var err error
    once.Do(func() {
        instance, err = newClient()
    })
    return instance, err
}

func newClient() (*K8sClient, error) {
    // 获取配置
    config, err := getConfig()
    if err != nil {
        return nil, err
    }
    
    // 配置 QPS
    config.QPS = 100
    config.Burst = 200
    
    // 创建 Clientset
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, err
    }
    
    // 创建 Dynamic Client
    dynamicClient, err := dynamic.NewForConfig(config)
    if err != nil {
        return nil, err
    }
    
    return &K8sClient{
        config:        config,
        clientset:     clientset,
        dynamicClient: dynamicClient,
    }, nil
}

func getConfig() (*rest.Config, error) {
    // 尝试 In-Cluster
    config, err := rest.InClusterConfig()
    if err == nil {
        return config, nil
    }
    
    // 回退到 kubeconfig
    kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
    return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// Clientset 返回 kubernetes.Clientset
func (c *K8sClient) Clientset() *kubernetes.Clientset {
    return c.clientset
}

// DynamicClient 返回 dynamic.Interface
func (c *K8sClient) DynamicClient() dynamic.Interface {
    return c.dynamicClient
}

// Config 返回 rest.Config
func (c *K8sClient) Config() *rest.Config {
    return c.config
}
```

### 使用示例

```go
package main

import (
    "context"
    "fmt"
    
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    
    "my-k8s-client/pkg/client"
)

func main() {
    // 获取客户端
    k8sClient, err := client.GetClient()
    if err != nil {
        panic(err)
    }
    
    // 使用 Clientset
    pods, err := k8sClient.Clientset().CoreV1().Pods("default").List(
        context.TODO(), 
        metav1.ListOptions{},
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("找到 %d 个 Pod\n", len(pods.Items))
}
```

## 下一步

- [资源的 CRUD 操作](./03-crud-operations.md)



