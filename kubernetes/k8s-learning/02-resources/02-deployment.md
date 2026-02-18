# 🚀 Deployment - 无状态应用部署

## 什么是 Deployment？

Deployment 是 Kubernetes 中最常用的工作负载资源，用于管理无状态应用的部署和更新。

```
┌────────────────────────────────────────────────────────────────┐
│                        Deployment                               │
│                                                                  │
│   ┌──────────────────────────────────────────────────────────┐ │
│   │                     ReplicaSet (v2)                      │ │
│   │   ┌─────────┐   ┌─────────┐   ┌─────────┐              │ │
│   │   │  Pod 1  │   │  Pod 2  │   │  Pod 3  │              │ │
│   │   └─────────┘   └─────────┘   └─────────┘              │ │
│   └──────────────────────────────────────────────────────────┘ │
│                                                                  │
│   ┌──────────────────────────────────────────────────────────┐ │
│   │                  ReplicaSet (v1) - 旧版本                 │ │
│   │   (保留用于回滚，副本数为 0)                               │ │
│   └──────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘

Deployment → 管理 ReplicaSet → ReplicaSet 管理 Pod
```

## Deployment 核心功能

| 功能 | 说明 |
|------|------|
| 声明式更新 | 定义期望状态，自动调整 |
| 滚动更新 | 零停机更新应用 |
| 回滚 | 回滚到历史版本 |
| 扩缩容 | 调整 Pod 副本数 |
| 暂停恢复 | 暂停更新，批量修改后再恢复 |

## Deployment YAML 完整解析

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
  labels:
    app: nginx
spec:
  # 副本数
  replicas: 3
  
  # 选择器 - 必须与 template.metadata.labels 匹配
  selector:
    matchLabels:
      app: nginx
  
  # 更新策略
  strategy:
    type: RollingUpdate           # RollingUpdate 或 Recreate
    rollingUpdate:
      maxUnavailable: 25%         # 最大不可用数量
      maxSurge: 25%               # 最大超出副本数
  
  # 历史版本保留数量
  revisionHistoryLimit: 10
  
  # 进度截止时间（秒）
  progressDeadlineSeconds: 600
  
  # Pod 模板
  template:
    metadata:
      labels:
        app: nginx                # 必须包含 selector 中的标签
      annotations:
        prometheus.io/scrape: "true"
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "200m"
            memory: "256Mi"
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 创建 Deployment

### 命令行方式

```bash
# 快速创建
kubectl create deployment nginx --image=nginx

# 指定副本数
kubectl create deployment nginx --image=nginx --replicas=3

# 生成 YAML（不实际创建）
kubectl create deployment nginx --image=nginx --replicas=3 --dry-run=client -o yaml > deployment.yaml

# 指定端口
kubectl create deployment nginx --image=nginx --port=80
```

### YAML 文件方式

```bash
# 创建/更新
kubectl apply -f deployment.yaml

# 删除
kubectl delete -f deployment.yaml
```

## 扩缩容

### 手动扩缩容

```bash
# 方式 1：scale 命令
kubectl scale deployment nginx --replicas=5

# 方式 2：编辑
kubectl edit deployment nginx

# 方式 3：patch
kubectl patch deployment nginx -p '{"spec":{"replicas":5}}'

# 查看扩缩容状态
kubectl rollout status deployment nginx
```

### 自动扩缩容（HPA）

```bash
# 创建 HPA（需要 metrics-server）
kubectl autoscale deployment nginx --min=2 --max=10 --cpu-percent=80

# 查看 HPA
kubectl get hpa
kubectl describe hpa nginx

# 删除 HPA
kubectl delete hpa nginx
```

HPA YAML 配置：

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: nginx-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## 滚动更新

### 更新策略详解

```
RollingUpdate (默认):
┌────────────────────────────────────────────────────────────┐
│ 初始状态: Pod1(v1) Pod2(v1) Pod3(v1)                       │
│                                                             │
│ 步骤 1: Pod1(v1) Pod2(v1) Pod3(v1) + Pod4(v2)             │
│         创建新版本 Pod                                       │
│                                                             │
│ 步骤 2: Pod1(v1) Pod2(v1) [删除Pod3] + Pod4(v2)           │
│         删除旧版本 Pod                                       │
│                                                             │
│ 步骤 3: Pod1(v1) Pod2(v1) Pod4(v2) + Pod5(v2)             │
│         继续创建新版本                                       │
│                                                             │
│ ... 重复直到全部更新完成 ...                                 │
│                                                             │
│ 最终: Pod4(v2) Pod5(v2) Pod6(v2)                           │
└────────────────────────────────────────────────────────────┘

Recreate:
┌────────────────────────────────────────────────────────────┐
│ 初始状态: Pod1(v1) Pod2(v1) Pod3(v1)                       │
│                                                             │
│ 步骤 1: 删除所有旧 Pod                                      │
│         (服务中断)                                          │
│                                                             │
│ 步骤 2: 创建新 Pod                                          │
│         Pod1(v2) Pod2(v2) Pod3(v2)                         │
└────────────────────────────────────────────────────────────┘
```

### 触发更新

```bash
# 方式 1：更新镜像
kubectl set image deployment/nginx nginx=nginx:1.22

# 方式 2：编辑
kubectl edit deployment nginx

# 方式 3：apply 更新的 YAML
kubectl apply -f deployment.yaml

# 方式 4：patch
kubectl patch deployment nginx -p '{"spec":{"template":{"spec":{"containers":[{"name":"nginx","image":"nginx:1.22"}]}}}}'
```

### 查看更新状态

