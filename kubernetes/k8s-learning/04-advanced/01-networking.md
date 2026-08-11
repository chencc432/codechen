# 🌐 Kubernetes 网络模型

## 网络基础

### Kubernetes 网络要求

Kubernetes 对网络有以下基本要求：

1. **Pod 间通信**：所有 Pod 可以互相通信，无需 NAT
2. **Node 与 Pod 通信**：所有节点可以与所有 Pod 通信
3. **Pod 看到的 IP**：Pod 看到自己的 IP 和其他人看到的一致

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Kubernetes 网络模型                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │                     外部网络                                  │  │
│   └───────────────────────────┬─────────────────────────────────┘  │
│                               │                                      │
│                               ▼                                      │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │            Ingress / LoadBalancer / NodePort                 │  │
│   └───────────────────────────┬─────────────────────────────────┘  │
│                               │                                      │
│                               ▼                                      │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │                    Service (ClusterIP)                       │  │
│   │               kube-proxy (iptables/IPVS)                    │  │
│   └───────────────────────────┬─────────────────────────────────┘  │
│                               │                                      │
│                               ▼                                      │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │                      Pod 网络                                │  │
│   │                 CNI (Calico/Flannel/...)                    │  │
│   │                                                              │  │
│   │  ┌─────────┐     ┌─────────┐     ┌─────────┐              │  │
│   │  │ Pod A   │ ←──→│ Pod B   │ ←──→│ Pod C   │              │  │
│   │  │10.1.1.10│     │10.1.2.20│     │10.1.3.30│              │  │
│   │  └─────────┘     └─────────┘     └─────────┘              │  │
│   │                                                              │  │
│   └─────────────────────────────────────────────────────────────┘  │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## 网络层次

### 1. 容器到容器（同 Pod）

