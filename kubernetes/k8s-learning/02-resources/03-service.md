# 🌐 Service - 服务发现与负载均衡

## 为什么需要 Service？

Pod 是临时的，它们的 IP 地址会随着重建而变化。Service 提供了一个稳定的访问入口。

```
问题：
┌─────────────────────────────────────────────────────────┐
│                                                          │
│  Client ──X──> Pod (IP: 10.1.1.5)                      │
│                      │                                   │
│                      ▼                                   │
│               Pod 重启后                                 │
│               IP 变为 10.1.2.8                          │
│                      │                                   │
│  Client ────?──> ???                                    │
│                                                          │
└─────────────────────────────────────────────────────────┘

解决方案：
┌─────────────────────────────────────────────────────────┐
│                                                          │
│  Client ────> Service (ClusterIP: 10.96.0.100)         │
│                    │                                     │
│                    ▼                                     │
│              ┌─────────────┐                            │
│              │  Endpoints  │                            │
│              │ 10.1.1.5    │                            │
│              │ 10.1.2.6    │                            │
│              │ 10.1.3.7    │                            │
│              └─────────────┘                            │
│                    │                                     │
│            ┌───────┴───────┐                            │
│            ▼       ▼       ▼                            │
│         Pod 1   Pod 2   Pod 3                           │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Service 类型

### 类型总览

| 类型 | 说明 | 可访问范围 | 典型用途 |
|------|------|-----------|---------|
| ClusterIP | 集群内部 IP | 集群内部 | 内部服务通信 |
| NodePort | 节点端口 | 集群外部 | 开发测试 |
| LoadBalancer | 云负载均衡器 | 外网 | 生产环境外部访问 |
| ExternalName | DNS CNAME | 集群内部 | 访问外部服务 |

### 1. ClusterIP（默认）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP                 # 默认类型，可省略
  selector:
    app: my-app                   # 选择标签为 app=my-app 的 Pod
  ports:
  - name: http
    port: 80                      # Service 端口
    targetPort: 8080              # Pod 端口
    protocol: TCP
```

```
访问方式：
┌────────────────────────────────────────────────────────────┐
│ 集群内部                                                    │
│                                                             │
│  curl http://my-service                     # 同命名空间    │
│  curl http://my-service.namespace           # 跨命名空间    │
│  curl http://my-service.namespace.svc.cluster.local        │
│  curl http://10.96.0.100                    # ClusterIP    │
└────────────────────────────────────────────────────────────┘
```

### 2. NodePort

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-nodeport-service
spec:
  type: NodePort
  selector:
    app: my-app
  ports:
  - port: 80                      # Service 端口
    targetPort: 8080              # Pod 端口
    nodePort: 30080               # 节点端口 (30000-32767)
```

```
访问方式：
┌────────────────────────────────────────────────────────────┐
│                                                             │
│  外部: http://<任意节点IP>:30080                            │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Node 1 (192.168.1.10)                  │  │
│  │                     :30080                           │  │
│  │                        │                             │  │
│  │                        ▼                             │  │
│  │                    Service                           │  │
│  │                        │                             │  │
│  │               ┌────────┼────────┐                   │  │
│  │               ▼        ▼        ▼                   │  │
│  │            Pod 1    Pod 2    Pod 3                  │  │
│  └─────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

### 3. LoadBalancer

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-lb-service
  annotations:
    # 云厂商特定注解
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  type: LoadBalancer
  selector:
    app: my-app
  ports:
  - port: 80
    targetPort: 8080
  # 可选：指定负载均衡器 IP
  loadBalancerIP: 203.0.113.10
