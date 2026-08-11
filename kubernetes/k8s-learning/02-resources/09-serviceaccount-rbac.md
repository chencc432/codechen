# 🛡️ ServiceAccount 与 RBAC 权限控制

## 为什么需要 ServiceAccount 和 RBAC？

Kubernetes 集群里不只有"人"在操作，更多的是"程序"在运行。

```
人在用 kubectl 操作集群
  └─→ 你：kubectl get pods
  └─→ CI/CD：kubectl apply -f deployment.yaml

程序在集群内部访问 API
  └─→ Pod 里的应用：想通过 API 查看其他 Pod 的状态
  └─→ 控制器：Deployment Controller 需要 watch Pod 变化
  └─→ 监控系统：Prometheus 需要拉取 Pod 指标
```

问题来了：**怎么控制这些"人"和"程序"各自能做什么？**

- 你的开发同事只能看 Pod，不能删 Namespace
- CI/CD 系统只能部署到 staging 命名空间，不能碰 production
- 你的 Pod 里的应用只能读自己的配置，不能看别人的 Secret

这就是 ServiceAccount 和 RBAC 解决的问题。

### 一句话总结

| 概念 | 回答的问题 |
|------|-----------|
| **认证（Authentication）** | 你是谁？→ ServiceAccount 是你的身份 |
| **授权（Authorization）** | 你能做什么？→ RBAC 是你的权限 |

## ServiceAccount — 身份

### 什么是 ServiceAccount？

ServiceAccount 是 Pod 在 Kubernetes 集群中的身份。

```
                ┌────────────────────┐
                │   API Server       │
                └────────┬───────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
     ┌────┴────┐   ┌────┴────┐   ┌────┴────┐
     │  Pod A  │   │  Pod B  │   │  Pod C  │
     │  SA: app│   │SA:monitor│   │  SA:default│
     └─────────┘   └─────────┘   └─────────┘
```

**关键点**：ServiceAccount 是"Pod 的身份"，不是"人的身份"。人用 kubectl 时用的是 User 证书，不是 ServiceAccount。

### default ServiceAccount

每个命名空间创建时，Kubernetes 会自动创建一个 `default` ServiceAccount。

```bash
kubectl get serviceaccount
# 输出：
# NAME      SECRETS   AGE
# default   1         10d
```

如果你创建 Pod 时没有指定 `serviceAccountName`，Pod 就会使用这个命名空间的 `default` ServiceAccount。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  # 没写 serviceAccountName → 自动使用 default
  containers:
  - name: app
    image: nginx
```

**`default` ServiceAccount 通常没有额外权限**——它只能做最基本的匿名查询。如果你的 Pod 需要访问 API（比如读取 ConfigMap、列出 Pod），你需要创建一个专门的 ServiceAccount 并给它授权。

### 创建 ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  namespace: default
```

```bash
kubectl apply -f serviceaccount.yaml
kubectl get sa             # 简写
kubectl describe sa myapp-sa
```

### 在 Pod 中使用 ServiceAccount

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  serviceAccountName: myapp-sa   # 指定身份
  containers:
  - name: app
    image: myapp
```

### ServiceAccount 的自动挂载机制

当你创建一个 ServiceAccount 时，Kubernetes 会：

1. 自动创建一个 Secret（存着 API Server 的 CA 证书和 Token）
2. 把这个 Secret 的名字记录在 ServiceAccount 的 `secrets` 字段中
3. Pod 启动时，kubelet 把这个 Secret 以 Volume 形式挂载到 `/var/run/secrets/kubernetes.io/serviceaccount/`

```bash
# 进入 Pod 看看挂载了什么
kubectl exec myapp -- ls /var/run/secrets/kubernetes.io/serviceaccount/
# 输出：
# ca.crt        # API Server 的 CA 证书，用于验证 API Server 的身份
# namespace     # 当前 Pod 所属的命名空间
# token         # 访问 API Server 的 Bearer Token

# 查看 token 内容
kubectl exec myapp -- cat /var/run/secrets/kubernetes.io/serviceaccount/token
# 输出一个 JWT 格式的 token
```

**Token 自动轮换**（K8s 1.21+）：Token 是自动轮换的，不需要手动管理。Kubernetes 会定期更新 Token，确保安全性。

**关闭自动挂载**：如果你的 Pod 不需要访问 API Server，可以关闭自动挂载：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  automountServiceAccountToken: false   # 不挂载 Token
  containers:
  - name: app
    image: myapp
```

