# client-go 入门：机制全景

`client-go` 是 Kubernetes 官方 Go 客户端。写自动化工具、Operator、平台组件，几乎都绕不开它。

本篇不急着堆 CRUD 示例，先把**整套机制地图**讲清楚：每块解决什么问题、彼此怎么配合、什么时候该用哪一块。后面章节再落到代码。

## 先建立心智模型

访问 API Server 只有几件事：

1. **读一次**：List / Get
2. **持续读**：Watch（带 `resourceVersion` 的增量流）
3. **写**：Create / Update / Patch / Delete / Apply
4. **发现能力**：集群里有哪些 Group/Version/Resource（Discovery）
5. **在本地跟上变化**：Informer（List+Watch + 本地缓存 + 回调）
6. **可靠地处理变化**：WorkQueue（去重、限速、重试）+ 控制器循环

Informer / WorkQueue / Controller 不是“高级装饰”，而是官方控制器（Deployment、ReplicaSet 等）同一套骨架。学 client-go，本质上是学这套**控制面编程模型**。

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         你的程序                                          │
│                                                                            │
│   一次性脚本 / CLI          长期运行的控制器 / Operator                   │
│         │                              │                                   │
│         ▼                              ▼                                   │
│   Clientset / Dynamic           Informer + Lister                          │
│   RESTClient                    WorkQueue + Worker                         │
│   Discovery / RESTMapper        EventRecorder / LeaderElection             │
│         │                              │                                   │
│         └──────────────┬───────────────┘                                   │
│                        ▼                                                   │
│                 rest.Config + HTTP Transport                               │
│                        │                                                   │
│                        ▼                                                   │
│                 Kubernetes API Server                                      │
└──────────────────────────────────────────────────────────────────────────┘
```

## 包与组件地图

按依赖层次看，比按“功能清单”更好记：

| 层次 | 代表包 / 类型 | 职责 |
|------|----------------|------|
| 配置 | `rest.Config`、`clientcmd` | kubeconfig / In-Cluster、QPS、TLS、超时 |
| 传输 | `rest.RESTClient`、`transport` | HTTP、鉴权、User-Agent、重试底层 |
| 类型化 CRUD | `kubernetes.Clientset` | 内置资源强类型 API |
| 动态 CRUD | `dynamic.Interface`、`unstructured` | CRD / 任意 GVR，运行时才知道类型 |
| API 发现 | `discovery`、`restmapper` | 集群支持哪些 API、GVK ↔ GVR 映射 |
| 编解码 | `runtime.Scheme`、`serializer` | Go 类型 ↔ JSON/YAML/Protobuf |
| 监听缓存 | `tools/cache`、`informers` | Reflector、DeltaFIFO、Indexer、SharedInformer |
| 任务队列 | `util/workqueue` | 去重、限速、重试 |
| 辅助 | `tools/record`、`tools/leaderelection`、`util/retry` | Event、选主、冲突重试 |

后面章节会逐个展开；本篇只要求你能指出它们在地图上的位置。

## 四类客户端怎么选

### 1. Clientset（内置资源首选）

```go
clientset, err := kubernetes.NewForConfig(config)
pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
```

- **优点**：编译期类型检查，IDE 友好，官方资源覆盖全
- **缺点**：不知道的 CRD 用不了；生成代码体积大
- **适合**：操作 Pod / Deployment / Service 等内置资源

### 2. Dynamic Client（CRD / 泛型工具）

```go
dyn, err := dynamic.NewForConfig(config)
gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
obj, err := dyn.Resource(gvr).Namespace("default").Get(ctx, "nginx", metav1.GetOptions{})
```

- 返回 `unstructured.Unstructured`（本质是 `map[string]interface{}`）
- **适合**：通用控制器、多 CRD 平台、kubectl 类工具
- **代价**：没有编译期字段检查，字段路径容易写错

### 3. RESTClient（最底层）

直接拼路径、动词、参数。几乎只在封装特殊子资源、非标准端点，或你必须精确控制请求时使用。日常业务优先 Clientset / Dynamic。

### 4. Metadata Client（只要元数据）

只拉取 `PartialObjectMetadata`（name、labels、ownerReferences 等），不反序列化完整 `spec/status`。

- **适合**：大规模集群里只按标签/Owner 做索引，内存敏感的控制器
- 常和 `metadatainformer` 一起用

**经验法则**：

- 已知内置类型 → Clientset
- 任意 / CRD / 插件化 → Dynamic + Discovery/RESTMapper
- 只要 metadata → Metadata client / Metadata Informer
- 需要极致控制 HTTP → RESTClient

## 读路径：List / Watch / Informer

这是整门课最重要的一条线。

### List：快照

一次拿到当前对象集合，附带一个 `resourceVersion`（RV）。RV 是 etcd/API 的一致性锚点，后续 Watch 从某个 RV 接着看。

### Watch：增量流

从某个 RV 开始，服务端推送 `ADDED` / `MODIFIED` / `DELETED` / `BOOKMARK` / `ERROR`。

直接用 Watch 的问题：

- 连接会断，要自己重连
- RV 过期会 `410 Gone`，必须重新 List
- 没有本地缓存，查询仍打 API
- 同一资源多处关心会开多条 Watch

### Informer：官方标准答案

Informer = **Reflector（List+Watch+重连）+ DeltaFIFO + Indexer（本地缓存）+ EventHandler +（可选）Shared 复用**。

```
API Server
    │ List + Watch
    ▼
