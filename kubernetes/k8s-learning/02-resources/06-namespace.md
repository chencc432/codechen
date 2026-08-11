# 🏢 Namespace - 资源隔离

## 什么是 Namespace？

Namespace 是 Kubernetes 中用于隔离资源的一种机制，可以将一个物理集群划分为多个虚拟集群。

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐     │
│  │    default      │  │   production    │  │   development   │     │
│  │   Namespace     │  │   Namespace     │  │   Namespace     │     │
│  │                 │  │                 │  │                 │     │
│  │  ┌───┐ ┌───┐   │  │  ┌───┐ ┌───┐   │  │  ┌───┐ ┌───┐   │     │
│  │  │Pod│ │Svc│   │  │  │Pod│ │Svc│   │  │  │Pod│ │Svc│   │     │
│  │  └───┘ └───┘   │  │  └───┘ └───┘   │  │  └───┘ └───┘   │     │
│  │  ┌───┐ ┌───┐   │  │  ┌───┐ ┌───┐   │  │  ┌───┐ ┌───┐   │     │
│  │  │CM │ │Sec│   │  │  │CM │ │Sec│   │  │  │CM │ │Sec│   │     │
│  │  └───┘ └───┘   │  │  └───┘ └───┘   │  │  └───┘ └───┘   │     │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘     │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                      kube-system                              │   │
│  │  (系统组件: kube-dns, kube-proxy, metrics-server 等)          │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

## 默认命名空间

```bash
# Kubernetes 自带的命名空间
default        # 默认命名空间，未指定时使用
kube-system    # Kubernetes 系统组件
kube-public    # 公开资源，所有用户可读
kube-node-lease # 节点心跳（租约）数据
```

## Namespace 的作用

1. **资源隔离**：不同命名空间的资源相互独立
2. **权限控制**：可以为不同命名空间设置不同的 RBAC 权限
3. **资源配额**：限制每个命名空间的资源使用量
4. **环境分离**：开发、测试、生产环境分离

## 命名空间操作

### 创建命名空间

```bash
# 命令行创建
kubectl create namespace development
kubectl create ns staging          # 简写

# YAML 创建
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    environment: production
    team: backend
EOF
```

### 查看命名空间

```bash
# 列出所有命名空间
kubectl get namespaces
kubectl get ns

# 查看命名空间详情
kubectl describe namespace production

# 查看命名空间的资源
kubectl get all -n production
```

### 删除命名空间

```bash
# 删除命名空间（会删除其中所有资源！）
kubectl delete namespace development

# 强制删除卡住的命名空间
kubectl delete namespace stuck-ns --force --grace-period=0
```

## 跨命名空间操作

### 指定命名空间

```bash
# 查看特定命名空间的资源
kubectl get pods -n kube-system
kubectl get all -n production

# 在特定命名空间创建资源
kubectl create deployment nginx --image=nginx -n development
kubectl apply -f deployment.yaml -n production

# 查看所有命名空间的资源
kubectl get pods --all-namespaces
kubectl get pods -A                           # 简写
```

### 设置默认命名空间

```bash
# 方式 1：修改当前 context
kubectl config set-context --current --namespace=development

# 方式 2：创建新 context
kubectl config set-context dev-context \
  --cluster=my-cluster \
  --user=my-user \
  --namespace=development

# 切换 context
kubectl config use-context dev-context

# 查看当前默认命名空间
kubectl config view --minify | grep namespace

# 使用 kubens 工具（需要安装）
kubens development
```

## 资源配额 (ResourceQuota)

限制命名空间中的资源使用量。

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-quota
  namespace: development
spec:
  hard:
    # 计算资源
    requests.cpu: "4"
    requests.memory: 8Gi
    limits.cpu: "8"
    limits.memory: 16Gi
    
    # 对象数量
    pods: "20"
    services: "10"
    secrets: "10"
    configmaps: "10"
    persistentvolumeclaims: "5"
    
    # 特定类型限制
    count/deployments.apps: "5"
    count/replicasets.apps: "10"
```

### 查看配额使用情况

```bash
kubectl get resourcequota -n development
kubectl describe resourcequota compute-quota -n development
```

## 限制范围 (LimitRange)

设置命名空间中 Pod/容器的默认资源限制。

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: development
spec:
  limits:
  # 容器默认值
  - type: Container
    default:           # 默认 limits
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:    # 默认 requests
      cpu: "100m"
      memory: "128Mi"
    min:               # 最小值
      cpu: "50m"
      memory: "64Mi"
    max:               # 最大值
      cpu: "2"
      memory: "2Gi"
  
  # Pod 级别限制
  - type: Pod
    max:
      cpu: "4"
      memory: "4Gi"
  
  # PVC 限制
  - type: PersistentVolumeClaim
    min:
      storage: 1Gi
    max:
      storage: 100Gi
```

