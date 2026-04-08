# ⚙️ StorageClass、CSI 与动态供给

## 为什么这部分是现代 Kubernetes 存储的核心

如果说 `PV` 和 `PVC` 解释的是“卷如何声明和绑定”，那么 `StorageClass` 和 `CSI` 解释的就是：

> 这些卷到底是谁来创建、按什么规则创建、在不同存储系统上如何统一接入。

在早期或简单环境里，存储管理员常常需要手工预创建很多 PV。  
但在现代 Kubernetes 环境中，更常见的方式是：

- 业务只写 PVC
- 系统根据 `StorageClass` 自动创建卷
- 通过 CSI 驱动与底层存储打通

这就是动态供给的核心价值。

## 1. StorageClass 是什么

`StorageClass` 可以理解为：

> 一种存储供应模板，定义“这类卷应该怎么被创建”。

它通常会描述：

- 用哪个供应器（provisioner）
- 参数是什么
- 绑定时机是什么
- 是否允许扩容
- 回收策略如何

一个简化例子：

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard-ssd
provisioner: csi.example.com
allowVolumeExpansion: true
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

## 2. 动态供给到底发生了什么

当业务提交一个 PVC 时，如果它指定了某个 `StorageClass`，系统会大致执行下面的流程：

1. PVC 被创建
2. Kubernetes 发现它引用了某个 `StorageClass`
3. 对应的 provisioner / CSI 驱动收到请求
4. 后端存储系统创建实际卷
5. Kubernetes 生成或绑定对应 PV
6. PVC 进入 `Bound`

这就是为什么很多现代环境里你几乎不需要手写 PV。

## 3. StorageClass 里的关键字段

### 3.1 `provisioner`

指定由谁来创建卷。

例如可能是：

- 云厂商 CSI 驱动
- NFS 外部 provisioner
- Ceph CSI
- 其他第三方存储驱动

### 3.2 `reclaimPolicy`

控制卷回收行为，和 PV 上的语义一致，常见是：

- `Delete`
- `Retain`

### 3.3 `allowVolumeExpansion`

表示是否允许后续扩容。

如果为 `true`，并且底层存储与驱动支持，就可能支持 PVC 在线或离线扩容。

### 3.4 `volumeBindingMode`

这是一个非常值得重点理解的字段。

常见值：

| 值 | 含义 |
|----|------|
| `Immediate` | PVC 创建后立即绑定或创建卷 |
| `WaitForFirstConsumer` | 等到真正有 Pod 消费时再决定卷和节点/拓扑关系 |

## 4. 为什么 `WaitForFirstConsumer` 很重要

如果底层存储有拓扑限制，比如：

- 卷只能创建在某个可用区
- Pod 只能调度到某些节点

那么如果 PVC 一创建就立刻创建卷，可能会导致：

- 卷在 A 区
- Pod 被调度倾向或限制在 B 区
- 最终 Pod 用不了该卷

`WaitForFirstConsumer` 的好处就是：

- 先看真正的 Pod 调度位置
- 再根据这个消费位置创建更合适的卷

这在云环境里尤其重要。

## 5. 默认 StorageClass 是什么

很多集群会有一个默认 `StorageClass`。  
当 PVC 没显式指定 `storageClassName` 时，就可能会使用默认类。

查看方式：

```bash
kubectl get storageclass
```

你通常会看到某个类带 `(default)` 标记。

### 实战提醒

默认类虽然方便，但也有风险：

- 团队成员不清楚实际会创建什么类型的卷
- 不同业务误用同一类存储
- 成本和性能不可控

所以生产环境里，很多团队会更倾向于显式指定 `storageClassName`。

## 6. CSI 到底是什么

CSI 全称是 **Container Storage Interface**。

它可以理解成：

> Kubernetes 和各种存储系统之间的一套标准接口。

有了 CSI 以后：

- Kubernetes 不需要为每个存储系统都内置特定代码
- 存储供应商可以通过统一接口接入
- 存储能力更容易标准化和演进

所以今天谈 Kubernetes 存储，绕不开 CSI。

## 7. CSI 解决了什么问题

在 CSI 出现前，很多存储逻辑更依赖 Kubernetes 内置插件。  
CSI 的价值在于：

- 存储驱动可以独立演进
- 社区和厂商可以更快交付能力
- 卷创建、挂载、扩容、快照等能力更容易标准化

