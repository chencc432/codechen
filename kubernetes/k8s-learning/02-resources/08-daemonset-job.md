# 🔄 DaemonSet 与 Job/CronJob

## DaemonSet

### 为什么需要 DaemonSet？

有些应用既不是"无状态"（Deployment），也不是"有状态"（StatefulSet），而是"跟节点相关"的。

典型的场景是：

- **日志收集**：每个节点上都要跑一个 Fluentd/Filebeat，把节点上的日志收集到中央存储
- **监控代理**：每个节点上都要跑一个 Prometheus Node Exporter，采集节点指标
- **网络插件**：每个节点上都要跑一个 Calico/Flannel 代理，处理容器网络
- **存储守护进程**：每个节点上都要跑一个 CSI 驱动，挂载存储卷
- **安全代理**：每个节点上都要跑一个安全扫描或入侵检测 agent

这些应用的共同特点是：**Pod 的数量不取决于"副本数"，而取决于"节点数"**。

DaemonSet 就是为这类场景设计的——它确保每个符合条件的节点上恰好运行一个 Pod 副本。

### 核心设计思想

```
┌─────────────────────────────────────────────────────────────────────┐
│                        DaemonSet                                     │
│                                                                       │
│  集群节点：                                                           │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐           │
│  │   Node-1     │    │   Node-2     │    │   Node-3     │           │
│  │  ┌────────┐  │    │  ┌────────┐  │    │  ┌────────┐  │           │
│  │  │ Pod    │  │    │  │ Pod    │  │    │  │ Pod    │  │           │
│  │  │fluentd │  │    │  │fluentd │  │    │  │fluentd │  │           │
│  │  └────────┘  │    │  └────────┘  │    │  └────────┘  │           │
│  └──────────────┘    └──────────────┘    └──────────────┘           │
│                                                                       │
│  DaemonSet 的 Pod 数量 = 符合条件的节点数量                            │
│                                                                       │
│  新节点加入集群 → 自动创建 Pod                                         │
│  节点被移除 → 自动删除 Pod                                             │
│  节点增加标签 → 根据条件匹配/不匹配，动态调整                          │
└─────────────────────────────────────────────────────────────────────┘
```

**Deployment vs DaemonSet 的本质区别**：

| 对比维度 | Deployment | DaemonSet |
|---------|-----------|-----------|
| Pod 数量由什么决定 | `spec.replicas` | 符合条件的节点数 |
| 新增节点 | 不影响（除非 HPA 触发） | 自动在新节点上创建 Pod |
| 删除节点 | Pod 被调度到其他节点 | 节点上的 Pod 被删除，不迁移 |
| 缩容 | 减少 replicas | 给节点加污点或移除标签 |
| 典型用途 | 业务应用 | 系统组件 |

### 调度机制

DaemonSet 的调度方式和 Deployment 不同，理解这一点很重要。

#### DaemonSet 默认的调度器

DaemonSet 使用自己的调度逻辑，而不是 Kubernetes 的默认调度器（kube-scheduler）。它的调度规则很简单：

```
对每个节点：
  如果节点满足 Pod 的调度约束 → 在该节点上创建一个 Pod
  如果节点不满足调度约束 → 不在该节点上创建 Pod
```

**这意味着 DaemonSet 的 Pod 不需要经过 kube-scheduler 的调度队列**，它们由 DaemonSet 控制器直接分配到节点上。

#### 控制 Pod 运行在哪些节点上

DaemonSet 的调度约束包括：

```yaml
spec:
  template:
    spec:
      # 方式 1：nodeSelector — 最简单的节点选择
      nodeSelector:
        kubernetes.io/os: linux
        node-type: worker

      # 方式 2：nodeAffinity — 更灵活的节点选择
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/worker
                operator: Exists

      # 方式 3：tolerations — 允许调度到有污点的节点
      tolerations:
      - operator: Exists          # 容忍所有污点
```

