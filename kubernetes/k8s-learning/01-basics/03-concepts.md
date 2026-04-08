# 📚 Kubernetes 核心概念与术语

## 📖 阅读指南

**如果你是初学者**：
1. 先阅读"对象模型"部分，理解 Kubernetes 的基本结构
2. 重点学习"核心概念详解"（Label、Namespace、Selector），这些是最常用的
3. 遇到配置看不懂时，查看"配置快速参考"部分的完整示例
4. 最后做"实践练习"，动手验证理解

**如果你已经有一定基础**：
1. 可以直接查看"配置快速参考"快速查找配置
2. 关注"常见配置错误"避免踩坑
3. 参考"常用术语对照表"确认理解

**配置示例说明**：
- 每个配置示例都包含详细的中文注释
- `# ✅` 表示正确示例，`# ❌` 表示错误示例
- 重要提示用 **粗体** 或 ⚠️ 标记

## 🎯 学习目标

读完这份文档后，你应该能回答下面这些问题：

- Kubernetes 中的 `Pod`、`Deployment`、`Service`、`Namespace` 分别是什么，它们之间如何关联
- 为什么 Kubernetes 对象总是长得像 `apiVersion + kind + metadata + spec`
- `Label` 和 `Annotation` 都是键值对，它们的本质区别是什么
- `selector` 为什么几乎是所有资源关联机制的核心
- `requests`、`limits`、QoS、调度、驱逐之间到底是什么关系
- `nodeSelector`、`Affinity`、`Taint/Toleration` 各自解决什么问题，应该如何组合使用
- Pod 从创建到销毁会经历什么状态，问题应该去哪里查
- Service 为什么能稳定访问一组会不断变化 IP 的 Pod

如果你能把这些问题讲清楚，就说明你已经建立了 Kubernetes 最核心的一层认知框架。

## 概念体系总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Kubernetes 概念体系                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   工作负载                    服务发现                 配置与存储       │
│   ┌─────────┐                ┌─────────┐            ┌─────────┐     │
│   │   Pod   │                │ Service │            │ConfigMap│     │
│   └────┬────┘                └────┬────┘            └─────────┘     │
│        │                          │                 ┌─────────┐     │
│   ┌────┴────┐                ┌────┴────┐            │ Secret  │     │
│   │ Deploy- │                │ Ingress │            └─────────┘     │
│   │  ment   │                └─────────┘            ┌─────────┐     │
│   └────┬────┘                ┌─────────┐            │ Volume  │     │
│        │                     │Endpoint │            └─────────┘     │
│   ┌────┴────┐                └─────────┘                            │
│   │Replica- │                                                        │
│   │   Set   │                                                        │
│   └─────────┘                                                        │
│                                                                       │
│   集群管理                    调度控制                 安全与权限       │
│   ┌─────────┐                ┌─────────┐            ┌─────────┐     │
│   │  Node   │                │ Taint   │            │  RBAC   │     │
│   └─────────┘                │Tolerate │            └─────────┘     │
│   ┌─────────┐                └─────────┘            ┌─────────┐     │
│   │Namespace│                ┌─────────┐            │ Service │     │
│   └─────────┘                │Affinity │            │ Account │     │
│                              └─────────┘            └─────────┘     │
└─────────────────────────────────────────────────────────────────────┘
```

### 如何建立 Kubernetes 的整体心智模型

很多初学者会觉得 Kubernetes 概念很多、名词很多、字段很多，越学越乱。其实可以把它先拆成 4 个层次来理解：

1. **声明层**：你通过 YAML 或 `kubectl` 告诉 Kubernetes "我想要什么"
2. **控制层**：控制器不断把"实际状态"拉回到"期望状态"
3. **运行层**：容器真正运行在 Node 上，由 kubelet、容器运行时、网络插件等负责
4. **访问层**：Service、DNS、Ingress 等机制让其他系统稳定访问这些工作负载

换句话说，Kubernetes 的核心不是"启动容器"，而是：

- 你声明目标状态
- 系统持续对比当前状态
- 系统自动做出修正
- 外部或内部流量通过稳定抽象访问这些不断变化的实例

后面所有概念，其实都可以放回这 4 层中理解。

### 最容易混淆的一句话总结

- `Pod`：真正跑容器的地方
- `Deployment`：管理一组 Pod，并负责滚动更新
- `Service`：给一组 Pod 提供稳定访问入口
- `Label/Selector`：把资源"贴标签"并"选出来"
- `Namespace`：做逻辑隔离
- `requests/limits`：描述资源需求与上限
- `Affinity/Taint`：决定 Pod 更适合去哪里，或者不能去哪里
- `Status/Conditions/Events`：告诉你现在到底发生了什么

## 0. 先学会看 YAML

如果你一开始就把每个字段都硬背，很容易崩溃。更高效的方式是：先学会把任意一个 Kubernetes YAML 拆成固定的阅读顺序。

### 0.1 看 YAML 的推荐顺序

拿到任意资源清单后，建议总是按下面顺序读：

1. 看 `apiVersion`
2. 看 `kind`
3. 看 `metadata.name`
4. 看 `metadata.namespace`
5. 看 `metadata.labels`
6. 看 `spec`
7. 最后再看和这个资源有关的 `status`

因为这套顺序基本等于：

- 这是什么资源
- 它叫什么
- 它属于哪里
- 它如何被别人识别
- 它想达到什么状态
- 它现在实际怎么样

### 0.2 一个对象最常见的骨架

```yaml
apiVersion: apps/v1           # API 组 + 版本
kind: Deployment              # 资源类型
metadata:                     # 对象身份信息
  name: web
  namespace: default
  labels:
    app: web
spec:                         # 你定义的目标状态
  replicas: 3
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
status:                       # 系统回填的实际状态
  replicas: 3
  availableReplicas: 3
```

### 0.3 不同字段分别回答什么问题

| 字段 | 本质问题 | 典型内容 |
|------|----------|----------|
| `apiVersion` | 去哪个 API 找这个对象 | `v1`、`apps/v1`、`batch/v1` |
| `kind` | 这是什么类型的资源 | `Pod`、`Deployment`、`Service` |
| `metadata` | 它是谁 | 名称、命名空间、标签、注解 |
| `spec` | 你希望它变成什么样 | 副本数、镜像、端口、选择器 |
| `status` | 它现在实际是什么样 | 是否就绪、分配了哪个 IP、是否创建成功 |

### 0.4 初学者最值得记住的一条规则

**`spec` 是你写给 Kubernetes 的目标，`status` 是 Kubernetes 回给你的结果。**

只要记住这一句，后面大部分资源都能看懂。

## 1. 对象模型

### 1.1 什么是 Kubernetes 对象？

Kubernetes 对象是 Kubernetes 系统中的持久化实体。Kubernetes 使用这些对象来表示集群的状态：

- 哪些容器化应用正在运行
- 这些应用使用什么资源
- 关于应用行为的策略

### 1.2 对象规约（Spec）与状态（Status）

每个 Kubernetes 对象都包含两个核心字段：

**Spec（规约）**：你定义的期望状态，告诉 Kubernetes "我想要什么"
**Status（状态）**：Kubernetes 维护的实际状态，告诉你 "现在是什么样"

```yaml
# 这是一个 Pod 对象的完整示例
apiVersion: v1              # API 版本，告诉 Kubernetes 使用哪个版本的 API
kind: Pod                   # 资源类型，这里是 Pod（容器组）
metadata:                   # 元数据部分：对象的标识信息
  name: my-pod              # 对象名称，在同一个命名空间内必须唯一
  namespace: default        # 命名空间，如果不指定默认是 default
spec:                       # 规约部分：你定义的期望状态（你写的）
  containers:               # 容器列表
  - name: nginx             # 容器名称
    image: nginx:1.21       # 使用的镜像和版本
status:                     # 状态部分：当前实际状态（Kubernetes 自动维护，你不需要写）
  phase: Running            # Pod 当前阶段：Pending/Running/Succeeded/Failed
  podIP: 10.244.1.5        # Pod 被分配的 IP 地址
  conditions:               # Pod 的各种条件状态
  - type: Ready             # 就绪状态
    status: "True"          # True 表示就绪，False 表示未就绪
