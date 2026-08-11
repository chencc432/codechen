# 🧠 AI Infra 在 Kubernetes 上的实践

## 前言：AI Infra 给 Kubernetes 带来了什么特殊挑战

Kubernetes 最初是为无状态 Web 服务设计的。但 AI 训练/推理工作负载的到来，给这个体系带来了几个根本性的新问题：

```
传统 Kubernetes 工作负载                    AI 训练工作负载
─────────────────────                      ─────────────────────
CPU 为主，内存为辅                          GPU 是核心资源，昂贵且稀缺
Pod 是无状态的，死了就重建                  Pod 挂了意味着训练中断，需要 checkpoint
所有 Pod 可以独立调度                       N 个 worker 必须同时启动（Gang Scheduling）
网络用默认 CNI 就够了                      需要 RDMA/InfiniBand 高性能网络
资源需求相对稳定                            显存和算力需求动态变化
不需要知道硬件拓扑                          需要感知 NUMA、NVLink、PCIe 拓扑
单次运行时间短                              训练可能跑几天甚至几周
```

这套文档的目的是：**把 Kubernetes 上跑 AI 训练/推理所需要但原生 Kubernetes 没直接提供的知识，系统性地串起来**。

**本文假设你已经熟悉 Kubernetes 的基础概念**（Pod、Deployment、DaemonSet、StatefulSet、调度、网络）。如果还不熟悉，建议先读前面的基础文档。

## 1. GPU 资源管理

### 1.1 从 Device Plugin 到 GPU Operator

我们之前讲过 Device Plugin 把 GPU 注册为可调度资源。但在生产环境中，只装 Device Plugin 是不够的：

```
Device Plugin 只做一件事：
  ┌──────────────┐
  │ 检测 GPU →   │  → 把 nvidia.com/gpu 注册到 kubelet
  │ 向 kubelet   │
  │ 注册资源     │
  └──────────────┘

GPU Operator 做了一整套事：
  ┌──────────────┐
  │ Device       │  → GPU 资源注册
  │ Plugin       │
  ├──────────────┤
  │ Driver       │  → 自动安装 NVIDIA 驱动
  │ Installer    │
  ├──────────────┤
  │ DCGM         │  → GPU 监控和指标暴露
  │ Exporter     │
  ├──────────────┤
  │ MIG Manager  │  → 管理 MIG 配置
  ├──────────────┤
  │ GPU Feature  │  → 检测节点 GPU 能力
  │ Discovery    │
  ├──────────────┤
  │ Time-Slicing │  → GPU 时间片共享
  │ Controller   │
  └──────────────┘
```

**安装 GPU Operator**：

```bash
# 安装 Helm（如果还没装）
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod +x get_helm.sh
./get_helm.sh

# 添加 NVIDIA Helm 仓库
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update

# 安装 GPU Operator
helm install gpu-operator nvidia/gpu-operator \
  --namespace nvidia-gpu-operator \
  --create-namespace \
  --set driver.enabled=true \
  --set toolkit.enabled=true
```

**GPU Operator 的价值**：你不需要手动在每个节点上装驱动、配置 Device Plugin、部署监控。Operator 会自动化这些操作。

### 1.2 GPU 监控与可观测性

训练任务跑着跑着变慢了，你怎么知道是 GPU 的问题？

**DCGM Exporter**（Data Center GPU Manager）是 NVIDIA 官方的 GPU 指标暴露工具，GPU Operator 默认会安装：

```bash
# 查看 DCGM Exporter 是否运行
kubectl get pods -n nvidia-gpu-operator -l app=nvidia-dcgm-exporter

# 暴露指标
kubectl port-forward -n nvidia-gpu-operator svc/nvidia-dcgm-exporter 9400:9400

# 访问指标
curl http://localhost:9400/metrics
```

**关键 GPU 指标**：

