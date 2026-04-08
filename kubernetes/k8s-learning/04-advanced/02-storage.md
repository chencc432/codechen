# 💾 Kubernetes 存储系统详解

## 为什么 Kubernetes 中的存储是重点

很多初学者前面学 `Pod`、`Deployment`、`Service` 时会觉得 Kubernetes 已经差不多了，但一旦开始部署数据库、日志系统、消息队列，就会发现真正困难的问题往往出在存储上。

原因很简单：

- 容器本身是易失的
- Pod 可能被删除、重建、迁移
- 节点也可能故障或被替换
- 但业务数据往往必须保留

所以 Kubernetes 的存储体系，本质上是在解决一个核心矛盾：

> 计算实例可以随时变化，但数据必须稳定、可恢复、可挂载、可管理。

## 先建立整体心智模型

在 Kubernetes 里，可以先把存储相关对象拆成 4 层：

1. **容器挂载层**：`volumeMounts`
2. **Pod 卷定义层**：`volumes`
3. **存储声明层**：`PVC`（PersistentVolumeClaim）
4. **存储供给层**：`PV`（PersistentVolume）和 `StorageClass`

可以把它简单理解为：

```text
容器 -> 挂载卷 -> Pod里的volume -> PVC申请存储 -> PV提供存储 -> StorageClass决定如何动态创建
```

## 1. 临时存储与持久存储

### 1.1 临时存储

最常见的是 `emptyDir`。它的特点是：

- Pod 启动时创建
- Pod 删除时一起消失
- 适合缓存、临时文件、中间计算结果

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: emptydir-demo
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: cache
      mountPath: /cache
  volumes:
  - name: cache
    emptyDir: {}
```

适合场景：

- 临时工作目录
- 多容器共享的临时文件
- 不需要跨 Pod 保留的数据

不适合场景：

- 数据库数据目录
- 业务必须保留的数据
- 需要 Pod 重建后仍然存在的数据

### 1.2 持久存储

持久存储的目标是：

- Pod 删除后数据依然保留
- Pod 重建后还能重新挂载原来的数据
- 存储和计算解耦

这就需要 `PV`、`PVC`、`StorageClass` 这一套机制。

## 2. Volume、PV、PVC、StorageClass 的关系

### 2.1 最容易记住的理解方式

| 对象 | 角色 | 你可以把它理解成 |
|------|------|------------------|
| `Volume` | Pod 内的卷定义 | “我要把一块存储挂到 Pod 里” |
| `PV` | 集群中的存储资源 | “集群里实际可用的一块盘” |
| `PVC` | 对存储资源的申请 | “我要申请一块符合条件的盘” |
| `StorageClass` | 存储供应模板 | “这类盘应该如何被动态创建” |

### 2.2 一个完整链路

```text
应用容器
  -> volumeMounts 指定挂载路径
  -> Pod volumes 引用 pvc
  -> PVC 申请存储容量和访问模式
  -> StorageClass 动态创建 PV
  -> PV 绑定给 PVC
```

这条链理解了，后面大多数存储 YAML 就不会太乱。

## 3. PersistentVolume（PV）

`PV` 是集群级别的存储资源，通常由：

- 管理员预先创建
- 或者由 `StorageClass` 动态创建

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-demo
spec:
  capacity:
    storage: 10Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /data/pv-demo
```

### 3.1 PV 中最重要的字段

| 字段 | 作用 |
|------|------|
| `capacity.storage` | 存储容量 |
| `accessModes` | 访问模式 |
| `storageClassName` | 对应的存储类 |
| `persistentVolumeReclaimPolicy` | 回收策略 |
| `hostPath` / `nfs` / `csi` | 底层存储类型 |

### 3.2 Reclaim Policy 是什么

`persistentVolumeReclaimPolicy` 决定 PVC 删除后，这块 PV 怎么处理。

| 策略 | 含义 | 常见场景 |
|------|------|----------|
| `Retain` | 保留数据，不自动删 | 生产数据库、关键数据 |
| `Delete` | 删除后端存储 | 临时环境、动态卷 |
| `Recycle` | 旧机制，已基本不用 | 不建议依赖 |

实践里最重要的判断是：

- 如果是关键数据，优先确认是不是 `Retain`
- 不要在不清楚策略的情况下随便删除 PVC

## 4. PersistentVolumeClaim（PVC）

`PVC` 是工作负载最常接触的存储申请方式。通常业务不会直接关心 PV，而是写 PVC。

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-data
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

### 4.1 PVC 在做什么

你可以把 PVC 理解为：

> “我需要一块 10Gi、满足某种访问模式的持久存储，请系统帮我分配。”

然后 Kubernetes 会：

- 去找已有 PV
- 或通过 `StorageClass` 动态创建合适的 PV
- 最后将它绑定到这个 PVC

### 4.2 Pod 如何使用 PVC

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pvc-demo
spec:
  containers:
  - name: app
    image: nginx
    volumeMounts:
    - name: data
      mountPath: /usr/share/nginx/html
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: app-data
```

这里最关键的是：

- 容器不直接挂 PV
- 容器挂的是 Pod 中的 `volume`
- 这个 `volume` 再去引用 PVC

## 5. StorageClass

`StorageClass` 用来描述“这类存储应该如何动态供应”。

如果没有它，很多时候你需要手动创建 PV；有了它，PVC 提交后系统就能自动创建底层存储。

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
```