```

**重要提示**：
- 你只需要写 `spec` 部分，`status` 是 Kubernetes 自动生成的
- 当你创建或更新对象时，只需要提供 `apiVersion`、`kind`、`metadata` 和 `spec`
- `status` 字段是只读的，Kubernetes 会根据实际情况自动更新

#### 为什么 Kubernetes 要把 Spec 和 Status 分开？

这是 Kubernetes 最核心的设计思想之一，原因有 3 个：

1. **职责分离**：用户声明需求，系统负责落地
2. **持续纠偏**：实际状态偏离期望状态时，控制器可以持续修正
3. **可观测性更强**：你可以同时看到"我想要什么"和"系统现在做到什么程度"

举个例子：

- 你在 `Deployment.spec.replicas` 里写 `3`
- 当前集群里因为节点故障只剩下 2 个 Pod
- 于是 `status.availableReplicas` 可能显示 `2`
- Deployment 控制器会继续想办法把它恢复到 3

这就是声明式系统的典型行为。

#### 声明式和命令式的区别

| 方式 | 你告诉系统什么 | 典型例子 |
|------|----------------|----------|
| 命令式 | "帮我执行这个动作" | `docker run nginx` |
| 声明式 | "我希望最终状态是这样" | `replicas: 3` |

Kubernetes 主要是**声明式系统**。你不是让它"现在启动 3 个容器"，而是告诉它"请一直维持 3 个副本"。

### 1.3 对象标识

每个对象都有唯一标识：

| 标识 | 说明 | 示例 |
|------|------|------|
| Name | 同一命名空间内唯一 | `nginx-deployment` |
| UID | 整个集群唯一 | `a1b2c3d4-e5f6-...` |
| Namespace | 资源所属的命名空间 | `default`, `kube-system` |

### 1.4 apiVersion 到底是什么意思

很多人第一次看到 `apiVersion` 会误以为它只是"软件版本号"，其实不是。它更准确地表示：

- 资源属于哪个 **API 组（API Group）**
- 使用这个组里的哪个 **版本（Version）**

常见例子：

| 写法 | 含义 | 常见资源 |
|------|------|----------|
| `v1` | 核心 API 组 | `Pod`、`Service`、`ConfigMap`、`Namespace` |
| `apps/v1` | `apps` 组的 v1 版本 | `Deployment`、`StatefulSet`、`DaemonSet` |
| `batch/v1` | `batch` 组的 v1 版本 | `Job`、`CronJob` |
| `networking.k8s.io/v1` | 网络相关 API 组 | `Ingress`、`NetworkPolicy` |

所以你可以把 `apiVersion` 理解成：

> "我要使用 Kubernetes 哪一类接口来解析这个对象。"

### 1.5 metadata 不只是 name

初学时最容易忽略 `metadata`，但它其实非常重要。因为资源之间的大量关联都不是靠字段嵌套，而是靠元数据完成的。

`metadata` 常见字段包括：

| 字段 | 作用 | 常见用途 |
|------|------|----------|
| `name` | 对象名字 | 人类识别、API 路径定位 |
| `namespace` | 命名空间 | 逻辑隔离 |
| `labels` | 可筛选的键值对 | 选择器、分组、关联 |
| `annotations` | 非筛选元数据 | 记录额外信息，给工具使用 |
| `ownerReferences` | 拥有者引用 | 建立资源归属关系，支持级联删除 |
| `finalizers` | 最终清理钩子 | 防止资源在清理完成前被直接删除 |

#### ownerReferences 是什么

例如 Deployment 创建 ReplicaSet，ReplicaSet 再创建 Pod。被创建出来的下游资源，通常会带有 `ownerReferences`，告诉系统：

- 这个 Pod 归谁管理
- 如果上游对象被删除，下游对象是否应该跟着删

这也是为什么很多资源不是你手动删 Pod 之后就算结束了，因为控制器会看到：

- "Pod 少了"
- "但 Deployment 还要求 3 个副本"
- "那我再补一个"

#### finalizers 是什么

`finalizers` 可以理解为删除前必须完成的"收尾工作列表"。只要列表还没清空，对象就会停留在 `Terminating` 状态。

典型场景：

- 先清理云盘、负载均衡器、DNS 等外部资源
- 清理完成后再真正删除 Kubernetes 对象

当你看到资源一直删不掉时，`finalizers` 往往是排查重点之一。

## 2. 核心概念详解

### 2.1 Label（标签）

**什么是标签？**
标签是附加到 Kubernetes 对象上的键值对，就像给资源贴标签一样，用于组织和选择资源。

你可以把 Label 理解成资源的"可查询属性"。它最大的价值不在于"写个描述"，而在于后续可以被：

- `kubectl -l` 查询
- `Service` 选择 Pod
- `Deployment` 识别自己要管理的 Pod
- 监控、发布、运维工具按维度分组

**为什么需要标签？**
- 组织资源：给资源分类（比如：这是哪个应用、什么环境）
- 选择资源：快速找到需要的资源（比如：找出所有生产环境的 Pod）
- 关联资源：让不同的资源关联起来（比如：Service 通过标签找到对应的 Pod）

```yaml
# 在 Pod 上添加标签的示例
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  labels:                    # 标签部分：键值对形式
    app: nginx               # 键: app，值: nginx（表示这是 nginx 应用）
    environment: production  # 键: environment，值: production（表示这是生产环境）
    tier: frontend           # 键: tier，值: frontend（表示这是前端服务）
    version: v1.0.0          # 键: version，值: v1.0.0（表示版本号）
spec:
  containers:
  - name: nginx
    image: nginx:1.21
```

**标签命名规范**：
- 键名：最多 63 个字符，可以包含字母、数字、连字符、下划线、点
- 值：最多 63 个字符，可以包含字母、数字、连字符、下划线、点
- 键名可以带前缀，推荐格式：`<dns-prefix>/<name>`
- 更推荐使用社区通用标签，如 `app.kubernetes.io/name`

#### 如何设计一套好用的标签

好的标签体系，应该既方便查询，也方便资源之间建立稳定关系。实践中常见的维度有：

- **应用维度**：这是谁
- **环境维度**：它属于哪个环境
- **层级维度**：它处于前端、后端还是缓存
- **版本维度**：当前实例属于哪个发布版本
- **归属维度**：属于哪个团队或业务线

推荐优先考虑下面这组通用标签：

```yaml
metadata:
  labels:
    app.kubernetes.io/name: nginx
    app.kubernetes.io/instance: nginx-prod
    app.kubernetes.io/component: frontend
    app.kubernetes.io/part-of: website
    app.kubernetes.io/version: "1.21.6"
    app.kubernetes.io/managed-by: helm
```

这组标签的好处是：

- 团队协作时更统一
- 生态工具更容易识别
- 后续做发布、监控、成本归因时更方便

#### 标签选择器

标签选择器用于根据标签来筛选资源。有两种方式：

**方式 1：等值选择器（matchLabels）** - 简单直接，最常用

```yaml
# 示例：Service 选择所有 app=nginx 的 Pod
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:                  # 选择器：告诉 Service 要选择哪些 Pod
    matchLabels:            # 等值匹配：键值必须完全相等
      app: nginx            # 只选择标签 app=nginx 的 Pod
  ports:
  - port: 80                # Service 端口
    targetPort: 80          # Pod 的端口
```

**方式 2：集合选择器（matchExpressions）** - 更灵活，支持复杂条件

```yaml
# 示例：选择环境是 production 或 staging，且不是 backend 层的 Pod
selector:
  matchExpressions:         # 集合匹配：支持多种操作符
  - key: environment        # 标签键名
    operator: In            # 操作符：In（在列表中）、NotIn（不在列表中）、Exists（存在）、DoesNotExist（不存在）
    values:                 # 值列表
    - production
    - staging
  - key: tier
    operator: NotIn         # 不在列表中
    values:
    - backend
```

**操作符说明**：
- `In`：标签值在给定的值列表中
- `NotIn`：标签值不在给定的值列表中
- `Exists`：标签键存在（不需要指定 values）
- `DoesNotExist`：标签键不存在（不需要指定 values）

#### Service 和 Deployment 的 selector 写法不一样

这点非常容易混淆：

- `Deployment.spec.selector` 使用的是 **LabelSelector 结构**，所以会看到 `matchLabels`、`matchExpressions`
- `Service.spec.selector` 通常是一个 **简单键值映射**，直接写 `app: nginx`

也就是说：

```yaml
# Deployment 的写法
spec:
  selector:
    matchLabels:
      app: nginx
```

```yaml
# Service 的写法
spec:
  selector:
    app: nginx
```

初学者经常把这两种写法混在一起，这是最常见的 YAML 误区之一。

#### selector 为什么这么重要

从系统设计角度看，selector 是 Kubernetes 里最核心的"关联机制"之一：

- Service 靠它找到 Pod
- Deployment 靠它识别自己要管理的 Pod
- ReplicaSet 靠它维持副本数
- 你自己也可以靠它批量查询和操作资源

你完全可以把 selector 当成 Kubernetes 里的"条件查询语句"。

**实际应用场景**：
- Service 选择 Pod：`matchLabels: {app: nginx}` 表示 Service 会将流量转发到所有 `app=nginx` 的 Pod
- Deployment 管理 Pod：`selector.matchLabels` 必须与 `template.metadata.labels` 匹配

#### kubectl 使用标签

```bash
# 按标签筛选
kubectl get pods -l app=nginx
kubectl get pods -l 'environment in (production, staging)'
kubectl get pods -l app=nginx,tier=frontend

# 添加/修改标签
kubectl label pods nginx-pod version=v2

# 删除标签
kubectl label pods nginx-pod version-

# 查看所有标签
kubectl get pods --show-labels
```


### 2.2 Annotation（注解）

**什么是注解？**
注解也是键值对，但用于存储非标识性的元数据，通常给工具、库或系统组件使用。

**Label vs Annotation 的区别**：
- **Label**：用于选择和分组资源（可以被选择器使用）
- **Annotation**：用于存储额外信息（不能被选择器使用，但可以存储更多数据）

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  labels:
    app: nginx              # 标签：用于选择
  annotations:              # 注解：用于存储元数据
    # 描述信息
    description: "This is the main nginx server"
    
    # Kubernetes 系统自动添加的注解
    kubernetes.io/created-by: "deployment-controller"
    
    # Prometheus 监控工具使用的注解（告诉 Prometheus 要监控这个 Pod）
    prometheus.io/scrape: "true"    # 是否抓取指标
    prometheus.io/port: "9090"      # 指标端口
    
    # 自定义注解
    imageregistry: "https://hub.docker.com/"
    contact: "team@example.com"
    changelog: "Updated to fix memory leak issue"
spec:
  containers:
  - name: nginx
    image: nginx:1.21
```

**常见使用场景**：
- 监控配置：告诉监控系统如何收集指标
- 构建信息：记录镜像仓库、构建时间等
- 部署信息：记录部署工具、部署时间等
- 描述信息：存储人类可读的描述

#### Annotation 的典型特点

`Annotation` 常被称为"写给系统和工具看的附加说明"。它通常具有以下特征：

- 不参与资源选择
- 更适合存储描述性、集成性信息
- 往往由控制器、发布工具、监控系统自动读写

