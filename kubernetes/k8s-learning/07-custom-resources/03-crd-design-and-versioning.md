# 🏗️ CRD 设计、版本演进与最佳实践

## 为什么 CRD 设计比“能用”更重要

很多团队第一次做 CRD 时，往往只追求一件事：

> 先把资源定义出来，能跑就行。

但 CRD 和普通脚本、普通 YAML 不一样。  
一旦它被用户使用，它其实就开始承担：

- API 契约
- 平台抽象
- 长期兼容性
- 使用体验
- 自动化治理入口

所以一个设计不好的 CRD，后续通常会产生这些问题：

- 字段命名混乱
- `spec` 和 `status` 职责不清
- 版本升级困难
- 兼容性极差
- 用户看不懂、平台自己也难维护

因此，CRD 设计本质上不是“写一个 YAML 结构”，而是在设计一个长期可演进的 API。

## 1. 先从资源建模开始

在真正写 CRD 前，建议先回答下面几个问题：

1. 这个对象到底代表什么
2. 用户真正想声明什么
3. 控制器负责自动决定什么
4. 当前状态里哪些信息值得反馈给用户
5. 这个对象是否会长期存在并演进

如果这些问题都回答不清，就不要急着写字段。

## 2. `spec` 和 `status` 的边界怎么划

这是 CRD 设计中最重要的一件事。

### 2.1 `spec`

应该放：

- 用户期望的目标状态
- 用户有权声明的配置
- 希望控制器据此执行的输入

例如：

- 副本数
- 版本
- 存储大小
- 备份是否启用
- 暴露方式

### 2.2 `status`

应该放：

- 控制器观测到的实际状态
- 已经创建了哪些资源
- 当前是否 Ready
- 当前 phase、conditions、错误信息

例如：

- `readyReplicas`
- `phase`
- `conditions`
- `observedGeneration`
- `lastBackupTime`

### 2.3 一个典型例子

```yaml
apiVersion: mysql.example.com/v1
kind: MySQLCluster
metadata:
  name: prod-db
spec:
  replicas: 3
  version: "8.0"
  storage:
    size: 100Gi
status:
  observedGeneration: 4
  readyReplicas: 2
  phase: Provisioning
  conditions:
  - type: Ready
    status: "False"
    reason: ReplicaNotReady
```

这类分层会让用户一眼看懂：

- 我想要什么
- 现在做到了什么

## 3. 字段设计的基本原则

### 3.1 命名要稳定、直观、符合 Kubernetes 风格

好的字段命名应该：

- 读起来像声明，而不是像脚本参数
- 尽量短而清晰
- 与 Kubernetes 生态已有命名风格保持一致

例如更推荐：

- `replicas`
- `storage`
- `resources`
- `serviceAccountName`
- `imagePullSecrets`

而不太推荐：

- `pod_num`
- `enable_the_backup_feature`
- `mysql_cluster_storage_in_gigabytes`

### 3.2 结构要可扩展

不要只为今天的需求设计字段，还要给未来演进留空间。

例如：

```yaml
spec:
  backup:
    enabled: true
    schedule: "0 2 * * *"
```

通常比下面这种更好：

```yaml
spec:
  backupEnabled: true
  backupSchedule: "0 2 * * *"
```

前者更容易未来扩展更多子字段。

### 3.3 避免把控制器内部状态放进 `spec`

例如这些字段通常不应该放在 `spec`：

- 当前 leader 是谁
- 当前有几个 ready 副本
- 当前 phase
- 最近一次错误信息

这些应该属于 `status`。

## 4. Conditions 为什么几乎是必备设计

如果你的自定义资源有一定复杂度，`conditions` 几乎是必备。

它的价值在于：

- 提供结构化状态
- 方便 UI、CLI、平台统一解析
- 避免所有状态都塞进一个字符串 `phase`

一个常见例子：

```yaml
status:
  conditions:
  - type: Ready
    status: "True"
    reason: AllReplicasReady
    message: "all replicas are healthy"
  - type: Progressing
    status: "False"
    reason: Stable
```

### `phase` 和 `conditions` 的关系

可以这样理解：

- `phase`：更像一个粗粒度状态概览
- `conditions`：更像结构化、可组合、可诊断的状态切面

如果必须二选一，复杂资源通常更推荐优先设计好 `conditions`。

## 5. `observedGeneration` 为什么重要

当用户修改了 `spec`，你需要有一种方式告诉用户：

> “控制器到底有没有处理到这次最新变更？”

这就是 `observedGeneration` 的意义。

通常：

- `metadata.generation` 代表对象 spec 的版本递增
- `status.observedGeneration` 代表控制器已经处理到的版本

这样用户就能判断：

- 是 spec 已更新但控制器还没处理
- 还是控制器已经处理完，只是当前状态尚未收敛

## 6. Finalizer 设计建议

如果你的 CR 删除时需要清理外部资源，通常要设计 Finalizer。

### 常见需要 Finalizer 的场景

- 外部数据库账号
- 云盘
- 负载均衡器
- DNS 记录
- 对象存储桶
- 第三方系统中的租户或项目

### 设计时的重点

- Finalizer 名称应稳定、唯一
- 清理逻辑必须幂等
- 清理失败要有可观察状态
- 不要让资源永远卡在 `Terminating`

