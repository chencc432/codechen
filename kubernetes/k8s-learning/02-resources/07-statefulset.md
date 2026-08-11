# 🗄️ StatefulSet - 有状态应用

## 为什么需要 StatefulSet？

Deployment 把 Pod 当成"长得一样的牛"——谁是谁不重要，死了换一头就行。但现实世界里有大量应用不是这样的：

- **数据库**：MySQL 主库和从库身份不同，不能互换
- **消息队列**：Kafka 每个 broker 有自己的数据分区
- **分布式存储**：Elasticsearch 每个节点有自己的分片数据
- **缓存**：Redis 集群每个实例都有自己的数据分片

这些应用的共同特点是：**每个实例有独立的身份、独立的存储、独立的网络标识**，它们不能像 Deployment 那样被随意替换。

StatefulSet 就是为这类"有状态应用"设计的控制器。

## 核心设计思想

StatefulSet 和 Deployment 最根本的区别，可以用一句话概括：

> **Deployment 管理的是"一群 Pod"，StatefulSet 管理的是"一组有编号的 Pod"。**

这个"编号"是所有特殊行为的起点。

### 1. 有序编号，永不重复

```
┌──────────────────────────────────────────────────────────────────┐
│  StatefulSet 创建的 Pod 命名规则：<sts-name>-<序号>               │
│                                                                    │
│  sts name: mysql                                                  │
│  replicas: 3                                                      │
│                                                                    │
│  mysql-0  ─── 身份固定：即使被删除重建，还是叫 mysql-0           │
│  mysql-1  ─── 身份固定                                            │
│  mysql-2  ─── 身份固定                                            │
│                                                                    │
│  Deployment 是随机后缀：nginx-7d9f8c6b9-abcde                     │
│               重建后变成：nginx-7d9f8c6b9-xyzab（完全新身份）    │
└──────────────────────────────────────────────────────────────────┘
```

这个编号有两个重要含义：

- **序号就是身份**：Pod 的序号不随重建而改变
- **序号决定顺序**：创建时从 0 开始递增，删除时从最大开始递减

### 2. 稳定的网络标识

每个 Pod 都有一个稳定的 DNS 名称，格式如下：

```
<pod-name>.<service-name>.<namespace>.svc.cluster.local

实际例子：
mysql-0.mysql.default.svc.cluster.local
mysql-1.mysql.default.svc.cluster.local
mysql-2.mysql.default.svc.cluster.local
```

这个 DNS 名称在 Pod 重建后仍然有效，因为**新 Pod 会继承旧 Pod 的名称和 IP 绑定**（取决于网络插件）。

**为什么需要稳定的网络标识？**

以 MySQL 主从为例：

```
应用层                             数据库层
                    ┌──────────────────────────┐
                    │  mysql-0.mysql           │  ← 写请求到这里（主库）
                    │  (始终是主库)            │
                    │                          │
                    │  mysql-1.mysql           │  ← 读请求到这里（从库1）
                    │  (始终是从库1)           │
                    │                          │
                    │  mysql-2.mysql           │  ← 读请求到这里（从库2）
                    │  (始终是从库2)           │
                    └──────────────────────────┘
```

如果 Pod 重启后 IP 变了，应用层仍然可以通过固定的 DNS 名称找到它。

### 3. 独立的持久化存储

每个 Pod 都有自己的 PVC，不会共享。

```
StatefulSet
  ├── mysql-0 ─── PVC-mysql-0 ─── PV-A
  ├── mysql-1 ─── PVC-mysql-1 ─── PV-B
  └── mysql-2 ─── PVC-mysql-2 ─── PV-C
```

这个设计是通过 `volumeClaimTemplates` 实现的——它像 Pod template 一样，是 StatefulSet 独有的字段：