三种方式的组合理解：

- **nodeSelector**：简单粗暴的键值匹配，节点必须包含指定的标签
- **nodeAffinity**：支持更丰富的表达式（In、NotIn、Exists、Gt、Lt）
- **tolerations**：不决定"去哪个节点"，而是决定"能否去有污点的节点"

#### 实际场景示例

**场景 1：只在 worker 节点上运行日志收集**

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: fluentd
  template:
    metadata:
      labels:
        name: fluentd
    spec:
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule      # 避免调度到 master 节点
      nodeSelector:
        kubernetes.io/os: linux
      containers:
      - name: fluentd
        image: fluent/fluentd:v1.14
        volumeMounts:
        - name: varlog
          mountPath: /var/log
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
```

**场景 2：网络插件需要运行在所有节点（包括 master）**

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: calico-node
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  template:
    metadata:
      labels:
        k8s-app: calico-node
    spec:
      hostNetwork: true          # 使用主机网络
      tolerations:
      - operator: Exists         # 容忍所有污点，包括 master 的 NoSchedule
      nodeSelector:
        kubernetes.io/os: linux
      containers:
      - name: calico-node
        image: calico/node:v3.26
```

**注意 `hostNetwork: true`**：网络插件必须使用主机网络，因为它在 Pod 网络创建之前就需要运行。

### 更新策略

#### RollingUpdate（默认）

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
```

和 Deployment 不同，DaemonSet 的滚动更新是按节点逐个更新的，而不是按百分比批量更新。

```
初始化：所有节点上都是 fluentd:v1.14
更新到：fluentd:v1.15

步骤 1：Node-1 上的 Pod 更新为 v1.15 → 等待 Ready
步骤 2：Node-2 上的 Pod 更新为 v1.15 → 等待 Ready
步骤 3：Node-3 上的 Pod 更新为 v1.15 → 等待 Ready
...

maxUnavailable: 1 表示任何时候最多有 1 个节点上的 DaemonSet Pod 不可用
```

**maxUnavailable 的含义**：

- 绝对数值：`maxUnavailable: 1` — 最多 1 个节点上的 Pod 不可用
- 百分比：`maxUnavailable: 25%` — 最多 25% 的节点上的 Pod 不可用（向上取整）

#### OnDelete

```yaml
spec:
  updateStrategy:
    type: OnDelete
```

更新模板后不会自动重建 Pod，需要手动删除每个节点上的 Pod，DaemonSet 控制器才会用新模板重建。

适合需要逐个节点验证更新、或者配合外部自动化工具的场景。

### 拓扑约束

#### 只在一部分节点上运行

```yaml
spec:
  template:
    spec:
      nodeSelector:
        disktype: ssd
        gpu: "true"
```

DaemonSet 只在有 `disktype=ssd` 和 `gpu=true` 标签的节点上创建 Pod。

如果集群中只有 3 个节点有这些标签，DaemonSet 就只创建 3 个 Pod。

#### 使用节点亲和性更精确地控制

```yaml
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-type
                operator: In
                values:
                - gpu-worker
                - high-mem-worker
```

#### 通过污点和容忍度控制

```yaml
spec:
  template:
    spec:
      # 只运行在带有 special=true 污点的节点上
      tolerations:
      - key: special
        operator: Equal
        value: "true"
        effect: NoSchedule
```

这种模式常用于：给某些节点打上特殊污点，DaemonSet 用容忍度"选择"这些节点。

### 和 Deployment 的滚动更新对比

```
Deployment 滚动更新：
  ReplicaSet v1 缩容 + ReplicaSet v2 扩容 同时进行，按百分比控制

DaemonSet 滚动更新：
  逐个节点更新，上一个节点 Ready 后再更新下一个节点
