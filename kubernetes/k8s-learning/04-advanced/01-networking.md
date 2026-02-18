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

同一 Pod 内的容器共享网络命名空间，通过 `localhost` 通信。

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
  - name: sidecar
    image: sidecar
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