Reflector ──▶ DeltaFIFO ──▶ Processor
                               │
                    ┌──────────┴──────────┐
                    ▼                     ▼
                 Indexer              EventHandler
                 (Lister 读这里)       (通常只入队 key)
                                          │
                                          ▼
                                      WorkQueue
                                          │
                                          ▼
                                      Worker / Reconcile
```

**结论（请记住）**：

- 临时脚本、一次性查询 → List/Get 即可
- 长期跟着资源变化走 → **用 Informer，不要手写 Watch 循环**
- Handler 里做重活 → **必须配 WorkQueue**，Handler 只入队

详情见 [Informer 机制详解](./04-informer.md)。更底层的 Reflector/DeltaFIFO 行为见 [控制器深度专题](../11-controller-deep-dive/02-informer-internals.md)。

## 写路径：Update / Patch / Apply

| 方式 | 语义 | 典型场景 |
|------|------|----------|
| Update | 整对象替换（乐观锁，靠 `resourceVersion`） | 简单改动、对象较小 |
| Patch | 只提交变更片段 | 改注解、改副本数、并发友好 |
| Apply（SSA） | 声明式字段归属（fieldManager） | 多控制器共建同一对象 |

冲突（`409 Conflict`）在控制器里很常见：别人刚改过同一对象。标准做法是读最新 → 再改 → 重试（`util/retry` / `retry.RetryOnConflict`）。

CRUD 与 Patch/SSA 见 [资源的 CRUD 操作](./03-crud-operations.md)。

## 控制器编程模型（预告）

Kubernetes 控制器是 **level-triggered（电平触发）**，不是 edge-triggered（边沿触发）：

- 你关心的是「当前实际状态是否等于期望」，不是「刚才发生了哪一次事件」
- 事件只是「请你再看一眼」的信号；丢事件也没关系，下次 resync 或再一次 Update 还会再对一遍
- 所以 Reconcile 必须**幂等**

标准骨架：

1. Informer 监听，本地缓存
2. EventHandler 把 `namespace/name` 丢进 WorkQueue
3. Worker 取 key → Lister 读缓存 → 调 API 写回
4. 失败则 `AddRateLimited` 重试

实战见 [自定义控制器](./05-controller-demo.md)；深度见 [控制器深度专题](../11-controller-deep-dive/README.md)。

## 其他必会机制（本课都会覆盖）

下面这些不是“选修彩蛋”，写生产级组件迟早碰到：

| 机制 | 解决什么问题 | 章节 |
|------|----------------|------|
| **Discovery / RESTMapper** | 运行时知道 GVK 对应哪个 REST 路径 | [Discovery 与 Dynamic](./06-discovery-and-dynamic.md) |
| **Scheme** | Go 类型注册与编解码 | [常用机制与工具包](./07-common-mechanisms.md) |
| **EventRecorder** | 在 `kubectl describe` 里留下可读事件 | 同上 |
| **Leader Election** | 多副本只跑一个活跃控制器 | 同上（深度版在 11 专题） |
| **Fake client** | 单测不连真集群 | 同上 |
| **wait / retry** | 轮询条件、冲突重试 | 同上 |
| **排障清单** | 热循环、缓存、选主、RBAC | [排障与生产清单](./08-debugging-and-pitfalls.md) |
| **QPS / Burst / Timeout** | 别打爆 API Server | [客户端配置](./02-client-setup.md) |

## 版本对应关系

`client-go` / `apimachinery` / `api` 三者版本必须对齐，并与集群大版本匹配：

| Kubernetes | client-go / api / apimachinery |
|------------|--------------------------------|
| 1.28 | v0.28.x |
| 1.29 | v0.29.x |
| 1.30 | v0.30.x |
| 1.31 | v0.31.x |

```bash
go get k8s.io/client-go@v0.29.0
go get k8s.io/apimachinery@v0.29.0
go get k8s.io/api@v0.29.0
go mod tidy
```

小版本可以略有差异，但 **major.minor 不一致**时编译错误或运行期序列化问题非常常见。

## 环境准备与第一个程序

### 初始化

```bash
mkdir my-k8s-client && cd my-k8s-client
go mod init my-k8s-client
go get k8s.io/client-go@v0.29.0
go get k8s.io/apimachinery@v0.29.0
go get k8s.io/api@v0.29.0
```

### 列出 Pod（建立手感）

```go
package main

