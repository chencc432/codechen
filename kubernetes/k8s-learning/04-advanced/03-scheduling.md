# 🧭 Kubernetes 调度机制与策略

## 为什么调度值得单独学

很多人刚接触 Kubernetes 时，会觉得 Pod 提交以后系统“自己会找地方跑起来”。这句话不算错，但太粗了。

更准确地说，Kubernetes 调度器在回答一个很具体的问题：

> 这个 Pod 应该被放到哪一个节点，才能同时满足约束、资源、策略和整体集群效率？

调度问题一旦理解不清，后面几类现象都会很难排查：

- Pod 一直 `Pending`
- 明明有节点，但就是不调度
- Pod 总跑到“不想让它去”的机器上 
- 同类副本都扎堆在同一台机器
- GPU、SSD、数据库等专用节点被普通业务抢占

所以调度不是边缘知识，而是 Kubernetes 的核心运行机制之一。

## 先建立一个整体心智模型

调度器工作时，可以把它想成 3 步：

1. **先排除不能放的节点**
2. **再给剩下的节点打分**
3. **最后选择最合适的节点**

也就是：

```text
待调度 Pod
  -> 过滤（哪些节点绝对不行）
  -> 打分（哪些节点更合适）
  -> 绑定（最终选中一个节点）
```

## 1. 调度器到底看什么

调度器通常会综合考虑以下因素：

- 节点剩余 CPU、内存是否足够
- Pod 的 `requests` 是否满足
- `nodeSelector` 是否匹配
- `nodeAffinity` 是否满足
- Pod 亲和性 / 反亲和性是否满足
- 是否存在污点以及 Pod 是否容忍
- 某些端口、卷、拓扑约束是否冲突
- 高可用、负载均衡、镜像本地性等偏好

这意味着：

- 调度不是只看资源
- 也不是只看标签
- 而是“硬约束 + 软偏好 + 集群当前状态”的综合结果



## 2. requests 为什么会影响调度

很多人知道 `resources.requests`，但不知道它对调度器有多重要。

最关键的一句是：

> 调度器主要看 `requests` 来判断节点是否“装得下”这个 Pod。

例如：

```yaml
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "1"
    memory: "1Gi"
```

在调度阶段，更关键的是：

- 这个节点能不能至少提供 `500m CPU + 512Mi 内存`

所以常见现象是：

- 节点总资源很多
- 但由于可分配资源不足
- Pod 仍然会一直 `Pending`



## 3. 最基础的调度控制：nodeSelector

`nodeSelector` 是最简单的节点约束方式，本质上就是：

> “Pod 只能调度到带某些标签的节点上。”

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  nodeSelector:
    disktype: ssd
  containers:
  - name: nginx
    image: nginx
```



### 3.1 适用场景

- 指定 SSD 节点
- 指定 GPU 节点
- 指定 Linux/Windows 节点
- 指定某类业务专用节点



### 3.2 优缺点

优点：

- 简单
- 易读
- 易上手

缺点：

- 只能做精确匹配
- 不能表达“最好而不是必须”
- 复杂场景扩展性差



## 4. 更灵活的方式：Node Affinity

`nodeAffinity` 可以理解为 `nodeSelector` 的增强版。

它支持两种模式：

- **硬性要求**：必须满足
- **软性偏好**：最好满足

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: affinity-demo
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: disktype
            operator: In
            values:
            - ssd
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        preference:
          matchExpressions:
          - key: zone
            operator: In
            values:
            - zone-a
  containers:
  - name: nginx
    image: nginx
```



### 4.1 最值得记住的区别

- `required...`：不满足就不能调度
- `preferred...`：满足更好，不满足也可能调度



### 4.2 常见操作符


| 操作符            | 含义      |
| -------------- | ------- |
| `In`           | 在给定列表中  |
| `NotIn`        | 不在给定列表中 |
| `Exists`       | 标签存在    |
| `DoesNotExist` | 标签不存在   |
| `Gt`           | 大于      |
| `Lt`           | 小于      |




## 5. Pod 亲和性与反亲和性

如果说 `nodeAffinity` 是“Pod 想靠近什么样的节点”，那么 `podAffinity` / `podAntiAffinity` 更像是：

- 想靠近哪些 Pod
- 想远离哪些 Pod



### 5.1 Pod Affinity

适合“放近一点”的场景：