最常见的误用是：

- 把会参与筛选的字段放到 Annotation 里
- 把大量无结构、长期没人用的信息都堆在 Annotation 里

简单判断规则：

- **以后要拿来筛选、关联、分组**：放 `Label`
- **只是补充说明、给工具消费、记录上下文**：放 `Annotation`

#### Label vs Annotation

| 特性 | Label | Annotation |
|------|-------|------------|
| 用途 | 标识和选择 | 存储元数据 |
| 选择器 | 支持 | 不支持 |
| 长度限制 | 较严格 | 较宽松 |
| 典型用例 | 分组、筛选 | 配置、描述 |

### 2.3 Namespace（命名空间）

命名空间用于在集群中创建虚拟的隔离环境。

但要注意，Namespace 是**逻辑隔离**，不是"物理隔离"。它主要隔离的是：

- 名称范围
- 权限边界
- 配额边界
- 默认查询范围

它**默认不会**自动隔离：

- 节点
- 网络访问
- 底层物理资源

如果你希望不同命名空间之间网络互相隔离，还需要 `NetworkPolicy`；如果你希望资源使用量隔离，还需要 `ResourceQuota` 和 `LimitRange`。

```bash
# 默认命名空间
- default         # 默认命名空间，用户资源默认在这里
- kube-system     # Kubernetes 系统组件
- kube-public     # 公开资源，所有用户可读
- kube-node-lease # 节点心跳数据
```

#### 命名空间操作

```bash
# 查看命名空间
kubectl get namespaces
kubectl get ns

# 创建命名空间
kubectl create namespace dev
kubectl create ns staging

# 在特定命名空间操作
kubectl get pods -n kube-system
kubectl apply -f deployment.yaml -n dev

# 设置默认命名空间
kubectl config set-context --current --namespace=dev

# 查看当前默认命名空间
kubectl config view --minify | grep namespace

# 删除命名空间（会删除其中所有资源！）
kubectl delete namespace dev
```

#### 命名空间 YAML

```yaml
# 通过 YAML 文件创建命名空间
apiVersion: v1              # 核心 API 版本
kind: Namespace             # 资源类型：命名空间
metadata:
  name: development         # 命名空间名称（必须唯一）
  labels:                   # 可以给命名空间也打标签
    environment: development
    team: backend
```

**什么时候使用命名空间？**
- 环境隔离：开发、测试、生产环境分开
- 团队隔离：不同团队使用不同的命名空间
- 资源配额：可以为每个命名空间设置资源限制
- 权限控制：可以为不同命名空间设置不同的访问权限

#### 什么时候不需要强行拆很多 Namespace

初学团队容易把 Namespace 用得过细，比如一个很小的单体系统拆十几个命名空间。这样会带来：

- 查询复杂度升高
- 配置文件里命名空间到处不一致
- 排查问题时上下文切换变多

比较常见、也更实用的划分方式是：

- 按环境拆：`dev`、`test`、`prod`
- 按团队拆：`platform`、`backend`、`data`
- 按强隔离业务域拆：例如 `payments`、`risk-control`

#### Namespace 解决了什么，不解决什么

| 问题 | Namespace 是否直接解决 | 说明 |
|------|------------------------|------|
| 同名资源冲突 | ✅ | 不同命名空间可以有同名 Pod/Service |
| 默认查询范围 | ✅ | `kubectl get pods` 默认只看当前命名空间 |
| 团队权限隔离 | ✅ | 常与 RBAC 一起使用 |
| 资源配额控制 | ✅ | 常与 `ResourceQuota` 一起使用 |
| 网络完全隔离 | ❌ | 需要 `NetworkPolicy` |
| 节点物理隔离 | ❌ | 需要调度规则、节点池、污点等 |

### 2.4 Selector（选择器）

选择器用于选择具有特定标签的资源。这是 Kubernetes 中资源关联的核心机制。

#### 场景 1：Service 选择 Pod

Service 通过选择器找到要代理的 Pod：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service       # Service 名称
spec:
  selector:                 # 选择器：告诉 Service 要选择哪些 Pod
    app: nginx              # 选择所有标签 app=nginx 的 Pod
  ports:
  - port: 80                # Service 对外暴露的端口
    targetPort: 80          # Pod 内部容器监听的端口
    protocol: TCP           # 协议类型
```

**工作原理**：
1. Service 的选择器 `app: nginx` 会找到所有标签为 `app=nginx` 的 Pod
2. Service 会创建一个虚拟 IP（ClusterIP）
3. 访问这个虚拟 IP 的流量会被负载均衡到所有匹配的 Pod

这里隐含着一个非常重要的好处：

- Pod 可以被替换、重建、扩缩容
- Pod IP 可以变化
- 但 Service 的名字和 ClusterIP 可以保持稳定

所以应用之间不应该直接依赖 Pod IP，而应该优先依赖 Service。

#### 场景 2：Deployment 管理 Pod

Deployment 通过选择器管理它创建的 Pod：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3               # 要创建 3 个 Pod 副本
  selector:                 # 选择器：告诉 Deployment 要管理哪些 Pod
    matchLabels:
      app: nginx            # 管理所有标签 app=nginx 的 Pod
  template:                 # Pod 模板：用于创建新的 Pod
    metadata:
      labels:               # Pod 的标签（必须与 selector.matchLabels 匹配！）
        app: nginx          # ⚠️ 这个标签必须与上面的 selector.matchLabels 一致
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
```

**重要规则**：
- `selector.matchLabels` 必须与 `template.metadata.labels` 完全匹配
- 如果不匹配，Deployment 无法管理它创建的 Pod
- 这是 Kubernetes 的设计要求，确保 Deployment 能正确识别和管理 Pod

**为什么需要匹配？**
Deployment 需要知道哪些 Pod 是它创建的，通过标签匹配来识别。如果标签不匹配，Deployment 会认为这些 Pod 不是它管理的，可能会创建新的 Pod。

#### Deployment 的 selector 为什么要谨慎设计

`Deployment.spec.selector` 一旦设计不当，后面会带来很多维护问题：

- 选择范围太宽：可能"误管"本不属于它的 Pod
- 选择范围太窄：可能管理不到自己应该管理的 Pod
- 变更 selector：很多场景下会受到限制，运维成本较高

因此实践建议是：

- selector 使用稳定、不轻易变化的标签
- 不要把版本号这种经常变化的标签放进 Deployment selector
- 版本号适合放在 Pod template labels 里供观察或灰度使用，而不是作为 Deployment 的核心识别条件

一个更稳妥的例子：

```yaml
spec:
  selector:
    matchLabels:
      app: nginx
      component: web
  template:
    metadata:
      labels:
        app: nginx
        component: web
        version: v2
```

这里：

- `app + component` 是稳定身份
- `version` 是变化属性

这样更利于长期维护。

## 3. 资源管理概念

### 3.1 资源请求与限制

**为什么需要资源管理？**
- 防止某个 Pod 占用过多资源，影响其他 Pod
- 帮助 Kubernetes 调度器选择合适的节点
- 确保集群资源被合理分配

很多人一开始只把 `resources` 看成"性能配置"，其实它同时影响两个阶段：

1. **调度阶段**：调度器主要看 `requests`
2. **运行阶段**：运行时主要约束 `limits`

也就是说，`requests` 更像"我至少要这么多"，`limits` 更像"你最多只能用这么多"

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
spec:
  containers:
  - name: app
    image: nginx
    resources:              # 资源限制配置
      requests:             # 资源请求：调度时保证的最小资源（必须满足才能调度）
        cpu: "250m"         # CPU 请求：0.25 个 CPU 核心（1000m = 1 CPU）
        memory: "128Mi"     # 内存请求：128 MiB（必须保证有这么多内存）
      limits:               # 资源限制：容器能使用的最大资源（超过会被限制或杀死）
        cpu: "500m"         # CPU 限制：最多使用 0.5 个 CPU 核心
        memory: "256Mi"     # 内存限制：最多使用 256 MiB（超过会被 OOMKilled）
```

**requests 和 limits 的区别**：

| 字段 | 含义 | 作用 | 示例 |
|------|------|------|------|
| **requests** | 资源请求 | 调度器保证的最小资源，Pod 必须能获得这么多资源才会被调度 | `cpu: "250m"` 表示至少需要 0.25 CPU |
| **limits** | 资源限制 | 容器能使用的最大资源，超过会被限制或终止 | `cpu: "500m"` 表示最多使用 0.5 CPU |

**实际例子**：
- 如果节点只有 0.2 CPU 可用，Pod 不会被调度（因为 requests 是 0.25 CPU）
- 如果 Pod 运行后尝试使用 0.6 CPU，会被限制在 0.5 CPU（因为 limits 是 0.5 CPU）
- 如果 Pod 内存超过 256Mi，容器会被杀死（OOMKilled）

#### requests 和 limits 的更细理解

| 维度 | requests | limits |
|------|----------|--------|
| 调度器是否关心 | 非常关心 | 通常不作为主决策依据 |
| CPU 行为 | 保障调度预留 | 超过后会被限速（throttling） |
| 内存行为 | 保障调度预留 | 超过后可能被 OOMKill |
| 对 QoS 的影响 | 有 | 有 |
| 是否建议总是设置 | 建议至少设置 | 视场景而定，但生产环境通常建议明确设置 |

#### 一个非常重要的现实差异：CPU 和内存超限的后果不同

- **CPU 超过 limit**：通常不是立刻杀死容器，而是被限速
- **内存超过 limit**：更容易直接触发 OOM，被系统杀死

所以很多线上问题的典型表现是：

- CPU 偏高：服务变慢，但还活着
- 内存打满：容器反复重启，出现 `OOMKilled`

#### CPU 单位详解

CPU 可以用两种方式表示：

```
# 方式 1：使用毫核（millicores，推荐）
1 CPU = 1000m
0.5 CPU = 500m
0.25 CPU = 250m
0.1 CPU = 100m

