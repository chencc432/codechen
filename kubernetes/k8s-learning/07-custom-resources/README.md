# 🧩 Kubernetes 自定义资源专题

## 这个专题解决什么问题

当你学完 Pod、Deployment、Service 这些内置资源后，很容易产生一个新的问题：

> 如果我的业务概念并不是 Kubernetes 自带的资源类型，那我应该怎么把它变成 Kubernetes 原生的一部分？

例如：

- 一个 `MySQLCluster`
- 一个 `RedisShard`
- 一个 `DataPipeline`
- 一个 `BackupPolicy`
- 一个 `TenantQuota`

这些对象都不是 Kubernetes 内置资源，但在真实平台或中间件场景里又非常常见。  
这时就需要 **自定义资源（Custom Resource）** 和 **自定义控制器（Custom Controller）**。

这个专题的目标，是把下面这条链讲清楚：

```text
业务领域模型
  -> 自定义资源定义（CRD）
  -> 自定义资源实例（CR）
  -> 控制器 Watch / Reconcile
  -> 自动创建和维护底层 Kubernetes 对象
  -> 最终形成 Operator 模式
```

## 学完后你应该掌握什么

读完这个专题后，你应该能回答这些问题：

- CRD 和 CR 到底分别是什么
- 为什么说“单独的 CRD 只是一个数据模型，还不是完整能力”
- 自定义控制器的工作循环到底是如何运作的
- Operator 和普通控制器是什么关系
- 一个好的自定义资源 Schema 应该怎么设计
- 什么时候应该创建新资源，什么时候只需要用 ConfigMap、Helm、脚本
- 多版本 CRD、状态字段、打印列、Finalizer、Webhook 分别解决什么问题

## 推荐阅读顺序

建议按下面顺序阅读：

1. [CRD 基础与 API 扩展](./01-crd-basics.md)
2. [控制器、Reconcile 与 Operator 模式](./02-controller-and-operator.md)
3. [CRD 设计、版本演进与最佳实践](./03-crd-design-and-versioning.md)
4. [真实案例拆解：从领域模型到 CRD 设计](./04-real-world-crd-case-study.md)
5. [Kubebuilder 工作流：从脚手架到控制器骨架](./05-kubebuilder-workflow.md)
6. [从 0 到 1 设计一个完整 Operator 的实战样例](./06-operator-end-to-end-example.md)
7. [Operator 测试、调试与发布](./07-operator-testing-debugging-release.md)

## 适合谁读

这个专题特别适合下面几类读者：

- 已经会写和使用 Kubernetes YAML，但希望进一步扩展平台能力
- 想理解 Helm、Operator、平台控制面背后原理的人
- 想写内部平台、数据库 Operator、自动化治理工具的人
- 想系统理解 `client-go`、Informer、Controller Runtime 应用场景的人

## 学习前建议先具备的基础

在进入这个专题之前，建议你至少已经读过：

- [核心概念与术语](../01-basics/03-concepts.md)
- [核心组件详解](../01-basics/02-components.md)
- [Informer 机制详解](../05-client-go/04-informer.md)
- [实战项目：自定义控制器](../05-client-go/05-controller-demo.md)

## 这个专题和现有文档的关系

仓库里原来已经在 `client-go` 的实战文档中带了一部分 CRD 附录内容，但它更偏“控制器实战上下文中的 CRD 介绍”。  
这个专题则是专门面向 **Kubernetes 自定义 API 扩展** 的系统讲解，重点在：

- 先讲概念边界
- 再讲控制器行为模型
- 再讲设计方法与工程实践

所以你可以把它看成：

- `05-client-go`：偏开发实现
- `07-custom-resources`：偏专题化、体系化理解

## 一条最重要的理解主线

如果你只先记住一句话，那就是：

> CRD 负责把“你的领域对象”注册进 Kubernetes API；控制器负责把“这个对象的期望状态”变成真实运行结果。

也就是说：

- **CRD** 负责“定义名词”
- **Controller** 负责“实现动词”

只有两者结合，Kubernetes 才真的能理解并执行你的领域模型。

## 目录总览

### 1. CRD 基础与 API 扩展

这篇会回答：

- Kubernetes API 为什么可以被扩展
- `group`、`version`、`kind`、`plural` 到底什么意思
- CRD 和普通资源在使用体验上有什么相同和不同
- Schema 校验、subresources、打印列的价值是什么

### 2. 控制器、Reconcile 与 Operator 模式

这篇会回答：

- Informer / WorkQueue / Reconcile 是怎么串起来的
- 控制器为什么天然适合做“期望状态 -> 实际状态”的持续修正
- Operator 为什么不是一个魔法词，而是一种模式组合
- 什么场景适合自己写控制器，什么场景不适合

### 3. CRD 设计、版本演进与最佳实践

这篇会回答：

- 好的 `spec` 和 `status` 应该怎么拆
- Finalizer、Conditions、ObservedGeneration、Printer Columns 怎么设计
- v1 到 v2 的版本演进如何做
- Conversion Webhook 为什么重要
- 设计 CRD 时最常见的反模式有哪些

### 4. 真实案例拆解：从领域模型到 CRD 设计

这篇会回答：

- 一个真实的领域对象应该如何抽象成 CRD
- 如何判断哪些字段该暴露，哪些不该暴露
- `spec`、`status`、下游资源之间如何建立映射
- 为什么成熟 CRD 设计一定离不开案例化思考

### 5. Kubebuilder 工作流：从脚手架到控制器骨架

这篇会回答：

- 今天主流 Operator 工程为什么常从 Kubebuilder 开始
- 项目骨架、API 类型、Reconciler、RBAC、生成清单是怎么串起来的
- Kubebuilder 与 `client-go`、`controller-runtime` 的关系是什么
- 如何从“会写控制器”进一步走向“能组织一个可维护的控制器工程”

### 6. 从 0 到 1 设计一个完整 Operator 的实战样例

这篇会回答：

- 如果把前面所有章节真正串起来，一个完整 Operator 应该按什么顺序设计
- 从领域建模到 CRD，再到 Reconcile 和状态更新，完整闭环是什么样
- 为什么第一次练手最好先做一个“轻量但完整”的 Operator
- 一个教学级案例如何一步步演进成更像生产工程的 Operator

### 7. Operator 测试、调试与发布

这篇会回答：

- 一个 Operator 从“能写”到“能上线”必须补哪些工程能力
- 应该如何设计测试分层：单元、集成、端到端
- 调试时为什么要优先看 `status`、日志、下游资源和事件
- 发布、升级、回滚时最容易忽略的风险是什么

## 学习建议

这个专题最好边读边做两件事：

1. 多看几个真实 Operator 的 CRD 例子
2. 尝试把自己熟悉的一个业务对象抽象成自定义资源

比如你可以思考：

- 如果把“数据库实例”定义成一个 CR，它的 `spec` 应该有哪些字段？
- 哪些字段是用户希望声明的？
- 哪些字段应该由控制器写进 `status`？
- 删除这个资源前，是否需要 Finalizer 清理外部资源？

这种练习会让你比单纯看 YAML 更快建立设计能力。

## 下一步

- 从 [CRD 基础与 API 扩展](./01-crd-basics.md) 开始