| 指标 | 说明 | 正常值 | 告警阈值 |
|------|------|--------|----------|
| `DCGM_FI_DEV_GPU_UTIL` | GPU 利用率 | 80-100%（训练） | < 30%（可能代码有问题） |
| `DCGM_FI_DEV_FB_USED` | 显存使用量 | 视模型而定 | > 90%（可能 OOM） |
| `DCGM_FI_DEV_MEM_TEMP` | 显存温度 | < 80°C | > 90°C（降频风险） |
| `DCGM_FI_DEV_POWER_USAGE` | 功耗 | 视 GPU 型号而定 | 接近 TDP |
| `DCGM_FI_DEV_XID_ERRORS` | GPU 错误 | 0 | > 0（需要排查） |
| `DCGM_FI_DEV_PCIE_REPLAY_COUNT` | PCIe 重试次数 | 0 | > 0（硬件问题） |
| `DCGM_FI_DEV_NVLINK_CRC_FLIT_ERRORS` | NVLink 错误 | 0 | > 0（通信链路问题） |

**Prometheus 采集配置**：

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nvidia-dcgm
  namespace: nvidia-gpu-operator
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: dcgm-exporter
  endpoints:
  - port: metrics
    interval: 15s
```

**Grafana Dashboard**：推荐使用 NVIDIA 官方提供的 [DCGM Exporter Dashboard](https://grafana.com/grafana/dashboards/12239)，ID 是 `12239`。

### 1.3 GPU 显存管理

显存是训练任务最稀缺的资源。以下场景在实践中非常常见：

**场景：模型训练 OOM**

```bash
kubectl describe pod training-job
# Event: OOMKilled

# 排查步骤：
# 1. 查看训练日志
kubectl logs training-job --previous

# 2. 查看 GPU 显存使用历史（如果已经配置了 Prometheus + DCGM）
#    检查 DCGM_FI_DEV_FB_USED 是否接近显存上限

# 3. 常见原因：
#    - batch size 太大
#    - 模型太大
#    - 显存泄漏（PyTorch 的 cache 没释放）
#    - 数据加载时的临时显存分配
```

**显存优化建议**：

```yaml
# 在 Pod 上设置显存上限（需要 GPU Operator 支持）
apiVersion: v1
kind: Pod
metadata:
  name: training-job
  annotations:
    nvidia.com/mps-percentage: "50"   # 限制 GPU 使用 50% 的计算能力
spec:
  containers:
  - name: trainer
    image: pytorch:latest
    resources:
      requests:
        nvidia.com/gpu: 1
      limits:
        nvidia.com/gpu: 1
```

**实际建议**：在代码层面设置 PyTorch 的显存分配策略：

```python
# 训练代码开头设置
import torch

# 限制 PyTorch 的显存缓存
torch.cuda.set_per_process_memory_fraction(0.9)  # 最多使用 90% 显存

# 启用显存优化
torch.backends.cuda.matmul.allow_tf32 = True
torch.backends.cudnn.benchmark = True
```

## 2. 高性能网络

### 2.1 分布式训练的网络需求

多机多卡训练时，网络是最大的瓶颈之一。以 AllReduce 通信为例：

```
单机 8 卡（NVLink 互联）：
  GPU 之间通信延迟：< 1μs
  带宽：~600GB/s（NVLink 4.0）

多机 8 卡（普通以太网）：
  机器之间通信延迟：> 100μs
  带宽：~25GB/s（100GbE）
```

这个数量级的差距意味着：**分布式训练的网络设计，直接决定了训练效率**。

### 2.2 从以太网到 RDMA

**普通以太网（TCP/IP）的通信路径**：

```
应用层数据
  → TCP 协议栈
  → 内核网络协议栈
  → 网卡驱动
  → 物理网卡
  → 网络传输
  → 对端网卡
  → 对端驱动
  → 对端内核
  → 对端应用
```

这个路径上每一层都有 CPU 参与和内存拷贝，延迟很高。

**RDMA（Remote Direct Memory Access）的通信路径**：

```
应用层数据
  → RDMA 网卡（绕过内核，直接读写对端内存）
  → 网络传输
  → 对端 RDMA 网卡
  → 对端应用