# 方式 2：使用小数
1 CPU = 1
0.5 CPU = 0.5
0.25 CPU = 0.25
```

**实际例子**：
- `cpu: "250m"` = 0.25 CPU = 四分之一 CPU 核心
- `cpu: "1000m"` = 1 CPU = 一个完整的 CPU 核心
- `cpu: "500m"` = 0.5 CPU = 半个 CPU 核心

**补充说明**：

- CPU 既可以写成 `500m`
- 也可以写成 `0.5`

但在团队协作中，**更推荐写成毫核形式**，因为更直观、更统一，也更不容易在对比时看错。

#### 内存单位详解

内存单位有两种进制：

**二进制单位（推荐，更精确）**：
```
Ki = 1024 字节（Kibibyte）
Mi = 1024 Ki = 1,048,576 字节（Mebibyte）
Gi = 1024 Mi = 1,073,741,824 字节（Gibibyte）
Ti = 1024 Gi
```

**十进制单位**：
```
K = 1000 字节（Kilobyte）
M = 1000 K = 1,000,000 字节（Megabyte）
G = 1000 M = 1,000,000,000 字节（Gigabyte）
T = 1000 G
```

**实际例子**：
- `memory: "128Mi"` = 128 MiB = 134,217,728 字节（约 134 MB）
- `memory: "1Gi"` = 1 GiB = 1,073,741,824 字节（约 1.07 GB）
- `memory: "512M"` = 512 MB = 512,000,000 字节（十进制）

**推荐使用二进制单位（Ki、Mi、Gi）**，因为更符合计算机的实际存储方式。

#### 一套简单但实用的资源配置思路

如果你现在还不知道一个应用该怎么配资源，可以先按下面思路走：

1. 先根据历史监控数据估算正常运行区间
2. `requests` 先设置在日常稳定值附近
3. `limits` 留出合理弹性空间
4. 观察一段时间，再用真实数据回调

例如：

- 常态 CPU 约 150m，偶尔冲到 300m
- 常态内存约 180Mi，峰值约 260Mi

那么起步配置可以是：

```yaml
resources:
  requests:
    cpu: "150m"
    memory: "192Mi"
  limits:
    cpu: "400m"
    memory: "320Mi"
```

这比完全不配资源，或者随手写一个非常大的值，都更合理。

### 3.2 QoS 类别（服务质量等级）

**什么是 QoS？**
QoS（Quality of Service）是 Kubernetes 根据资源配置自动为 Pod 分配的服务质量等级，影响资源不足时 Pod 的驱逐优先级。

**三种 QoS 类别**：

| QoS 类别 | 条件 | 驱逐优先级 | 说明 |
|----------|------|-----------|------|
| **Guaranteed** | 所有容器都设置了 `requests = limits`（且不为 0） | 最低（最后被驱逐） | 资源有保障，最稳定 |
| **Burstable** | 至少一个容器设置了 `requests`，但不满足 Guaranteed | 中等 | 有最低保障，可以超用 |
| **BestEffort** | 没有设置任何资源限制 | 最高（最先被驱逐） | 无保障，资源不足时优先被驱逐 |

**判断规则**：

```yaml
# 例子 1：Guaranteed（所有容器 requests = limits）
spec:
  containers:
  - name: app
    resources:
      requests:
        cpu: "250m"
        memory: "128Mi"
      limits:
        cpu: "250m"        # ✅ requests = limits
        memory: "128Mi"    # ✅ requests = limits
# 结果：QoS = Guaranteed

# 例子 2：Burstable（有 requests 但不等于 limits）
spec:
  containers:
  - name: app
    resources:
      requests:
        cpu: "250m"
        memory: "128Mi"
      limits:
        cpu: "500m"        # ❌ requests ≠ limits
        memory: "256Mi"    # ❌ requests ≠ limits
# 结果：QoS = Burstable

# 例子 3：BestEffort（没有资源限制）
spec:
  containers:
  - name: app
    image: nginx
    # 没有 resources 字段
# 结果：QoS = BestEffort
```

**实际影响**：

当节点资源不足时，Kubernetes 会按以下顺序驱逐 Pod：
1. **BestEffort** Pod（最先被驱逐）
2. **Burstable** Pod（中等优先级）
3. **Guaranteed** Pod（最后被驱逐，最稳定）

**查看 Pod 的 QoS 类别**：
```bash
# 方式 1：使用 kubectl get
kubectl get pods -o wide

# 方式 2：使用 kubectl describe（在 Status 部分）
kubectl describe pod <pod-name>
# 输出示例：
# QoS Class:       Burstable

# 方式 3：使用 JSONPath
kubectl get pod <pod-name> -o jsonpath='{.status.qosClass}'
```

**最佳实践**：
- **生产环境**：使用 Guaranteed，确保资源有保障
- **开发测试**：可以使用 Burstable，灵活使用资源
- **临时任务**：可以使用 BestEffort，但要注意可能被驱逐

#### 不要把 QoS 当成性能等级

QoS 的核心意义是**资源紧张时的驱逐优先级**，而不是"谁性能更高"。

例如：

- `Guaranteed` 不代表应用一定更快
- `BestEffort` 也不代表应用一定慢

QoS 主要决定的是：在节点资源紧张、发生压力时，谁更容易先被赶走。

## 4. 调度相关概念

### 调度到底在解决什么问题

当你提交一个 Pod 后，调度器要回答的问题本质上是：

> "集群里哪一台 Node 最适合承载这个 Pod？"

它通常会综合考虑：

- 节点是否还有足够资源
- 是否满足节点选择条件
- 是否违反污点约束
- 是否存在亲和性/反亲和性要求
- 是否有拓扑分布偏好

所以调度不是简单的"随便找台机器放上去"，而是一套持续评估约束和偏好的决策过程。

### 4.1 节点选择器（nodeSelector）

**什么是节点选择器？**
节点选择器是最简单的调度方式，要求 Pod 只能调度到具有特定标签的节点上。

**使用场景**：
- 需要 GPU 的应用调度到有 GPU 的节点
- 需要 SSD 的应用调度到有 SSD 的节点
- 特定环境的应用调度到特定节点

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
spec:
  nodeSelector:            # 节点选择器：硬性要求，必须满足
    disktype: ssd          # 只调度到标签 disktype=ssd 的节点
    gpu: nvidia-tesla-v100 # 只调度到标签 gpu=nvidia-tesla-v100 的节点
  containers:
  - name: nginx
    image: nginx:1.21
```

**使用步骤**：
1. 给节点打标签：`kubectl label nodes node1 disktype=ssd`
2. 在 Pod 中指定 nodeSelector
3. Pod 只会被调度到有匹配标签的节点

**注意事项**：
- 这是硬性要求，如果没有匹配的节点，Pod 会一直处于 Pending 状态
- 如果需要更灵活的调度，使用亲和性（Affinity）

#### nodeSelector 的定位

`nodeSelector` 的优点是：

- 简单
- 可读
- 非常适合固定规则

缺点是：

- 只能做等值匹配
- 不能表达偏好
- 条件复杂时扩展性差

所以你可以把它理解为：

- `nodeSelector`：适合"必须在某类节点上"
- `Affinity`：适合"最好在某类节点上，或者要表达复杂条件"

### 4.2 亲和性（Affinity）

#### 节点亲和性（Node Affinity）

节点亲和性比 nodeSelector 更灵活，支持硬性要求和软性偏好。

**两种类型**：
1. **硬性要求**（required）：必须满足，否则不调度
2. **软性偏好**（preferred）：尽量满足，但不强制

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
spec:
  affinity:                # 亲和性配置
    nodeAffinity:          # 节点亲和性
      # 硬性要求：必须调度到 Linux 节点
      requiredDuringSchedulingIgnoredDuringExecution:  # 调度时必须满足，运行后忽略
        nodeSelectorTerms: # 节点选择条件列表（满足任意一个即可）
        - matchExpressions: # 匹配表达式
          - key: kubernetes.io/os  # 节点标签键
            operator: In           # 操作符：In（在列表中）
            values:                # 值列表
            - linux
      
      # 软性偏好：优先调度到 zone-a 区域的节点
      preferredDuringSchedulingIgnoredDuringExecution:  # 调度时优先考虑，运行后忽略
      - weight: 100        # 权重：1-100，数字越大优先级越高
        preference:        # 偏好条件
          matchExpressions:
          - key: zone      # 节点标签键
            operator: In   # 操作符
            values:
            - zone-a       # 优先选择 zone-a 的节点
  containers:
  - name: nginx
    image: nginx:1.21