同一 Pod 内的容器共享网络命名空间，通过 `localhost` 通信。下面 YAML 里的 `sidecar` 只是占位名，表示「同 Pod 里的辅助容器」；Sidecar 是多容器模式，不是单独的资源类型，说明见 [Pod - Sidecar 模式](../02-resources/01-pod.md#sidecar-模式边车)。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-container
spec:
  containers:
  - name: app
    image: myapp
    ports:
    - containerPort: 8080
  - name: log-collector          # 示例：边车容器名
    image: fluent/fluent-bit     # 换成真实镜像；没有官方镜像叫 sidecar
    # 可以通过 localhost:8080 访问 app
```

### 2. Pod 到 Pod（同节点/跨节点）

由 CNI 插件负责实现，所有 Pod 都在一个扁平的网络空间中。

```bash
# 测试 Pod 间通信
kubectl run test --image=busybox -it --rm -- ping <other-pod-ip>
```

### 3. Pod 到 Service

通过 kube-proxy 实现，将 Service ClusterIP 转换为后端 Pod IP。

### 4. 外部到 Service

通过 NodePort、LoadBalancer 或 Ingress 实现。

## CNI 插件

### 常见 CNI 插件对比

| 插件 | 模式 | 特点 |
|------|------|------|
| Calico | BGP/IPIP | 高性能、支持网络策略 |
| Flannel | VXLAN/host-gw | 简单、轻量 |
| Weave | Mesh overlay | 加密、简单 |
| Cilium | eBPF | 高性能、可观测性 |

### 查看 CNI 配置

```bash
# 查看 CNI 配置
cat /etc/cni/net.d/*.conf

# 查看 Pod 网络
kubectl get pods -o wide
```

## Service 网络

### Service 类型

```yaml
# ClusterIP (默认)
spec:
  type: ClusterIP
  clusterIP: 10.96.0.100  # 可选，自动分配

# NodePort
spec:
  type: NodePort
  ports:
  - port: 80
    nodePort: 30080  # 30000-32767

# LoadBalancer
spec:
  type: LoadBalancer
  loadBalancerIP: 203.0.113.10  # 可选

# ExternalName
spec:
  type: ExternalName
  externalName: db.example.com
```

### kube-proxy 模式

```bash
# 查看 kube-proxy 模式
kubectl get configmap kube-proxy -n kube-system -o yaml | grep mode

# iptables 模式（默认）
# IPVS 模式（高性能）
```

## 网络策略 (NetworkPolicy)

### 默认策略

```yaml
# 默认拒绝所有入站流量
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
spec:
  podSelector: {}
  policyTypes:
  - Ingress

# 默认允许所有入站流量
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-all-ingress
spec:
  podSelector: {}
  ingress:
  - {}
  policyTypes:
  - Ingress
```

### 复杂策略示例

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: app-network-policy
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: web
  policyTypes:
  - Ingress
  - Egress
  
  ingress:
  # 允许来自同命名空间 app=api 的流量
  - from:
    - podSelector:
        matchLabels:
          app: api
    ports:
    - protocol: TCP
      port: 80
  
  # 允许来自特定命名空间的流量
  - from:
    - namespaceSelector:
        matchLabels:
          environment: staging
    ports:
    - protocol: TCP
      port: 80
  
  egress:
  # 允许访问数据库
  - to:
    - podSelector:
        matchLabels:
          app: database
    ports:
    - protocol: TCP
      port: 5432
  
  # 允许 DNS 查询
  - to:
    - namespaceSelector: {}
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
```

## DNS 服务

### CoreDNS

```bash
# 查看 CoreDNS
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 查看 CoreDNS 配置
kubectl get configmap coredns -n kube-system -o yaml
```

### DNS 解析规则

```
# Service DNS
<service>.<namespace>.svc.cluster.local

# Pod DNS (如果启用)
<pod-ip-dashed>.<namespace>.pod.cluster.local

# Headless Service
<pod-name>.<service>.<namespace>.svc.cluster.local
```

### 测试 DNS

```bash
# 测试 DNS 解析
kubectl run test --image=busybox -it --rm -- nslookup kubernetes.default

# 完整域名测试
kubectl run test --image=busybox -it --rm -- nslookup my-service.my-namespace.svc.cluster.local
```

## 网络排查

```bash
# 检查 Pod 网络
kubectl get pods -o wide

# 测试连通性
kubectl run test --image=nicolaka/netshoot -it --rm -- bash
# 然后使用 ping, curl, dig, tcpdump 等工具

# 检查 Service 端点
kubectl get endpoints <service-name>

# 查看 iptables 规则
sudo iptables -t nat -L -n | grep <service-ip>
```

## 下一步

- [存储系统详解](./02-storage.md)

## 14. 高性能网络：RDMA 与 InfiniBand

前面的章节主要覆盖了 Kubernetes 的基础网络模型和 Service 通信。但 AI 训练场景对网络有更高的要求，这里单独展开。

### 14.1 为什么训练需要高性能网络

分布式训练的核心是**集合通信（Collective Communication）**：每个 GPU 计算完梯度后，需要把所有 GPU 的梯度汇总起来。

```
AllReduce 通信模式：

Worker 0: 梯度 A  ──────┐
Worker 1: 梯度 B  ──────┤
Worker 2: 梯度 C  ──────┼──→ 汇总 → 平均 → 分发 → 所有 Worker 拿到相同梯度
Worker 3: 梯度 D  ──────┘

通信量 = 模型参数 × 每个参数的字节数 × 通信次数
一个 7B 参数的模型，FP32 下每次 AllReduce 传输约 28GB 数据
```

如果网络是瓶颈，GPU 就在空等数据。**网络越慢，GPU 利用率越低**。

### 14.2 NCCL 的通信模式

NCCL 会根据 GPU 之间的拓扑自动选择最优通信路径：

```
同节点内（GPU 0 ↔ GPU 1）：
  ┌─────────────────────────────────────┐
  │  GPU 0 ──── NVLink ──── GPU 1      │
  │  GPU 2 ──── NVLink ──── GPU 3      │
  │  GPU 4 ──── NVLink ──── GPU 5      │
  │  GPU 6 ──── NVLink ──── GPU 7      │
  └─────────────────────────────────────┘
  通信路径：NVLink（~600GB/s）

跨节点（Node A 的 GPU 0 ↔ Node B 的 GPU 0）：
  Node A                     Node B
  ┌──────────┐              ┌──────────┐
  │ GPU 0    │              │ GPU 0    │
  │   │      │              │   │      │
  │ PCIe    │              │ PCIe    │
  │   │      │              │   │      │
  │ IB/RoCE │─────网络─────│ IB/RoCE │
  │  网卡   │              │  网卡   │
  └──────────┘              └──────────┘
  通信路径：PCIe → 网卡 → 网络 → 网卡 → PCIe（~200Gb/s 如果 IB）
```

### 14.3 Kubernetes 中配置 RDMA

**使用 HostNetwork（最简单）**：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: training-job
spec:
  hostNetwork: true              # 直接使用主机网络栈
  nodeSelector:
    accelerator: nvidia
  tolerations:
  - key: nvidia.com/gpu
    operator: Exists
    effect: NoSchedule
  containers:
  - name: trainer
    image: pytorch:latest
    env:
    - name: NCCL_IB_HCA
      value: "mlx5_0"
    - name: NCCL_SOCKET_IFNAME
      value: "eth0"
    - name: OMPI_MCA_btl
      value: "^openib"
    resources:
      requests:
        nvidia.com/gpu: 8
      limits:
        nvidia.com/gpu: 8
```

**使用 Multus CNI（多网卡，生产推荐）**：

[Multus](https://github.com/k8snetworkplumbingwg/multus-cni) 允许 Pod 同时挂载多个网络接口——一个用于管理/控制面通信，另一个用于 RDMA 数据面。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: training-job
  annotations:
    k8s.v1.cni.cncf.io/networks: rdma-net   # 附加 RDMA 网卡
spec:
  containers:
  - name: trainer
    image: pytorch:latest
    env:
    - name: NCCL_IB_HCA
      value: "mlx5_0,mlx5_1"   # 指定 RDMA 网卡
```

### 14.4 高性能网络监控

**关键指标**：

| 指标 | 工具/命令 | 说明 |
|------|----------|------|
| IB 端口状态 | `ibstat` | 检查 IB 链路是否正常 |
| IB 吞吐 | `ib_read_bw` | 测量 IB 读写带宽 |
| 网络延迟 | `ibping` | 测量 IB 延迟 |
| NCCL 通信效率 | `nsys profile` | 分析 NCCL 通信耗时 |
| 网卡丢包 | `ethtool -S <iface>` | 检查丢包和重传 |

### 14.5 常见问题

```
问题：NCCL 通信超时
可能原因：
  - NCCL_IB_HCA 配置错误
  - IB 链路故障（光模块、线缆）
  - 不同节点 IB 固件版本不一致
  - 交换机端口配置问题

问题：NCCL 未使用 IB（回退到 TCP）
可能原因：
  - NCCL_IB_DISABLE 被设置为 1
  - IB 网卡未被 NCCL 识别
  - 缺少 libibverbs

问题：跨节点通信性能差
可能原因：
  - 节点间 IB 交换机带宽不足
  - 网络拓扑不是 fat-tree
  - 多链路没有做负载均衡
```