```

**RDMA 的三种实现**：

| 方案 | 网络硬件 | 带宽 | 延迟 | 成本 |
|------|---------|------|------|------|
| InfiniBand | 专用 IB 交换机 + HCA 卡 | 400Gb/s+ | < 1μs | 高 |
| RoCEv2 | 普通以太网交换机（需支持 PFC/DCQCN） | 100-200Gb/s | 2-5μs | 中 |
| TCP/IP | 普通以太网 | 25-100Gb/s | 10-100μs | 低 |

**训练场景的选择建议**：

- 大规模分布式训练（64+ GPU）：InfiniBand 或 RoCEv2
- 中小规模训练（8-32 GPU）：RoCEv2 或高速以太网
- 单机多卡训练：NVLink 已经足够，不需要 RDMA

### 2.3 NCCL 与 Kubernetes 网络

NCCL（NVIDIA Collective Communications Library）是 NVIDIA 的集合通信库，PyTorch DDP 和 Horovod 都依赖它。

**NCCL 在 Kubernetes 中的通信路径**：

```
Pod A（GPU 0）                  Pod B（GPU 1）
  │                                │
  │  NCCL 通信                      │
  │    │                            │
  │    ├── 同节点：共享内存/NVLink  │
  │    │                            │
  │    └── 跨节点：RDMA/IB/RoCE    │
  │         │                      │
  │         ▼                      │
  │    ┌──────────┐          ┌──────────┐
  │    │ 网卡     │◄───IB────│ 网卡     │
  │    └──────────┘          └──────────┘
```

**Kubernetes 中 NCCL 的关键配置**：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: distributed-trainer
  annotations:
    # 告知 NCCL 使用哪块网卡进行通信
    k8s.v1.cni.cncf.io/networks: ib-net
spec:
  hostNetwork: true                  # 使用主机网络（绕过 CNI 的额外开销）
  containers:
  - name: trainer
    image: pytorch:latest
    env:
    # NCCL 环境变量
    - name: NCCL_IB_DISABLE
      value: "0"                     # 启用 InfiniBand
    - name: NCCL_IB_HCA
      value: "mlx5_0"               # 指定 InfiniBand 网卡
    - name: NCCL_SOCKET_IFNAME
      value: "eth0"                 # 控制面通信网卡
    - name: NCCL_DEBUG
      value: "INFO"                  # NCCL 调试日志
    - name: NCCL_IB_TIMEOUT
      value: "22"                   # IB 超时时间
    - name: NCCL_IB_RETRY_CNT
      value: "7"                    # 重试次数
    resources:
      requests:
        nvidia.com/gpu: 8
      limits:
        nvidia.com/gpu: 8
```

**NCCL 环境变量速查表**：

| 环境变量 | 作用 | 推荐值 |
|---------|------|--------|
| `NCCL_DEBUG` | 调试日志级别 | `INFO`（排查时），`WARN`（生产） |
| `NCCL_IB_DISABLE` | 是否禁用 InfiniBand | `0`（启用） |
| `NCCL_IB_HCA` | 指定 InfiniBand 网卡 | `mlx5_0` 或 `^mlx5_1,mlx5_2` |
| `NCCL_SOCKET_IFNAME` | 控制面通信网卡 | `eth0` |
| `NCCL_IB_TIMEOUT` | InfiniBand 超时 | `22`（约 4 秒） |
| `NCCL_IB_RETRY_CNT` | 重试次数 | `7` |
| `NCCL_IB_GID_INDEX` | GID 索引 | `3`（RoCEv2 时） |
| `NCCL_TOPO_FILE` | 自定义拓扑文件 | 需要时指定 |
| `NCCL_ALGO` | 通信算法 | `Ring` 或 `Tree` |
| `NCCL_PROTO` | 通信协议 | `Simple` 或 `LL` |

### 2.4 网络拓扑感知调度