```

**字段名解释**（看起来很复杂，但记住规律）：
- `requiredDuringSchedulingIgnoredDuringExecution`：调度时必须满足，运行后忽略
- `preferredDuringSchedulingIgnoredDuringExecution`：调度时优先考虑，运行后忽略

**操作符说明**：
- `In`：标签值在列表中
- `NotIn`：标签值不在列表中
- `Exists`：标签键存在
- `DoesNotExist`：标签键不存在
- `Gt`：大于（用于数值）
- `Lt`：小于（用于数值）

**实际例子**：
- 硬性要求：必须调度到有 GPU 的节点
- 软性偏好：优先调度到 SSD 节点（如果没有，普通节点也可以）

#### 如何理解 required 和 preferred

最实用的记忆方式是：

- `required...`：像门禁，不满足就进不去
- `preferred...`：像加分项，满足更好，不满足也可能调度

如果一个 Pod 一直 `Pending`，而你又配置了 `requiredDuringSchedulingIgnoredDuringExecution`，就要优先怀疑：

- 节点标签是否真的存在
- 值是否拼错
- 规则是否过于严格

#### Pod 亲和性/反亲和性（Pod Affinity/Anti-Affinity）

**Pod 亲和性**：让 Pod 与某些 Pod 调度在一起（比如：应用和缓存调度在同一节点）
**Pod 反亲和性**：让 Pod 与某些 Pod 分开调度（比如：多个副本分散到不同节点，提高可用性）

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web-pod
spec:
  affinity:
    # Pod 亲和性：与 cache Pod 调度在一起（同一节点）
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:  # 硬性要求
      - labelSelector:        # 选择哪些 Pod
          matchLabels:
            app: cache        # 选择标签 app=cache 的 Pod
        topologyKey: kubernetes.io/hostname  # 拓扑键：同一节点（hostname）
        # 其他常用拓扑键：
        # - kubernetes.io/hostname：同一节点
        # - topology.kubernetes.io/zone：同一可用区
        # - topology.kubernetes.io/region：同一区域
    
    # Pod 反亲和性：与 web Pod 分开调度（不同节点）
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:  # 软性偏好
      - weight: 100
        podAffinityTerm:     # 反亲和性条件
          labelSelector:
            matchLabels:
              app: web       # 避免与标签 app=web 的 Pod 在同一节点
          topologyKey: kubernetes.io/hostname
  containers:
  - name: nginx
    image: nginx:1.21
```

**实际应用场景**：

**场景 1：Pod 亲和性** - 应用和缓存在一起
```yaml
# Web 应用希望和 Redis 缓存调度在同一节点，减少网络延迟
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
        app: redis
        topologyKey: kubernetes.io/hostname
```
    
**场景 2：Pod 反亲和性** - 高可用部署
```yaml
# 同一个应用的多个副本分散到不同节点，避免单点故障
    podAntiAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
  - labelSelector:
            matchLabels:
        app: nginx
          topologyKey: kubernetes.io/hostname
```

**topologyKey 说明**：
- `kubernetes.io/hostname`：同一节点
- `topology.kubernetes.io/zone`：同一可用区（可用区是云提供商的物理隔离区域）
- `topology.kubernetes.io/region`：同一区域（区域是地理区域，如北京、上海）

#### 亲和性和反亲和性的典型用法

最常见的组合并不是"全都上亲和性"，而是按目标区分：

- **低延迟需求**：应用和缓存放近一点，用 Pod Affinity
- **高可用需求**：同类副本尽量打散，用 Pod Anti-Affinity
- **资源匹配需求**：特定工作负载去特定机器，用 Node Affinity

经验上：

- 追求可用性时，反亲和性比亲和性更常用
- 大规模集群里，过度复杂的亲和性规则会增加调度成本

### 4.3 污点与容忍度（Taint & Toleration）

#### 污点（Taint）- 在节点上设置

**什么是污点？**
污点是节点上的标记，表示"这个节点不欢迎某些 Pod"。只有能容忍这个污点的 Pod 才能调度到这个节点。

**使用场景**：
- 专用节点：某些节点只给特定应用使用（如 GPU 节点）
- 维护节点：节点需要维护，不想调度新 Pod
- 测试节点：某些节点只用于测试

```bash
# 添加污点（格式：key=value:effect）
kubectl taint nodes node1 key=value:NoSchedule

# 污点效果（effect）说明：
# NoSchedule：不调度新 Pod 到这个节点（现有 Pod 不受影响）
# PreferNoSchedule：尽量不调度，但如果没有其他节点可选，也可以调度
# NoExecute：不调度新 Pod，并且驱逐现有不能容忍的 Pod

# 实际例子
kubectl taint nodes gpu-node1 gpu=true:NoSchedule        # GPU 节点，只允许有容忍度的 Pod
kubectl taint nodes maintenance-node maintenance=true:NoExecute  # 维护节点，驱逐所有 Pod

# 查看节点的污点
kubectl describe node node1 | grep Taints

# 删除污点（在 key 后面加 -）
kubectl taint nodes node1 key:NoSchedule-
```

#### 容忍度（Toleration）- 在 Pod 上设置

**什么是容忍度？**
容忍度是 Pod 上的配置，表示"这个 Pod 可以容忍某些污点"，从而能够调度到有污点的节点。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-pod
spec:
  tolerations:             # 容忍度列表
  # 方式 1：精确匹配污点
  - key: "gpu"             # 污点的键
    operator: "Equal"      # 操作符：Equal（精确匹配）或 Exists（存在即可）
    value: "true"          # 污点的值（operator=Equal 时需要）
    effect: "NoSchedule"   # 污点的效果
  
  # 方式 2：容忍所有污点（不推荐，太宽泛）
  - operator: "Exists"     # 只要污点的键存在就容忍（忽略 value 和 effect）
  
  # 方式 3：容忍特定键的所有效果
  - key: "maintenance"
    operator: "Exists"
    effect: "NoExecute"    # 只容忍 NoExecute 效果
  containers:
  - name: app
    image: nginx:1.21
```

**实际例子**：

**场景 1：GPU 节点专用**
```bash
# 1. 给 GPU 节点打污点
kubectl taint nodes gpu-node1 gpu=true:NoSchedule

# 2. 在需要 GPU 的 Pod 中添加容忍度
```
```yaml
spec:
  tolerations:
  - key: "gpu"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

**场景 2：维护节点**
```bash
# 节点需要维护，驱逐所有 Pod
kubectl taint nodes node1 maintenance=true:NoExecute
  
# 系统 Pod（如 kube-proxy）需要容忍所有污点才能继续运行
```

**污点和容忍度的关系**：
- 节点有污点 → 普通 Pod 不能调度
- Pod 有容忍度 → 可以调度到有对应污点的节点
- 没有污点的节点 → 所有 Pod 都可以调度（除非有其他限制）

#### 不要把 toleration 误解成"强制调度"

这是非常常见的误区：

- `toleration` 的意思是"允许我待在这种节点上"
- 它**不是**"请一定把我调度到这种节点上"

如果你想表达"只能去 GPU 节点"，通常需要组合：

- 节点有污点，避免普通 Pod 进入
- Pod 有对应 toleration，允许进入
- Pod 还有 `nodeSelector` 或 `nodeAffinity`，明确指定目标节点类别

这三者合起来，约束才完整。

#### 一个完整的专用节点思路

以 GPU 节点为例，比较完整的设计通常是：

1. 节点打标签：`accelerator=nvidia`
2. 节点打污点：`gpu=true:NoSchedule`
3. GPU 工作负载添加：
   - `tolerations`
   - `nodeSelector` 或 `nodeAffinity`

这样可以同时实现：

- 普通 Pod 进不来
- GPU Pod 可以进来
- GPU Pod 不会被误调度到普通节点

## 5. 生命周期概念

### 5.1 Pod 生命周期

**Pod 生命周期阶段**：
Pod 从创建到终止会经历不同的阶段（Phase），可以通过 `kubectl get pods` 查看。

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌──────────┐
│ Pending │───>│ Running │───>│Succeeded│ or │  Failed  │
└─────────┘    └─────────┘    └─────────┘    └──────────┘
     │              │
     │              ▼
     │         ┌─────────┐
     └────────>│ Unknown │
               └─────────┘
```

**各阶段说明**：

| 阶段 | 说明 | 常见原因 | 查看方法 |
|------|------|---------|---------|
| **Pending** | 等待中 | 等待调度、拉取镜像、初始化 | `kubectl get pods` 显示 Pending |
| **Running** | 运行中 | 至少一个容器正在运行 | `kubectl get pods` 显示 Running |
| **Succeeded** | 成功终止 | 所有容器正常退出（退出码 0） | 通常用于 Job 类型的 Pod |
| **Failed** | 失败终止 | 至少一个容器异常退出（退出码非 0） | 需要查看日志排查 |
| **Unknown** | 未知状态 | 无法与节点通信，无法获取状态 | 通常是节点问题 |

**查看 Pod 阶段**：
```bash
# 方式 1：简单查看
kubectl get pods
# NAME         READY   STATUS    RESTARTS   AGE
# nginx-pod    1/1     Running   0          5m

# 方式 2：查看详细信息
kubectl describe pod <pod-name>
# Status:  Running
# Phase:   Running

# 方式 3：使用 JSONPath
kubectl get pod <pod-name> -o jsonpath='{.status.phase}'
```

**阶段转换示例**：

```bash
# 1. 创建 Pod
kubectl create pod nginx-pod --image=nginx

# 2. 查看状态（通常是 Pending）
kubectl get pods
# NAME         READY   STATUS    RESTARTS   AGE
# nginx-pod    0/1     Pending   0          5s

# 3. 等待调度和启动（变为 Running）
kubectl get pods
# NAME         READY   STATUS    RESTARTS   AGE
# nginx-pod    1/1     Running   0          30s

# 4. 如果容器退出（根据退出码和重启策略）
# - 正常退出（退出码 0）→ Succeeded
# - 异常退出（退出码非 0）→ Failed
# - 有重启策略 → 可能重新变为 Running
```

**常见问题排查**：

```bash
# 问题 1：Pod 一直处于 Pending
# 原因：无法调度（资源不足、节点选择器不匹配、污点等）
# 解决：
kubectl describe pod <pod-name>  # 查看 Events 部分
kubectl get events --sort-by='.lastTimestamp'  # 查看最近事件

# 问题 2：Pod 处于 Failed
# 原因：容器启动失败、镜像拉取失败、应用崩溃
# 解决：
kubectl logs <pod-name>           # 查看容器日志
kubectl describe pod <pod-name>   # 查看详细错误信息

