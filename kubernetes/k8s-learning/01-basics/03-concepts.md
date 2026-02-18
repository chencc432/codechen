# 📚 Kubernetes 核心概念与术语

## 概念体系总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Kubernetes 概念体系                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   工作负载                    服务发现                 配置与存储       │
│   ┌─────────┐                ┌─────────┐            ┌─────────┐     │
│   │   Pod   │                │ Service │            │ConfigMap│     │
│   └────┬────┘                └────┬────┘            └─────────┘     │
│        │                          │                 ┌─────────┐     │
│   ┌────┴────┐                ┌────┴────┐            │ Secret  │     │
│   │ Deploy- │                │ Ingress │            └─────────┘     │
│   │  ment   │                └─────────┘            ┌─────────┐     │
│   └────┬────┘                ┌─────────┐            │ Volume  │     │
│        │                     │Endpoint │            └─────────┘     │
│   ┌────┴────┐                └─────────┘                            │
│   │Replica- │                                                        │
│   │   Set   │                                                        │
│   └─────────┘                                                        │
│                                                                       │
│   集群管理                    调度控制                 安全与权限       │
│   ┌─────────┐                ┌─────────┐            ┌─────────┐     │
│   │  Node   │                │ Taint   │            │  RBAC   │     │
│   └─────────┘                │Tolerate │            └─────────┘     │
│   ┌─────────┐                └─────────┘            ┌─────────┐     │
│   │Namespace│                ┌─────────┐            │ Service │     │
│   └─────────┘                │Affinity │            │ Account │     │
│                              └─────────┘            └─────────┘     │
└─────────────────────────────────────────────────────────────────────┘
```

## 1. 对象模型

### 1.1 什么是 Kubernetes 对象？

Kubernetes 对象是 Kubernetes 系统中的持久化实体。Kubernetes 使用这些对象来表示集群的状态：

- 哪些容器化应用正在运行
- 这些应用使用什么资源
- 关于应用行为的策略

### 1.2 对象规约（Spec）与状态（Status）

每个 Kubernetes 对象都包含两个核心字段：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
spec:                    # 规约 - 期望状态（你定义的）
  containers:
  - name: nginx
    image: nginx:1.21
status:                  # 状态 - 当前状态（系统维护的）
  phase: Running
  podIP: 10.244.1.5
  conditions:
  - type: Ready
    status: "True"
```

### 1.3 对象标识

每个对象都有唯一标识：

| 标识 | 说明 | 示例 |
|------|------|------|
| Name | 同一命名空间内唯一 | `nginx-deployment` |
| UID | 整个集群唯一 | `a1b2c3d4-e5f6-...` |
| Namespace | 资源所属的命名空间 | `default`, `kube-system` |

## 2. 核心概念详解

### 2.1 Label（标签）

标签是附加到对象的键值对，用于组织和选择资源。

```yaml
# 添加标签
metadata:
  labels:
    app: nginx              # 应用名称
    environment: production # 环境
    tier: frontend          # 层级
    version: v1.0.0         # 版本
```

#### 标签选择器

```yaml
# 等值选择器
selector:
  matchLabels:
    app: nginx

# 集合选择器
selector:
  matchExpressions:
  - key: environment
    operator: In
    values: ["production", "staging"]
  - key: tier
    operator: NotIn
    values: ["backend"]
```

#### kubectl 使用标签

```bash
# 按标签筛选
kubectl get pods -l app=nginx
kubectl get pods -l 'environment in (production, staging)'
kubectl get pods -l app=nginx,tier=frontend

# 添加/修改标签
kubectl label pods nginx-pod version=v2

# 删除标签
kubectl label pods nginx-pod version-

# 查看所有标签
kubectl get pods --show-labels
```


### 2.2 Annotation（注解）

注解用于存储非标识性的元数据，通常是给工具或库使用。

```yaml
metadata:
  annotations:
    description: "This is the main nginx server"
    kubernetes.io/created-by: "deployment-controller"
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    imageregistry: "https://hub.docker.com/"
```

#### Label vs Annotation

| 特性 | Label | Annotation |
|------|-------|------------|
| 用途 | 标识和选择 | 存储元数据 |
| 选择器 | 支持 | 不支持 |
| 长度限制 | 较严格 | 较宽松 |
| 典型用例 | 分组、筛选 | 配置、描述 |

### 2.3 Namespace（命名空间）

命名空间用于在集群中创建虚拟的隔离环境。

```bash
# 默认命名空间
- default         # 默认命名空间，用户资源默认在这里
- kube-system     # Kubernetes 系统组件
- kube-public     # 公开资源，所有用户可读
- kube-node-lease # 节点心跳数据
```