在 8 节点 × 8 GPU 的集群中，如果调度器把 8 个 worker 分散到 8 个节点，但如果它们使用的网卡不在同一个交换机下，NCCL 通信效率会大幅下降。

**Kubernetes 的网络拓扑感知**依赖节点标签：

```bash
# 标记节点所在的网络拓扑
kubectl label node node-1 topology.kubernetes.io/rack=rack-a
kubectl label node node-2 topology.kubernetes.io/rack=rack-a
kubectl label node node-3 topology.kubernetes.io/rack=rack-b
kubectl label node node-4 topology.kubernetes.io/rack=rack-b
```

```yaml
# 训练任务：尽量调度到同一个机架
apiVersion: v1
kind: Pod
spec:
  affinity:
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            job: distributed-training
        topologyKey: topology.kubernetes.io/rack
```

## 3. CPU 与内存的细粒度管理

### 3.1 CPU Manager

默认情况下，Kubernetes 的 CPU 分配是"共享"的——一个 Pod 的 CPU 时间片可能来自不同的物理核心。这对训练任务来说不是好事，因为 CPU 缓存亲和性会降低性能。

**CPU Manager 的 `static` 策略**：保证 Pod 可以独占一组 CPU 核心。

```yaml
# kubelet 配置
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cpuManagerPolicy: static           # 启用 CPU Manager 静态策略
cpuManagerReconcilePeriod: 5s
# 预留系统组件使用的 CPU
reservedSystemCPUs: "0-1"          # 预留 2 个核心给系统
```

**启用后，Guaranteed QoS 的 Pod 会独占 CPU**：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: training-pod
spec:
  containers:
  - name: trainer
    image: pytorch:latest
    resources:
      requests:
        cpu: "4"                   # 整数 CPU 请求
        memory: "16Gi"
      limits:
        cpu: "4"                   # requests = limits 才能触发独占
        memory: "16Gi"
```

**CPU Manager 的影响**：

```
没有 CPU Manager：
  Pod 的 CPU 时间片可能来自 4 个不同的物理核心
  L1/L2 缓存频繁失效
  性能损耗：5-15%

有 CPU Manager（static）：
  Pod 独占 4 个物理核心
  L1/L2 缓存命中率大幅提升
  性能更稳定
```

### 3.2 Topology Manager

CPU Manager 管 CPU 亲和性，但训练任务还关心 GPU 和 CPU 之间的拓扑关系。

```
典型硬件拓扑：
  ┌──────────────────────────────┐
  │  NUMA Node 0                 │
  │  ┌──────┐  ┌──────┐         │
  │  │ CPU  │  │ GPU 0├───┐     │
  │  │ 0-15 │  │      │   │     │
  │  └──────┘  └──────┘   │     │
  │                 │     │     │
  │  ┌──────┐  ┌──────┐   │     │
  │  │ CPU  │  │ GPU 1├───┘     │
  │  │ 0-15 │  │      │         │
  │  └──────┘  └──────┘         │
  ├──────────────────────────────┤
  │  NUMA Node 1                 │
  │  ┌──────┐  ┌──────┐         │
  │  │ CPU  │  │ GPU 2├───┐     │
  │  │16-31 │  │      │   │     │
  │  └──────┘  └──────┘   │     │
  │                 │     │     │
  │  ┌──────┐  ┌──────┐   │     │
  │  │ CPU  │  │ GPU 3├───┘     │
  │  │16-31 │  │      │         │
  │  └──────┘  └──────┘         │
  └──────────────────────────────┘