- Web 和缓存放同一节点，减少网络延迟
- 某些强耦合服务希望在同一可用区



### 5.2 Pod Anti-Affinity

适合“打散部署”的场景：

- 多副本不要集中在同一节点
- 多可用区分布，提高可用性

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
spec:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: web
          topologyKey: kubernetes.io/hostname
  containers:
  - name: nginx
    image: nginx
```



### 5.3 topologyKey 是什么

它定义了“靠近”或“分开”的粒度。

常见值：

- `kubernetes.io/hostname`：节点级别
- `topology.kubernetes.io/zone`：可用区级别
- `topology.kubernetes.io/region`：区域级别

最常用、也最容易理解的是：

- 同一节点：`kubernetes.io/hostname`



## 6. 污点与容忍度

这是调度里最常被误解的一组概念。

### 6.1 污点（Taint）

污点加在节点上，表达的是：

> “这台节点不欢迎普通 Pod。”

```bash
kubectl taint nodes node1 gpu=true:NoSchedule
```



### 6.2 容忍度（Toleration）

容忍度写在 Pod 上，表达的是：

> “我可以接受这种污点，不要把我拦在门外。”

```yaml
tolerations:
- key: "gpu"
  operator: "Equal"
  value: "true"
  effect: "NoSchedule"
```



### 6.3 最常见的误区

`toleration` 只是“允许进入”，不是“强制去那里”。

如果你希望 Pod 一定跑到 GPU 节点，通常要组合：

- 节点打污点
- Pod 配容忍度
- 再加 `nodeSelector` 或 `nodeAffinity`



## 7. 调度优先级和抢占

在集群资源紧张时，并不是所有 Pod 地位都一样。

Kubernetes 可以通过优先级类（PriorityClass）表达：

- 哪些 Pod 更重要
- 资源不够时谁优先被调度
- 极端情况下谁可以抢占低优先级 Pod 的资源



### 7.1 什么时候会用到

- 核心系统组件优先级高于普通业务
- 生产业务高于测试业务
- 监控、网关、关键控制面组件要优先活下来



### 7.2 抢占需要谨慎

抢占可以解决高优先级业务上不去的问题，但也可能带来：

- 低优先级业务被驱逐
- 突发连锁影响
- 调度行为更难预测

所以生产里通常需要明确优先级设计，而不是随意设置。

## 8. 常见调度场景



### 8.1 GPU 调度

GPU 在 Kubernetes 里是最特殊的资源之一。它昂贵、稀缺、不可分（在默认配置下），而且通常需要配合物理拓扑、驱动版本、显存容量等多重约束来调度。

#### 8.1.1 为什么 GPU 调度和普通资源不一样

CPU 和内存是"可压缩"的——调度器可以分配 `500m` CPU，Pod 实际用多少都行，不会出大问题。

GPU 不一样：

- GPU 是**不可压缩资源**：你不能让两个 Pod 共享同一个 GPU（除非显式配置，见下文）
- GPU 是**整卡分配**的：要么整张卡给一个 Pod，要么不给
- GPU 有**驱动版本和 CUDA 版本兼容性**问题
- GPU 有**显存**瓶颈：算力够但显存不够也不行
- GPU 有**拓扑**约束：多卡通信时，卡的位置（同一 PCIe 交换机、NVLink）会影响性能

所以 Kubernetes 对 GPU 的调度策略，本质上是在回答这些问题：

> 这个 Pod 需要 GPU 吗？需要哪类 GPU？需要多少？节点上还剩多少 GPU？怎么保证 GPU 只给真正需要的 Pod 用？



#### 8.1.2 GPU 在 Kubernetes 中如何"出现"

Kubernetes 本身不懂 GPU。它需要通过 **Device Plugin** 机制把 GPU 注册为可调度资源。

```
nvidia-device-plugin（DaemonSet）
  │
  ├── 检测节点上的 NVIDIA GPU
  ├── 向 kubelet 注册 GPU 资源
  └── kubelet 更新 Node 状态
        │
        └── Node 的可分配资源里出现 nvidia.com/gpu: 4
