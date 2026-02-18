# 🔐 Kubernetes 安全与权限控制

## 安全层次

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Kubernetes 安全体系                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   传输安全                                                            │
│   └─→ TLS 加密所有 API 通信                                          │
│                                                                       │
│   认证 (Authentication)                                               │
│   └─→ 验证"你是谁"                                                   │
│       • X509 证书                                                     │
│       • Bearer Token                                                  │
│       • ServiceAccount                                                │
│                                                                       │
│   授权 (Authorization)                                                │
│   └─→ 验证"你能做什么"                                               │
│       • RBAC (推荐)                                                   │
│       • ABAC                                                          │
│       • Webhook                                                       │
│                                                                       │
│   准入控制 (Admission Control)                                        │
│   └─→ 验证和修改请求                                                 │
│       • 资源配额                                                      │
│       • Pod 安全策略                                                  │
│       • 变更/验证 Webhook                                             │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## RBAC (基于角色的访问控制)

### 核心概念

```
┌─────────────────────────────────────────────────────────────────────┐
│                         RBAC 模型                                    │
│                                                                       │
│   ┌──────────┐              ┌──────────┐                           │
│   │   User   │              │  Role    │                           │
│   │ServiceAcc│───绑定───→   │ClusterRole│───定义───→ 权限          │
│   │  Group   │              │          │                           │
│   └──────────┘              └──────────┘                           │
│        │                         │                                   │
│        │    ┌──────────────┐    │                                   │
│        └───→│ RoleBinding  │←───┘                                   │
│             │ClusterRoleBind│                                       │
│             └──────────────┘                                       │
└─────────────────────────────────────────────────────────────────────┘

Role/RoleBinding:        命名空间级别
ClusterRole/ClusterRoleBinding: 集群级别
```

### Role（命名空间级角色）

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
  namespace: default
rules:
- apiGroups: [""]           # "" 表示核心 API 组
  resources: ["pods"]
  verbs: ["get", "list", "watch"]

- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "create", "update", "patch", "delete"]
```

### ClusterRole（集群级角色）

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: node-reader
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "watch"]

- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
  # 只能访问特定名称的资源
  resourceNames: ["specific-pod"]
```

### RoleBinding

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: default
subjects:
# 用户
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io

# ServiceAccount
- kind: ServiceAccount
  name: myapp
  namespace: default

# 用户组
- kind: Group
  name: developers
  apiGroup: rbac.authorization.k8s.io

roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

### ClusterRoleBinding

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cluster-admin-binding
subjects:
- kind: User
  name: admin
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
```

### 常用 Verbs

| Verb | 说明 |
|------|------|
| get | 获取单个资源 |
| list | 列出资源 |
| watch | 监听资源变化 |
| create | 创建资源 |
| update | 更新资源 |
| patch | 部分更新资源 |
| delete | 删除资源 |
| deletecollection | 批量删除 |

### RBAC 命令

```bash
# 查看角色
kubectl get roles
kubectl get clusterroles

# 查看绑定
kubectl get rolebindings
kubectl get clusterrolebindings

# 创建角色
kubectl create role pod-reader --verb=get,list,watch --resource=pods

# 创建绑定
kubectl create rolebinding read-pods --role=pod-reader --user=jane

# 检查权限
kubectl auth can-i get pods
kubectl auth can-i get pods --as jane
kubectl auth can-i get pods --as system:serviceaccount:default:myapp

# 查看用户权限
kubectl auth can-i --list
kubectl auth can-i --list --as jane
```

## ServiceAccount

### 创建 ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp
  namespace: default
automountServiceAccountToken: true
imagePullSecrets:
- name: regcred
```

### 在 Pod 中使用

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  serviceAccountName: myapp
  automountServiceAccountToken: true
  containers:
  - name: app
    image: myapp
```

### 完整示例

```yaml
# 1. 创建 ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  namespace: default

---
# 2. 创建 Role
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: myapp-role
  namespace: default
rules:
- apiGroups: [""]
  resources: ["pods", "configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list"]

---
# 3. 创建 RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: myapp-rolebinding
  namespace: default
subjects:
- kind: ServiceAccount
  name: myapp-sa
  namespace: default
roleRef:
  kind: Role
  name: myapp-role
  apiGroup: rbac.authorization.k8s.io

---
# 4. 使用 ServiceAccount
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      serviceAccountName: myapp-sa
      containers:
      - name: app
        image: myapp
```

## Pod 安全

### SecurityContext（容器安全上下文）

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: secure-pod
spec:
  # Pod 级别
  securityContext:
    runAsUser: 1000
    runAsGroup: 3000
    fsGroup: 2000
    runAsNonRoot: true
  
  containers:
  - name: app
    image: myapp
    # 容器级别
    securityContext:
      readOnlyRootFilesystem: true
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
        add:
        - NET_BIND_SERVICE
```

### Pod Security Standards (K8s 1.25+)

```yaml
# 在命名空间上设置安全标准
apiVersion: v1
kind: Namespace
metadata:
  name: secure-ns
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

安全级别：
- **privileged**：不受限制
- **baseline**：防止已知的特权升级
- **restricted**：严格限制，遵循最佳实践

## Secret 安全

### 加密 Secret（etcd 加密）

```yaml
# /etc/kubernetes/encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
- resources:
  - secrets
  providers:
  - aescbc:
      keys:
      - name: key1
        secret: <base64-encoded-key>
  - identity: {}
```

### 使用外部密钥管理

推荐使用：
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- External Secrets Operator

## 常用安全检查

```bash
# 检查 RBAC
kubectl auth can-i --list --as system:serviceaccount:default:myapp

# 检查 Pod 安全上下文
kubectl get pod <pod-name> -o jsonpath='{.spec.securityContext}'

# 检查 ServiceAccount
kubectl get serviceaccount
kubectl get sa <sa-name> -o yaml

# 检查角色权限
kubectl describe role <role-name>
kubectl describe clusterrole <clusterrole-name>
```

## 安全最佳实践

1. **最小权限原则**：只授予必需的权限
2. **使用 ServiceAccount**：为每个应用创建专用 SA
3. **不使用 default SA**：禁用 default SA 的 token 挂载
4. **启用 RBAC**：始终使用 RBAC 进行授权
5. **定期审计**：审计 RBAC 配置和使用情况
6. **保护 etcd**：加密 Secret 数据
7. **网络隔离**：使用 NetworkPolicy
8. **容器安全**：使用 SecurityContext 限制容器权限

## 下一步

- [Ingress 与流量管理](./05-ingress.md)



