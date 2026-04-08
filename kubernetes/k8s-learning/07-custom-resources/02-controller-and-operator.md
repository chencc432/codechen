# 🔁 控制器、Reconcile 与 Operator 模式

## 为什么说“只有 CRD 还不够”

很多人在第一次接触 CRD 时，会误以为：

> “我定义了一个 `MySQLCluster` 资源，Kubernetes 就会自动帮我创建数据库集群。”

其实这并不会自动发生。

CRD 做的事情只是：

- 注册一个新 API 类型
- 允许用户创建这个类型的资源对象
- 让这些对象可以被查询、Watch、校验和存储

但它不会自动：

- 创建 Deployment
- 创建 StatefulSet
- 创建 Service
- 创建 PVC
- 检查状态并修复故障

这些“行为”来自 **控制器（Controller）**。

## 1. 控制器的本质

控制器是 Kubernetes 的核心思想之一。它一直在做一件事：

> 对比期望状态和当前状态，如果不一致，就采取行动让系统逐步收敛。

这也是为什么 Kubernetes 是声明式系统：

- 用户写下“我想要什么”
- 控制器负责不断尝试把现实修正成那个样子

内置资源如此，自定义资源也如此。

## 2. 一个最小的心智模型

对于自定义资源，可以把控制器理解成：

```text
用户创建 MySQLCluster
  -> API Server 存储 CR
  -> 控制器监听到变化
  -> 控制器读取 spec
  -> 控制器创建/更新底层资源
  -> 控制器把结果写回 status
```

如果只记一句话：

> 控制器就是“把自定义资源的声明，翻译成一组真实 Kubernetes 资源和持续运维动作”的那层逻辑。

## 3. Reconcile 到底是什么

`Reconcile` 是控制器世界里最重要的词之一。

它可以翻译成：

- 调谐
- 收敛
- 对账

从工程角度说，它通常是在做：

1. 读取资源当前对象
2. 判断资源是否被删除
3. 检查 spec 中声明的目标
4. 查询底层实际资源状态
5. 计算差异
6. 执行创建、更新、删除、修复动作
7. 更新 status / conditions

这个过程会持续重复。

## 4. Reconcile 为什么不是“一次性执行脚本”

因为真实系统总在变化：

- Pod 可能异常退出
- 节点可能故障
- 用户可能修改 spec
- 外部资源可能创建失败
- 网络和云平台可能短暂抖动

如果你只在“创建那一刻”跑一次脚本，后面状态漂移了就没人处理。  
而控制器的价值就是持续观察和持续修正。

## 5. 控制器常见的内部结构

虽然不同框架写法不同，但核心思路通常类似：

```text
Informer 监听资源变化
  -> EventHandler 收到事件
  -> WorkQueue 入队 key
  -> Worker 从队列消费
  -> Reconcile / SyncHandler 执行业务逻辑
```

### 5.1 Informer

负责：

- Watch API 变化
- 本地缓存资源
- 减少频繁直接打 API Server

### 5.2 WorkQueue

负责：

- 排队处理对象
- 失败重试
- 限速重试

### 5.3 Reconcile / SyncHandler

负责：

- 真正的业务控制逻辑
- 状态比对
- 创建、更新、删除资源

## 6. 一个简化的控制循环

下面是一个容易理解的伪代码流程：

```text
for each key in queue:
  load custom resource
  if resource not found:
    return

  if deletionTimestamp exists:
    handle finalizer cleanup
    return

  ensure child resources exist
  ensure child resources spec is correct
  collect current runtime status
  update custom resource status
```

这个流程里最关键的思想是：

- 不要假设资源一定存在
- 不要假设上一次操作一定成功
- 每次都应该把对象重新收敛到期望状态

## 7. 自定义资源控制器通常管什么

### 7.1 管理下游 Kubernetes 资源

最常见的事情是：

- 创建 Deployment / StatefulSet
- 创建 Service
- 创建 ConfigMap / Secret
- 创建 PVC

### 7.2 更新状态

控制器往往会更新：

- `phase`
- `readyReplicas`
- `conditions`
- 关联资源引用
- 最后一次成功同步时间

### 7.3 处理删除

如果资源删除前需要：

- 删云盘
- 删 DNS
- 删数据库账号
- 回收外部系统资源

那么控制器通常要结合 Finalizer 完成。

## 8. Finalizer 在控制器里为什么重要

当 CR 被删除时，如果你什么都不做，Kubernetes 可能会直接删对象记录。  
但现实中很多自定义资源背后可能还挂着外部资源。

这时就需要 Finalizer：

