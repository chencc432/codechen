# 📘 CRD 基础与 API 扩展

## 为什么 Kubernetes 需要自定义资源

Kubernetes 内置资源已经很多了，但它们主要解决的是通用编排问题：

- 容器怎么跑
- 服务怎么暴露
- 配置怎么注入
- 存储怎么挂载
- 权限怎么控制

可是真实业务和平台场景里，经常需要表达更高层的领域对象：

- 一个数据库集群
- 一个数据同步任务
- 一个租户配额策略
- 一个备份计划
- 一个证书签发请求

这些对象如果只用内置资源拼出来，往往会出现几个问题：

- 语义分散，理解成本高
- 配置文件很多，难以统一管理
- 用户无法直接以领域对象视角操作
- 平台逻辑散落在脚本、Helm 模板和外部工具中

所以 Kubernetes 提供了 **Custom Resource Definition（CRD）**，让你把自己的业务对象变成 Kubernetes API 的一部分。

## 1. 先把几个核心名词讲清楚

### 1.1 CRD 是什么

CRD 全称是 **CustomResourceDefinition**，它是一种特殊资源，用来告诉 Kubernetes：

> “请帮我注册一种新的资源类型，让它像 Pod、Service 一样可以被 API Server 识别。”

例如，你定义了一个 CRD 叫 `MySQLCluster`，那么之后就可以像这样操作：

```bash
kubectl get mysqlclusters
kubectl describe mysqlcluster prod-db
kubectl delete mysqlcluster prod-db
```

### 1.2 CR 是什么

CR 全称是 **Custom Resource**，它是基于 CRD 创建出来的资源实例。

也就是说：

- **CRD** 是“类型定义”
- **CR** 是“这个类型的具体对象”

可以类比成数据库：

| 概念 | 类比 |
|------|------|
| CRD | 表结构定义 |
| CR | 表里的一条记录 |

### 1.3 GVK 和 GVR

Kubernetes 资源世界里两个非常重要的概念是：

- `GroupVersionKind`（GVK）
- `GroupVersionResource`（GVR）

#### GVK

表示“这是什么类型的对象”：

```yaml
apiVersion: mysql.example.com/v1
kind: MySQLCluster
```

这里：

- `Group` = `mysql.example.com`
- `Version` = `v1`
- `Kind` = `MySQLCluster`

#### GVR

表示 API 路径上的资源名：

```text
/apis/mysql.example.com/v1/namespaces/default/mysqlclusters
```

这里的 `mysqlclusters` 就是 `resource`，通常对应 `plural`。

## 2. Kubernetes API 为什么可以扩展

Kubernetes 的强大之处，不只在于它自带一套资源，而在于它本身就是一个**可扩展 API 平台**。

这意味着：

- API Server 不只是“固定写死的资源网关”
- 它还能注册新资源
- 还能对新资源做校验、存储、查询、Watch

所以从平台角度看，Kubernetes 其实像一个：

> 可声明、可扩展、可持续调谐的分布式控制面框架

这也是为什么它后来能承载：

- 数据库 Operator
- 监控 Operator
- 服务网格控制平面
- 内部平台资源模型

## 3. 一个 CRD 最小示例

下面是一个最小可理解的 CRD：

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: mysqlclusters.mysql.example.com
spec:
  group: mysql.example.com
  names:
    kind: MySQLCluster
    plural: mysqlclusters
    singular: mysqlcluster
    shortNames:
    - mysql
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              replicas:
                type: integer
                minimum: 1
              version:
                type: string
```

## 4. 这个 CRD 里最关键的字段

### 4.1 `metadata.name`

格式通常是：

```text
<plural>.<group>
```

例如：

```text
mysqlclusters.mysql.example.com
```

### 4.2 `spec.group`

定义 API 组。这个字段会出现在 `apiVersion` 中。

例如：

```yaml
apiVersion: mysql.example.com/v1
```

### 4.3 `spec.names`

这里定义资源的多种名字形式：

| 字段 | 作用 |
|------|------|
| `kind` | 资源类型名，通常首字母大写 |
| `plural` | 复数资源名，用于 API 路径 |
| `singular` | 单数名 |
| `shortNames` | kubectl 简写 |

### 4.4 `scope`

作用域决定资源是命名空间级还是集群级：

| 值 | 含义 |
|----|------|
| `Namespaced` | 资源属于某个命名空间 |
| `Cluster` | 资源是集群级别 |

判断方法通常是：

- 如果这个对象天然属于某个租户、某个业务空间，通常用 `Namespaced`
- 如果这个对象本身描述的是全局能力或全局策略，可能用 `Cluster`

### 4.5 `versions`

定义这个 CRD 有哪些 API 版本。

最关键的字段有两个：

| 字段 | 作用 |
|------|------|
| `served` | 该版本是否对外提供 API |
| `storage` | etcd 中最终以哪个版本存储 |

通常只能有一个版本 `storage: true`。

## 5. CRD 注册后会发生什么

当你 `kubectl apply -f crd.yaml` 后，大致会发生这些事情：

1. API Server 接收到 CRD 定义
2. API Extension 机制注册新资源类型
3. Kubernetes 为它创建新的 REST API 入口
4. 后续你就可以提交该类型的资源实例
5. 这些 CR 会像其他资源一样存储在 etcd 中

也就是说，CRD 最核心的价值是：

- 不是生成 YAML 模板
- 不是创建控制器
- 而是**扩展 API 面**

## 6. 创建 CR 实例是什么样子

一旦 CRD 注册成功，就可以创建对应的 CR：

```yaml
apiVersion: mysql.example.com/v1
kind: MySQLCluster
metadata:
  name: production-db
  namespace: default