```

**安装 Device Plugin**：

```bash
# 安装 NVIDIA Device Plugin（需要先安装 NVIDIA 驱动和 nvidia-docker）
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.15.0/nvidia-device-plugin.yml
```

安装完成后，GPU 节点会显示：

```bash
kubectl get node <gpu-node> -o json | jq '.status.allocatable'
# 输出示例：
# {
#   "cpu": "16",
#   "memory": "65892884Ki",
#   "nvidia.com/gpu": "4",    ← GPU 资源出现了
#   "pods": "110"
# }
```



#### 8.1.3 GPU 资源模型

在 YAML 中请求 GPU 的方式：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-pod
spec:
  containers:
  - name: cuda
    image: nvidia/cuda:12.0-base
    command: ["nvidia-smi"]
    resources:
      requests:
        nvidia.com/gpu: 1      # 请求 1 张 GPU
      limits:
        nvidia.com/gpu: 1      # 必须和 requests 一致
```

**GPU 资源的关键规则**：

- `requests` 和 `limits` 必须相等（GPU 不支持超卖）
- `nvidia.com/gpu` 是一个整数资源，不能写 `0.5`
- 一个 Pod 可以请求多张卡：`nvidia.com/gpu: 4`
- 请求了 GPU 的 Pod 会自动挂载 NVIDIA 驱动库和 CUDA 工具包



#### 8.1.4 GPU 隔离策略

GPU 是稀缺资源，核心策略是：**确保只有需要 GPU 的 Pod 才能进入 GPU 节点**。

标准做法是**标签 + 污点 + 容忍度 + nodeSelector 四件套**：

```bash
# 步骤 1：给 GPU 节点打标签
kubectl label node gpu-node-1 accelerator=nvidia

# 步骤 2：给 GPU 节点打污点（阻止普通 Pod 进入）
kubectl taint node gpu-node-1 nvidia.com/gpu=present:NoSchedule
```

```yaml
# 步骤 3：需要 GPU 的 Pod 加上容忍度和 nodeSelector
apiVersion: v1
kind: Pod
metadata:
  name: gpu-job
spec:
  tolerations:
  - key: "nvidia.com/gpu"
    operator: "Equal"
    value: "present"
    effect: "NoSchedule"
  nodeSelector:
    accelerator: nvidia
  containers:
  - name: cuda
    image: nvidia/cuda:12.0-base
    command: ["nvidia-smi"]
    resources:
      requests:
        nvidia.com/gpu: 1
      limits:
        nvidia.com/gpu: 1
```

**不配污点的后果**：

```
普通 Pod 被调度到 GPU 节点
  └── GPU 被占着但没人用
  └── 真正需要 GPU 的 Pod 反而没地方跑
  └── GPU 资源浪费 + 调度效率低
```

**不配容忍度的后果**：

```
GPU Pod 无法调度
  └── 节点有污点，Pod 没有容忍度
  └── GPU Pod 一直 Pending
  └── kubectl describe pod 显示：node(s) had taint that the pod didn't tolerate
```



#### 8.1.5 多卡 Pod 的调度行为

当 Pod 请求 `nvidia.com/gpu: 4` 时，调度器会：

1. 找到至少有 4 张空闲 GPU 的节点
2. 如果节点有 8 张卡，但已有 Pod 占了 6 张，还剩 2 张 → 不够，不调度
3. 如果节点有 8 张卡，空闲 8 张 → 满足，该节点进入候选

**一个重要细节**：默认情况下，Kubernetes 不关心这 4 张卡是不是在同一个 PCIe 交换机下，也不关心它们是不是通过 NVLink 连接的。它只关心"数量够不够"。

#### 8.1.6 MIG（Multi-Instance GPU）

NVIDIA A100 和 H100 支持 MIG 技术，可以把一张物理 GPU 切分成多个独立的 GPU 实例。

```
物理 GPU：NVIDIA A100 (80GB)
  │
  ├── MIG 实例 1：1g.10gb  (1 个计算单元 + 10GB 显存)
  ├── MIG 实例 2：1g.10gb  (1 个计算单元 + 10GB 显存)
  ├── MIG 实例 3：2g.20gb  (2 个计算单元 + 20GB 显存)
  └── MIG 实例 4：3g.40gb  (3 个计算单元 + 40GB 显存)
```

**在 Kubernetes 中使用 MIG**：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mig-pod
  annotations:
    nvidia.com/mig-config: "1g.10gb"   # 指定 MIG 配置
spec:
  containers:
  - name: cuda
    image: nvidia/cuda:12.0-base
    resources:
      requests:
        nvidia.com/gpu: 1
      limits:
        nvidia.com/gpu: 1
