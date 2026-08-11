# 客户端配置与连接

所有 client-go 调用都建立在 `rest.Config` 之上：鉴权、TLS、超时、QPS、User-Agent 都在这里定调。配错了，后面 Informer / 控制器再漂亮也稳不住。

本章覆盖：配置来源、关键字段、限流与超时、多集群、生产封装建议。

## 配置从哪来

### 1. kubeconfig（本地 / CI）

```go
kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
```

也可用环境变量 `KUBECONFIG`（多文件用 `:` / `;` 分隔）。`BuildConfigFromFlags(masterURL, kubeconfigPath)` 的第一个参数一般留空，除非你要强制覆盖 API Server 地址。

### 2. In-Cluster（跑在 Pod 里）

```go
config, err := rest.InClusterConfig()
```

自动读取：

| 路径 / 环境 | 内容 |
|-------------|------|
| `/var/run/secrets/kubernetes.io/serviceaccount/token` | SA JWT |
| `.../ca.crt` | 集群 CA |
| `KUBERNETES_SERVICE_HOST/PORT` | API 地址 |

权限完全取决于挂载的 ServiceAccount + RBAC，不是“进了集群就万能”。

### 3. 自动回退（推荐写法）

```go
func buildConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
```

先 In-Cluster，失败再 kubeconfig：同一份二进制本地调试、集群部署都能跑。

### 4. 手动拼装（特殊场景）

仅在你明确持有 token/证书、或对接非标准入口时使用：

```go
config := &rest.Config{
	Host:        "https://kubernetes.default.svc",
	BearerToken: token,
	TLSClientConfig: rest.TLSClientConfig{
		CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
	},
}
```

`Insecure: true` 只适合临时排障，不要进生产。

## rest.Config 关键字段

```go
config := &rest.Config{
	Host:        "https://api.example.com:6443",
	BearerToken: "token",
	Username:    "user", // 与 Password 一起用；现代集群更常见 token / exec / cert
	Password:    "password",

	TLSClientConfig: rest.TLSClientConfig{
		Insecure: false,
		CAFile:   "/path/ca.crt",
		CertFile: "/path/client.crt",
		KeyFile:  "/path/client.key",
	},

	// 客户端侧令牌桶限流（每个 rest.Config / 客户端一份）
	QPS:   50,
	Burst: 100,

	// 单次请求超时（含等响应）；Watch 长连接要特别小心
	Timeout: 30 * time.Second,

	UserAgent: "my-controller/1.2.3",
}
```

### QPS / Burst 怎么理解

- **QPS**：稳态每秒请求上限（client-go 本地令牌桶）
- **Burst**：短时间可打出的峰值

默认大约 `QPS=5, Burst=10`，控制器稍忙就容易自己把自己限死，表现为 Reconcile 变慢、队列堆积。

经验：

| 场景 | 起点 |
|------|------|
| 小工具 / 脚本 | 默认或略提高 |
| 单资源控制器 | `QPS=20~50, Burst=40~100` |
| 多资源 / 大规模 | `50~100+`，并配合命名空间/标签过滤减流量 |

注意：这是**客户端**限流。API Server 还有 Priority and Fairness / max-requests-inflight。把 QPS 开到 1000 不等于集群扛得住，也可能拖垮别人。

Informer 的 List/Watch 也走同一套限流；factory 很多时要统一规划，避免每个组件各建一份超高 QPS 客户端。

### Timeout 与 Watch

`Timeout` 作用于普通请求。Watch 是长连接，Reflector 有自己的超时/重连逻辑。不要指望靠把 `Timeout` 设成几小时来“养” Watch。

### UserAgent

```go
config.UserAgent = "pod-annotator/1.0.0"
```

审计日志、API 指标里能直接看到是谁在打集群。多组件共用一个匿名 UA 时，出问题很难归因。

### WrapTransport

注入 header、metrics、tracing：

```go
config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
	return &uaRoundTripper{base: rt, ua: config.UserAgent}
}
```

详见 [常用机制与工具包](./07-common-mechanisms.md)。

