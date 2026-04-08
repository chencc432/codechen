# 🧰 从 0 到 1 设计一个完整 Operator 的实战样例

## 为什么这一篇很重要

前面的章节已经把 CRD、控制器、Operator、Kubebuilder、版本演进这些概念分开讲清楚了。  
但如果你真的要开始做一个 Operator，脑子里往往仍然会卡在一个更现实的问题上：

> 我应该先做什么，再做什么？一个“完整但不过度复杂”的 Operator 到底是如何被一步步设计出来的？

这篇的目标，就是把这个过程完整串起来。  
它不是只讲某一个 YAML，也不是只讲某一段 Reconcile 代码，而是从：

- 领域建模
- API 设计
- 工程骨架
- Reconcile 逻辑
- 状态更新
- 删除清理
- 部署验证

这一整条链路，带你走一遍。

## 1. 选一个适合作为教学案例的对象

如果第一次就做数据库主从切换、备份恢复、自动选主这种复杂 Operator，学习成本会非常高。  
所以这篇故意选一个更适合教学的领域对象：

> `AppService`

它代表的是一种“平台上的应用服务对象”，用户希望通过它来声明：

- 镜像是什么
- 副本数是多少
- 暴露端口是什么
- 是否对外暴露
- 资源限制是什么

而 Operator 负责自动创建并维护：

- `Deployment`
- `Service`
- 状态信息

这个案例足够简单，能看清楚控制器全链路；又足够完整，能覆盖 Operator 的关键能力。

## 2. 先不要急着写代码，先明确领域边界

### 2.1 用户真正关心什么

一个平台用户通常不想直接写一套 Deployment + Service + labels + selector + probes 的完整 YAML。  
他更希望写的是：

- 应用名
- 镜像
- 副本数
- 服务端口
- 资源需求
- 暴露策略

所以 `AppService` 的核心价值是：

> 把“应用上线”这个高层领域动作，抽象成一个更友好的平台 API。

### 2.2 控制器负责什么

控制器更适合负责：

- 生成标准化的 Deployment
- 生成 Service
- 保持 labels/selector 一致
- 根据 Deployment 状态回填 Ready 副本数
- 删除时做资源清理

### 2.3 用户不应该直接关心什么

这个案例里，用户通常不需要直接管理：

- Deployment 的底层名字规则
- Service 的 selector 细节
- ownerReferences
- 状态计算逻辑

## 3. 先设计目标 API

一个比较合理的资源实例可能长这样：

```yaml
apiVersion: platform.example.com/v1alpha1
kind: AppService
metadata:
  name: demo-app
spec:
  image: nginx:1.27
  replicas: 2
  port: 80
  expose: true
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
status:
  observedGeneration: 1
  readyReplicas: 2
  phase: Ready
  serviceName: demo-app
```

这个对象的好处是：

- 用户一眼就能看懂
- 足够平台化
- 不会暴露太多底层实现细节

## 4. 设计 CRD 时，先想清楚 `spec` 与 `status`

### `spec`

这个案例里，`spec` 推荐包含：

- `image`
- `replicas`
- `port`
- `expose`
- `resources`

这些都属于：

- 用户想声明的目标状态

### `status`

推荐包含：

- `observedGeneration`
- `readyReplicas`
- `phase`
- `serviceName`
- `conditions`

这些都属于：

- 控制器根据实际运行情况观测出来的结果

## 5. 用 Kubebuilder 初始化工程

一个典型流程可能是：

```bash
kubebuilder init --domain example.com --repo github.com/acme/appservice-operator
kubebuilder create api --group platform --version v1alpha1 --kind AppService
```

完成后通常会得到：

- API 类型定义文件
- Reconciler 骨架
- config 目录
- RBAC 模板
- Makefile

这时你已经有了一个标准化 Operator 工程底座。

## 6. 定义 API 类型

Go 类型可以先写成下面这样：

```go
type AppServiceSpec struct {
    Image    string                      `json:"image,omitempty"`
    Replicas *int32                      `json:"replicas,omitempty"`
    Port     int32                       `json:"port,omitempty"`
    Expose   bool                        `json:"expose,omitempty"`
    Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type AppServiceStatus struct {
    ObservedGeneration int64              `json:"observedGeneration,omitempty"`
    ReadyReplicas      int32              `json:"readyReplicas,omitempty"`
    Phase              string             `json:"phase,omitempty"`
    ServiceName        string             `json:"serviceName,omitempty"`
    Conditions         []metav1.Condition `json:"conditions,omitempty"`
}
```

### 这一步最重要的不是语法，而是设计意识

你应该始终问自己：

- 这个字段是不是用户真正关心的
- 这个字段是不是控制器应该负责的
- 这个结构未来还能不能扩展

## 7. 给 CRD 加注解和打印列

如果你想让这个资源更像原生 Kubernetes 资源，最好给它加一些生成注解。