```

**MIG 的价值**：

- 一张 A100 可以同时服务多个小模型推理任务
- 每个 MIG 实例有独立的显存和计算单元，互不干扰
- 提高 GPU 利用率，降低碎片化

**MIG 的限制**：

- 只支持 A100、H100 及后续架构
- 需要在节点层面预先配置 MIG 策略
- 不是所有 CUDA 应用都兼容 MIG
- MIG 配置变更需要重启节点



#### 8.1.7 GPU Time-Slicing（时间片）

如果不需要 MIG 的硬件隔离，可以用 GPU Operator 的 Time-Slicing 功能，让多个 Pod 共享同一张 GPU。

```
物理 GPU：NVIDIA A100 (80GB)
  │
  ├── Pod A：独占 GPU 50% 的时间片
  ├── Pod B：独占 GPU 30% 的时间片
  └── Pod C：独占 GPU 20% 的时间片
```

**Time-Slicing 配置示例**：

```yaml
# ClusterPolicy 配置
apiVersion: nvidia.com/v1
kind: ClusterPolicy
metadata:
  name: gpu-cluster-policy
spec:
  timeSlicing:
    resources:
    - name: nvidia.com/gpu
      replicas: 4   # 一张物理卡暴露为 4 个可调度资源
```

**Time-Slicing vs MIG**：


| 对比维度  | MIG           | Time-Slicing   |
| ----- | ------------- | -------------- |
| 隔离级别  | 硬件隔离（显存+计算）   | 时间片（计算共享，显存共享） |
| 适用硬件  | A100/H100 及以上 | 所有 NVIDIA GPU  |
| 性能隔离  | 强             | 弱（可能互相干扰）      |
| 显存隔离  | 独立显存          | 共享显存           |
| 适用场景  | 多租户、安全隔离要求高   | 开发测试、小模型推理     |
| 配置复杂度 | 高（需要重启节点）     | 低（动态配置）        |




#### 8.1.8 GPU 拓扑感知调度

对于多卡训练（如分布式深度学习），GPU 之间的通信效率至关重要。

```
同一个 PCIe 交换机下的 GPU：
  GPU 0 ────┐
  GPU 1 ────┼── PCIe Switch ── CPU
  GPU 2 ────┘
  GPU 3

NVLink 直连的 GPU：
  GPU 0 ─── NVLink ─── GPU 1
    │                     │
  NVLink                NVLink
    │                     │
  GPU 2 ─── NVLink ─── GPU 3
```

**NVLink 比 PCIe 快 5-10 倍**，所以对于多卡训练任务，调度器最好能把 Pod 调度到 NVLink 互联的 GPU 上。

**NVIDIA GPU Operator 的拓扑调度**：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: distributed-training
  annotations:
    nvidia.com/gpu-pod-scheduling: "true"   # 启用拓扑感知
spec:
  containers:
  - name: train
    image: pytorch:latest
    resources:
      requests:
        nvidia.com/gpu: 4
      limits:
        nvidia.com/gpu: 4
```

启用后，调度器会：

1. 检查节点上 GPU 的拓扑结构
2. 找到 4 张 NVLink 互连的 GPU
3. 把 Pod 调度到这些 GPU 所在的节点
4. 确保 Pod 分配到的 4 张卡之间通信效率最高



#### 8.1.9 GPU 调度常见问题

**问题 1：Pod 请求 GPU 但一直 Pending**

```bash
kubectl describe pod gpu-pod
# 输出：
# Events:
#   Type     Reason            Age   From               Message
#   ----     ------            ----  ----               -------
#   Warning  FailedScheduling  12s   default-scheduler  0/5 nodes are available: 3 Insufficient nvidia.com/gpu, 2 node(s) had taint that the pod didn't tolerate.
```

排查方向：

- 检查节点是否有 GPU：`kubectl describe node <node> | grep nvidia.com/gpu`
- 检查 Device Plugin 是否运行：`kubectl get pods -n kube-system | grep nvidia`
- 检查 GPU 节点是否有污点：`kubectl describe node <node> | grep Taints`
- 检查 Pod 是否配置了 tolerations

**问题 2：Pod 在 GPU 节点上但找不到 GPU**

```bash
# 进入 Pod 后执行 nvidia-smi 报错
kubectl exec gpu-pod -- nvidia-smi
# 输出：nvidia-smi: command not found 或 Failed to initialize NVML: Driver/library version mismatch
```