1. CR 创建时加上 finalizer
2. 用户删除 CR 时，对象进入删除中状态
3. 控制器看到删除时间戳，执行清理逻辑
4. 清理完成后移除 finalizer
5. 对象才真正被删除

这类场景在下面几种资源特别常见：

- 云数据库
- 负载均衡器
- 域名记录
- 对象存储桶
- 外部账号和权限对象

## 9. OwnerReferences 与 Finalizer 的区别

这两个概念经常被混淆。

### OwnerReferences

更像是：

> “这些下游资源是谁创建和拥有的。”

用于：

- 建立父子关系
- 支持级联删除

### Finalizer

更像是：

> “删除之前还有收尾动作没做完。”

用于：

- 阻止对象在清理完成前被立即删除

所以：

- `OwnerReferences` 解决归属问题
- `Finalizer` 解决删除前收尾问题

## 10. Operator 到底是什么

Operator 不是 Kubernetes 里一个特殊 API 类型，而是一种模式。

最常见的理解方式是：

> Operator = CRD + Controller + 面向某个领域对象的自动化运维知识

它和普通控制器相比，差别不在“有没有 Reconcile”，而在于它更强调：

- 领域对象建模
- 持续运维能力
- 专家经验自动化

### 10.1 Operator 常见负责的能力

- 初始化部署
- 滚动升级
- 主从切换
- 扩缩容
- 备份恢复
- 健康检查
- 参数校验
- 版本兼容处理

例如数据库 Operator 通常不是只会“创建 Pod”，而是会负责：

- 创建主从结构
- 维护拓扑
- 处理 failover
- 更新状态

## 11. 什么时候值得写 Operator

适合写 Operator 的场景通常有下面这些特征：

- 对象不是一次性创建，而是要长期维护
- 运维逻辑复杂，人工成本高
- 领域知识稳定，可被抽象
- 需要把专家经验沉淀为平台能力

### 典型场景

- 数据库集群
- 消息队列集群
- 证书生命周期管理
- 备份与恢复系统
- 多租户平台资源治理

### 不太适合的场景

- 只是简单资源打包发布
- 没有持续控制逻辑
- 只是想少写几个 YAML
- 对象生命周期很短、很杂、经常变

## 12. 控制器实现时最常见的工程问题

### 12.1 幂等性

Reconcile 必须尽量幂等。  
因为它可能：

- 被重复触发
- 在失败后重试
- 在并发和抖动条件下反复进入

所以你不能把它当“一次性任务脚本”写。

### 12.2 状态来源要清晰

控制器更新状态时，必须清楚：

- 哪些字段来自用户 spec
- 哪些字段来自底层真实状态
- 哪些字段只是内部计算结果

否则状态会越来越混乱。

### 12.3 错误与重试

有些错误适合重试：

- 网络波动
- API 冲突
- 依赖资源暂时未就绪

有些错误则应该尽快写入状态而不是盲目重试：

- spec 非法
- 配置缺失
- 不可恢复的约束不满足

## 13. Controller Runtime 和 Kubebuilder 为什么流行

早期直接用 `client-go` 写控制器会看到很多模板代码：

- Informer 组装
- 队列管理
- 事件处理
- 类型注册

后来社区逐渐形成了更高层抽象：

- `controller-runtime`
- `Kubebuilder`
- `Operator SDK`

它们的价值在于：

- 降低样板代码
- 推动社区写法统一
- 更容易进入生产化工程模式

所以今天如果你是：

- 为了学习底层原理：读 `client-go` 很有价值
- 为了实际做 Operator：通常更推荐 `controller-runtime` / `Kubebuilder`

## 14. 常见误区

### 误区 1：Operator 就是“比控制器高级一点的控制器”

不准确。  
更准确地说，Operator 是控制器模式在“复杂领域运维自动化”上的强化使用。

### 误区 2：控制器只在创建资源时工作

不对。  
控制器的价值就在于“持续调谐”，而不是创建一次就结束。

### 误区 3：状态写不写都行

短期可以偷懒，长期会严重影响可用性和排障效率。  
一个没有好好维护 `status` 的 Operator，用户体验通常会很差。

## 15. 一页总结

- CRD 定义资源类型，控制器实现资源行为
- Reconcile 的核心是“期望状态与实际状态的持续收敛”
- 控制器通常由 Informer、Queue、Reconcile 组成
- Finalizer 解决删除前收尾，OwnerReferences 解决资源归属
- Operator 本质上是“领域模型 + 控制器 + 运维知识自动化”
- 复杂领域对象适合 Operator，简单打包发布未必需要

## 下一步

- [CRD 设计、版本演进与最佳实践](./03-crd-design-and-versioning.md)