# 问题 3：Pod 处于 Unknown
# 原因：节点失联、网络问题
# 解决：
kubectl get nodes                 # 检查节点状态
kubectl describe node <node-name> # 查看节点详情
```

### 5.1.1 Pod 生命周期不等于容器进程生命周期

这也是一个特别容易混淆的点：

- Pod 是 Kubernetes 的调度与管理单元
- 容器是 Pod 内具体运行的进程载体

一个 Pod 里可以有多个容器，它们共享：

- 网络命名空间
- 存储卷
- 生命周期边界

因此，排查问题时要分清：

- 是 Pod 没被调度成功
- 还是 Pod 已经起来了，但其中某个容器没启动成功
- 还是容器启动成功了，但没有 Ready

### 5.2 容器状态

容器有三种状态，可以通过 `kubectl describe pod <pod-name>` 查看：

```yaml
# 容器状态说明
Waiting:     等待启动（正在拉取镜像、等待依赖、等待初始化）
Running:     正在运行（容器正常执行中）
Terminated:  已终止（容器退出，可能是正常退出或出错）
```

**查看容器状态**：
```bash
# 查看 Pod 状态
kubectl get pods

# 查看详细信息（包含容器状态）
kubectl describe pod <pod-name>

# 输出示例：
# Containers:
#   nginx:
#     Container ID:   docker://abc123...
#     Image:          nginx:1.21
#     Image ID:       docker-pullable://nginx@sha256:...
#     State:          Running                    # 容器状态
#       Started:      Mon, 01 Jan 2024 10:00:00
#     Ready:          True                       # 是否就绪
#     Restart Count:  0                          # 重启次数
```

**状态转换**：
```
Waiting → Running → Terminated
   ↓         ↓
   └─────────┘ (重启时)
```

#### Waiting 不一定是坏事，但要看 Reason

容器处于 `Waiting` 很常见，关键是看 `reason`：

| Reason | 通常含义 | 典型排查方向 |
|--------|----------|--------------|
| `ContainerCreating` | 正在创建容器 | 正常启动过程 |
| `ImagePullBackOff` | 镜像拉取失败并退避重试 | 镜像名、仓库认证、网络 |
| `ErrImagePull` | 拉取镜像时报错 | 镜像不存在、权限不足 |
| `CrashLoopBackOff` | 容器不断崩溃并退避重启 | 启动命令、配置、依赖、探针 |

所以只看 `STATUS` 列还不够，通常还要结合：

- `kubectl describe pod`
- `kubectl logs`
- `kubectl get events`

### 5.3 重启策略（restartPolicy）

**什么是重启策略？**
重启策略定义了容器退出后是否自动重启。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
spec:
  restartPolicy: Always    # 重启策略：Always / OnFailure / Never
  containers:
  - name: nginx
    image: nginx:1.21
```

**三种重启策略**：

| 策略 | 说明 | 使用场景 | 示例 |
|------|------|---------|------|
| **Always** | 总是重启（无论退出码是什么） | Deployment、StatefulSet、DaemonSet | Web 服务、API 服务 |
| **OnFailure** | 失败时重启（退出码非 0） | Job、CronJob | 批处理任务、定时任务 |
| **Never** | 从不重启 | 一次性任务 | 测试任务、调试任务 |

**实际例子**：

```yaml
# 例子 1：Web 服务（需要持续运行）
apiVersion: v1
kind: Pod
metadata:
  name: web-server
spec:
  restartPolicy: Always    # 总是重启，确保服务持续运行
  containers:
  - name: nginx
    image: nginx:1.21

# 例子 2：批处理任务（失败才重启）
apiVersion: batch/v1
kind: Job
metadata:
  name: data-processor
spec:
  template:
    spec:
      restartPolicy: OnFailure  # 失败时重启，成功就结束
      containers:
      - name: processor
        image: data-processor:latest

# 例子 3：测试任务（不重启）
apiVersion: v1
kind: Pod
metadata:
  name: test-runner
spec:
  restartPolicy: Never     # 不重启，查看原始结果
  containers:
  - name: test
    image: test-runner:latest
```

**重要提示**：
- Pod 级别的策略，不是容器级别
- `Always` 是 Deployment 等控制器管理的 Pod 的默认值
- `OnFailure` 是 Job 的默认值
- `Never` 需要显式指定

### 5.3.1 Pod 删除时到底发生了什么

很多人以为 `kubectl delete pod` 就是"立刻杀掉"，其实通常不是。

默认情况下，Pod 删除流程大致如下：

1. API Server 给对象打上删除时间
2. Pod 进入 `Terminating`
3. kubelet 给容器发送终止信号
4. 等待 `terminationGracePeriodSeconds`
5. 到时仍未退出，再强制终止

这就是为什么很多服务型应用需要做：

- 优雅停机
- 连接摘除
- 请求收尾

否则发布或缩容时就容易丢请求。

### 5.4 Pod 条件（Conditions）

**什么是 Pod 条件？**
Pod 条件描述了 Pod 在不同阶段的状态，帮助你诊断 Pod 的问题。

```yaml
status:
  conditions:              # Pod 条件列表
  - type: PodScheduled     # 条件类型
    status: "True"         # 状态：True（满足）/ False（不满足）/ Unknown（未知）
    lastTransitionTime: "2024-01-01T10:00:00Z"
    reason: ""             # 原因说明
  - type: Initialized
    status: "True"
  - type: ContainersReady
    status: "True"
  - type: Ready
    status: "True"
```

**四种条件类型**：

| 条件类型 | 说明 | 何时为 True |
|---------|------|-----------|
| **PodScheduled** | Pod 已被调度到节点 | Pod 已分配到某个节点 |
| **Initialized** | Init 容器已完成 | 所有 Init 容器执行完成 |
| **ContainersReady** | 所有容器就绪 | 所有容器通过就绪探针 |
| **Ready** | Pod 就绪，可接收流量 | Pod 可以处理请求（最重要） |

**查看 Pod 条件**：
```bash
# 方式 1：使用 kubectl get（简洁）
kubectl get pods
# READY 列显示：1/1 表示 1 个容器就绪 / 总共 1 个容器

# 方式 2：使用 kubectl describe（详细）
kubectl describe pod <pod-name>
# 在 Conditions 部分可以看到所有条件

# 方式 3：使用 JSONPath（精确）
kubectl get pod <pod-name> -o jsonpath='{.status.conditions[*].type}{"\n"}{.status.conditions[*].status}'
```

**常见问题诊断**：

```bash
# 问题 1：PodScheduled = False
# 原因：没有可用节点、资源不足、节点选择器不匹配
# 解决：检查节点状态、资源请求、节点选择器

# 问题 2：Initialized = False
# 原因：Init 容器执行失败
# 解决：查看 Init 容器日志：kubectl logs <pod-name> -c <init-container-name>

# 问题 3：ContainersReady = False
# 原因：容器未就绪（可能正在启动、就绪探针失败）
# 解决：查看容器状态和日志

# 问题 4：Ready = False
# 原因：就绪探针失败、容器崩溃
# 解决：检查就绪探针配置、查看容器日志
```

**实际例子**：
```bash
# 查看 Pod 的详细条件
kubectl describe pod nginx-pod

# 输出示例：
# Conditions:
#   Type              Status
#   ----              ------
#   PodScheduled      True      # ✅ 已调度
#   Initialized       True      # ✅ 初始化完成
#   ContainersReady   True      # ✅ 容器就绪
#   Ready             True      # ✅ Pod 就绪
```

**条件状态说明**：
- `True`：条件满足，Pod 处于该状态
- `False`：条件不满足，Pod 有问题
- `Unknown`：无法确定条件状态（通常是节点通信问题）

#### Ready 和 Running 不是一回事

这是线上排查里最常见的误解之一：

- `Running`：容器进程已经起来了
- `Ready`：Pod 已经准备好接收流量了

一个 Pod 完全可能：

- 已经是 `Running`
- 但 `Ready=False`

这种情况下：

- 进程可能活着
- 但 Service 不一定会把流量转给它

所以排查"服务为什么访问不到"时，不能只看 `Running`，一定要看 `READY` 列和 `conditions`。

## 6. 服务发现概念

### 6.1 Service 类型

Service 是访问 Pod 的稳定端点。有 4 种类型，适用于不同场景。

#### 类型 1：ClusterIP（默认）- 集群内部访问

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  type: ClusterIP        # 默认类型，可以不写
  selector:
    app: nginx
  ports:
  - port: 80            # Service 端口
    targetPort: 80      # Pod 端口
```

**特点**：
- 只在集群内部可访问
- 分配一个虚拟 IP（ClusterIP）
- 最常用，适合集群内部服务间通信

#### 类型 2：NodePort - 通过节点端口访问

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  type: NodePort        # NodePort 类型
  selector:
    app: nginx
  ports:
  - port: 80            # Service 端口
    targetPort: 80      # Pod 端口
    nodePort: 30080     # 节点端口（可选，不指定会自动分配 30000-32767）
```

**特点**：
- 在每个节点上开放一个端口（30000-32767）
- 可以通过 `<节点IP>:<nodePort>` 从集群外部访问
- 适合开发测试，生产环境通常用 Ingress

**访问方式**：
- 集群内：`http://nginx-service:80`
- 集群外：`http://<任意节点IP>:30080`

#### 类型 3：LoadBalancer - 云负载均衡器

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  type: LoadBalancer    # LoadBalancer 类型
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
```

**特点**：
- 使用云提供商的负载均衡器（如 AWS ELB、阿里云 SLB）
- 自动分配外部 IP
- 适合生产环境，需要云提供商支持

#### 类型 4：ExternalName - 指向外部服务

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-db
spec:
  type: ExternalName    # ExternalName 类型
  externalName: database.example.com  # 外部域名
```