```yaml
spec:
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

每次创建新 Pod 时，StatefulSet 会自动为它创建一个 PVC，命名规则是：`<volume-name>-<sts-name>-<序号>`，例如 `data-mysql-0`、`data-mysql-1`。

**重要含义**：删除 StatefulSet **不会删除 PVC**，这是有意的安全设计。因为数据比 Pod 重要。如果你要彻底清理，需要手动删除 PVC。

```
kubectl delete sts mysql     # Pod 被删除，但 PVC 保留
kubectl delete pvc data-mysql-0 data-mysql-1 data-mysql-2  # 手动清理数据
```

## 执行顺序和策略

### 创建和删除顺序

```
OrderedReady（默认策略）：

创建时：
  mysql-0 创建 → 等待 Ready → mysql-1 创建 → 等待 Ready → mysql-2 创建 → 等待 Ready

删除时：
  mysql-2 删除 → 确认已删除 → mysql-1 删除 → 确认已删除 → mysql-0 删除 → 确认已删除
```

**为什么是这个顺序？**

想象一个 3 节点的 MySQL 集群：

- 创建时：先启动主库（mysql-0），从库加入时才知道主库是谁
- 删除时：先删从库（mysql-2, mysql-1），最后删主库（mysql-0），避免数据不一致

### Parallel（并行策略）

```yaml
spec:
  podManagementPolicy: Parallel
```

所有 Pod 同时创建或删除，不等待任何一个 Ready。适合以下场景：

- 节点之间没有启动依赖关系
- 你知道自己在做什么，需要快速扩缩容
- 配合自定义的初始化逻辑，不需要 Kubernetes 帮你控制顺序

### 扩缩容的特殊行为

**扩容**：新增 Pod 的序号从当前最大序号 +1 开始

```bash
# 当前有 mysql-0, mysql-1
kubectl scale sts mysql --replicas=5
# 新增：mysql-2, mysql-3, mysql-4（按顺序创建）
```

**缩容**：删除最大序号的 Pod，不会删除 PVC

```bash
# 当前有 mysql-0, mysql-1, mysql-2, mysql-3, mysql-4
kubectl scale sts mysql --replicas=2
# 删除顺序：mysql-4 → mysql-3 → mysql-2
# 保留：mysql-0, mysql-1
# PVC 全部保留（data-mysql-0 ~ data-mysql-4 都在）
```

**缩容后再扩容**：新 Pod 会拿到旧的 PVC

```bash
# 缩容到 2 后，PVC 保留
# 再扩容到 5：
# mysql-2 会重新绑定到旧的 PVC data-mysql-2
# mysql-3 会重新绑定到旧的 PVC data-mysql-3
# mysql-4 会重新绑定到旧的 PVC data-mysql-4
```

这是 StatefulSet 的灾难恢复机制——即使 Pod 全挂了，数据还在 PVC 里，重新创建 StatefulSet 后 Pod 会自动挂载回原来的数据。

## 更新策略

### RollingUpdate（默认）

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      partition: 0
```

**默认行为（partition=0）**：从最大序号开始，逆序逐个更新 Pod。

```
更新镜像 8.0 → 8.1：

初始状态：mysql-0(8.0), mysql-1(8.0), mysql-2(8.0)

步骤 1：  mysql-2 更新为 8.1 → 等待 Ready
步骤 2：  mysql-1 更新为 8.1 → 等待 Ready
步骤 3：  mysql-0 更新为 8.1 → 等待 Ready

完成：    mysql-0(8.1), mysql-1(8.1), mysql-2(8.1)
```

同样是逆序，这个设计和删除顺序一致——从最不重要的节点开始更新。

### 分区更新（金丝雀发布）

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      partition: 2  # 只更新序号 >= 2 的 Pod
```

```bash
# 阶段 1：只更新 mysql-2（金丝雀验证）
kubectl patch sts mysql -p '{"spec":{"updateStrategy":{"rollingUpdate":{"partition":2}}}}'
# 效果：mysql-2 更新为新版本，mysql-0, mysql-1 保持旧版本

# ... 验证 mysql-2 运行正常 ...

# 阶段 2：更新 mysql-1, mysql-2
kubectl patch sts mysql -p '{"spec":{"updateStrategy":{"rollingUpdate":{"partition":1}}}}'
# 效果：mysql-1, mysql-2 更新为新版本，mysql-0 保持旧版本