或者在 ServiceAccount 级别关闭：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
automountServiceAccountToken: false
```

### 什么时候需要创建专门的 ServiceAccount

| 场景 | 是否需要专门 SA | 原因 |
|------|---------------|------|
| 普通 Web 应用，不访问 API | 不需要 | 用 default 就够了 |
| 应用需要读 ConfigMap | 需要 | 需要给 SA 绑定读 ConfigMap 的 Role |
| 自定义控制器/Operator | 需要 | 需要读写各种资源 |
| 应用需要查询 Pod 状态 | 需要 | 需要给 SA 绑定 Pod 读权限 |
| 只需要访问外部数据库 | 不需要 | 数据库凭据在 Secret 里，不通过 API |

## RBAC — 权限

### 什么是 RBAC？

RBAC（Role-Based Access Control，基于角色的访问控制）是 Kubernetes 的授权机制。它的核心模型是：

```
谁（Subject）───绑定───→ 角色（Role）───定义───→ 权限（Rules）
```

具体到 Kubernetes 的资源：

```
Subject                 Binding                 Role/ClusterRole
─────────               ───────                 ──────────────
User                    RoleBinding             Role（命名空间级）
Group                   ClusterRoleBinding      ClusterRole（集群级）
ServiceAccount
```

### 两种作用域

Kubernetes 的权限分为两个层级：

```
命名空间级（Role）                   集群级（ClusterRole）
─────────────────                   ─────────────────
只能操作一个命名空间内的资源         可以操作整个集群的资源
例如：default 命名空间的 Pod         例如：Node、PV、Namespace
                                    也可以跨命名空间授权
```

**Role 和 ClusterRole 的区别**：

| 对比维度 | Role | ClusterRole |
|---------|------|-------------|
| 作用范围 | 单个命名空间 | 整个集群 |
| 可以授权哪些资源 | 命名空间级资源（Pod、Service、Deployment 等）| 集群级资源（Node、PV、Namespace）+ 命名空间级资源 |
| 常见用途 | 应用在某个命名空间内的权限 | 管理员权限、跨命名空间授权 |

### Role — 命名空间级角色

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pod-reader
  namespace: default
rules:
- apiGroups: [""]                  # 核心 API 组
  resources: ["pods"]              # 资源类型
  verbs: ["get", "list", "watch"]  # 允许的操作
```

**apiGroups 怎么填？**

| 资源 | apiGroups | 典型资源名 |
|------|-----------|-----------|
| Pod、Service、ConfigMap、Secret、Node | `""`（空字符串） | pods, services, configmaps |
| Deployment、StatefulSet、DaemonSet | `"apps"` | deployments, statefulsets |
| Job、CronJob | `"batch"` | jobs, cronjobs |
| Ingress | `"networking.k8s.io"` | ingresses |
| Role、RoleBinding | `"rbac.authorization.k8s.io"` | roles, rolebindings |

**最常用的做法**：不确定时，先 `kubectl api-resources` 查看：

```bash
kubectl api-resources -o wide | grep -i pod
# 输出：
# pods                              v1                   true        Pod
# 从输出能看出：pods 属于 apiVersion v1，对应 apiGroups 为 ""
```

### ClusterRole — 集群级角色

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
  verbs: ["get", "list"]
```

**ClusterRole 的特殊能力**：即使 Role 只能作用于命名空间内的资源，但 ClusterRole 可以通过 RoleBinding 绑定到某个命名空间，从而"借用" ClusterRole 的权限定义，但只作用于该命名空间。

```
ClusterRole 定义了"可以读 Pod"
    │
    ├──→ ClusterRoleBinding → 整个集群的所有 Pod 都能读
    │
    └──→ RoleBinding（在 default 命名空间）→ 只能读 default 命名空间的 Pod
```

### RoleBinding — 绑定角色到命名空间

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: default
subjects:
- kind: ServiceAccount
  name: myapp-sa
  namespace: default
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

**RoleBinding 可以绑定的对象**：

```yaml
subjects:
# 绑定 ServiceAccount（最常用）
- kind: ServiceAccount
  name: myapp-sa
  namespace: default

# 绑定 User（人用 kubectl 时）
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io

# 绑定用户组
- kind: Group
  name: developers
  apiGroup: rbac.authorization.k8s.io