**特点**：
- 不选择 Pod，而是指向外部服务
- 创建 DNS CNAME 记录
- 适合访问集群外的服务

**类型对比表**：

| 类型 | 集群内访问 | 集群外访问 | 使用场景 |
|------|-----------|-----------|---------|
| ClusterIP | ✅ | ❌ | 集群内部服务间通信（最常用）|
| NodePort | ✅ | ✅ | 开发测试，简单外部访问 |
| LoadBalancer | ✅ | ✅ | 生产环境，需要云支持 |
| ExternalName | ✅ | - | 访问外部服务 |

### 6.2 Endpoint

**什么是 Endpoint？**
Endpoint 是 Service 和 Pod 之间的桥梁，记录了 Service 选择的所有 Pod 的 IP 和端口。

在较新的 Kubernetes 版本和现代实现中，底层更常见的是 **EndpointSlice**。你可以简单理解为：

- `Endpoints`：早期、较直观的表示方式
- `EndpointSlice`：更适合大规模场景的切片化表示

但对初学者来说，先理解"Service 后面其实维护着一组后端地址"就够了。

**工作原理**：
1. Service 通过 selector 选择 Pod
2. Kubernetes 自动创建 Endpoint 对象，记录所有匹配 Pod 的 IP:Port
3. Service 将流量转发到 Endpoint 中记录的 Pod

```bash
# 查看 Service 的 Endpoints
kubectl get endpoints nginx-service

# 输出示例
NAME            ENDPOINTS                         AGE
nginx-service   10.244.1.5:80,10.244.2.6:80      5m
#              ↑ Pod1的IP:端口  ↑ Pod2的IP:端口
```

**实际例子**：
```bash
# 1. 创建 Service
kubectl create service clusterip nginx-service --tcp=80:80 --dry-run=client -o yaml

# 2. 查看 Endpoint（Service 会自动创建）
kubectl get endpoints nginx-service

# 3. 查看详细信息
kubectl describe endpoints nginx-service

# 输出示例：
# Name:         nginx-service
# Namespace:    default
# Labels:       <none>
# Annotations:  <none>
# Subsets:
#   Addresses:          10.244.1.5,10.244.2.6    # Pod 的 IP 地址
#   NotReadyAddresses:  <none>
#   Ports:
#     Name     Port  Protocol
#     <unset>  80    TCP
```

**重要提示**：
- Endpoint 是自动创建的，你不需要手动创建
- 当 Pod 被删除或创建时，Endpoint 会自动更新
- 如果 Service 没有匹配的 Pod，Endpoint 的 Addresses 会是空的
- 可以通过 `kubectl get endpoints` 检查 Service 是否找到了 Pod

#### 一条非常实用的排查链路

当你访问 Service 不通时，建议按这个顺序查：

1. `Service` 是否存在
2. `selector` 是否写对
3. `Pod labels` 是否匹配
4. `Endpoints` 是否为空
5. Pod 是否 `Ready`
6. `targetPort` 是否和容器监听端口一致

这个链路能解决大量"明明 Pod 在跑，但服务不通"的问题。

### 6.3 DNS 解析

**Kubernetes 内置 DNS**
Kubernetes 集群内置了 DNS 服务（通常是 CoreDNS），为 Service 提供 DNS 解析。

**DNS 命名规则**：

```bash
# 格式：<service-name>.<namespace>.svc.cluster.local

# 1. 完整域名（最明确，但通常不需要）
nginx-service.default.svc.cluster.local

# 2. 简化格式（跨命名空间）
nginx-service.production              # 访问 production 命名空间的 nginx-service

# 3. 最短格式（同命名空间，最常用）
nginx-service                         # 访问同命名空间的 nginx-service
```

**实际例子**：

```bash
# 场景 1：同命名空间访问（最常用）
# 在 default 命名空间的 Pod 中访问 default 命名空间的 Service
curl http://nginx-service
curl http://nginx-service:80

# 场景 2：跨命名空间访问
# 在 default 命名空间的 Pod 中访问 production 命名空间的 Service
curl http://nginx-service.production
curl http://nginx-service.production:80

# 场景 3：使用完整域名（很少用）
curl http://nginx-service.default.svc.cluster.local
```

**DNS 解析优先级**：
1. 先按 Pod 当前命名空间的 DNS 搜索域补全
2. 再按集群 DNS 搜索路径继续尝试
3. 如果写的是完整域名，则直接按完整域名解析

更实用的理解方式是：

- 同命名空间访问时，直接写 Service 名最方便
- 跨命名空间访问时，至少写成 `<service-name>.<namespace>`
- 需要最明确、最不歧义时，用完整 FQDN

**测试 DNS 解析**：
```bash
# 在 Pod 中测试 DNS
kubectl run -it --rm debug --image=busybox --restart=Never -- nslookup nginx-service

# 或者使用 dig
kubectl run -it --rm debug --image=busybox --restart=Never -- dig nginx-service.default.svc.cluster.local
```

**常见问题**：
- **DNS 解析失败**：检查 Service 是否存在，selector 是否正确
- **跨命名空间访问失败**：使用完整格式 `<service-name>.<namespace>`
- **DNS 延迟**：新创建的 Service 可能需要几秒钟才能被 DNS 解析

### 6.4 端口概念一次讲清

Kubernetes 里最容易看混的不是 Service 类型，而是各种端口字段。可以按这张表理解：

| 字段 | 所属资源 | 作用 |
|------|----------|------|
| `containerPort` | Pod/Container | 容器进程监听的端口说明 |
| `targetPort` | Service | Service 最终转发到后端 Pod 的端口 |
| `port` | Service | Service 自己暴露的端口 |
| `nodePort` | Service(NodePort) | 节点对外开放的端口 |

一个典型链路如下：

```text
客户端 -> Service:80 -> targetPort:8080 -> Pod容器监听 8080
```

如果 `Service` 能访问到，但应用响应异常，常见问题往往就在：

- `targetPort` 写错
- 应用没监听你以为的那个端口
- 就绪探针没通过，导致后端没进可用端点

## 7. 配置快速参考

### 7.1 完整的 Pod 配置示例

这是一个包含所有常见配置的完整 Pod 示例，帮助你理解各个字段：

```yaml
apiVersion: v1                    # API 版本
kind: Pod                         # 资源类型
metadata:                         # 元数据部分
  name: my-app                    # Pod 名称（必填）
  namespace: default              # 命名空间（可选，默认 default）
  labels:                         # 标签（用于选择和分组）
    app: my-app                   # 应用名称
    environment: production       # 环境
    version: v1.0.0                # 版本
  annotations:                    # 注解（存储元数据）
    description: "Main application pod"
    prometheus.io/scrape: "true"
spec:                             # 规约部分（你定义的期望状态）
  # 调度相关
  nodeSelector:                   # 节点选择器（简单方式）
    disktype: ssd
  affinity:                       # 亲和性（灵活方式）
    nodeAffinity:                 # 节点亲和性
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/os
            operator: In
            values: ["linux"]
  tolerations:                    # 容忍度（允许调度到有污点的节点）
  - key: "gpu"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
  
  # 容器配置
  containers:
  - name: app                     # 容器名称
    image: nginx:1.21             # 镜像（必填）
    imagePullPolicy: IfNotPresent # 镜像拉取策略：Always/Never/IfNotPresent
    
    # 资源限制
    resources:
      requests:                   # 资源请求（调度保证）
        cpu: "250m"               # CPU：0.25 核心
        memory: "128Mi"           # 内存：128 MiB
      limits:                     # 资源限制（最大使用）
        cpu: "500m"               # CPU：最多 0.5 核心
        memory: "256Mi"           # 内存：最多 256 MiB
    
    # 端口
    ports:
    - name: http                  # 端口名称
      containerPort: 80           # 容器端口
      protocol: TCP               # 协议：TCP/UDP
    
    # 环境变量
    env:
    - name: ENV_VAR               # 环境变量名
      value: "value"              # 环境变量值
    - name: CONFIG_MAP_VAR
      valueFrom:
        configMapKeyRef:
          name: my-config         # ConfigMap 名称
          key: config-key         # ConfigMap 的键
  
  # 重启策略
  restartPolicy: Always           # Always/OnFailure/Never
  
  # 其他
  terminationGracePeriodSeconds: 30  # 优雅终止时间（秒）
```

### 7.2 完整的 Service 配置示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service                # Service 名称
  namespace: default              # 命名空间
  labels:                         # 标签
    app: my-app
spec:
  type: ClusterIP                 # Service 类型：ClusterIP/NodePort/LoadBalancer/ExternalName
  selector:                       # 选择器（选择要代理的 Pod）
    app: my-app                   # 选择标签 app=my-app 的 Pod
  ports:                          # 端口列表
  - name: http                    # 端口名称
    port: 80                      # Service 端口（集群内访问的端口）
    targetPort: 80                # Pod 端口（容器监听的端口）
    protocol: TCP                 # 协议：TCP/UDP
    nodePort: 30080               # 节点端口（仅 NodePort 类型需要）
  sessionAffinity: None          # 会话亲和性：None/ClientIP