例如：

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
```

这样以后：

```bash
kubectl get appservices
```

就能得到更友好的输出。

## 8. 定义 Reconcile 的目标

在开始写 Reconcile 之前，先把这次控制器到底要做什么列清楚。

本案例最小闭环目标是：

1. 读到 `AppService`
2. 确保对应的 `Deployment` 存在
3. 如果 `spec.expose=true`，确保对应的 `Service` 存在
4. 收集 Deployment 当前状态
5. 写回 `status`
6. 删除时清理自己创建的资源

如果你不先明确目标，Reconcile 很容易越写越散。

## 9. Reconcile 逻辑的推荐拆法

不要把所有逻辑都塞进一个超大的 `Reconcile()` 函数里。  
更好的方式通常是拆成几类函数：

### 9.1 资源构造函数

例如：

- `desiredDeployment(app *AppService) *appsv1.Deployment`
- `desiredService(app *AppService) *corev1.Service`

### 9.2 状态计算函数

例如：

- `calculatePhase(...)`
- `buildConditions(...)`

### 9.3 删除清理函数

例如：

- `handleDeletion(...)`
- `cleanupFinalizer(...)`

这种拆法会让控制器更容易测试和维护。

## 10. 一个典型的 Reconcile 流程

下面是一个更接近真实控制器的处理顺序：

### 10.1 读取对象

先根据 `req.NamespacedName` 拿到 `AppService`。

### 10.2 对象不存在就结束

如果对象已经不存在，通常直接返回。

### 10.3 处理 Finalizer

如果需要清理下游对象或外部资源，应确保有 Finalizer。

### 10.4 处理删除逻辑

如果发现对象正在删除：

- 先执行清理
- 再移除 Finalizer

### 10.5 确保 Deployment 存在

如果不存在则创建，存在则比对和修正。

### 10.6 确保 Service 存在

如果 `spec.expose=true`，则创建或修正；否则可按策略删除或忽略。

### 10.7 更新状态

根据 Deployment 的 ready 副本、可用状态等，写回：

- `readyReplicas`
- `phase`
- `conditions`
- `observedGeneration`

## 11. ownerReferences 为什么一定要加

当控制器创建下游 Deployment / Service 时，通常应该设置 ownerReferences 指向 `AppService`。

这样做的好处是：

- 资源归属明确
- 删除上游对象时可自动级联删除
- 更符合 Kubernetes 原生资源管理方式

否则就容易出现“孤儿资源”。

## 12. 为什么还要 Finalizer

很多人会问：

> 已经有 ownerReferences 了，为什么还要 Finalizer？

因为两者解决的问题不同：

- `ownerReferences`：解决资源归属和级联删除
- `Finalizer`：解决删除前还要做额外收尾动作

在这个 `AppService` 例子里，如果你只需要级联删除 Deployment/Service，可能 ownerReferences 就够。  
但如果未来还涉及：

- 外部 DNS
- 网关注册
- 第三方平台记录

那 Finalizer 就很重要。

## 13. 状态设计要尽量让用户“看一眼就懂”

例如你可以定义非常简单但有效的 phase：

- `Pending`
- `Progressing`
- `Ready`
- `Failed`

同时配合条件：

```yaml
conditions:
- type: Ready
  status: "True"
  reason: DeploymentAvailable
```

这样：

- `phase` 给整体摘要
- `conditions` 给结构化细节

## 14. 如何判断控制器是否已经处理了最新 spec

这时就要用 `observedGeneration`。

例如：

- 当前对象 `metadata.generation = 5`
- `status.observedGeneration = 4`

说明：

- 用户已经改过第 5 次 spec
- 但控制器状态还没追上

这个字段对调试非常有帮助。

## 15. 样例 CR 是什么

完成 CRD 和控制器后，通常会配一个样例资源文件，帮助用户验证最小工作流。

例如：

```yaml
apiVersion: platform.example.com/v1alpha1
kind: AppService
metadata:
  name: demo-app
spec:
  image: nginx:1.27
  replicas: 2
  port: 80
  expose: true
```

这个文件非常重要，因为它往往是：

- 用户第一次接触你的 API 的入口

## 16. 本地调试的推荐顺序

一个比较稳妥的教学顺序是：

1. 先 `make install` 安装 CRD
2. 本地运行控制器
3. 提交 sample CR
4. 观察 Deployment / Service 是否自动创建
5. 修改 CR 的副本数，看控制器是否更新 Deployment
6. 删除 CR，看下游资源是否被正确清理

这套路径能验证最关键的闭环。

## 17. 从“能跑”到“像样”的提升方向

一个最小 Operator 跑起来后，通常还可以继续补这些能力：

### 17.1 校验

例如：

- `port` 不能小于 1
- `image` 不能为空
- `replicas` 不能为负数

### 17.2 默认值

例如：

- 不填副本数时默认 1
- 不填 phase 时由控制器初始化

### 17.3 更好的状态

例如：

- `conditions`
- `observedGeneration`
- 错误信息

### 17.4 更严格的资源更新策略

例如：

- 只允许某些字段变更
- 某些字段变更触发重建

## 18. 常见反模式

### 18.1 一开始就做成“大而全”的 Operator

第一次练手最容易犯的错就是：

- 一上来就想做数据库 Operator
- 一上来就想支持多拓扑、多版本、多云平台

这样会让你根本学不清核心链路。

### 18.2 Reconcile 不幂等

控制器经常会被重复触发，所以必须尽量做到：

- 多次执行结果一致
- 即使重试也不会破坏资源

### 18.3 没有状态设计

只有“创建成功/失败”的日志，而没有标准化状态字段，会让这个 Operator 很难使用。

## 19. 这篇案例真正想教会你的是什么

不是“写一个 AppService 多简单”，而是：

- 如何从领域问题开始
- 如何收敛为一个 CRD
- 如何把 CRD 映射成下游资源
- 如何把控制器做成一个完整闭环

只要这条链理解了，之后做更复杂的 Operator 只是领域复杂度升级，而不是思路完全变掉。

## 20. 一页总结

- 一个完整 Operator 的设计顺序通常是：领域建模 -> CRD 设计 -> 工程骨架 -> Reconcile -> 状态 -> 删除清理 -> 部署验证
- 初学者最适合先做一个轻量但闭环完整的案例
- Reconcile 的价值不是“创建资源”，而是“持续让系统收敛到期望状态”
- 好的 Operator 一定同时包含：清晰的 API、可靠的控制逻辑、可读的状态反馈

## 下一步

- 返回 [自定义资源专题总览](./README.md)