```

如果 Pod 分配到 NUMA Node 0 的 CPU 和 NUMA Node 1 的 GPU，跨 NUMA 访问会导致性能下降（10-30%）。

**Topology Manager 解决这个问题**：

```yaml
# kubelet 配置
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
topologyManagerPolicy: single-numa-node   # 强制所有资源在同一 NUMA 节点
```

**Topology Manager 的策略**：

| 策略 | 行为 | 适用场景 |
|------|------|----------|
| `none` | 不限制 | 默认 |
| `best-effort` | 尽量拓扑对齐，不对齐也行 | 性能敏感但可用性优先 |
| `restricted` | 必须对齐，不对齐就拒绝 | 高性能训练 |
| `single-numa-node` | 所有资源必须在同一 NUMA 节点 | 对延迟最敏感的场景 |

### 3.3 HugePages

训练任务（尤其是推理）可能大量使用大页内存来加速模型加载。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: inference-pod
spec:
  containers:
  - name: triton
    image: nvcr.io/nvidia/tritonserver:latest
    resources:
      requests:
        memory: "32Gi"
        hugepages-2Mi: "2Gi"       # 请求 2Gi 的 2MB 大页
      limits:
        memory: "32Gi"
        hugepages-2Mi: "2Gi"
    volumeMounts:
    - name: hugepage
      mountPath: /dev/hugepages
  volumes:
  - name: hugepage
    emptyDir:
      medium: HugePages
```

## 4. 训练任务调度最佳实践

### 4.1 训练任务的多级调度策略

一个生产级 AI 训练集群的调度策略通常是多层的：

```
第 1 层：Node Level（节点级）
  污点 + 标签 + nodeSelector
  → 确保 GPU 节点只给 GPU 任务用
  → 不同类型 GPU 用标签区分

第 2 层：Queue Level（队列级）
  Volcano Queue + ResourceQuota
  → 不同团队/项目分配不同资源配额
  → 队列权重控制资源分配比例

第 3 层：Job Level（任务级）
  PriorityClass + Gang Scheduling
  → 高优任务插队
  → 分布式训练任务同时调度

第 4 层：Cluster Level（集群级）
  Cluster Autoscaler + Node Pool
  → 按需自动扩缩 GPU 节点
  → 缩容时保护运行中的训练任务
```

### 4.2 完整的训练调度配置示例

```yaml
# 1. 节点配置
# GPU 节点打标签和污点
# kubectl label node gpu-node-1 accelerator=nvidia gpu-type=a100
# kubectl taint node gpu-node-1 nvidia.com/gpu=present:NoSchedule

# 2. 优先级
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: production-training
value: 10000
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: experimental-training
value: 1000

# 3. 队列（Volcano）
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: production
spec:
  weight: 2
  capability:
    nvidia.com/gpu: 32
---
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: research
spec:
  weight: 1
  capability:
    nvidia.com/gpu: 16

# 4. 训练任务
apiVersion: batch.volcano.sh/v1alpha1
kind: Job
metadata:
  name: gpt-training
spec:
  minAvailable: 4
  queue: production
  priorityClassName: production-training
  tasks:
  - name: worker
    replicas: 4
    template:
      spec:
        nodeSelector:
          gpu-type: a100
        tolerations:
        - key: nvidia.com/gpu
          operator: Exists
          effect: NoSchedule
        containers:
        - name: trainer
          image: pytorch:latest
          resources:
            requests:
              nvidia.com/gpu: 8
              cpu: "64"
              memory: "256Gi"
            limits:
              nvidia.com/gpu: 8
              cpu: "64"
              memory: "256Gi"
          env:
          - name: NCCL_DEBUG
            value: "WARN"
          - name: NCCL_IB_HCA
            value: "mlx5_0"
```

### 4.3 Checkpoint 管理

训练任务跑几天后突然挂了，如果没有 checkpoint，一切从头开始。

**Kubernetes 上的 checkpoint 策略**：

```yaml
# 方案 1：使用 PVC 持久化 checkpoint
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: training-checkpoint
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 500Gi
  storageClassName: nfs-client
---
apiVersion: v1
kind: Pod
metadata:
  name: training-job
spec:
  containers:
  - name: trainer
    image: pytorch:latest
    volumeMounts:
    - name: checkpoint
      mountPath: /checkpoint
  volumes:
  - name: checkpoint
    persistentVolumeClaim:
      claimName: training-checkpoint
```