import (
	"context"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("找到 %d 个 Pod\n", len(pods.Items))
	for _, pod := range pods.Items {
		fmt.Printf("  - %s/%s (%s)\n", pod.Namespace, pod.Name, pod.Status.Phase)
	}
}
```

In-Cluster 与配置细节见下一章。集群内运行时优先 `rest.InClusterConfig()`，本地开发再回退 kubeconfig。

## 推荐项目结构

```
my-k8s-client/
├── go.mod
├── main.go
├── pkg/
│   ├── client/          # rest.Config / Clientset / Dynamic 初始化
│   ├── controllers/     # Informer + Queue + Reconcile
│   └── util/            # retry、索引函数等
└── examples/
```

控制器类项目更推荐直接用 Kubebuilder / Operator SDK 生成骨架；本模块用“手写 client-go”是为了先把机制吃透。

## 常见错误速查

| 现象 | 原因 | 处理 |
|------|------|------|
| 类型对不上 / 编译失败 | client-go 与 apimachinery 版本不一致 | 三者对齐到同一 `v0.x.y` |
| `Unable to connect` | kubeconfig / 网络 / TLS | 检查 context、证书、API 地址 |
| `forbidden` | RBAC 不足 | 补 ServiceAccount + Role/ClusterRole |
| Informer Lister 空 | 未 `WaitForCacheSync` | Start 后必须等同步 |
| 疯狂 409 Conflict | 直接改缓存对象或未重试 | `DeepCopy` + `RetryOnConflict` |
| Watch 自己写挂了 | 未处理重连 / 410 | 改用 Informer |

## 本模块学习路线

按顺序读，不要跳着只抄 CRUD：

1. **本篇** — 机制地图与选型
2. [客户端配置与连接](./02-client-setup.md) — rest.Config、QPS、多集群
3. [资源的 CRUD 操作](./03-crud-operations.md) — Get/List/Create/Update/Patch/Apply
4. [Informer 机制详解](./04-informer.md) — Reflector、缓存、Lister、WorkQueue
5. [实战：自定义控制器](./05-controller-demo.md) — 完整控制循环
6. [Discovery 与 Dynamic Client](./06-discovery-and-dynamic.md) — CRD / 泛型编程
7. [常用机制与工具包](./07-common-mechanisms.md) — Event、选主、Scheme、Fake、retry
8. [排障与生产清单](./08-debugging-and-pitfalls.md) — 故障剧本与上线检查表

学完本模块后，建议继续：

- [自定义资源专题](../07-custom-resources/README.md)
- [控制器深度专题](../11-controller-deep-dive/README.md)

## 下一步

- [客户端配置与连接](./02-client-setup.md)
