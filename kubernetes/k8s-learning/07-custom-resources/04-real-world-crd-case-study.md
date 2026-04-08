# 🧪 真实案例拆解：从领域模型到 CRD 设计

## 为什么要做案例拆解

前面几篇已经把 CRD、控制器、Operator、版本演进这些概念讲清楚了，但很多人到真正动手设计时还是会卡住：

> “理论我懂了，可我还是不知道一个真实 CRD 到底应该长什么样。”

这很正常。  
因为 CRD 设计最难的部分，从来不是记字段，而是：

- 抽象业务概念
- 划定职责边界
- 决定哪些能力该暴露、哪些不该暴露
- 决定用户声明什么、控制器负责什么

所以这一篇不再主要讲概念，而是用一个完整案例，把“从领域对象到 CRD 结构”的思考过程展开。

## 1. 先选一个合适的案例

我们选一个典型但又足够复杂的对象：

> **MySQLCluster**

为什么选它？

因为它天然具备 CRD 设计里最典型的挑战：

- 有明确领域语义
- 有部署、扩容、升级、备份等长期生命周期
- 有状态，不只是简单创建 Pod
- 有用户输入，也有控制器观测结果

它非常适合用来学习“成熟资源模型该怎么设计”。

## 2. 先别写 YAML，先画领域边界

如果你一上来就开始写：

```yaml
spec:
  replicas: 3
  image: xxx
```

那大概率会陷入“想到什么加什么字段”的局面。  
更好的方法是先回答：

### 2.1 用户真正想表达什么

一个使用者通常关心的是：

- 我要几个实例
- 我要哪个数据库版本
- 存储要多大
- 要不要备份
- 要不要高可用
- 暴露方式是什么

### 2.2 控制器应该自动决定什么

控制器更适合负责：

- 具体 StatefulSet / Service 创建细节
- Pod labels、selector 等底层编排结构
- 某些默认值补全
- 状态检测和修复
- 主从拓扑维护

### 2.3 用户通常不应该直接管什么

如果你的 CRD 让用户去关心：

- 每个 Pod 名字
- StatefulSet 名称
- Headless Service 细节
- 每个探针参数的完整实现

那么这个抽象就很可能泄漏了太多底层实现。

## 3. 一个相对合理的资源目标模型

先不看实现，只站在 API 使用者角度，一个 `MySQLCluster` 可能长这样：

```yaml
apiVersion: database.example.com/v1alpha1
kind: MySQLCluster
metadata:
  name: orders-db
spec:
  replicas: 3
  version: "8.0.36"
  storage:
    size: 100Gi
    storageClassName: fast-ssd
  backup:
    enabled: true
    schedule: "0 2 * * *"
  service:
    type: ClusterIP
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
    limits:
      cpu: "2"
      memory: "4Gi"
```

这个对象的可读性就很强，因为它表达的是：

- 用户想要的数据库集群样子

而不是：

- 底层要创建哪几个 Kubernetes 细节对象

## 4. 为什么 `spec` 里应该这么分层

### 4.1 `replicas`

表示规模，是最直接的业务期望。

### 4.2 `version`

表示数据库引擎版本，是升级和兼容性设计的核心字段之一。

### 4.3 `storage`

把存储放成嵌套对象而不是平铺字段，是因为后面通常还会继续长出：

- `size`
- `storageClassName`
- `accessModes`
- `volumeMode`
- 甚至更多高级配置

### 4.4 `backup`

备份往往不是一个布尔值能讲完的能力，所以做成结构更合理。

### 4.5 `service`

暴露策略是业务关心的，但用户未必要关心底层要创建几个 Service。

## 5. 接下来决定 `status` 放什么

一个成熟的自定义资源，`status` 不只是“顺手写点状态”，而是要承担：

- 提供用户反馈
- 帮助排障
- 支持平台 UI
- 支持自动化判断

一个更合理的 `status` 可能像这样：

```yaml
status:
  observedGeneration: 6
  phase: Ready
  readyReplicas: 3
  endpoint: orders-db.default.svc.cluster.local
  conditions:
  - type: Ready
    status: "True"
    reason: AllReplicasReady
  - type: BackupConfigured
    status: "True"
    reason: ScheduleAccepted
```

这里几个字段各自有明确职责：

- `observedGeneration`：控制器处理到哪一版 spec 了
- `phase`：粗粒度摘要
- `readyReplicas`：当前可用副本数
- `endpoint`：对用户非常实用的接入信息
- `conditions`：结构化状态

## 6. 什么时候该拆成多个资源

一个常见问题是：

> 备份策略、参数模板、升级计划、账号管理，是不是都该塞进 `MySQLCluster` 里？

不一定。

要看这些能力是否：