**推荐做法**：

- Checkpoint 放到 ReadWriteMany 的 PVC 上（NFS、Lustre、JuiceFS）
- 每 N 个 epoch 保存一次 checkpoint
- 保留最近 3-5 个 checkpoint，删除旧的
- 训练脚本启动时自动检测 checkpoint 并恢复

### 4.4 推理部署的特殊考虑

推理和训练不同，推理关注的是**延迟**和**吞吐**，而不是显存利用率。

**推理服务的典型配置**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: triton-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: triton
  template:
    metadata:
      labels:
        app: triton
    spec:
      containers:
      - name: triton
        image: nvcr.io/nvidia/tritonserver:23.10-py3
        args:
        - tritonserver
        - --model-repository=/models
        - --model-control-mode=poll
        args:
        - tritonserver
        - --model-repository=/models
        - --model-control-mode=poll
        ports:
        - containerPort: 8000
          name: http
        - containerPort: 8001
          name: grpc
        - containerPort: 8002
          name: metrics
        resources:
          requests:
            nvidia.com/gpu: 1
            cpu: "4"
            memory: "16Gi"
          limits:
            nvidia.com/gpu: 1
            cpu: "4"
            memory: "16Gi"
        env:
        - name: CUDA_VISIBLE_DEVICES
          value: "0"
        readinessProbe:
          httpGet:
            path: /v2/health/ready
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /v2/health/live
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 15
```

**推理 vs 训练的调度差异**：

| 对比维度 | 训练 | 推理 |
|---------|------|------|
| GPU 需求 | 多卡，长时间 | 单卡或少卡，持续运行 |
| 调度策略 | Gang Scheduling | 普通 Deployment |
| 网络需求 | RDMA/InfiniBand | 普通以太网 |
| 中断容忍 | 低（需要 checkpoint） | 低（需要 PDB）|
| 资源需求 | 稳定，可预测 | 波峰波谷明显 |
| 扩缩容 | 手动或按队列 | HPA 自动扩缩 |
| 监控重点 | 显存、GPU 利用率 | 延迟、吞吐、QPS |

## 5. 训练集群规划建议

### 5.1 节点规划

| 节点类型 | 配置建议 | 用途 |
|---------|---------|------|
| GPU 计算节点 | 8×A100 80GB + 512GB 内存 + 4×100GbE/IB | 训练 |
| CPU 节点 | 64 核 + 256GB 内存 | 数据预处理、控制面 |
| 存储节点 | 大容量 SSD/NVMe + 网络存储 | Checkpoint、数据集 |
| 管理节点 | 16 核 + 64GB 内存 | Kubernetes 控制面 |

### 5.2 网络规划

```
                        ┌─────────────┐
                        │ 管理网络    │
                        │ (1GbE)      │
                        └──────┬──────┘
                               │
┌──────────────┐      ┌───────┴───────┐      ┌──────────────┐
│  存储网络    │──────│  GPU 计算节点  │──────│  训练网络    │
│  (25/100GbE) │      │  8×A100       │      │  (IB/RoCEv2) │
└──────────────┘      └───────────────┘      └──────────────┘
                               │
                        ┌──────┴──────┐
                        │  推理网络    │
                        │  (HTTP/gRPC) │
                        └─────────────┘
```

**推荐网络隔离**：

- **管理网络**：kubelet、API Server 通信
- **训练网络**：NCCL 集合通信（RDMA/InfiniBand）
- **存储网络**：数据集、checkpoint 读写
- **推理网络**：对外服务的 HTTP/gRPC 流量

### 5.3 存储规划

| 数据类型 | 存储方案 | 性能要求 | 容量 |
|---------|---------|---------|------|
| 数据集 | NFS/Lustre/JuiceFS | 高吞吐 | TB 级 |
| Checkpoint | RWX PVC | 中等 | 几百 GB |
| 模型仓库 | 对象存储 + 缓存 | 高 IOPS | 几十 GB |
| 日志 | 对象存储 | 低 | 取决于尺寸 |

## 6. 常见问题排查

### 6.1 NCCL 通信超时

```bash
# 现象：训练日志中频繁出现 NCCL timeout
# 日志：NCCL WARN NET/IB : Got completion with error 12