```

这种差异的本质原因是：

- **Deployment 的 Pod 是无状态的**，可以同时启动多个新版本，销毁多个旧版本，流量由 Service 负载均衡
- **DaemonSet 的 Pod 是跟节点绑定的**，多个节点同时更新可能影响集群的稳定性（比如网络插件全部更新可能造成网络中断）

### DaemonSet YAML 完整解析

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd
  namespace: kube-system
  labels:
    app: fluentd
spec:
  selector:
    matchLabels:
      app: fluentd
  
  # 更新策略
  updateStrategy:
    type: RollingUpdate          # RollingUpdate 或 OnDelete
    rollingUpdate:
      maxUnavailable: 1
  
  # Pod 模板
  template:
    metadata:
      labels:
        app: fluentd
    spec:
      # 容忍度
      tolerations:
      - operator: Exists
      
      # 节点选择器
      nodeSelector:
        kubernetes.io/os: linux
      
      containers:
      - name: fluentd
        image: fluent/fluentd:v1.14
        resources:
          limits:
            memory: 200Mi
          requests:
            cpu: 100m
            memory: 200Mi
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: containers
          mountPath: /var/lib/docker/containers
          readOnly: true
      
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: containers
        hostPath:
          path: /var/lib/docker/containers
```

### 常见问题

#### 1. DaemonSet Pod 一直 Pending

```
原因：Pod 无法调度到节点
排查方向：
  - 节点是否有足够的资源（CPU、内存、磁盘）
  - nodeSelector 是否匹配
  - 是否有未容忍的污点
  - Pod 是否设置了 resource requests 导致节点资源不足
```

#### 2. DaemonSet Pod 被驱逐

```
原因：节点的资源压力（磁盘、内存不足）
排查方向：
  - kubectl describe node 查看节点状态
  - 检查节点的压力条件
  - 考虑给 DaemonSet Pod 设置 priorityClassName
```

#### 3. DaemonSet 在新节点上没有自动创建 Pod

```
原因：新节点带有 DaemonSet 无法容忍的污点
排查方向：
  - 查看新节点的 taints
  - 检查 DaemonSet 的 tolerations 是否覆盖
  - 检查 nodeSelector 是否匹配新节点标签
```

### DaemonSet 操作

```bash
# 创建
kubectl apply -f daemonset.yaml

# 查看
kubectl get daemonset -n kube-system
kubectl get ds                           # 简写
kubectl describe ds fluentd -n kube-system

# 查看 Pod（每个节点一个）
kubectl get pods -l app=fluentd -o wide

# 更新
kubectl set image ds/fluentd fluentd=fluent/fluentd:v1.15

# 查看滚动更新状态
kubectl rollout status ds/fluentd -n kube-system

# 删除
kubectl delete ds fluentd
```

### 实践练习

```bash
# 创建日志收集 DaemonSet
cat << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-collector
spec:
  selector:
    matchLabels:
      app: log-collector
  template:
    metadata:
      labels:
        app: log-collector
    spec:
      containers:
      - name: collector
        image: busybox
        command: ["sh", "-c", "while true; do echo 'Collecting logs from \$(hostname)'; sleep 60; done"]
        volumeMounts:
        - name: logs
          mountPath: /var/log
          readOnly: true
      volumes:
      - name: logs
        hostPath:
          path: /var/log
EOF

# 查看（每个节点一个 Pod）
kubectl get ds
kubectl get pods -l app=log-collector -o wide

# 观察 Pod 分布
kubectl get pods -l app=log-collector -o wide | awk '{print $1, $7}'

# 清理
kubectl delete ds log-collector
```

## Job

### 什么是 Job？

