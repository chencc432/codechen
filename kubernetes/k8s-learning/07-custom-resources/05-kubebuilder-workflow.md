# 🛠️ Kubebuilder 工作流：从脚手架到控制器骨架

## 为什么要单独讲 Kubebuilder

前面已经提到：

- `client-go` 能让你直接理解底层机制
- `controller-runtime` 提供了更现代、更简洁的控制器抽象
- `Kubebuilder` 则是在这之上提供脚手架和工程化工作流

很多团队今天真正开始做 Operator 时，最常用的入口就是 Kubebuilder。  
所以如果要把“自定义资源专题”做得更像教材，Kubebuilder 这一章几乎是绕不开的。

## 1. Kubebuilder 到底是什么

可以把 Kubebuilder 理解成：

> 围绕 `controller-runtime` 构建的一套 Operator / API 扩展脚手架与开发工作流。

它主要解决的是：

- 项目骨架怎么起
- API 类型怎么生成
- CRD 清单怎么生成
- Controller 模板怎么组织
- RBAC、Webhook、测试、部署文件怎么配套生成

它不是一个“运行时平台”，而更像：

- 开发框架
- 代码生成工具
- 工程模板集合

## 2. 为什么它比手写 `client-go` 更适合工程化

如果你纯手写控制器，往往需要自己处理：

- 类型注册
- Scheme 管理
- Informer 组织
- Queue 逻辑
- Reconcile 模板
- RBAC 清单
- CRD 生成

这些工作并不是没价值，但在真正做业务控制器时，很多其实属于重复工程劳动。

Kubebuilder 的价值就在于：

- 让你更快进入“业务模型”和“控制逻辑”本身
- 让项目结构更贴近社区主流实践
- 降低样板代码和目录组织成本

## 3. 一个典型的开发流程

使用 Kubebuilder 开发一个新 Operator，大致会经历下面这些步骤：

1. 初始化项目
2. 创建 API 类型
3. 生成 Go 类型和 CRD
4. 编写 Reconciler
5. 生成部署与 RBAC 清单
6. 本地运行和测试
7. 构建镜像并部署到集群

## 4. 初始化项目

一个典型的开始命令可能像这样：

```bash
kubebuilder init --domain example.com --repo github.com/acme/mysql-operator
```

这一步通常会完成：

- 初始化 Go 模块
- 创建项目目录骨架
- 配置基础 Makefile
- 准备 `controller-runtime` 工程结构

### 你可以把它理解成

它不是“帮你写完业务”，而是：

> 先搭出一个标准化的 Operator 工程底座。

## 5. 创建 API 类型

接下来通常会创建一个 API：

```bash
kubebuilder create api --group database --version v1alpha1 --kind MySQLCluster
```

这一步通常会做几件事：

- 创建 API 类型定义文件
- 创建 Reconciler 骨架
- 更新 Scheme 注册
- 生成对应目录结构

从这里开始，你的项目就有了：

- 资源类型
- 控制器入口
- 与 CRD 生成相关的源码结构

## 6. 典型目录结构怎么理解

一个典型 Kubebuilder 项目通常会有类似结构：

```text
.
├── api/
│   └── v1alpha1/
│       ├── groupversion_info.go
│       ├── mysqlcluster_types.go
│       └── zz_generated.deepcopy.go
├── cmd/
│   └── main.go
├── config/
│   ├── crd/
│   ├── rbac/
│   ├── manager/
│   └── samples/
├── internal/
│   └── controller/
│       └── mysqlcluster_controller.go
├── Makefile
└── PROJECT
```

### 6.1 `api/`

这里定义资源类型，也就是：

- `Spec`
- `Status`
- 类型注解

### 6.2 `internal/controller/`

这里放控制器逻辑，也就是：

- `Reconcile()`
- 资源获取
- 下游资源管理
- 状态更新

### 6.3 `config/`

这里通常会放：

- CRD
- RBAC
- manager 部署清单
- 示例资源

### 6.4 `cmd/main.go`

这里通常负责启动 controller manager，并注册所有 controller。

## 7. API 类型文件里通常写什么

例如在 `mysqlcluster_types.go` 中，你通常会看到类似结构：

```go
type MySQLClusterSpec struct {
    Replicas *int32 `json:"replicas,omitempty"`
    Version  string `json:"version,omitempty"`
}

type MySQLClusterStatus struct {
    ReadyReplicas int32              `json:"readyReplicas,omitempty"`
    Conditions    []metav1.Condition `json:"conditions,omitempty"`
}

type MySQLCluster struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MySQLClusterSpec   `json:"spec,omitempty"`
    Status MySQLClusterStatus `json:"status,omitempty"`
}
```

这里本质上就是把 CRD 资源模型用 Go 类型表达出来。

## 8. 注解为什么这么重要