## 网络策略 (NetworkPolicy)

控制命名空间内 Pod 的网络访问。

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all
  namespace: production
spec:
  # 选择所有 Pod
  podSelector: {}
  
  # 禁止所有入站流量
  policyTypes:
  - Ingress
  - Egress

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-same-namespace
  namespace: production
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector: {}      # 允许同命名空间的 Pod
```

## 完整的环境隔离示例

### 创建开发环境

```yaml
# development-ns.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: development
  labels:
    environment: dev

---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: dev-quota
  namespace: development
spec:
  hard:
    requests.cpu: "2"
    requests.memory: 4Gi
    limits.cpu: "4"
    limits.memory: 8Gi
    pods: "10"

---
apiVersion: v1
kind: LimitRange
metadata:
  name: dev-limits
  namespace: development
spec:
  limits:
  - type: Container
    default:
      cpu: "200m"
      memory: "256Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
```

### 创建生产环境

```yaml
# production-ns.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    environment: prod

---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: prod-quota
  namespace: production
spec:
  hard:
    requests.cpu: "16"
    requests.memory: 32Gi
    limits.cpu: "32"
    limits.memory: 64Gi
    pods: "100"
    services: "20"

---
apiVersion: v1
kind: LimitRange
metadata:
  name: prod-limits
  namespace: production
spec:
  limits:
  - type: Container
    default:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "200m"
      memory: "256Mi"
    max:
      cpu: "4"
      memory: "4Gi"
```

## 常用操作命令汇总

```bash
# ============ 命名空间操作 ============
kubectl get ns
kubectl create ns <name>
kubectl delete ns <name>
kubectl describe ns <name>

# ============ 跨命名空间操作 ============
kubectl get pods -n <namespace>
kubectl get all -A
kubectl apply -f file.yaml -n <namespace>

# ============ 设置默认命名空间 ============
kubectl config set-context --current --namespace=<namespace>
kubectl config view --minify | grep namespace

# ============ 资源配额 ============
kubectl get resourcequota -n <namespace>
kubectl describe resourcequota <name> -n <namespace>

# ============ 限制范围 ============
kubectl get limitrange -n <namespace>
kubectl describe limitrange <name> -n <namespace>
```

## 实践练习

### 练习 1：创建和管理命名空间

```bash
# 1. 创建命名空间
kubectl create namespace test-ns

# 2. 在命名空间中创建资源
kubectl create deployment nginx --image=nginx -n test-ns

# 3. 查看资源
kubectl get all -n test-ns

# 4. 设置默认命名空间
kubectl config set-context --current --namespace=test-ns

# 5. 现在不需要 -n 参数
kubectl get pods

# 6. 恢复默认
kubectl config set-context --current --namespace=default

# 7. 清理
kubectl delete namespace test-ns
```

### 练习 2：配置资源配额

```bash
# 1. 创建命名空间
kubectl create namespace quota-test

# 2. 创建资源配额
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: ResourceQuota
metadata:
  name: test-quota
  namespace: quota-test
spec:
  hard:
    pods: "3"
    requests.cpu: "1"
    requests.memory: 1Gi
EOF

# 3. 查看配额
kubectl get resourcequota -n quota-test
kubectl describe resourcequota test-quota -n quota-test

# 4. 尝试创建超过配额的 Pod
kubectl create deployment nginx --image=nginx --replicas=5 -n quota-test

# 5. 查看状态（会受到配额限制）
kubectl get deployment -n quota-test
kubectl describe deployment nginx -n quota-test

# 6. 清理
kubectl delete namespace quota-test
```

## 最佳实践

1. **按环境划分**：development, staging, production
2. **按团队划分**：team-a, team-b
3. **按项目划分**：project-x, project-y
4. **始终设置资源配额**：防止资源耗尽
5. **配置默认资源限制**：使用 LimitRange
6. **使用网络策略**：实现网络隔离

## 下一步

- [ServiceAccount 与 RBAC 权限控制](./09-serviceaccount-rbac.md)
- [StatefulSet - 有状态应用](./07-statefulset.md)