因此现代环境通常会看到：

- `StorageClass`
- `CSIDriver`
- CSI Controller 组件
- CSI Node 组件

## 8. CSI 架构可以怎么简单理解

从使用者视角，不需要一开始就陷入全部组件细节。  
先记住这条链就够了：

```text
PVC
  -> StorageClass
  -> CSI provisioner/controller
  -> 后端存储系统创建卷
  -> 节点侧 CSI 负责挂载
  -> Pod 使用卷
```

### 两类核心职责

#### 控制面职责

通常包括：

- 创建卷
- 删除卷
- 扩容卷
- 创建快照

#### 节点侧职责

通常包括：

- 将卷附加到节点
- 在节点上挂载卷
- 让容器可见

## 9. PVC 如何与 StorageClass 配合

一个典型 PVC 示例：

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: db-data
spec:
  accessModes:
  - ReadWriteOnce
  storageClassName: standard-ssd
  resources:
    requests:
      storage: 50Gi
```

这里最关键的是：

- 用户声明容量和访问模式
- 用户指定所需存储类别
- 系统再决定如何创建实际卷

## 10. 扩容能力为什么不是所有卷都有

很多人以为只要改大 PVC 的 `storage` 就能扩容。  
实际上必须同时满足：

1. StorageClass 允许扩容
2. 底层存储系统支持扩容
3. CSI 驱动支持扩容
4. 文件系统扩展流程可用

所以“能不能扩容”不是 PVC 一侧说了算，而是整条链都得支持。

## 11. 快照能力为什么越来越重要

现代生产环境里，卷快照已经越来越关键，尤其适合：

- 备份前冻结某个时间点
- 升级前保留回滚点
- 数据复制与迁移
- 故障恢复演练

不过要真正支持快照，也通常需要：

- CSI 驱动支持
- Snapshot Controller
- 对应的 VolumeSnapshot 类资源

这意味着：

- 不是所有环境都天然支持
- 不能想当然地认为“PVC 一定能做快照”

## 12. 常见使用场景怎么选

### 12.1 开发测试环境

目标通常是：

- 快速搭起来
- 成本低
- 能跑即可

可能会看到：

- 本地路径
- 简单共享存储
- 默认 StorageClass

### 12.2 生产数据库

目标通常是：

- 数据可靠
- 可恢复
- 可扩容
- 有备份策略

这时更关注：

- 高可靠存储类
- 明确的回收策略
- 可扩容能力
- 快照与备份机制

### 12.3 共享文件场景

目标通常是：

- 多节点同时读写

这时往往要重点确认：

- 是否需要 `RWX`
- 后端共享文件系统是否稳定
- 性能是否满足业务需求

## 13. 常见问题与排查思路

### 问题 1：PVC 一直 `Pending`

优先检查：

- `storageClassName` 是否存在
- 默认 StorageClass 是否存在
- provisioner 是否正常
- 请求的容量和模式是否合理

### 问题 2：PVC 绑定了，但 Pod 起不来

优先检查：

- 节点附加卷是否失败
- CSI Node 插件是否异常
- 拓扑或调度限制是否冲突
- 文件系统挂载是否失败

### 问题 3：卷创建成功，但性能很差

优先检查：

- StorageClass 是否选错性能等级
- 后端卷类型是否满足业务
- 是否用了共享存储但业务是随机高 IO
- 节点和存储所在区域是否合理

## 14. 最佳实践

- 生产环境尽量显式指定 `storageClassName`
- 关键业务优先确认扩容、快照、回收策略是否支持
- 结合业务场景选存储类型，不要只看“能挂上”
- 理解 `WaitForFirstConsumer`，尤其是云环境和多可用区场景
- 不要把默认 StorageClass 当作长期存储策略
- 新存储类上线前，最好先做小规模验证和恢复演练

## 15. 一页总结

- `StorageClass` 定义“这类卷该如何创建”
- 动态供给让业务只写 PVC，不必手工准备 PV
- CSI 是 Kubernetes 对接现代存储生态的标准接口
- `WaitForFirstConsumer` 对有拓扑约束的环境很关键
- 扩容、快照、回收能力取决于整条链是否支持

## 下一步

- [存储运维手册：扩容、迁移、备份、排障](./03-storage-operations-runbook.md)