```

### ClusterRoleBinding — 绑定角色到整个集群

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

### 完整的权限模型

```
权限查找流程：

1. 请求到达 API Server
2. API Server 确定请求者的身份（ServiceAccount / User）
3. 查找该身份绑定的所有 Role / ClusterRole
4. 合并所有匹配的规则
5. 如果任何一条规则允许 → 允许
6. 如果没有任何规则允许 → 拒绝
```

**注意**：RBAC 是"白名单"机制——没有明确允许的就是拒绝。

## 内置 ClusterRole

Kubernetes 自带了一些 ClusterRole，可以直接使用：

| ClusterRole | 说明 | 常见用途 |
|-------------|------|----------|
| `cluster-admin` | 超级管理员，所有权限 | 集群管理员 |
| `admin` | 命名空间管理员，可以管理该命名空间的大部分资源 | 项目负责人 |
| `edit` | 可以读写命名空间内的资源，但不能修改 RBAC | 开发者 |
| `view` | 只读权限，不能读 Secret | 只读用户 |

```bash
# 查看内置 ClusterRole
kubectl get clusterrole

# 查看具体权限
kubectl describe clusterrole view
```

## 常用 Verbs 详解

| Verb | 对应的 HTTP 方法 | 说明 |
|------|-----------------|------|
| `get` | GET | 获取单个资源详情 |
| `list` | GET (collection) | 列出资源列表 |
| `watch` | GET (watch) | 监听资源变化（长连接）|
| `create` | POST | 创建资源 |
| `update` | PUT | 全量更新资源 |
| `patch` | PATCH | 部分更新资源 |
| `delete` | DELETE | 删除单个资源 |
| `deletecollection` | DELETE (collection) | 批量删除资源 |

**实用建议**：

- 只读权限：`["get", "list", "watch"]`
- 读写权限：`["get", "list", "watch", "create", "update", "patch", "delete"]`

## 完整示例

### 示例 1：让应用能读自己的 ConfigMap

```yaml
# 1. 创建 ServiceAccount
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  namespace: default

---
# 2. 创建 Role（只允许读 ConfigMap）
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: configmap-reader
  namespace: default
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list"]

---
# 3. 绑定
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: myapp-read-configmaps
  namespace: default
subjects:
- kind: ServiceAccount
  name: myapp-sa
  namespace: default
roleRef:
  kind: Role
  name: configmap-reader
  apiGroup: rbac.authorization.k8s.io

---
# 4. 在 Pod 中使用
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  serviceAccountName: myapp-sa
  containers:
  - name: app
    image: busybox
    command: ["sh", "-c", "sleep 3600"]
```

测试一下：

```bash
# 进入 Pod
kubectl exec -it myapp -- sh

# 在 Pod 里尝试访问 API
# 可以读 ConfigMap
curl -k --header "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
  https://kubernetes.default.svc/api/v1/namespaces/default/configmaps

# 但不能读 Pod（没有授权）
curl -k --header "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
  https://kubernetes.default.svc/api/v1/namespaces/default/pods
# 返回 403 Forbidden
```

### 示例 2：跨命名空间查看 Pod

```yaml
# 1. 创建 ClusterRole（定义"可以读 Pod"这个权限）
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-reader
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]

---
# 2. 在各自命名空间绑定
# 在 default 命名空间绑定
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: monitor-read-pods
  namespace: default
subjects:
- kind: ServiceAccount
  name: monitor-sa
  namespace: monitoring
roleRef:
  kind: ClusterRole
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io

---
# 在 production 命名空间绑定
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: monitor-read-pods
  namespace: production
subjects:
- kind: ServiceAccount
  name: monitor-sa
  namespace: monitoring
roleRef:
  kind: ClusterRole
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

这里的关键理解：**一个 ClusterRole + 多个 RoleBinding** 可以实现跨命名空间的权限授权。