spec:
  replicas: 3
  version: "8.0"
```

之后就可以：

```bash
kubectl get mysqlclusters
kubectl get mysqlcluster production-db -o yaml
```

## 7. Schema 校验为什么重要

如果不给 CRD 写清楚 Schema，就会遇到很多问题：

- 用户随便写字段也能提交
- 类型错误很晚才暴露
- 控制器要处理大量非法输入
- 资源模型容易逐渐失控

所以 `openAPIV3Schema` 不只是“文档说明”，它还是：

- 字段结构定义
- 类型约束
- 最小最大值校验
- 枚举约束
- 默认值策略的基础之一

例如：

```yaml
schema:
  openAPIV3Schema:
    type: object
    required:
    - spec
    properties:
      spec:
        type: object
        required:
        - replicas
        properties:
          replicas:
            type: integer
            minimum: 1
            maximum: 9
          engine:
            type: string
            enum:
            - mysql
            - mariadb
```

## 8. `spec` 和 `status` 应该怎么理解

CRD 一样遵循 Kubernetes 的通用设计哲学：

- `spec`：用户声明的期望状态
- `status`：控制器回填的实际状态

举个例子：

```yaml
apiVersion: mysql.example.com/v1
kind: MySQLCluster
metadata:
  name: prod-db
spec:
  replicas: 3
  version: "8.0"
status:
  readyReplicas: 2
  phase: Provisioning
```

这表示：

- 用户要 3 个副本
- 但当前只有 2 个已就绪

这和 Deployment 的设计思路完全一致。

## 9. Subresources 是什么

CRD 还可以开启子资源：

```yaml
subresources:
  status: {}
  scale: {}
```

### 9.1 `status`

启用后可以让：

- 用户主要改 `spec`
- 控制器主要改 `status`

这样职责更清晰，也更安全。

### 9.2 `scale`

如果你的 CR 有“副本数”这类语义，可以暴露给 Kubernetes 的扩缩容接口使用。

例如 HPA 等工具可能需要这种标准化能力。

## 10. Printer Columns 为什么很有用

一个成熟的 CRD 通常不会满足于 `kubectl get` 只能显示 NAME 和 AGE，而是会定义附加列：

```yaml
additionalPrinterColumns:
- name: Ready
  type: integer
  jsonPath: .status.readyReplicas
- name: Version
  type: string
  jsonPath: .spec.version
- name: Phase
  type: string
  jsonPath: .status.phase
```

这样你平时查看时会更像原生资源：

```bash
kubectl get mysqlclusters
```

输出可以更友好：

```text
NAME      READY   VERSION   PHASE        AGE
prod-db   2       8.0       Provisioning 3m
```

这会显著提升使用体验。

## 11. 什么情况下适合用 CRD

不是所有场景都值得引入 CRD。适合使用 CRD 的场景通常有这些特点：

- 有稳定、明确的业务领域对象
- 这个对象需要长期维护和演进
- 希望用户通过 Kubernetes 原生方式声明需求
- 后面有控制逻辑要围绕这个对象持续执行
- 希望统一平台抽象，而不是到处散落脚本和模板

### 典型适合场景

- 数据库集群
- 中间件实例
- 备份策略
- 灰度发布策略
- 多租户配额对象
- 内部平台产品对象

### 不太适合的场景

- 一次性、临时脚本任务
- 只是简单配置拼装
- 没有持续控制逻辑
- 对象语义非常不稳定，今天改、明天废弃

## 12. CRD 与 ConfigMap、Helm、Webhook 的边界

很多团队会困惑：到底该用 CRD 还是其他方式？

### CRD vs ConfigMap

- `ConfigMap` 更适合“给程序传配置”
- `CRD` 更适合“把领域对象定义成平台 API”

### CRD vs Helm

- Helm 更像“打包和分发模板”
- CRD 更像“声明新的资源类型”

### CRD vs Webhook

- CRD 负责“定义资源”
- Webhook 更适合“在创建或更新时做校验或修改”

它们常常是配合关系，不是替代关系。

## 13. 常见误区

### 误区 1：有了 CRD 就等于有了 Operator

不对。  
**只有 CRD 没有控制器时，它通常只是存了一份结构化数据。**

### 误区 2：所有平台概念都要做成 CRD

也不对。  
如果对象本身不稳定、没有持续控制逻辑，强行做 CRD 反而会让维护成本变高。

### 误区 3：Schema 可以后面再补

短期可能省事，长期通常会造成：

- 字段随意膨胀
- 类型不一致
- 兼容性失控

Schema 应该尽早设计清楚。

## 14. 一页总结

- CRD 的作用是把新的资源类型注册进 Kubernetes API
- CRD 是类型定义，CR 是实例对象
- `group/version/kind` 决定资源身份，`plural` 决定 API 路径
- `spec` 表示期望状态，`status` 表示实际状态
- Schema 校验是保证资源长期可维护的基础
- 单独 CRD 只解决“定义资源”，不自动解决“执行逻辑”

## 下一步

- [控制器、Reconcile 与 Operator 模式](./02-controller-and-operator.md)