# 排查步骤
# 1. 检查 NCCL 网卡选择
kubectl exec training-pod -- env | grep NCCL_IB_HCA

# 2. 检查 IB 链路
kubectl exec training-pod -- ibstatus

# 3. 检查 NCCL 拓扑
kubectl exec training-pod -- nvidia-smi topo -m

# 4. 常见原因
#    - NCCL_IB_HCA 配置错误，NCCL 用了错误的网卡
#    - 不同节点之间的 IB 链路不通
#    - IB 交换机端口配置问题
#    - NCCL 版本和驱动版本不匹配
```

### 6.2 GPU 显存泄漏

```bash
# 现象：训练过程中显存使用持续增长，最终 OOM

# 排查步骤
# 1. 监控显存使用趋势
kubectl exec training-pod -- nvidia-smi --query-gpu=memory.used --format=csv -l 5

# 2. 检查 PyTorch 显存缓存
kubectl exec training-pod -- python -c "
import torch
print(torch.cuda.memory_summary())
"

# 3. 常见原因
#    - PyTorch 的显存缓存没有释放（默认会缓存）
#    - DataLoader 的 num_workers 太多
#    - 没有调用 torch.cuda.empty_cache()
#    - 模型的 forward 中创建了临时张量没有释放
```

### 6.3 训练速度波动

```bash
# 现象：同一个模型，同一个配置，某些 epoch 训练速度明显变慢

# 排查步骤
# 1. 检查是否有其他 Pod 共享了 GPU 节点
kubectl get pods --field-selector spec.nodeName=<gpu-node>

# 2. 检查 GPU 温度是否过高（降频）
kubectl exec training-pod -- nvidia-smi --query-gpu=temperature.gpu --format=csv

# 3. 检查 NCCL 通信是否有异常重试
kubectl exec training-pod -- env NCCL_DEBUG=WARN

# 4. 检查节点 CPU 是否被打满（CPU 竞争影响数据加载）
kubectl top node <gpu-node>

# 5. 常见原因
#    - GPU 温度过高触发降频（> 85°C）
#    - 数据加载速度跟不上 GPU 计算（IO 瓶颈）
#    - 其他 Pod 共享 GPU 节点，CPU 资源竞争
#    - NCCL 通信链路问题（重试导致延迟）
```

### 6.4 推理服务延迟抖动

```bash
# 现象：推理服务的 P99 延迟突然升高

# 排查步骤
# 1. 检查 GPU 是否被其他任务抢占
kubectl describe pod inference-server

# 2. 检查模型是否触发了重新加载
kubectl logs inference-server | grep -i "loading\|reload"

# 3. 检查是否触发了 GPU 降频
kubectl exec inference-server -- nvidia-smi -q -d TEMPERATURE

# 4. 常见原因
#    - GPU 时间片被其他 Pod 抢占（没有独占 GPU）
#    - Triton/TorchServe 的模型动态加载导致 CPU 突增
#    - 推理请求突增导致 GPU 计算排队
#    - 磁盘 IO 瓶颈（日志写太多）
```

## 7. 总结

AI Infra 在 Kubernetes 上的实践，核心是解决四个问题：

1. **GPU 资源管理**：Device Plugin → GPU Operator → MIG/Time-Slicing → 监控
2. **高性能网络**：以太网 → RDMA → InfiniBand → NCCL 调优
3. **细粒度调度**：CPU Manager → Topology Manager → NUMA → HugePages
4. **训练任务调度**：PriorityClass → Gang Scheduling → Queue → Checkpoint

## 下一步

- [Kubeflow 官方文档](https://www.kubeflow.org/)
- [Volcano 官方文档](https://volcano.sh/)
- [NVIDIA GPU Operator 文档](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/)