## 鉴权方式速览

| 方式 | 典型来源 | 场景 |
|------|----------|------|
| ServiceAccount token | In-Cluster | 集群内控制器 |
| kubeconfig user.exec | `aws eks get-token` 等 | 云厂商本地登录 |
| 客户端证书 | kubeconfig `client-certificate` | 老式/私有集群 |
| Bearer token | 手动 / OIDC | CI、调试 |
| Impersonation | `config.Impersonate` | 平台代用户操作（需权限） |

```go
config.Impersonate = rest.ImpersonationConfig{
	UserName: "jane@example.com",
	Groups:   []string{"system:authenticated", "devs"},
}
```

只有当你自己的身份拥有 `impersonate` 权限时才有效；审计会留下“A 假扮 B”。

## 多集群与 Context

### 按 Context 建客户端

```go
func NewClientWithContext(kubeconfig, context string) (*kubernetes.Clientset, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}
```

多集群平台常见模式：`map[clusterName]*Clientset`，每个集群独立 InformerFactory，**不要**假设一个 factory 能跨集群。

### 多文件合并

```go
loadingRules := &clientcmd.ClientConfigLoadingRules{
	Precedence: []string{"/etc/k8s/a.conf", "/etc/k8s/b.conf"},
}
```

## 创建客户端的正确姿势

```go
// 一般够用：内部自建 HTTP Client
clientset, err := kubernetes.NewForConfig(config)

// 需要共享 Transport / 自定义 http.Client 时：
httpClient, err := rest.HTTPClientFor(config)
clientset, err = kubernetes.NewForConfigAndClient(config, httpClient)
dyn, err := dynamic.NewForConfigAndClient(config, httpClient)
```

同一进程里 Clientset + Dynamic + Metadata 建议：

1. 一份 `rest.Config`（拷贝后再改 QPS，避免互相覆盖）
2. 尽量共享 `http.Client` / Transport，减少连接数
3. 给不同组件设不同 UserAgent 后缀（如 `my-op/informer`、`my-op/worker`）便于观察

```go
cfg := rest.CopyConfig(base)
cfg.QPS, cfg.Burst = 50, 100
cfg.UserAgent = rest.DefaultKubernetesUserAgent() + "/pod-annotator"
```

## 生产向封装示例

```go
package client

import (
	"fmt"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Clients struct {
	Config    *rest.Config
	Clientset kubernetes.Interface
	Dynamic   dynamic.Interface
}

func New(appName string, qps float32, burst int) (*Clients, error) {
	cfg, err := buildConfig()
	if err != nil {
		return nil, err
	}
	cfg = rest.CopyConfig(cfg)
	cfg.QPS, cfg.Burst = qps, burst
	cfg.UserAgent = fmt.Sprintf("%s/%s", appName, "1.0.0")

	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfigAndClient(cfg, httpClient)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfigAndClient(cfg, httpClient)
	if err != nil {
		return nil, err
	}
	return &Clients{Config: cfg, Clientset: cs, Dynamic: dyn}, nil
}

func buildConfig() (*rest.Config, error) {
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	return clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
}
```

是否做成进程级单例看场景：测试里单例很难隔离；长期运行的控制器用“启动时构造一次、往下注入”通常比全局 `sync.Once` 更清晰。

## 常见配置问题

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| `Unauthorized` / `401` | token 过期、kubeconfig 用户错 | 重登；检查 SA secret 是否仍挂载 |
| `certificate signed by unknown authority` | CA 不匹配 | 换正确 CA；别随手 Insecure |
| 请求极慢、队列堆积 | QPS 太低 | 提高 QPS/Burst，并减少无谓 List |
| `context deadline exceeded` | Timeout 过短或 API 卡住 | 区分单次超时 vs 集群故障 |
| In-Cluster 失败 | 不在 Pod 内 / 无 SA 卷 | 本地改走 kubeconfig |
| `Forbidden` | RBAC | 补 Role/ClusterRole 动词与资源 |

## 下一步

- [CRUD 与写路径机制](./03-crud-operations.md)