### 5.1 为什么 StorageClass 很重要

它让存储从“人工分盘”变成“按需申请”。

最直接的好处是：

- 业务 YAML 更简单
- 集群管理更自动化
- 不同性能等级的存储可以标准化

常见做法：

- `standard`：通用型
- `fast-ssd`：高性能 SSD
- `backup`：低成本慢速存储

## 6. 访问模式（Access Modes）

访问模式是存储排查里最常见的卡点之一。

| 模式 | 含义 | 常见理解 |
|------|------|----------|
| `ReadWriteOnce` | 单节点读写 | 最常见，很多云盘默认就是这个 |
| `ReadOnlyMany` | 多节点只读 | 适合分发静态内容 |
| `ReadWriteMany` | 多节点读写 | 适合共享文件系统，但不是所有存储都支持 |
| `ReadWriteOncePod` | 单 Pod 独占读写 | 更严格的单 Pod 绑定场景 |

最容易误解的一点是：

- `ReadWriteOnce` 不代表“只能一个 Pod 用”
- 它更准确地表示“同一时刻只能被单个节点以读写方式挂载”

所以如果多个 Pod 在同一节点上，有时仍可能使用同一个 `RWO` 卷，但跨节点就通常不行。

## 7. 常见底层存储类型

### 7.1 hostPath

适合本地学习和实验：

- 简单
- 好理解
- 不适合生产

原因是它依赖单机本地路径，一旦 Pod 调度到别的节点，就找不到原来的数据。

### 7.2 NFS

适合共享文件场景：

- 多节点共享
- 部署简单
- 性能和稳定性依赖 NFS 服务本身

### 7.3 云盘 / CSI 存储

生产里更常见：

- 云厂商块存储
- 分布式存储系统
- 通过 CSI 驱动接入

现在更推荐从“CSI 驱动 + StorageClass”这个方向理解 Kubernetes 存储生态。

## 8. StatefulSet 与存储为什么经常一起出现

数据库、队列、中间件这些有状态应用，经常不是单独讲存储，而是和 `StatefulSet` 一起讲。

因为它们通常同时需要：

- 稳定身份
- 稳定网络标识
- 稳定持久卷

典型场景：

- MySQL 主从
- Redis 持久化
- Kafka Broker
- Elasticsearch 节点

这也是为什么：

- 无状态服务更常用 `Deployment`
- 有状态服务更常用 `StatefulSet + PVC`

## 9. 动态卷与静态卷

### 9.1 静态卷

管理员先创建 PV，业务再创建 PVC 去绑定。

优点：

- 可控
- 适合严格管理环境

缺点：

- 运维成本更高
- 不够灵活

### 9.2 动态卷

业务只写 PVC，系统根据 `StorageClass` 自动创建 PV。

优点：

- 更自动化
- 更适合云原生场景

缺点：

- 需要底层驱动和存储类正确配置
- 出问题时排查链路更长

## 10. 排障时应该怎么查

当你发现 Pod 因存储问题起不来，最常见的排查顺序是：

1. `kubectl get pvc`
2. `kubectl describe pvc <pvc-name>`
3. `kubectl get pv`
4. `kubectl describe pv <pv-name>`
5. `kubectl get storageclass`
6. `kubectl describe pod <pod-name>`

### 10.1 常见问题 1：PVC 一直 Pending

常见原因：

- 没有可匹配的 PV
- `StorageClass` 不存在
- 申请容量过大
- 访问模式不匹配
- 动态供应器异常

### 10.2 常见问题 2：Pod 卡在 ContainerCreating

常见原因：

- 卷还没挂载成功
- 节点无法附加云盘
- PVC 未绑定
- 权限或底层驱动异常

### 10.3 常见问题 3：数据“丢了”

优先确认：

- 用的是不是 `emptyDir`
- 是不是误删了 PVC
- PV 的回收策略是不是 `Delete`
- Pod 是否重建到了新卷

## 11. 一个最小持久化示例

下面给一个从 PVC 到 Pod 的最小例子：

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nginx-data
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx-with-storage
spec:
  containers:
  - name: nginx
    image: nginx
    volumeMounts:
    - name: html
      mountPath: /usr/share/nginx/html
  volumes:
  - name: html
    persistentVolumeClaim:
      claimName: nginx-data
```

## 12. 最佳实践

- 学习环境可以使用 `hostPath`，生产环境尽量使用标准化的 `StorageClass`
- 有状态应用优先考虑 `StatefulSet + PVC`
- 删除 PVC 前先确认回收策略和数据重要性
- 不要把 `emptyDir` 当持久化方案使用
- 生产存储变更前，先做备份和恢复演练
- 重点业务要明确“数据恢复目标”和“存储故障处理流程”

## 13. 一页总结

如果你只想先记住最关键的内容，请记下面这些结论：

- `emptyDir` 是临时卷，不是持久存储
- 真正的持久化通常通过 `PVC -> PV` 完成
- `StorageClass` 决定动态卷如何创建
- 访问模式经常是卷无法绑定或无法挂载的关键原因
- 生产中最常见组合是 `StatefulSet + PVC + StorageClass`
- 排查存储问题时，不要只看 Pod，也要一起看 PVC、PV、StorageClass

## 下一步

- [调度机制与策略](./03-scheduling.md)
- [StatefulSet - 有状态应用](../02-resources/07-statefulset.md)