## 7. 打印列和状态可读性

一个 CRD 设计得成熟不成熟，`kubectl get` 的体验通常就能看出来。

建议为关键状态设计打印列：

```yaml
additionalPrinterColumns:
- name: Ready
  type: string
  jsonPath: .status.conditions[?(@.type=="Ready")].status
- name: Version
  type: string
  jsonPath: .spec.version
- name: Age
  type: date
  jsonPath: .metadata.creationTimestamp
```

好的打印列可以显著降低排查和使用成本。

## 8. 版本演进为什么必须提前考虑

只要你的 CRD 面向长期使用，就几乎必然会遇到：

- 字段新增
- 字段废弃
- 结构重组
- 语义变化

这时如果一开始完全没考虑版本演进，后面就会非常痛苦。

### 8.1 常见版本状态

| 版本 | 含义 |
|------|------|
| `v1alpha1` | 早期实验版本，允许较大变化 |
| `v1beta1` | 相对稳定，语义开始收敛 |
| `v1` | 稳定版本，兼容性要求高 |

### 8.2 一个多版本示例

```yaml
versions:
- name: v1alpha1
  served: true
  storage: false
- name: v1beta1
  served: true
  storage: false
- name: v1
  served: true
  storage: true
```

这表示：

- 用户仍可访问早期版本
- 但 etcd 中最终以 `v1` 存储

## 9. 什么时候需要 Conversion Webhook

如果多个版本之间的字段结构不完全一致，Kubernetes 需要知道如何在不同版本之间转换。  
简单场景下可以依赖结构兼容；复杂场景下则需要 **Conversion Webhook**。

典型需要它的场景：

- 字段重命名
- 嵌套结构重组
- 枚举值变化
- 一个字段拆成多个字段
- 多版本之间语义不完全同构

如果你预期 CRD 会长期演进，Conversion 设计最好尽早考虑。

## 10. 兼容性应该怎么做

设计 CRD 时，最容易踩坑的是“今天方便，明天破坏兼容”。

### 更安全的做法

- 新增字段优于修改旧字段语义
- 废弃字段时保留过渡期
- 用默认值兼容历史行为
- 在 status / conditions 中明确反馈转换或兼容状态

### 风险较高的做法

- 直接删除用户正在使用的字段
- 复用旧字段做完全不同的语义
- 没有版本策略就直接升级控制器

## 11. 什么时候应该拆成多个资源

一个常见反模式是把所有东西都塞进一个超大的 CR。

例如：

- 部署策略
- 备份策略
- 网络规则
- 权限配置
- 监控规则
- 生命周期策略

全部塞进一个对象，会带来：

- Schema 过大
- 职责不清
- 控制器过重
- 权限边界模糊

更合理的做法是判断：

- 这些配置是否总是一起变化
- 是否属于同一生命周期
- 是否应由同一控制器负责

如果不是，就考虑拆资源。

## 12. 常见反模式

### 反模式 1：把底层实现细节全部暴露给用户

如果用户写一个 `MySQLCluster` 却需要同时配置：

- StatefulSet 名称
- Pod labels
- Service selector
- PVC 模板细节

那这个抽象通常就不够好。

CRD 应该暴露“用户关心的领域语义”，而不是把底层实现原样搬出来。

### 反模式 2：没有明确状态模型

只让用户靠日志猜系统状态，会极大降低可维护性。

### 反模式 3：字段语义模糊

例如一个字段叫 `mode`，却没人知道它具体控制什么。  
模糊字段会迅速让 API 失控。

### 反模式 4：删除逻辑没设计

如果对象背后会创建外部资源，却没 Finalizer，后面基本一定出问题。

## 13. 一个相对合理的 CRD 结构示例

```yaml
apiVersion: cache.example.com/v1
kind: RedisCluster
metadata:
  name: redis-prod
spec:
  replicas: 3
  version: "7.2"
  storage:
    size: 20Gi
    storageClassName: fast-ssd
  backup:
    enabled: true
    schedule: "0 3 * * *"
status:
  observedGeneration: 7
  readyReplicas: 3
  phase: Ready
  conditions:
  - type: Ready
    status: "True"
    reason: AllReplicasReady
  endpoint: redis-prod.default.svc
```

这个结构的优点是：

- `spec` 可读性强
- `status` 职责清晰
- 有扩展空间
- 用户和控制器职责分离

## 14. 最佳实践清单

- 把 CRD 当长期 API 契约设计，不要当一次性配置文件
- 先定义领域模型，再定义字段结构
- 明确划分 `spec` 和 `status`
- 复杂资源优先设计 `conditions`
- 关键资源优先考虑 `observedGeneration`
- 涉及外部资源时优先设计 Finalizer
- 提前规划版本演进和兼容策略
- 打印列、状态字段、错误信息都要面向使用者友好

## 15. 一页总结

- CRD 设计本质上是在设计 API，而不只是写 YAML
- `spec` 放用户意图，`status` 放控制器观测结果
- `conditions`、`observedGeneration`、Finalizer 是成熟 CRD 的重要组成
- 版本演进必须提前考虑，否则后面兼容性成本很高
- 好的抽象应该暴露领域语义，而不是泄漏所有底层细节

## 下一步

- 返回 [自定义资源专题总览](./README.md)