#### 命名空间操作

```bash
# 查看命名空间
kubectl get namespaces
kubectl get ns

# 创建命名空间
kubectl create namespace dev
kubectl create ns staging

# 在特定命名空间操作
kubectl get pods -n kube-system
kubectl apply -f deployment.yaml -n dev

# 设置默认命名空间
kubectl config set-context --current --namespace=dev

# 查看当前默认命名空间
kubectl config view --minify | grep namespace

# 删除命名空间（会删除其中所有资源！）
kubectl delete namespace dev
```

#### 命名空间 YAML

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: development
  labels:
    environment: development
```

### 2.4 Selector（选择器）

选择器用于选择具有特定标签的资源。

```yaml
# Service 选择 Pod
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx        # 选择 app=nginx 的 Pod
  ports:
  - port: 80

---
# Deployment 选择 Pod（模板）
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  selector:
    matchLabels:
      app: nginx      # 必须与 template.labels 匹配
  template:
    metadata:
      labels:
        app: nginx    # Pod 的标签
```

## 3. 资源管理概念

### 3.1 资源请求与限制

```yaml
spec:
  containers:
  - name: app
    image: nginx
    resources:
      requests:       # 调度时保证的最小资源
        cpu: "250m"   # 0.25 CPU 核心
        memory: "128Mi"
      limits:         # 最大可用资源
        cpu: "500m"
        memory: "256Mi"
```

#### CPU 单位

```
1 CPU = 1000m (毫核)
0.5 CPU = 500m
0.1 CPU = 100m
```

#### 内存单位

```
Ki = 1024
Mi = 1024 Ki
Gi = 1024 Mi

K = 1000
M = 1000 K
G = 1000 M
```

### 3.2 QoS 类别

根据资源配置，Pod 被分为不同 QoS 类别：

| QoS 类别 | 条件 | 驱逐优先级 |
|----------|------|-----------|
| Guaranteed | 所有容器都设置了 requests = limits | 最低（最后被驱逐）|
| Burstable | 至少一个容器设置了 requests | 中等 |
| BestEffort | 没有设置任何资源限制 | 最高（最先被驱逐）|

## 4. 调度相关概念

### 4.1 节点选择器（nodeSelector）

```yaml
spec:
  nodeSelector:
    disktype: ssd         # 只调度到有此标签的节点
    gpu: nvidia-tesla-v100
```

### 4.2 亲和性（Affinity）

#### 节点亲和性

```yaml
spec:
  affinity:
    nodeAffinity:
      # 硬性要求
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/os
            operator: In
            values: ["linux"]
      # 软性偏好
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        preference:
          matchExpressions:
          - key: zone
            operator: In
            values: ["zone-a"]
```

#### Pod 亲和性/反亲和性

```yaml
spec:
  affinity:
    # Pod 亲和性 - 与某些 Pod 调度在一起
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app: cache
        topologyKey: kubernetes.io/hostname
    
    # Pod 反亲和性 - 与某些 Pod 分开调度
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: web
          topologyKey: kubernetes.io/hostname
```

### 4.3 污点与容忍度（Taint & Toleration）

#### 污点（Taint）- 在节点上设置

```bash
# 添加污点
kubectl taint nodes node1 key=value:NoSchedule

# 污点效果
NoSchedule      # 不调度新 Pod
PreferNoSchedule # 尽量不调度
NoExecute       # 不调度且驱逐现有 Pod

# 删除污点
kubectl taint nodes node1 key:NoSchedule-
```

#### 容忍度（Toleration）- 在 Pod 上设置

```yaml
spec:
  tolerations:
  - key: "key"
    operator: "Equal"
    value: "value"
    effect: "NoSchedule"
  
  # 容忍所有污点
  - operator: "Exists"
```

## 5. 生命周期概念

### 5.1 Pod 生命周期

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌──────────┐
│ Pending │───>│ Running │───>│Succeeded│ or │  Failed  │
└─────────┘    └─────────┘    └─────────┘    └──────────┘
     │              │
     │              ▼
     │         ┌─────────┐
     └────────>│ Unknown │
               └─────────┘

Pending:   等待调度或拉取镜像
Running:   至少一个容器运行中
Succeeded: 所有容器成功终止
Failed:    至少一个容器失败终止
Unknown:   无法获取 Pod 状态
```

### 5.2 容器状态

```yaml
Waiting:     等待启动（拉取镜像、等待依赖）
Running:     正在运行
Terminated:  已终止（正常退出或出错）
```