```bash
# 查看滚动更新状态
kubectl rollout status deployment nginx

# 查看更新历史
kubectl rollout history deployment nginx

# 查看特定版本详情
kubectl rollout history deployment nginx --revision=2
```

### 暂停和恢复

```bash
# 暂停更新（可以进行多次修改）
kubectl rollout pause deployment nginx

# 进行多次修改...
kubectl set image deployment nginx nginx=nginx:1.23
kubectl set resources deployment nginx -c nginx --limits=cpu=200m,memory=512Mi

# 恢复更新（一次性应用所有修改）
kubectl rollout resume deployment nginx
```

## 回滚

```bash
# 回滚到上一版本
kubectl rollout undo deployment nginx

# 回滚到指定版本
kubectl rollout undo deployment nginx --to-revision=2

# 查看回滚状态
kubectl rollout status deployment nginx
```

### 配置 revision 保留

```yaml
spec:
  revisionHistoryLimit: 10    # 保留 10 个历史版本
```

## 更新策略配置

### maxUnavailable 和 maxSurge

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      # 最大不可用：更新过程中最多有多少 Pod 不可用
      maxUnavailable: 25%     # 或具体数字如 1
      
      # 最大超出：更新过程中最多超出期望副本数多少
      maxSurge: 25%           # 或具体数字如 1
```

示例配置：

```yaml
# 保守策略：确保始终有足够的 Pod 可用
maxUnavailable: 0
maxSurge: 1

# 激进策略：快速更新
maxUnavailable: 50%
maxSurge: 50%

# 滚动策略（默认）
maxUnavailable: 25%
maxSurge: 25%
```

## Deployment 状态

### 查看状态

```bash
# 基本状态
kubectl get deployment nginx

# 输出示例
NAME    READY   UP-TO-DATE   AVAILABLE   AGE
nginx   3/3     3            3           10m

# READY: 就绪的副本数/期望副本数
# UP-TO-DATE: 已更新到最新版本的副本数
# AVAILABLE: 可用的副本数
```

### Deployment 条件

```yaml
status:
  conditions:
  - type: Available           # Deployment 可用性
    status: "True"
    reason: MinimumReplicasAvailable
  - type: Progressing         # 更新进度
    status: "True"
    reason: NewReplicaSetAvailable
```

## 常用操作命令汇总

```bash
# ============ 创建和删除 ============
kubectl create deployment nginx --image=nginx
kubectl delete deployment nginx

# ============ 查看 ============
kubectl get deployments
kubectl get deployment nginx -o wide
kubectl get deployment nginx -o yaml
kubectl describe deployment nginx

# ============ 扩缩容 ============
kubectl scale deployment nginx --replicas=5
kubectl autoscale deployment nginx --min=2 --max=10 --cpu-percent=80

# ============ 更新 ============
kubectl set image deployment/nginx nginx=nginx:1.22
kubectl set resources deployment nginx -c nginx --limits=cpu=200m,memory=512Mi
kubectl set env deployment nginx ENV_VAR=value

# ============ 回滚 ============
kubectl rollout status deployment nginx
kubectl rollout history deployment nginx
kubectl rollout undo deployment nginx
kubectl rollout undo deployment nginx --to-revision=2

# ============ 暂停/恢复 ============
kubectl rollout pause deployment nginx
kubectl rollout resume deployment nginx

# ============ 重启 ============
kubectl rollout restart deployment nginx
```

## 实践练习

### 练习 1：基本 Deployment 操作

```bash
# 1. 创建 Deployment
kubectl create deployment web --image=nginx:1.20 --replicas=3

# 2. 查看状态
kubectl get deployment web
kubectl get pods -l app=web

# 3. 扩容
kubectl scale deployment web --replicas=5
kubectl get pods -l app=web -w

# 4. 缩容
kubectl scale deployment web --replicas=2

# 5. 清理
kubectl delete deployment web
```

### 练习 2：滚动更新和回滚

```bash
# 1. 创建初始版本
kubectl create deployment nginx --image=nginx:1.20 --replicas=3

# 2. 查看 ReplicaSet
kubectl get rs -l app=nginx

# 3. 更新镜像
kubectl set image deployment/nginx nginx=nginx:1.21

# 4. 观察滚动更新
kubectl rollout status deployment nginx
kubectl get rs -l app=nginx    # 观察新旧 RS

# 5. 更新到错误版本
kubectl set image deployment/nginx nginx=nginx:nonexistent

# 6. 查看状态（会卡住）
kubectl rollout status deployment nginx

# 7. 回滚
kubectl rollout undo deployment nginx

# 8. 验证
kubectl get pods -l app=nginx

# 9. 清理
kubectl delete deployment nginx
```

### 练习 3：完整 Deployment YAML

创建文件 `my-deployment.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  labels:
    app: my-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: app
        image: nginx:1.21
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
```

```bash
# 应用
kubectl apply -f my-deployment.yaml

# 验证
kubectl get deployment my-app
kubectl describe deployment my-app

# 更新（修改 YAML 后）
kubectl apply -f my-deployment.yaml

# 清理
kubectl delete -f my-deployment.yaml
```

## 最佳实践

1. **始终设置资源限制**：防止 Pod 耗尽节点资源
2. **配置健康检查**：确保流量只发送到健康的 Pod
3. **使用合适的更新策略**：根据应用特性选择
4. **设置合理的 revisionHistoryLimit**：节省存储，保留足够的回滚版本
5. **使用标签管理**：便于筛选和管理资源
6. **配置 PodDisruptionBudget**：确保高可用

## 下一步

- [Service - 服务发现与负载均衡](./03-service.md)