```

```
访问方式：
┌────────────────────────────────────────────────────────────┐
│                                                             │
│   Internet                                                  │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────────────┐                                       │
│  │ Load Balancer   │ ← 云厂商提供                          │
│  │ (外网 IP)        │                                       │
│  └────────┬────────┘                                       │
│           │                                                 │
│           ▼                                                 │
│      ┌─────────┐                                           │
│      │ Service │                                           │
│      └────┬────┘                                           │
│           │                                                 │
│      ┌────┴────┐                                           │
│      ▼    ▼    ▼                                           │
│    Pod1 Pod2 Pod3                                          │
└────────────────────────────────────────────────────────────┘
```

### 4. ExternalName

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-db
spec:
  type: ExternalName
  externalName: db.example.com    # 外部服务域名
```

```bash
# 访问方式：在集群内通过 Service 名访问外部服务
curl http://external-db  # 会解析到 db.example.com
```

## Service YAML 完整示例

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
  labels:
    app: my-app
  annotations:
    description: "My application service"
spec:
  # 类型
  type: ClusterIP
  
  # 选择器
  selector:
    app: my-app
    version: v1
  
  # 端口配置
  ports:
  - name: http
    port: 80                      # Service 端口
    targetPort: 8080              # Pod 端口
    protocol: TCP
  - name: https
    port: 443
    targetPort: 8443
  
  # 会话亲和性
  sessionAffinity: ClientIP       # None 或 ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800       # 3小时
  
  # 外部流量策略
  externalTrafficPolicy: Cluster  # Cluster 或 Local
  
  # 内部流量策略 (K8s 1.22+)
  internalTrafficPolicy: Cluster  # Cluster 或 Local
```

## 创建 Service 的方式

### 方式 1：命令行

```bash
# 暴露 Deployment
kubectl expose deployment nginx --port=80 --target-port=80

# 指定类型
kubectl expose deployment nginx --port=80 --type=NodePort

# 暴露 Pod（不推荐）
kubectl expose pod nginx --port=80

# 生成 YAML
kubectl expose deployment nginx --port=80 --dry-run=client -o yaml
```

### 方式 2：YAML 文件

```bash
kubectl apply -f service.yaml
```

## Endpoints

Service 通过 Endpoints 关联 Pod。

```bash
# 查看 Service 的 Endpoints
kubectl get endpoints my-service

# 输出示例
NAME         ENDPOINTS                                   AGE
my-service   10.244.1.5:8080,10.244.2.6:8080,10.244.3.7:8080   5m

# 详细信息
kubectl describe endpoints my-service
```

### 手动 Endpoints

有时需要将 Service 指向集群外部的服务：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
spec:
  ports:
  - port: 80
  # 注意：没有 selector

---
apiVersion: v1
kind: Endpoints
metadata:
  name: external-service      # 必须与 Service 同名
subsets:
- addresses:
  - ip: 192.168.1.100         # 外部服务 IP
  - ip: 192.168.1.101
  ports:
  - port: 80
```

## DNS 服务发现

### DNS 记录格式

```
<service-name>.<namespace>.svc.cluster.local

示例：
my-service.default.svc.cluster.local
nginx.production.svc.cluster.local
```

### DNS 解析示例

```bash
# 在 Pod 中测试 DNS
kubectl run test --image=busybox -it --rm -- nslookup my-service

# 同命名空间简写
curl http://my-service

# 跨命名空间
curl http://my-service.other-namespace

# 完整域名
curl http://my-service.other-namespace.svc.cluster.local
```

### Headless Service（无头服务）

不分配 ClusterIP，直接返回 Pod IP 列表。

```yaml
apiVersion: v1
kind: Service
metadata:
  name: headless-service
spec:
  clusterIP: None              # 关键配置
  selector:
    app: my-app
  ports:
  - port: 80
```

```bash
# DNS 返回所有 Pod IP
nslookup headless-service
# 返回: 10.244.1.5, 10.244.2.6, 10.244.3.7
```

用途：
- StatefulSet 的服务发现
- 客户端需要直接访问所有 Pod

## 会话保持（Session Affinity）

```yaml
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800    # 会话保持时间
```

效果：来自同一客户端的请求会被转发到同一个 Pod。

## 流量策略

### externalTrafficPolicy