排查方向：

- 节点上的 NVIDIA 驱动版本是否和容器镜像中的 CUDA 版本兼容
- 是否安装了 nvidia-docker 和 nvidia-container-runtime
- Device Plugin 是否正常工作
- 是否忘了设置 `nvidia.com/gpu` 的 resources（只写 tolerations 是不够的）

**问题 3：GPU 利用率低**

```
现象：GPU 节点上跑了很多 Pod，但 GPU 利用率只有 10-20%

原因：
  - 很多 Pod 申请了 GPU 但实际没怎么用（比如推理任务间歇性调用）
  - 没有使用 MIG 或 Time-Slicing 来共享 GPU
  - 单卡显存很大但 Pod 只需要一小部分

优化方向：
  - 考虑 GPU 共享方案（MIG、Time-Slicing）
  - 对于推理任务，考虑在 Pod 级别做批处理
  - 监控 GPU 利用率，识别低效占用
```

**问题 4：不同 GPU 型号混用**

```
场景：集群中有 A100、V100、T4 三种 GPU
问题：Pod 请求 nvidia.com/gpu: 1，可能被调度到任意一种 GPU 上
      但 CUDA 程序可能对 GPU 架构有要求（比如需要 A100 的 MIG 功能）

解决方案：通过节点标签区分 GPU 类型
```

```bash
# 给不同 GPU 节点打不同标签
kubectl label node a100-node gpu-type=a100
kubectl label node v100-node gpu-type=v100
kubectl label node t4-node  gpu-type=t4
```

```yaml
# Pod 指定 GPU 类型
apiVersion: v1
kind: Pod
metadata:
  name: a100-job
spec:
  nodeSelector:
    gpu-type: a100
  containers:
  - name: cuda
    image: nvidia/cuda:12.0-base
    resources:
      requests:
        nvidia.com/gpu: 1
      limits:
        nvidia.com/gpu: 1
```



#### 8.1.10 GPU 调度最佳实践

1. **污点隔离是必须的**：GPU 节点必须打污点，防止普通 Pod 占用
2. **标签区分 GPU 类型**：不同型号的 GPU 用不同标签，让 Pod 精确选择
3. **requests 和 limits 必须相等**：GPU 不支持超卖
4. **监控 GPU 利用率**：用 `dcgm-exporter` 采集 GPU 指标，识别浪费
5. **小模型推理用 MIG 或 Time-Slicing**：提高 GPU 利用率
6. **大模型训练用拓扑感知**：确保多卡通信效率
7. **不要忽略显存**：`nvidia-smi` 显示的显存使用比"算力"更容易成为瓶颈
8. **驱动版本一致性**：同一个 GPU 节点池的驱动版本保持一致，避免兼容性问题



### 8.2 高可用多副本

目标：

- 多个副本不要集中到一个节点

常见做法：

- 使用 `podAntiAffinity`
- 或者结合拓扑分布策略



### 8.3 数据库与状态服务

目标：

- 固定在某类高性能节点
- 与普通业务隔离

常见做法：

- 节点标签
- 节点污点
- 严格的 `requiredDuringScheduling...`



### 8.4 调度优先级（PriorityClass）

在集群资源紧张时，不是所有 Pod 都该有相同的调度优先级。训练任务和高优在线服务应该能优先调度，低优任务可以排队等资源。

#### PriorityClass 是什么

PriorityClass 定义了一个优先级等级，调度器根据优先级决定：

- 先调度谁
- 资源不够时驱逐谁（抢占）

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000000               # 数值越大优先级越高
globalDefault: false          # 是否设为集群默认优先级
description: "High priority for production training jobs"
```

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: low-priority
value: 100
globalDefault: false
description: "Low priority for batch jobs"
```

**在 Pod 中指定优先级**：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: training-job
spec:
  priorityClassName: high-priority
  containers:
  - name: trainer
    image: pytorch:latest
```



#### 优先级如何影响调度行为

```
系统内置优先级：
  2147483647  → system-node-critical（节点关键组件，如 kubelet）
  2000001000  → system-cluster-critical（集群关键组件，如 DNS、控制器）
  1000000     → high-priority（用户定义的高优任务）
  100         → low-priority（用户定义的低优任务）
  0           → 默认值