### 5.3 重启策略

```yaml
spec:
  restartPolicy: Always    # 总是重启（默认，用于 Deployment）
  restartPolicy: OnFailure # 失败时重启（用于 Job）
  restartPolicy: Never     # 从不重启
```

### 5.4 Pod 条件（Conditions）

```yaml
status:
  conditions:
  - type: PodScheduled     # 已调度
    status: "True"
  - type: Initialized      # Init 容器已完成
    status: "True"
  - type: ContainersReady  # 所有容器就绪
    status: "True"
  - type: Ready            # Pod 就绪，可接收流量
    status: "True"
```

## 6. 服务发现概念

### 6.1 Service 类型

```yaml
ClusterIP (默认):
  - 只在集群内部可访问
  - 分配虚拟 IP

NodePort:
  - 在每个节点上开放端口
  - 端口范围: 30000-32767

LoadBalancer:
  - 使用云提供商的负载均衡器
  - 自动分配外部 IP

ExternalName:
  - DNS CNAME 记录
  - 指向外部服务
```

### 6.2 Endpoint

Endpoint 是 Service 和 Pod 之间的桥梁：

```bash
# 查看 Service 的 Endpoints
kubectl get endpoints nginx-service

# 输出示例
NAME            ENDPOINTS                         AGE
nginx-service   10.244.1.5:80,10.244.2.6:80      5m
```

### 6.3 DNS 解析

```bash
# 在集群内，Service 可通过 DNS 访问
<service-name>                          # 同命名空间
<service-name>.<namespace>              # 跨命名空间
<service-name>.<namespace>.svc.cluster.local  # 完整域名

# 示例
curl nginx-service                      # 同命名空间
curl nginx-service.production           # 访问 production 命名空间的服务
```

## 7. 常用术语对照表

| 术语 | 中文 | 说明 |
|------|------|------|
| Cluster | 集群 | Kubernetes 管理的一组节点 |
| Node | 节点 | 集群中的一台机器 |
| Pod | 容器组 | 最小的部署单元 |
| Container | 容器 | Pod 中运行的应用实例 |
| Deployment | 部署 | 无状态应用的部署管理 |
| Service | 服务 | 访问 Pod 的稳定端点 |
| Namespace | 命名空间 | 资源隔离 |
| Label | 标签 | 资源分类标识 |
| Selector | 选择器 | 选择特定资源 |
| ReplicaSet | 副本集 | 维护 Pod 副本数 |
| StatefulSet | 有状态集 | 有状态应用的部署管理 |
| DaemonSet | 守护进程集 | 每个节点运行一个 Pod |
| Job | 任务 | 一次性任务 |
| CronJob | 定时任务 | 周期性任务 |
| ConfigMap | 配置映射 | 非敏感配置数据 |
| Secret | 密钥 | 敏感数据 |
| Volume | 存储卷 | 持久化存储 |
| PV | 持久卷 | 集群级存储资源 |
| PVC | 持久卷声明 | 对 PV 的请求 |
| Ingress | 入口 | HTTP(S) 路由规则 |
| NetworkPolicy | 网络策略 | Pod 网络访问控制 |
| RBAC | 角色访问控制 | 权限管理 |

## 实践练习

### 练习 1：使用标签组织资源

```bash
# 创建带标签的 Pod
kubectl run nginx-prod --image=nginx --labels="app=nginx,env=production"
kubectl run nginx-dev --image=nginx --labels="app=nginx,env=development"

# 按标签筛选
kubectl get pods -l env=production
kubectl get pods -l 'env in (production, development)'

# 查看所有标签
kubectl get pods --show-labels

# 清理
kubectl delete pods -l app=nginx
```

### 练习 2：使用命名空间

```bash
# 创建命名空间
kubectl create namespace test-ns

# 在命名空间中创建资源
kubectl run nginx --image=nginx -n test-ns

# 查看特定命名空间的资源
kubectl get pods -n test-ns

# 查看所有命名空间的资源
kubectl get pods --all-namespaces
kubectl get pods -A

# 清理
kubectl delete namespace test-ns
```

### 练习 3：理解资源关系

```bash
# 创建 Deployment
kubectl create deployment web --image=nginx --replicas=3

# 查看创建的资源链
kubectl get deployment web
kubectl get replicaset -l app=web
kubectl get pods -l app=web

# 查看资源详情
kubectl describe deployment web

# 清理
kubectl delete deployment web
```

## 下一步

- [Pod - 最小调度单元](../02-resources/01-pod.md) - 深入学习 Pod