Job 用于运行一次性任务，确保指定数量的 Pod 成功完成。

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Job                                       │
│                                                                       │
│  完成数: 5                                                           │
│  并行数: 2                                                           │
│                                                                       │
│  ┌─────────┐ ┌─────────┐                                           │
│  │  Pod 1  │ │  Pod 2  │  ← 并行运行                                │
│  │   ✓     │ │   ✓     │                                           │
│  └─────────┘ └─────────┘                                           │
│                                                                       │
│  ┌─────────┐ ┌─────────┐                                           │
│  │  Pod 3  │ │  Pod 4  │  ← 前面完成后继续                          │
│  │   ✓     │ │   ✓     │                                           │
│  └─────────┘ └─────────┘                                           │
│                                                                       │
│  ┌─────────┐                                                        │
│  │  Pod 5  │             ← 最后一个                                  │
│  │   ✓     │                                                        │
│  └─────────┘                                                        │
│                                                                       │
│  Job 完成！                                                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Job 和普通 Pod 的区别

| 特性 | 普通 Pod | Job 管理的 Pod |
|------|---------|---------------|
| 重启策略 | 默认 Always | Never 或 OnFailure |
| 完成条件 | 不适用 | 容器退出码为 0 |
| 失败处理 | 持续重启 | 达到 backoffLimit 后停止 |
| 生命周期 | 永久运行 | 任务完成后终止 |

### Job YAML 完整解析

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-job
spec:
  # 完成数：需要成功完成的 Pod 数量
  completions: 5
  
  # 并行数：同时运行的 Pod 数量
  parallelism: 2
  
  # 重试次数
  backoffLimit: 4
  
  # 超时时间（秒）
  activeDeadlineSeconds: 300
  
  # 完成后保留时间（秒，K8s 1.23+）
  ttlSecondsAfterFinished: 100
  
  template:
    spec:
      restartPolicy: Never        # 或 OnFailure
      containers:
      - name: worker
        image: busybox
        command: ["sh", "-c", "echo Processing item && sleep 30"]
```

### Job 类型

#### 1. 单任务 Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: single-job
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: task
        image: busybox
        command: ["echo", "Hello Job"]
```

#### 2. 并行 Job（固定完成数）

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: parallel-job
spec:
  completions: 10        # 总共需要完成 10 个
  parallelism: 3         # 同时运行 3 个
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: busybox
        command: ["sh", "-c", "echo Task $RANDOM && sleep 5"]
```

#### 3. 工作队列 Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: queue-job
spec:
  parallelism: 3         # 只设置并行数
  # 不设置 completions，Pod 自己决定何时完成
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: myapp/queue-processor
```

### Job 生命周期

```
创建 Job
  │
  ├──→ Pod 创建（根据 parallelism 控制并发数）
  │       │
  │       ├──→ Pod 成功退出（exit 0）
  │       │       └──→ completions 计数 +1
  │       │
  │       ├──→ Pod 失败退出（exit != 0）
  │       │       └──→ 重试（backoffLimit 控制最大重试次数）
  │       │
  │       └──→ Pod 超时（activeDeadlineSeconds）
  │               └──→ Job 标记为 Failed
  │
  ├──→ completions 达到 → Job 标记为 Complete
  │
  └──→ backoffLimit 耗尽 → Job 标记为 Failed
```

### Job 操作

```bash
# 创建
kubectl apply -f job.yaml
kubectl create job my-job --image=busybox -- echo "Hello"

# 查看
kubectl get jobs
kubectl describe job batch-job
kubectl get pods -l job-name=batch-job

# 查看日志
kubectl logs job/batch-job

# 删除
kubectl delete job batch-job

# 级联删除 Pod
kubectl delete job batch-job --cascade=foreground
```

## CronJob

### 什么是 CronJob？