```

### 7.3 配置字段速查表

| 字段 | 位置 | 说明 | 必填 |
|------|------|------|------|
| `apiVersion` | 根 | API 版本 | ✅ |
| `kind` | 根 | 资源类型（Pod/Service/Deployment 等）| ✅ |
| `metadata.name` | metadata | 资源名称 | ✅ |
| `metadata.namespace` | metadata | 命名空间 | ❌ |
| `metadata.labels` | metadata | 标签 | ❌ |
| `spec.containers` | spec | 容器列表 | ✅（Pod）|
| `spec.containers[].image` | spec.containers | 镜像 | ✅ |
| `spec.selector` | spec | 选择器 | ✅（Service/Deployment）|
| `spec.type` | spec | Service 类型 | ❌（默认 ClusterIP）|
| `spec.restartPolicy` | spec | 重启策略 | ❌（默认 Always）|

### 7.3.1 看不懂 YAML 时的最短检查单

如果你拿到一个 Kubernetes YAML，觉得字段很多、完全看不懂，先只检查下面这几项：

1. `apiVersion` 和 `kind` 是否匹配
2. `metadata.name` 是否合理且唯一
3. `metadata.namespace` 是否写在你期望的环境里
4. `labels` 是否完整、可读、能支撑 selector
5. `selector` 和目标 Pod 的 labels 是否真的一致
6. 镜像名、端口、资源配置是否明显有误
7. 是否混入了不该手写的 `status`

仅靠这 7 步，已经能提前发现很多初学者常见错误。

### 7.4 常见配置错误

**错误 1：Deployment 的 selector 和 labels 不匹配**
```yaml
# ❌ 错误：selector 和 template.labels 不匹配
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: web        # 错误！应该是 nginx

# ✅ 正确：必须匹配
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx      # 正确！
```

**错误 2：资源单位写错**
```yaml
# ❌ 不推荐：团队里混用不同 CPU 表示法，可读性差
resources:
  requests:
    cpu: 0.5

# ✅ 更推荐：统一使用毫核，便于阅读和对比
resources:
  requests:
    cpu: "500m"        # 0.5 CPU = 500m
```

**说明**：
- `0.5` 在 Kubernetes 中是合法写法
- 但团队实践中通常更推荐统一写成 `500m`
- 真正常见的错误是把内存单位、CPU 单位、引号风格混得很乱，导致配置不一致或误读

**错误 3：Service 的 selector 找不到 Pod**
```yaml
# ❌ 错误：selector 的标签和 Pod 的标签不匹配
# Service
spec:
  selector:
    app: nginx

# Pod 的标签
metadata:
  labels:
    application: nginx  # 错误！应该是 app: nginx

# ✅ 正确：标签必须匹配
# Service
spec:
  selector:
    app: nginx

# Pod
metadata:
  labels:
    app: nginx          # 正确！
```

**错误 4：把 Service selector 写成 Deployment 风格**
```yaml
# ❌ 错误：Service 里误写 matchLabels
apiVersion: v1
kind: Service
spec:
  selector:
    matchLabels:
      app: nginx

# ✅ 正确：Service 直接写键值映射
apiVersion: v1
kind: Service
spec:
  selector:
    app: nginx
```

**错误 5：只看 Running，不看 Ready**
```text
# ❌ 错误理解
Pod 是 Running，所以服务一定可用

# ✅ 正确理解
Running 只代表进程起来了
Ready 才更接近“可以接流量”
```

**错误 6：给 Pod 配了 toleration，就以为一定会去目标节点**
```text
# ❌ 错误理解
有 toleration = 一定调度到带污点节点

# ✅ 正确理解
toleration 只是“允许去”
是否真的去，还要结合 nodeSelector / nodeAffinity / 资源情况 / 调度结果
```

### 7.5 常用排错路径

#### 场景 1：Pod 一直 Pending

优先检查：

1. `kubectl describe pod <pod-name>` 看 Events
2. 节点是否有足够资源
3. `nodeSelector` / `Affinity` 是否过严
4. 是否被污点拦住
5. PVC、镜像、Init 容器是否阻塞

#### 场景 2：Pod 在 Running，但服务访问不通

优先检查：

1. Pod 是否 `Ready`
2. Service `selector` 是否匹配 Pod labels
3. `kubectl get endpoints <service-name>` 是否有后端
4. `targetPort` 是否与应用实际监听端口一致
5. 容器内部服务是否真的在监听该端口

#### 场景 3：Pod 频繁重启

优先检查：

1. `kubectl describe pod <pod-name>`
2. `kubectl logs <pod-name> --previous`
3. 是否 `OOMKilled`
4. 启动命令、配置文件、依赖连接是否有误
5. 存活探针 / 就绪探针是否配置不当

#### 场景 4：Service 能解析域名，但请求超时

优先检查：

1. Pod 是否 Ready
2. 应用是否监听正确地址和端口
3. 网络策略是否拦截
4. 节点网络插件是否异常
5. 如果是云环境，外部 LB 或安全组是否放通

### 7.6 一张“从 YAML 到流量”的总链路

把前面所有概念串起来，最值得记住的是这条链：

1. 你写 `Deployment YAML`
2. Deployment 根据 `spec` 创建 ReplicaSet
3. ReplicaSet 创建 Pod
4. Pod 根据调度规则被分配到 Node
5. kubelet 在 Node 上拉镜像并启动容器
6. Pod 通过条件和探针逐步进入 Ready
7. Service 通过 selector 找到这些 Ready Pod
8. DNS 把 Service 名字解析成稳定地址
9. 流量最终转发到实际 Pod

如果这条链任何一个环节出问题，服务就可能表现异常。

所以排障时，不要只盯一个点，而要顺着整条链去看。

## 8. 常用术语对照表

| 术语 | 中文 | 说明 |
|------|------|------|
| Cluster | 集群 | Kubernetes 管理的一组节点 |
| Node | 节点 | 集群中的一台机器 |
| Pod | 容器组 | 最小的部署单元 |
| Container | 容器 | Pod 中运行的应用实例 |
| Deployment | 部署 | 无状态应用的部署管理 |
| Service | 服务 | 访问 Pod 的稳定端点 |
| Namespace | 命名空间 | 资源隔离 |
| Label | 标签 | 资源分类标识 |
| Selector | 选择器 | 选择特定资源 |
| ReplicaSet | 副本集 | 维护 Pod 副本数 |
| StatefulSet | 有状态集 | 有状态应用的部署管理 |
| DaemonSet | 守护进程集 | 每个节点运行一个 Pod |
| Job | 任务 | 一次性任务 |
| CronJob | 定时任务 | 周期性任务 |
| ConfigMap | 配置映射 | 非敏感配置数据 |
| Secret | 密钥 | 敏感数据 |
| Volume | 存储卷 | 持久化存储 |
| PV | 持久卷 | 集群级存储资源 |
| PVC | 持久卷声明 | 对 PV 的请求 |
| Ingress | 入口 | HTTP(S) 路由规则 |
| NetworkPolicy | 网络策略 | Pod 网络访问控制 |
| RBAC | 角色访问控制 | 权限管理 |

## 9. 实践练习

### 练习 1：使用标签组织资源

```bash
# 创建带标签的 Pod
kubectl run nginx-prod --image=nginx --labels="app=nginx,env=production"
kubectl run nginx-dev --image=nginx --labels="app=nginx,env=development"

# 按标签筛选
kubectl get pods -l env=production
kubectl get pods -l 'env in (production, development)'

# 查看所有标签
kubectl get pods --show-labels

# 清理
kubectl delete pods -l app=nginx
```

### 练习 2：使用命名空间

```bash
# 创建命名空间
kubectl create namespace test-ns

# 在命名空间中创建资源
kubectl run nginx --image=nginx -n test-ns

# 查看特定命名空间的资源
kubectl get pods -n test-ns

# 查看所有命名空间的资源
kubectl get pods --all-namespaces
kubectl get pods -A

# 清理
kubectl delete namespace test-ns
```

### 练习 3：理解资源关系

```bash
# 创建 Deployment
kubectl create deployment web --image=nginx --replicas=3

# 查看创建的资源链
kubectl get deployment web
kubectl get replicaset -l app=web
kubectl get pods -l app=web

# 查看资源详情
kubectl describe deployment web

# 清理
kubectl delete deployment web
```

### 练习 4：自己验证 Service 到 Pod 的关联

```bash
# 1. 创建一个带标签的 Deployment
kubectl create deployment web --image=nginx --replicas=2

# 2. 查看 Pod 标签
kubectl get pods --show-labels

# 3. 创建 Service
kubectl expose deployment web --port=80 --target-port=80 --type=ClusterIP

# 4. 查看 Service
kubectl get svc web

# 5. 查看后端地址
kubectl get endpoints web

# 6. 进入临时 Pod 测试访问
kubectl run -it --rm debug --image=busybox --restart=Never -- wget -qO- http://web

# 7. 清理
kubectl delete deployment web
kubectl delete svc web
```

这个练习最重要的不是命令本身，而是你要刻意观察：

- Deployment 和 Pod 的标签关系
- Service 和 Pod 的 selector 匹配关系
- Endpoints 是否随着 Pod 自动更新

## 10. 一页总结

如果你只想先记住最关键的内容，请记下面这些结论：

- Kubernetes 是**声明式系统**，你写的是目标状态，不是执行步骤
- 绝大多数资源都可以按 `apiVersion -> kind -> metadata -> spec -> status` 的顺序去理解
- `Label` 负责可筛选的身份标识，`Annotation` 负责补充说明
- `selector` 是资源关联的核心，`Service` 和 `Deployment` 都依赖它
- `requests` 影响调度，`limits` 影响运行时约束
- CPU 超限更常见的是被限速，内存超限更常见的是被 OOMKill
- `toleration` 只是允许进入某类节点，不代表一定会调度过去
- `Running` 不等于 `Ready`
- Service 提供的是稳定入口，真正后端是一组会变化的 Pod
- 排查问题时，要顺着"YAML -> 调度 -> 启动 -> Ready -> Service -> Endpoints -> DNS" 这条链去看

## 11. 下一步

- [Pod - 最小调度单元](../02-resources/01-pod.md) - 深入学习 Pod