```

**优先级的作用**：

1. **调度顺序**：高优先级的 Pod 会先进入调度队列
2. **抢占（Preemption）**：如果高优 Pod 调度不下去，调度器会驱逐低优 Pod 腾出资源
3. **Node Pressure 驱逐**：节点资源紧张时，kubelet 优先驱逐低优先级的 Pod



#### 抢占的代价

```text
高优 Pod 需要资源 → 调度器发现资源不足
  → 找到低优 Pod 所在的节点
  → 驱逐低优 Pod（发送 SIGTERM）
  → 低优 Pod 重新调度或 Terminating
  → 高优 Pod 调度成功
```

**注意**：抢占不是优雅的。低优 Pod 被驱逐时可能正在写数据，所以训练任务通常需要 checkpoint 机制。

#### 训练场景的优先级策略

```yaml
# 高优：生产训练任务
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: production-training
value: 10000

# 中优：开发调试
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: dev-training
value: 5000

# 低优：批量实验
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: batch-experiment
value: 1000
```



### 8.5 PodDisruptionBudget（PDB）

训练任务跑了几小时，因为节点维护被驱逐，这种体验非常糟糕。PDB 就是用来解决这个问题的。

#### PDB 是什么

PDB 确保在主动中断（voluntary disruption）时，你的应用不会因为太多副本同时下线而不可用。

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: training-pdb
spec:
  minAvailable: 2              # 至少保留 2 个副本
  selector:
    matchLabels:
      app: distributed-training
```

或者：

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: training-pdb
spec:
  maxUnavailable: 1            # 最多允许 1 个副本不可用
  selector:
    matchLabels:
      app: distributed-training
```



#### 什么时候 PDB 生效

PDB 只影响**主动中断**，不影响**被动中断**：


| 中断类型          | PDB 是否拦截 | 举例                         |
| ------------- | -------- | -------------------------- |
| 节点 drain      | 是        | `kubectl drain node` 做节点维护 |
| 节点 Deprecated | 是        | 云厂商标记节点不可用，主动驱逐            |
| 节点硬件故障        | 否        | 机器宕机，PDB 拦不住               |
| 节点被抢占         | 否        | 高优 Pod 抢占低优 Pod            |
| 用户手动删除 Pod    | 否        | 你手动 kubectl delete pod     |




#### 训练场景的 PDB 配置

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: dist-training-pdb
spec:
  # 分布式训练：至少保留所有 worker 的 75%
  minAvailable: 75%
  selector:
    matchLabels:
      job-type: distributed-training
```



### 8.6 ResourceQuota 与 LimitRange

当多个团队共享 GPU 集群时，需要限制每个团队能使用多少资源。

#### ResourceQuota（资源配额）

**按命名空间限制总资源**：

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: team-a-quota
  namespace: team-a
spec:
  hard:
    requests.cpu: "40"
    requests.memory: "80Gi"
    requests.nvidia.com/gpu: 8    # 最多使用 8 张 GPU
    limits.cpu: "80"
    limits.memory: "160Gi"
    persistentvolumeclaims: 10
    pods: "50"
```



#### LimitRange（默认资源限制）

**设置命名空间内 Pod 的默认资源请求/限制**：

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: gpu-limit
  namespace: team-a
spec:
  limits:
  - max:
      requests.nvidia.com/gpu: 4    # 单个 Pod 最多申请 4 张 GPU
      cpu: "16"
      memory: "64Gi"
    min:
      requests.nvidia.com/gpu: 0
      cpu: "100m"
      memory: "128Mi"
    default:
      requests.nvidia.com/gpu: 0
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      requests.nvidia.com/gpu: 0
      cpu: "100m"
      memory: "256Mi"
    type: Container
```



#### 多团队 GPU 分配合法的示例

```bash
# 查看团队资源使用情况
kubectl describe quota team-a-quota -n team-a
# 输出示例：
# Name:         team-a-quota
# Namespace:    team-a
# Resource                Used  Hard
# --------                ----  ----
# requests.nvidia.com/gpu  2    8
# requests.cpu             12   40
# requests.memory          24Gi 80Gi
```



### 8.7 Cluster Autoscaler

训练任务需要 GPU，但集群里没有空闲 GPU 节点。Cluster Autoscaler 能自动增加节点。

#### 什么是 Cluster Autoscaler

Cluster Autoscaler 自动调整集群节点数量：