### 示例 3：Operator 的权限（常见模式）

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-operator
  namespace: operators

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: my-operator-role
rules:
# 需要读自己管理的 CRD
- apiGroups: ["example.com"]
  resources: ["myresources"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# 需要创建 Deployment
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# 需要创建 Service
- apiGroups: [""]
  resources: ["services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# 需要读 Pod 状态
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]

# 需要读 Event（用于记录操作日志）
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: my-operator-binding
subjects:
- kind: ServiceAccount
  name: my-operator
  namespace: operators
roleRef:
  kind: ClusterRole
  name: my-operator-role
  apiGroup: rbac.authorization.k8s.io
```

## 权限检查

### 用 kubectl 检查权限

```bash
# 检查当前用户能不能做某件事
kubectl auth can-i get pods
kubectl auth can-i create deployments
kubectl auth can-i delete nodes

# 模拟其他身份检查
kubectl auth can-i get pods --as jane
kubectl auth can-i get pods --as system:serviceaccount:default:myapp-sa

# 列出所有权限
kubectl auth can-i --list
kubectl auth can-i --list --as system:serviceaccount:default:myapp-sa
```

### ServiceAccount 的命名格式

在 RBAC 中引用 ServiceAccount 时，完整的格式是：

```
system:serviceaccount:<namespace>:<name>
```

例如：

```bash
# 检查 default 命名空间的 myapp-sa 能不能读 Pod
kubectl auth can-i get pods \
  --as system:serviceaccount:default:myapp-sa
```

## 常用操作

```bash
# ============ ServiceAccount ============
kubectl create serviceaccount myapp-sa
kubectl get serviceaccount
kubectl get sa                          # 简写
kubectl describe sa myapp-sa
kubectl delete sa myapp-sa

# ============ Role ============
kubectl create role pod-reader \
  --verb=get,list,watch \
  --resource=pods

kubectl create role deployment-manager \
  --verb=create,delete,update,patch \
  --resource=deployments.apps

kubectl get roles
kubectl describe role pod-reader

# ============ ClusterRole ============
kubectl create clusterrole node-reader \
  --verb=get,list,watch \
  --resource=nodes

kubectl get clusterroles
kubectl describe clusterrole view

# ============ RoleBinding ============
kubectl create rolebinding read-pods \
  --role=pod-reader \
  --serviceaccount=default:myapp-sa

kubectl get rolebindings
kubectl describe rolebinding read-pods

# ============ ClusterRoleBinding ============
kubectl create clusterrolebinding admin-binding \
  --clusterrole=cluster-admin \
  --user=admin

kubectl get clusterrolebindings

# ============ 权限检查 ============
kubectl auth can-i get pods
kubectl auth can-i "*" pods
kubectl auth can-i get pods --as system:serviceaccount:default:myapp-sa
kubectl auth can-i --list
```

## 最佳实践

1. **最小权限原则**：只给需要的权限，不给多余的
2. **一个应用一个 ServiceAccount**：不要多个应用共享一个 SA
3. **避免使用 cluster-admin**：除非你真的需要集群管理
4. **使用 ClusterRole + RoleBinding 模式**：权限定义复用，作用域控制
5. **关闭不必要的 Token 挂载**：不需要访问 API 的 Pod 关掉自动挂载
6. **定期审计权限**：用 `kubectl auth can-i --list` 检查
7. **不使用 default ServiceAccount**：创建专门的 SA 并绑定必需的权限

## 常见问题

### 1. Pod 访问 API 返回 403

```
可能原因：
  - Pod 使用的 ServiceAccount 没有绑定相应的 Role
  - 绑定的 Role 权限不足（缺少某个 verb）
  - 绑定的 Role 作用在错误的命名空间

排查步骤：
  1. kubectl describe pod <pod-name> 查看 serviceAccount
  2. kubectl auth can-i get pods --as system:serviceaccount:<ns>:<sa>
  3. 检查 RoleBinding 的 roleRef 和 subjects 是否正确
```

### 2. kubectl 权限不足

```
可能原因：
  - kubeconfig 中的证书没有对应权限
  - 切换了集群/上下文

排查步骤：
  1. kubectl config view 查看当前配置
  2. kubectl auth can-i --list 查看当前用户权限
  3. 确认 kubeconfig 中的用户有正确的证书
```

### 3. 删除命名空间时卡住

```
可能原因：ServiceAccount 的 finalizer 等待清理

排查步骤：
  kubectl get sa -n <namespace> -o yaml
  查看是否有 finalizers 阻止删除
```

## 总结

```
认证（你是谁？）               授权（你能做什么？）
─────────────────              ─────────────────
ServiceAccount                Role / ClusterRole
  └── Pod 的身份                └── 定义权限集合
  └── 自动挂载 Token                    │
  └── 每个命名空间有 default    RoleBinding / ClusterRoleBinding
                                  └── 把身份和权限绑定起来
```

## 下一步

- [SecurityContext 与 Pod 安全标准](../04-advanced/04-security.md)