# 阶段 3：全部更新
kubectl patch sts mysql -p '{"spec":{"updateStrategy":{"rollingUpdate":{"partition":0}}}}'
# 效果：全部更新为新版本
```

**分区更新为什么重要？**

对于数据库类应用，你不可能像 Deployment 那样直接灰度 10% 的 Pod —— 因为你要确保主库最后更新，从库先更新验证。分区更新让你可以精确控制更新的范围。

### OnDelete

```yaml
spec:
  updateStrategy:
    type: OnDelete
```

手动删除 Pod 后，StatefulSet 才会用新模板重建。适合需要完全人工控制更新节奏的场景。

## Headless Service 的核心作用

StatefulSet 必须配合 Headless Service（`clusterIP: None`）使用，但很多人不理解为什么。

### Headless Service 做了什么

普通 Service 做的事情：

```
客户端请求 Service → kube-proxy 随机转发到某个 Pod
```

Headless Service 做的事情：

```
DNS 查询 Service → 返回所有 Pod 的 IP 列表（不负载均衡）
DNS 查询 <pod-name>.<service-name> → 返回该 Pod 的特定 IP
```

### 为什么 StatefulSet 需要它

两个原因：

**1. 为每个 Pod 提供稳定的 DNS 名称**

没有 Headless Service，`mysql-0.mysql` 这个 DNS 记录就不存在。应用层就无法通过固定域名找到特定实例。

**2. 让应用自己决定连接哪个 Pod**

数据库集群的客户端通常需要知道集群拓扑：

- 写操作 → 连接 `mysql-0.mysql`（主库）
- 读操作 → 连接 `mysql-read`（普通 Service，负载均衡到所有从库）
- 数据同步 → 从库连接 `mysql-0.mysql` 拉取 binlog

这些都是应用层自己控制的，Headless Service 只负责提供稳定的 DNS 解析。

### 一个 Service 的两种角色

```yaml
# Headless Service（用于 Pod 间发现）
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  clusterIP: None
  selector:
    app: mysql
  ports:
  - port: 3306

# 普通 Service（用于客户端负载均衡访问）
apiVersion: v1
kind: Service
metadata:
  name: mysql-read
spec:
  selector:
    app: mysql
  ports:
  - port: 3306
```

在 MySQL 主从集群中，这两个 Service 各有用途：

- `mysql`（Headless）：用于 Pod 之间的互相发现和主从同步
- `mysql-read`（普通 ClusterIP）：用于应用层做读负载均衡

## 完整示例：3 节点 ZooKeeper 集群

ZooKeeper 是 StatefulSet 的经典用例，比 MySQL 配置更简洁，适合用来理解核心概念。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: zk-headless
  labels:
    app: zk
spec:
  clusterIP: None
  selector:
    app: zk
  ports:
  - port: 2888
    name: server
  - port: 3888
    name: leader-election
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: zk-config
data:
  zoo.cfg: |
    tickTime=2000
    initLimit=10
    syncLimit=5
    dataDir=/data
    clientPort=2181
    server.1=zk-0.zk-headless:2888:3888
    server.2=zk-1.zk-headless:2888:3888
    server.3=zk-2.zk-headless:2888:3888
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: zk
spec:
  serviceName: zk-headless
  replicas: 3
  selector:
    matchLabels:
      app: zk
  template:
    metadata:
      labels:
        app: zk
    spec:
      containers:
      - name: zookeeper
        image: zookeeper:3.8
        ports:
        - containerPort: 2181
          name: client
        - containerPort: 2888
          name: server
        - containerPort: 3888
          name: leader-election
        env:
        - name: ZOO_MY_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: ZOO_SERVERS
          valueFrom:
            configMapKeyRef:
              name: zk-config
              key: zoo.cfg
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

## 常用操作

```bash
# ============ 创建和管理 ============
kubectl apply -f statefulset.yaml
kubectl delete statefulset mysql