```
Pending Pod（因为资源不足）
  → Cluster Autoscaler 检测到
  → 向云厂商申请新节点（通常按节点池/ASG 扩容）
  → 新节点加入集群
  → Pod 调度到新节点
  → 节点空闲时缩容
```



#### 安装

```bash
# 以 AWS 为例
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml
```



#### 关键配置

```yaml
# cluster-autoscaler 启动参数
command:
- ./cluster-autoscaler
- --cloud-provider=aws
- --nodes=1:10:gpu-node-group    # 最少 1 个，最多 10 个 GPU 节点
- --scale-down-delay-after-add=10m
- --scale-down-unneeded-time=10m
- --skip-nodes-with-system-pods=false
```



#### GPU 场景的自动扩缩容

```yaml
# 节点池配置（以云厂商为例）
# GPU 节点池：
#   实例类型：p3.8xlarge（4 张 V100）
#   最小节点数：0
#   最大节点数：20
#   标签：accelerator=nvidia
#   污点：nvidia.com/gpu=present:NoSchedule

# 当有 GPU Pod 提交时：
# 1. 现有 GPU 节点资源不足
# 2. Pod 进入 Pending
# 3. Cluster Autoscaler 检测到不可调度的 Pod
# 4. 触发 GPU 节点池扩容
# 5. 新节点加入 → Pod 调度成功
# 6. 训练完成后，节点空闲 → 自动缩容到 0
```

**Cluster Autoscaler vs Karpenter**：


| 对比维度  | Cluster Autoscaler | Karpenter         |
| ----- | ------------------ | ----------------- |
| 扩缩容粒度 | 节点池级别              | 单个节点级别            |
| 调度感知  | 不感知 Pod 调度需求       | 直接根据 Pod 需求选择实例类型 |
| 缩容速度  | 慢（10 分钟级别）         | 快（分钟级别）           |
| 配置复杂度 | 需要配置 ASG/节点池       | 只需配置 Provisioner  |
| 适用场景  | 大规模稳定集群            | 动态负载、训练集群         |




### 8.8 Gang Scheduling（群体调度）

分布式训练的一个核心矛盾是：

> 一个 PyTorch 训练任务需要 8 个 worker 同时启动，但只要 1 个 worker 没起来，其他 7 个就在空等。



#### 什么是 Gang Scheduling

Gang Scheduling 确保一组 Pod 要么全部调度成功，要么一个都不调度。

```
没有 Gang Scheduling：
  Worker 0 ✓  Worker 1 ✓  Worker 2 ✓  Worker 3 ✓
  Worker 4 ✓  Worker 5 ✓  Worker 6 ✓  Worker 7 ✗（资源不足）
  → 前 7 个 worker 空等，浪费资源

有 Gang Scheduling：
  Worker 0 ～ Worker 7 全部等待，直到资源足够同时调度 8 个
  → 要么全部启动，要么都不启动
```



#### Volcano 实现 Gang Scheduling