- 和主集群对象强耦合
- 生命周期一致
- 由同一控制器负责

### 更适合拆资源的场景

例如：

- 一个备份计划可能适用于多个数据库
- 一个参数模板可能被多个集群复用

那就更可能设计成：

- `MySQLCluster`
- `MySQLBackupPolicy`
- `MySQLParameterTemplate`

而不是把所有东西都塞进一个巨型资源里。

## 7. 设计一版更成熟的 API

我们把上面的思路再推进一步，让这个 CRD 更像真实生产对象：

```yaml
apiVersion: database.example.com/v1beta1
kind: MySQLCluster
metadata:
  name: orders-db
spec:
  topology:
    mode: primary-replica
    replicas: 3
  engine:
    version: "8.0.36"
  storage:
    size: 100Gi
    storageClassName: fast-ssd
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retentionDays: 7
  service:
    type: ClusterIP
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
    limits:
      cpu: "2"
      memory: "4Gi"
status:
  observedGeneration: 12
  phase: Ready
  readyReplicas: 3
  primaryEndpoint: orders-db.default.svc.cluster.local
  conditions:
  - type: Ready
    status: "True"
    reason: ClusterStable
```

这样做的好处是：

- 结构清晰
- 可扩展
- 字段职责稳定
- 后续演进空间更大

## 8. 从 CRD 到下游资源映射

一个很实用的问题是：

> 这个 CRD 最终会映射出哪些 Kubernetes 原生资源？

以 `MySQLCluster` 为例，控制器通常可能会管理：

- `StatefulSet`
- `Service`
- `Secret`
- `ConfigMap`
- `PersistentVolumeClaim`
- `PodDisruptionBudget`

你可以把控制器理解成：

> “把一个高层领域对象翻译成一组底层 Kubernetes 对象，并持续维护它们。”

## 9. 这个案例里最值得注意的设计取舍

### 9.1 不直接暴露太多底层字段

如果把 StatefulSet 的所有字段都暴露给用户，CRD 可能会变成一个“换皮 StatefulSet”，失去领域抽象价值。

### 9.2 不要过早做成超级通用模型

很多团队一开始就想把一个资源做成：

- 支持所有数据库
- 支持所有拓扑
- 支持所有备份方式
- 支持所有网络模型

结果就是：

- Schema 爆炸
- 控制器复杂度失控
- 用户也看不懂

更好的做法通常是：

- 从一个清晰、聚焦、稳定的领域对象开始
- 再逐步演进

### 9.3 用户最在意的是“好用”和“可观测”

很多设计者更关注“控制器内部写得爽不爽”，但从用户角度，最关键的是：

- 字段好不好理解
- 错误能不能看懂
- `kubectl get` 输出是否友好
- 出问题时能不能快速定位

## 10. 再看一个更轻量的案例

不是所有 CRD 都要像数据库那样重。  
例如一个 `BackupPolicy` 资源，可能就适合更轻量的设计：

```yaml
apiVersion: platform.example.com/v1
kind: BackupPolicy
metadata:
  name: daily-backup
spec:
  schedule: "0 3 * * *"
  retentionDays: 14
  targetNamespaces:
  - payments
  - orders
status:
  observedGeneration: 3
  lastBackupTime: "2026-03-24T03:00:00Z"
  conditions:
  - type: Ready
    status: "True"
```

这个案例说明：

- CRD 不一定都很重
- 好的资源设计取决于对象本身的职责，而不是取决于“能不能设计得复杂”

## 11. 教科书式设计检查表

当你准备设计一个新的 CRD，可以用这张检查表先过一遍：

### 领域建模检查

- 这个对象是不是稳定的领域概念
- 它是不是值得被长期作为平台 API 暴露
- 它是不是比 ConfigMap/Helm/脚本更适合作为资源建模

### `spec` 检查

- 用户真正关心的目标状态是否表达清楚
- 有没有把内部实现细节泄漏给用户
- 结构是否支持未来扩展

### `status` 检查

- 是否能表达当前整体状态
- 是否有结构化 conditions
- 是否能帮助用户排障

### 生命周期检查

- 删除时是否需要 Finalizer
- 升级时是否需要多版本兼容
- 是否需要 `observedGeneration`

## 12. 一页总结

- 好的 CRD 设计从领域抽象开始，而不是从 YAML 字段开始
- `spec` 应该表达用户意图，`status` 应该反馈控制器观测结果
- 资源设计要防止实现细节泄漏过多
- 不是所有相关能力都该塞进一个对象，必要时应拆资源
- 真正成熟的 CRD 设计必须兼顾可读性、可演进性、可观测性

## 下一步

- [Kubebuilder 工作流：从脚手架到控制器骨架](./05-kubebuilder-workflow.md)