# ============ 查看状态 ============
kubectl get statefulset
kubectl get sts                          # 简写
kubectl describe sts mysql

# ============ 查看 Pod（有序命名）============
kubectl get pods -l app=mysql

# ============ 扩缩容 ============
kubectl scale sts mysql --replicas=5

# ============ 更新 ============
kubectl set image sts/mysql mysql=mysql:8.1

# 使用分区更新
kubectl patch sts mysql -p '{"spec":{"updateStrategy":{"rollingUpdate":{"partition":2}}}}'

# ============ 查看 PVC（每个 Pod 独立）============
kubectl get pvc

# ============ 查看 Pod DNS 解析 ============
kubectl run -it --rm debug --image=busybox -- nslookup mysql-0.mysql
```

## 实践练习

### 练习：创建简单的 StatefulSet

```bash
# 1. 创建 Headless Service
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  clusterIP: None
  selector:
    app: nginx
  ports:
  - port: 80
EOF

# 2. 创建 StatefulSet
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  serviceName: nginx
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
EOF

# 3. 观察有序创建
kubectl get pods -w -l app=nginx

# 4. 测试 DNS
kubectl run test --image=busybox -it --rm -- nslookup nginx
kubectl run test --image=busybox -it --rm -- nslookup web-0.nginx

# 5. 扩容观察
kubectl scale sts web --replicas=5

# 6. 缩容观察
kubectl scale sts web --replicas=2

# 7. 清理
kubectl delete sts web
kubectl delete svc nginx
```

### 练习：观察 StatefulSet 的持久化特性

```bash
# 1. 创建一个带 PVC 的 StatefulSet
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: data-test
spec:
  serviceName: "none"
  replicas: 2
  selector:
    matchLabels:
      app: data-test
  template:
    metadata:
      labels:
        app: data-test
    spec:
      containers:
      - name: writer
        image: busybox
        command: ["sh", "-c", "echo 'Hello from \$HOSTNAME' > /data/hello.txt && sleep 3600"]
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 1Gi
EOF

# 2. 验证每个 Pod 写了不同的内容
kubectl exec data-test-0 -- cat /data/hello.txt
kubectl exec data-test-1 -- cat /data/hello.txt

# 3. 删除 StatefulSet（Pod 被删除，但 PVC 保留）
kubectl delete sts data-test

# 4. 查看 PVC 依然存在
kubectl get pvc | grep data-test

# 5. 重新创建 StatefulSet，Pod 会重新挂载到原来的 PVC
kubectl apply -f ...  # 重新 apply

# 6. 验证数据还在
kubectl exec data-test-0 -- cat /data/hello.txt

# 7. 彻底清理
kubectl delete sts data-test
kubectl delete pvc -l app=data-test
```

## 核心要点

1. **StatefulSet 的核心机制是"编号"**：Pod 名称、DNS 名称、PVC 名称都基于这个编号
2. **Headless Service 不是可选项，是必选项**：它为每个 Pod 提供稳定的 DNS 记录
3. **删除 StatefulSet 不会删除 PVC**：这是保护数据的安全设计
4. **分区更新是数据库场景的标配**：先更新从库验证，再更新主库
5. **OrderedReady 策略适合大多数有状态应用**：Parallel 只在没有启动依赖时使用

## 总结

| 对比维度 | Deployment | StatefulSet |
|---------|-----------|-------------|
| Pod 命名 | 随机后缀 | 有序编号 |
| 创建顺序 | 并行 | 顺序（0→1→2→...）|
| 删除顺序 | 并行 | 逆序（...→2→1→0）|
| 网络标识 | 不稳定 | 稳定 DNS 名称 |
| 存储 | 共享或临时 | 每个 Pod 独立 PVC |
| 扩缩容行为 | 普通增减 | 保留 PVC，按序号操作 |
| 更新控制 | 百分比灰度 | 分区更新（按序号范围）|
| 适用场景 | 无状态应用 | 数据库、消息队列、分布式系统 |

## 下一步

- [DaemonSet 与 Job](./08-daemonset-job.md)