[Volcano](https://github.com/volcano-sh/volcano) 是 Kubernetes 上最流行的批处理调度系统，支持 Gang Scheduling、队列、优先级等。

```bash
# 安装 Volcano
kubectl apply -f https://raw.githubusercontent.com/volcano-sh/volcano/main/installer/volcano-development.yaml
```

**Volcano 的 Job 定义**：

```yaml
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: pytorch-training
spec:
  # 最小可用成员数（Gang Scheduling 的核心）
  minAvailable: 4
  
  # 队列（用于资源配额管理）
  queue: training-queue
  
  # 任务定义
  tasks:
  - name: ps
    replicas: 1
    template:
      spec:
        containers:
        - name: ps
          image: pytorch:latest
          command: ["python", "distributed.py", "--role=ps"]
          resources:
            requests:
              cpu: "4"
              memory: "8Gi"
  
  - name: worker
    replicas: 4
    template:
      spec:
        containers:
        - name: worker
          image: pytorch:latest
          command: ["python", "distributed.py", "--role=worker"]
          resources:
            requests:
              nvidia.com/gpu: 1
              cpu: "8"
              memory: "32Gi"
```

**minAvailable 的含义**：最少需要 4 个 worker 全部就绪，Volcano 才会开始调度。如果当前集群只能提供 3 个 GPU worker 的资源，那 4 个 worker 全部排队等待。

#### Volcano 的队列管理

```yaml
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: training-queue
spec:
  weight: 1                # 队列权重（影响资源分配比例）
  capability:              # 队列资源上限
    cpu: "200"
    memory: "800Gi"
    nvidia.com/gpu: 32
```

多队列场景：

```yaml
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: production-queue
spec:
  weight: 2                # 生产队列权重更高，获得更多资源
  capability:
    nvidia.com/gpu: 16
---
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: research-queue
spec:
  weight: 1
  capability:
    nvidia.com/gpu: 16
```



#### 为什么原生 Kubernetes 不支持 Gang Scheduling

Kubernetes 的调度器是"逐个 Pod 调度"的，它不知道一个 Pod 是分布式训练任务的一部分。当集群资源只够 7 个 worker 时，它会先调度 7 个，然后第 8 个 Pending。

Volcano 通过一个"**Resource Reservation**"阶段来解决这个问题：先锁定足够的资源，再开始调度，如果资源不够，所有 Pod 都排队。

#### 什么时候需要 Gang Scheduling


| 场景                       | 需要 Gang Scheduling | 原因                       |
| ------------------------ | ------------------ | ------------------------ |
| 单卡训练/推理                  | 不需要                | 单个 Pod 独立运行              |
| 多卡单机训练                   | 不需要                | 一张机器上申请多张 GPU，调度器能处理     |
| 多机多卡分布式训练                | **需要**             | 需要 N 个 worker 同时启动       |
| AllReduce 通信训练           | **需要**             | 任何一个 worker 缺失都会导致集合通信卡住 |
| 弹性训练（torchrun --elastic） | 不需要                | 支持动态增删 worker            |




## 9. Pod 一直 Pending 时怎么查

这是最常见的调度排障问题。

### 9.1 第一优先级命令

```bash
kubectl describe pod <pod-name>
```

重点看：

- `Events`
- `Warning`
- `FailedScheduling`



### 9.2 常见失败原因


| 现象                                                   | 常见原因               |
| ---------------------------------------------------- | ------------------ |
| `0/3 nodes are available: insufficient cpu`          | 节点 CPU 不足          |
| `node(s) didn't match node selector`                 | `nodeSelector` 不匹配 |
| `node(s) didn't match Pod's node affinity`           | 节点亲和性过严            |
| `node(s) had taint ... that the pod didn't tolerate` | 没有容忍度              |
| PVC 相关错误                                             | 存储未准备好             |




### 9.3 排查顺序建议

1. `kubectl describe pod`
2. `kubectl get nodes --show-labels`
3. 检查资源请求是否过大
4. 检查 `nodeSelector` / `affinity`
5. 检查 `tolerations`
6. 如果用了卷，再检查 PVC / PV



## 10. 一个综合调度示例

下面这个例子同时表达了：

- 必须是 SSD 节点
- 优先选择 `zone-a`
- 能容忍专用节点污点

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: scheduling-complete-demo
spec:
  nodeSelector:
    disktype: ssd
  tolerations:
  - key: "dedicated"
    operator: "Equal"
    value: "backend"
    effect: "NoSchedule"
  affinity:
    nodeAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 80
        preference:
          matchExpressions:
          - key: topology.kubernetes.io/zone
            operator: In
            values:
            - zone-a
  containers:
  - name: app
    image: nginx
    resources:
      requests:
        cpu: "250m"
        memory: "256Mi"
```



## 11. 最佳实践

- 先从简单规则开始，不要一上来写过度复杂的亲和性
- 节点标签命名要稳定、统一、可预期
- 专用节点建议同时使用“标签 + 污点 + 容忍度”
- 副本型工作负载优先考虑打散策略，提升可用性
- 调度问题优先看 `describe pod` 的事件，而不是只盯着 YAML
- `requests` 要真实，过大和过小都会让调度质量变差



## 12. 一页总结

- 调度器核心任务是“为 Pod 选择最合适的节点”
- `requests` 是调度阶段最重要的资源依据
- `nodeSelector` 简单直接，`nodeAffinity` 更灵活
- `podAntiAffinity` 很适合做高可用打散
- 污点是节点的拒绝策略，容忍度是 Pod 的准入许可
- `Pending` 不等于系统坏了，通常是调度条件不满足
- 排查调度问题最重要的入口是 `kubectl describe pod`



## 下一步

- [安全与权限控制](./04-security.md)
- [核心概念与术语](../01-basics/03-concepts.md)