```yaml
spec:
  externalTrafficPolicy: Local   # 或 Cluster（默认）
```

| 策略 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| Cluster | 跨节点负载均衡 | 均匀分布 | 额外跳转，丢失源 IP |
| Local | 只转发到本节点 Pod | 保留源 IP，低延迟 | 可能不均匀 |

## 常用操作命令

```bash
# ============ 创建和删除 ============
kubectl expose deployment nginx --port=80
kubectl delete service nginx

# ============ 查看 ============
kubectl get services
kubectl get svc                              # 简写
kubectl get svc -o wide
kubectl describe svc nginx

# ============ 查看 Endpoints ============
kubectl get endpoints
kubectl describe endpoints nginx

# ============ 端口转发（调试）============
kubectl port-forward svc/nginx 8080:80

# ============ 获取 Service URL（Minikube）============
minikube service nginx --url
```

## 实践练习

### 练习 1：ClusterIP Service

```bash
# 1. 创建 Deployment
kubectl create deployment web --image=nginx --replicas=3

# 2. 创建 ClusterIP Service
kubectl expose deployment web --port=80 --target-port=80

# 3. 查看 Service
kubectl get svc web
kubectl get endpoints web

# 4. 测试访问（从集群内）
kubectl run test --image=busybox -it --rm -- wget -qO- http://web

# 5. 清理
kubectl delete svc web
kubectl delete deployment web
```

### 练习 2：NodePort Service

```bash
# 1. 创建 Deployment
kubectl create deployment nginx --image=nginx --replicas=2

# 2. 创建 NodePort Service
kubectl expose deployment nginx --port=80 --type=NodePort

# 3. 获取 NodePort
kubectl get svc nginx

# 4. 访问（获取节点 IP 和 NodePort）
# curl http://<node-ip>:<node-port>

# 或使用 Minikube
minikube service nginx --url

# 5. 清理
kubectl delete svc nginx
kubectl delete deployment nginx
```

### 练习 3：完整的服务配置

创建文件 `service-demo.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
      - name: hello
        image: gcr.io/google-samples/hello-app:1.0
        ports:
        - containerPort: 8080

---
apiVersion: v1
kind: Service
metadata:
  name: hello-service
spec:
  type: ClusterIP
  selector:
    app: hello
  ports:
  - name: http
    port: 80
    targetPort: 8080
```

```bash
# 应用
kubectl apply -f service-demo.yaml

# 验证
kubectl get deployment hello-app
kubectl get svc hello-service
kubectl get endpoints hello-service

# 测试
kubectl run test --image=busybox -it --rm -- wget -qO- http://hello-service

# 清理
kubectl delete -f service-demo.yaml
```

### 练习 4：Headless Service

创建文件 `headless-demo.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80

---
apiVersion: v1
kind: Service
metadata:
  name: web-headless
spec:
  clusterIP: None
  selector:
    app: web
  ports:
  - port: 80
```

```bash
# 应用
kubectl apply -f headless-demo.yaml

# 查看（没有 ClusterIP）
kubectl get svc web-headless

# DNS 测试
kubectl run test --image=busybox -it --rm -- nslookup web-headless

# 清理
kubectl delete -f headless-demo.yaml
```

## Service 与 Deployment 组合模板

```yaml
# deployment-service-template.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
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

---
apiVersion: v1
kind: Service
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  type: ClusterIP
  selector:
    app: myapp
  ports:
  - name: http
    port: 80
    targetPort: 80
```

## 最佳实践

1. **使用有意义的名称**：Service 名称即 DNS 名
2. **配置健康检查**：确保流量只发送到健康的 Pod
3. **选择合适的类型**：内部服务用 ClusterIP，外部访问用 LoadBalancer/Ingress
4. **使用标签选择器**：便于管理和调试
5. **配置资源端口名称**：方便识别和管理

## 下一步

- [ConfigMap 与 Secret](./04-configmap-secret.md)