CronJob 按照预定时间表创建 Job。

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CronJob                                     │
│                                                                       │
│  schedule: "*/5 * * * *"  (每 5 分钟)                                │
│                                                                       │
│  时间线:                                                              │
│  ─────┬─────┬─────┬─────┬─────┬─────┬─────→                        │
│       │     │     │     │     │     │                               │
│       ▼     ▼     ▼     ▼     ▼     ▼                               │
│     Job 1 Job 2 Job 3 Job 4 Job 5 Job 6                             │
│                                                                       │
│  00:00 00:05 00:10 00:15 00:20 00:25                                │
└─────────────────────────────────────────────────────────────────────┘
```

### Cron 表达式

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日 (1 - 31)
│ │ │ ┌───────────── 月 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6，0 = 周日)
│ │ │ │ │
* * * * *

示例：
*/5 * * * *     # 每 5 分钟
0 * * * *       # 每小时
0 0 * * *       # 每天凌晨
0 0 * * 0       # 每周日凌晨
0 0 1 * *       # 每月 1 日凌晨
0 9 * * 1-5     # 工作日早上 9 点
```

### CronJob YAML 完整解析

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-job
spec:
  # Cron 表达式
  schedule: "0 2 * * *"           # 每天凌晨 2 点
  
  # 时区（K8s 1.27+）
  timeZone: "Asia/Shanghai"
  
  # 并发策略
  concurrencyPolicy: Forbid       # Allow/Forbid/Replace
  
  # 保留历史 Job 数量
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  
  # 启动截止时间（秒）
  startingDeadlineSeconds: 200
  
  # 挂起
  suspend: false
  
  # Job 模板
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: backup
            image: backup-tool:latest
            command: ["/bin/sh", "-c"]
            args:
            - |
              echo "Starting backup at $(date)"
              # 执行备份逻辑
              echo "Backup completed"
```

### 并发策略

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| Allow | 允许并发运行（默认）| 任务之间不冲突 |
| Forbid | 禁止并发，跳过新调度 | 上一次还没跑完，这次就不跑了 |
| Replace | 取消当前运行的，启动新的 | 确保只运行最新一次 |

### CronJob 操作

```bash
# 创建
kubectl apply -f cronjob.yaml
kubectl create cronjob my-cron --image=busybox --schedule="*/5 * * * *" -- echo "Hello"

# 查看
kubectl get cronjobs
kubectl get cj                           # 简写
kubectl describe cj backup-job

# 手动触发一次
kubectl create job manual-backup --from=cronjob/backup-job

# 暂停/恢复
kubectl patch cj backup-job -p '{"spec":{"suspend":true}}'
kubectl patch cj backup-job -p '{"spec":{"suspend":false}}'

# 查看生成的 Job
kubectl get jobs

# 删除
kubectl delete cj backup-job
```

### 实践练习

#### 练习 1：创建 Job

```bash
# 创建并行 Job
cat << EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: parallel-job
spec:
  completions: 5
  parallelism: 2
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: busybox
        command: ["sh", "-c", "echo Processing \$HOSTNAME && sleep 10"]
EOF

# 观察执行
kubectl get pods -l job-name=parallel-job -w

# 查看完成状态
kubectl get job parallel-job

# 清理
kubectl delete job parallel-job
```

#### 练习 2：创建 CronJob

```bash
# 创建定时任务
cat << EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello-cron
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: hello
            image: busybox
            command: ["sh", "-c", "echo Hello at \$(date)"]
EOF

# 等待 1-2 分钟查看
kubectl get cj
kubectl get jobs
kubectl logs job/<job-name>

# 手动触发
kubectl create job manual-hello --from=cronjob/hello-cron

# 清理
kubectl delete cj hello-cron
kubectl delete job --all
```

## 总结

| 资源类型 | 用途 | 特点 | Pod 数量决定因素 |
|---------|------|------|----------------|
| DaemonSet | 每节点一个 Pod | 系统守护进程，节点级服务 | 符合条件的节点数 |
| Job | 一次性任务 | 确保任务完成，失败重试 | completions + parallelism |
| CronJob | 定时任务 | 按计划创建 Job，支持并发策略 | 受 schedule 和 concurrencyPolicy 控制 |

## 下一步

- [kubectl 命令完全手册](../03-practice/01-kubectl-commands.md)