Kubebuilder 项目里经常会看到很多形如：

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
```

这些注解并不是装饰，而是用来驱动代码生成和清单生成的。

它们可以决定：

- 是否生成 root object
- 是否启用 `status` 子资源
- `kubectl get` 显示哪些列
- 校验规则和 schema 信息

所以可以把这些注解理解成：

> “从 Go 类型推导出 Kubernetes API 清单和行为配置的元信息。”

## 9. 生成清单通常怎么做

典型项目里通常会通过 Makefile 调用生成命令：

```bash
make generate
make manifests
```

### `make generate`

通常负责：

- DeepCopy 等代码生成

### `make manifests`

通常负责：

- 生成或更新 CRD YAML
- 生成 RBAC 清单
- 生成 Webhook 相关配置

这也是 Kubebuilder 工作流很重要的一部分：

- 不是你手写所有最终 YAML
- 而是通过类型和注解驱动生成

## 10. Reconciler 骨架通常长什么样

一个简化的 Reconcile 结构通常会像这样：

```go
func (r *MySQLClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var cluster databasev1alpha1.MySQLCluster
    if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 1. 处理删除逻辑
    // 2. 补 Finalizer
    // 3. 确保下游资源存在
    // 4. 收集实际状态
    // 5. 更新 status

    return ctrl.Result{}, nil
}
```

这段代码虽然短，但背后其实已经体现了成熟控制器的骨架思想。

## 11. SetupWithManager 在干什么

Kubebuilder 生成的控制器里，通常还会有：

```go
func (r *MySQLClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&databasev1alpha1.MySQLCluster{}).
        Complete(r)
}
```

它的意思大致是：

- 告诉 manager 这个 controller 主要 watch 哪种资源
- 当这类资源变化时，触发对应 Reconcile

后续你也可以继续扩展它 watch：

- owned resources
- secondary resources
- predicates

## 12. Kubebuilder 工程里的 RBAC 是怎么来的

控制器通常要读写很多资源：

- 自定义资源本身
- `status`
- `finalizers`
- Deployment / StatefulSet / Service / PVC 等下游对象

Kubebuilder 常通过注解生成 RBAC：

```go
// +kubebuilder:rbac:groups=database.example.com,resources=mysqlclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.example.com,resources=mysqlclusters/status,verbs=get;update;patch
```

再由 `make manifests` 生成最终 ClusterRole/Role 清单。

这让权限定义更靠近源码，也更容易保持一致。

## 13. 本地运行与集群部署

常见的开发流程通常是：

### 13.1 本地运行控制器

```bash
make install
make run
```

常见含义：

- 安装 CRD 到当前 kubeconfig 指向的集群
- 在本地启动 controller manager 连接集群

### 13.2 部署到集群

```bash
make docker-build docker-push IMG=<your-image>
make deploy IMG=<your-image>
```

这通常会：

- 构建并推送镜像
- 部署 controller manager 到集群中

## 14. Kubebuilder 为什么适合“教科书式”学习

因为它把 Operator 开发过程拆得非常清楚：

- API 类型定义
- 清单生成
- 控制器逻辑
- 权限配置
- 部署管理

这让学习者能更好区分：

- 哪些是 API 设计问题
- 哪些是控制器逻辑问题
- 哪些是工程和部署问题

## 15. 初学者最容易踩的坑

### 15.1 以为生成了脚手架就等于完成了控制器

实际上脚手架只提供：

- 结构
- 模板
- 通用生成能力

真正有价值的部分仍然是：

- 领域建模
- Reconcile 逻辑
- 状态设计
- 运维流程设计

### 15.2 把所有逻辑直接塞进 Reconcile

短期省事，长期会非常难维护。  
更好的方式通常是：

- 拆出资源构造函数
- 拆出状态计算函数
- 拆出删除清理逻辑

### 15.3 不重视状态与事件

很多初学控制器只想着“能创建资源”，但真正生产可用的控制器必须重视：

- `status`
- `conditions`
- error handling
- requeue 策略

## 16. 推荐的学习姿势

如果你是第一次用 Kubebuilder，建议按下面顺序学：

1. 先理解 CRD 和 Reconcile 基础
2. 用 Kubebuilder 初始化一个最小项目
3. 只做一个非常小的 CR，比如 `AppService`
4. 先实现“创建一个 Deployment”的最小控制逻辑
5. 再逐步加 `status`、Finalizer、下游资源管理

不要一开始就做一个复杂数据库 Operator，否则学习曲线会非常陡。

## 17. 一页总结

- Kubebuilder 是围绕 `controller-runtime` 的 Operator 脚手架和工作流
- 它帮助你快速搭建 API、控制器、清单与部署结构
- Go 类型 + 注解 + 生成命令 是 Kubebuilder 的核心工作方式
- 脚手架解决的是工程起步问题，不替代领域设计和控制逻辑设计
- 想做生产级 Operator，Kubebuilder 往往是很好的起点

## 下一步

- 返回 [自定义资源专题总览](./README.md)
