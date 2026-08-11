# 🐳 Pod - Kubernetes 最小调度单元

## 什么是 Pod？

Pod 是 Kubernetes 中最小的可部署单元，一个 Pod 可以包含一个或多个容器。

```
┌─────────────────────────────────────────────────────────────┐
│                           Pod                                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  共享网络命名空间                      │   │
│  │                  (共享 IP 和端口)                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌───────────┐  ┌───────────┐  ┌───────────────────────┐  │
│  │ Container │  │ Container │  │    Init Container     │  │
│  │   (app)   │  │  (sidecar)│  │    (初始化容器)        │  │
│  └───────────┘  └───────────┘  └───────────────────────┘  │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    共享存储卷                         │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Pod 的核心特点

1. **共享网络**：同一 Pod 内的容器共享 IP 地址和端口，可以通过 `localhost` 互访
2. **共享存储**：可以定义共享的 Volume，多个容器可以访问
3. **共同调度**：Pod 内的所有容器总是调度到同一个节点
4. **生命周期共同体**：容器一起创建、一起销毁

## Pod YAML 完整解析

```yaml
apiVersion: v1                    # API 版本
kind: Pod                         # 资源类型
metadata:                         # 元数据
  name: my-pod                    # Pod 名称
  namespace: default              # 命名空间
  labels:                         # 标签
    app: myapp
    version: v1
  annotations:                    # 注解
    description: "This is my first pod"
spec:                             # 规约（期望状态）
  restartPolicy: Always           # 重启策略: Always/OnFailure/Never
  
  # 初始化容器（按顺序执行，全部成功后才启动主容器）
  initContainers:
  - name: init-myservice
    image: busybox:1.28
    command: ['sh', '-c', 'until nslookup myservice; do sleep 2; done']
  
  # 主容器
  containers:
  - name: main-container          # 容器名称
    image: nginx:1.21             # 镜像
    imagePullPolicy: IfNotPresent # 镜像拉取策略
    
    # 端口
    ports:
    - containerPort: 80           # 容器端口
      name: http                  # 端口名称
      protocol: TCP               # 协议
    
    # 环境变量
    env:
    - name: MY_ENV_VAR
      value: "hello"
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: SECRET_PASSWORD
      valueFrom:
        secretKeyRef:
          name: my-secret
          key: password
    
    # 资源限制
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
      limits:
        cpu: "200m"
        memory: "256Mi"
    
    # 存活探针
    livenessProbe:
      httpGet:
        path: /healthz
        port: 80
      initialDelaySeconds: 15
      periodSeconds: 10
      timeoutSeconds: 1
      failureThreshold: 3
    
    # 就绪探针
    readinessProbe:
      httpGet:
        path: /ready
        port: 80
      initialDelaySeconds: 5
      periodSeconds: 5
    
    # 卷挂载
    volumeMounts:
    - name: config-volume
      mountPath: /etc/config
    - name: data-volume
      mountPath: /data
  
  # 存储卷定义
  volumes:
  - name: config-volume
    configMap:
      name: my-config
  - name: data-volume
    emptyDir: {}
```

## 创建 Pod 的多种方式

### 方式 1：命令行快速创建

```bash
# 最简单的方式
kubectl run nginx --image=nginx

# 指定端口
kubectl run nginx --image=nginx --port=80

# 指定标签
kubectl run nginx --image=nginx --labels="app=nginx,env=dev"

# 运行命令
kubectl run busybox --image=busybox --command -- sleep 3600

# 交互式运行（调试用）
kubectl run -it debug --image=busybox --rm -- sh

# 生成 YAML（不实际创建）
kubectl run nginx --image=nginx --dry-run=client -o yaml
```

### 方式 2：YAML 文件创建

```bash
# 创建
kubectl apply -f pod.yaml

# 更新
kubectl apply -f pod.yaml

# 删除
kubectl delete -f pod.yaml
```

### 方式 3：从 Deployment 创建（推荐生产使用）

```bash
kubectl create deployment nginx --image=nginx
```

## Pod 生命周期详解

### 阶段（Phase）

```
┌─────────────────────────────────────────────────────────────────┐
│                        Pod 生命周期                              │
│                                                                   │
│   创建请求                                                        │
│      │                                                            │
│      ▼                                                            │
│  ┌─────────┐                                                     │
│  │ Pending │ ← 等待调度、拉取镜像、创建容器                        │
│  └────┬────┘                                                     │
│       │                                                           │
│       ▼                                                           │
│  ┌─────────┐                                                     │
│  │ Running │ ← 至少一个容器正在运行                               │
│  └────┬────┘                                                     │
│       │                                                           │
│       ├──────────────────────┐                                   │
│       │                      │                                    │
│       ▼                      ▼                                    │
│  ┌───────────┐        ┌──────────┐                              │
│  │ Succeeded │        │  Failed  │                              │
│  │  (成功)    │        │  (失败)  │                              │
│  └───────────┘        └──────────┘                              │
│                                                                   │
│  特殊状态：                                                       │
│  ┌─────────┐                                                     │
│  │ Unknown │ ← 无法获取状态（节点失联）                           │
│  └─────────┘                                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 容器状态

```yaml
# 查看容器状态
kubectl get pod my-pod -o jsonpath='{.status.containerStatuses}'

# 三种状态
Waiting:    # 等待中
  reason: ContainerCreating  # 正在创建
  reason: ImagePullBackOff   # 镜像拉取失败
  reason: CrashLoopBackOff   # 容器崩溃循环

Running:    # 运行中
  startedAt: "2024-01-01T00:00:00Z"

Terminated: # 已终止
  exitCode: 0              # 退出码
  reason: Completed        # 正常完成
  reason: Error            # 出错
  reason: OOMKilled        # 内存溢出被杀
```

## 健康检查详解

### 三种探针

```yaml
# 1. livenessProbe - 存活探针
# 检测容器是否存活，失败则重启容器
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 15   # 首次检查延迟
  periodSeconds: 10         # 检查间隔
  timeoutSeconds: 1         # 超时时间
  failureThreshold: 3       # 失败阈值
  successThreshold: 1       # 成功阈值

# 2. readinessProbe - 就绪探针
# 检测容器是否准备好接收流量，失败则从 Service 端点移除
readinessProbe:
  exec:
    command:
    - cat
    - /tmp/ready
  initialDelaySeconds: 5
  periodSeconds: 5

# 3. startupProbe - 启动探针
# 用于慢启动容器，成功后才开始 liveness 和 readiness 检查
startupProbe:
  httpGet:
    path: /startup
    port: 8080
  failureThreshold: 30
  periodSeconds: 10
```

### 探针类型

```yaml
# HTTP GET 探针
httpGet:
  path: /healthz
  port: 8080
  httpHeaders:
  - name: Custom-Header
    value: awesome

# TCP Socket 探针
tcpSocket:
  port: 3306

# Exec 探针（命令）
exec:
  command:
  - cat
  - /tmp/healthy

# gRPC 探针（K8s 1.24+）
grpc:
  port: 50051
```

## Init 容器

Init 容器在主容器启动之前按顺序运行，全部成功后主容器才会启动。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: init-demo
spec:
  initContainers:
  # 1. 等待服务可用
  - name: wait-for-service
    image: busybox:1.28
    command: ['sh', '-c', 'until nslookup myservice.default.svc.cluster.local; do echo waiting; sleep 2; done']
  
  # 2. 下载配置
  - name: download-config
    image: busybox:1.28
    command: ['sh', '-c', 'wget -O /config/app.conf http://config-server/app.conf']
    volumeMounts:
    - name: config
      mountPath: /config
  
  containers:
  - name: app
    image: myapp
    volumeMounts:
    - name: config
      mountPath: /etc/config
  
  volumes:
  - name: config
    emptyDir: {}
```

### Init 容器用途

1. **等待依赖服务就绪**
2. **下载或生成配置文件**
3. **设置文件权限**
4. **数据库初始化**

## 多容器 Pod 模式

### Sidecar 模式（边车）

**Sidecar（边车）不是一种独立的 Kubernetes 资源**，而是多容器 Pod 的一种设计模式：在同一个 Pod 里，再放一个「辅助容器」，和主业务容器一起跑，帮主容器干周边的事。

名字来自摩托车边车：主车负责前进，边车挂在旁边提供额外能力，但两者始终同生共死。

#### 和 Init 容器的区别

| | Init 容器 | Sidecar 容器 |
|---|---|---|
| 运行时机 | 主容器启动**之前**，按顺序跑完就退出 | 与主容器**同时**运行，生命周期一致 |
| 典型职责 | 等依赖、拉配置、初始化 | 日志收集、代理、监控、服务网格 |
| YAML 位置 | `spec.initContainers` | 通常写在 `spec.containers`（和主容器并列） |

> 从 Kubernetes 1.29 起，也可以把 sidecar 写成「带 `restartPolicy: Always` 的 init 容器」，让它在初始化阶段启动后一直活着；常见文档和示例里，仍多写成普通 `containers` 里的第二个容器。

#### 核心特点

1. **共享网络**：和主容器共用一个网络命名空间，可用 `localhost` 互访（这也是网络文档里「同 Pod 多容器」示例的写法来源）
2. **共享存储**：常通过 `emptyDir` 等 Volume 共享日志目录、socket 等
3. **共同调度、共同生命周期**：主容器挂了，整个 Pod（含 sidecar）一起重建；不能单独把 sidecar 调度到别的节点

#### 常见用途

1. **日志收集**：主容器写本地文件，sidecar（如 Fluent Bit / Fluentd）负责采集并推送到集中日志系统
2. **代理 / 服务网格**：如 Istio 的 Envoy sidecar，接管进出流量做 mTLS、限流、可观测
3. **监控与指标**：sidecar 暴露或刮取指标，主应用无需改代码
4. **配置 / 证书热更新**：sidecar 拉取最新配置或证书，主容器只读共享目录

#### 示例：日志收集 sidecar

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: sidecar-example
spec:
  containers:
  # 主容器 - Web 应用
  - name: web-app
    image: nginx
    volumeMounts:
    - name: logs
      mountPath: /var/log/nginx
  
  # Sidecar - 日志收集（name / image 换成真实镜像即可，没有叫 sidecar 的官方镜像）
  - name: log-collector
    image: fluentd
    volumeMounts:
    - name: logs
      mountPath: /var/log/nginx
  
  volumes:
  - name: logs
    emptyDir: {}
```

文档里若写到 `name: sidecar` / `image: sidecar`，那只是占位符，表示「这里放一个边车容器」，需要换成实际镜像（如 `fluent/fluent-bit`、`envoyproxy/envoy` 等）。

#### 运维注意

- `kubectl logs <pod>` 默认只看第一个容器；有 sidecar 时要加 `-c <容器名>`
- `kubectl exec` 同理，必须指定 `-c`，否则可能进错容器
- 资源 requests/limits 要分别给主容器和 sidecar 算，避免 sidecar 把节点资源吃满拖垮主业务

### Ambassador 模式

代理容器，简化主容器对外部服务的访问。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ambassador-example
spec:
  containers:
  - name: app
    image: myapp
    # 应用直接访问 localhost:6379
  
  - name: redis-ambassador
    image: redis-ambassador
    # 代理到实际的 Redis 集群
```

### Adapter 模式

转换容器，标准化主容器的输出。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: adapter-example
spec:
  containers:
  - name: app
    image: myapp
    volumeMounts:
    - name: logs
      mountPath: /var/log/app
  
  - name: log-adapter
    image: log-adapter
    # 将应用日志转换为标准格式
    volumeMounts:
    - name: logs
      mountPath: /var/log/app
  
  volumes:
  - name: logs
    emptyDir: {}
```

## Pod 常用操作命令

### 基本操作

```bash
# 创建 Pod
kubectl apply -f pod.yaml
kubectl run nginx --image=nginx

# 查看 Pod
kubectl get pods
kubectl get pods -o wide                    # 显示更多信息
kubectl get pods -w                         # 监听变化
kubectl get pods --show-labels              # 显示标签

# 查看 Pod 详情
kubectl describe pod nginx

# 查看 Pod 日志
kubectl logs nginx                          # 查看日志
kubectl logs nginx -c container-name        # 指定容器
kubectl logs nginx --previous               # 查看上一个容器的日志
kubectl logs nginx -f                       # 持续输出
kubectl logs nginx --tail=100               # 最后 100 行

# 删除 Pod
kubectl delete pod nginx
kubectl delete pod nginx --grace-period=0 --force  # 强制删除
```

### 调试操作

```bash
# 进入 Pod 执行命令
kubectl exec nginx -- ls /
kubectl exec -it nginx -- /bin/bash         # 交互式 shell
kubectl exec -it nginx -c container -- sh   # 指定容器

# 端口转发（本地调试）
kubectl port-forward nginx 8080:80

# 复制文件
kubectl cp nginx:/etc/nginx/nginx.conf ./nginx.conf
kubectl cp ./config.txt nginx:/tmp/config.txt

# 临时调试容器（K8s 1.25+）
kubectl debug nginx -it --image=busybox
```

### 状态检查

```bash
# 查看 Pod 状态
kubectl get pod nginx -o jsonpath='{.status.phase}'

# 查看 Pod 事件
kubectl get events --field-selector involvedObject.name=nginx

# 查看资源使用（需要 metrics-server）
kubectl top pod nginx

# 查看 Pod YAML
kubectl get pod nginx -o yaml
```

## 常见问题排查

### 问题 1：ImagePullBackOff

```bash
# 查看详情
kubectl describe pod nginx

# 常见原因
1. 镜像名称错误
2. 镜像仓库需要认证
3. 网络问题

# 解决方案
# 检查镜像名称
kubectl get pod nginx -o jsonpath='{.spec.containers[0].image}'

# 配置镜像拉取密钥
kubectl create secret docker-registry regcred \
  --docker-server=<registry> \
  --docker-username=<username> \
  --docker-password=<password>
```

### 问题 2：CrashLoopBackOff

```bash
# 查看日志
kubectl logs nginx --previous

# 常见原因
1. 应用启动失败
2. 配置错误
3. 资源不足

# 调试
kubectl exec -it nginx -- /bin/sh
```

### 问题 3：Pending 状态

```bash
# 查看事件
kubectl describe pod nginx

# 常见原因
1. 资源不足（CPU、内存）
2. 节点选择器无匹配
3. PVC 无法绑定

# 检查节点资源
kubectl describe nodes | grep -A 5 "Allocated resources"
```

## 实践练习

### 练习 1：创建简单 Pod

```bash
# 1. 创建 nginx Pod
kubectl run nginx-pod --image=nginx --port=80

# 2. 查看状态
kubectl get pods -w

# 3. 查看详情
kubectl describe pod nginx-pod

# 4. 访问 Pod
kubectl port-forward nginx-pod 8080:80
# 浏览器访问 http://localhost:8080

# 5. 清理
kubectl delete pod nginx-pod
```

### 练习 2：创建多容器 Pod

创建文件 `multi-container-pod.yaml`：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-container
spec:
  containers:
  - name: nginx
    image: nginx
    ports:
    - containerPort: 80
    volumeMounts:
    - name: shared-data
      mountPath: /usr/share/nginx/html
  
  - name: content-generator
    image: busybox
    command: ["/bin/sh", "-c"]
    args:
      - while true; do
          echo "Hello from content generator! $(date)" > /data/index.html;
          sleep 10;
        done
    volumeMounts:
    - name: shared-data
      mountPath: /data
  
  volumes:
  - name: shared-data
    emptyDir: {}
```

```bash
# 应用
kubectl apply -f multi-container-pod.yaml

# 测试
kubectl port-forward multi-container 8080:80
curl http://localhost:8080

# 清理
kubectl delete -f multi-container-pod.yaml
```

### 练习 3：使用健康检查

创建文件 `health-check-pod.yaml`：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: health-check-demo
spec:
  containers:
  - name: web
    image: nginx
    ports:
    - containerPort: 80
    livenessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 5
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /
        port: 80
      initialDelaySeconds: 2
      periodSeconds: 5
```

```bash
# 应用
kubectl apply -f health-check-pod.yaml

# 查看探针状态
kubectl describe pod health-check-demo

# 清理
kubectl delete -f health-check-pod.yaml
```

## 最佳实践

1. **不要直接使用 Pod**：生产环境使用 Deployment 管理 Pod
2. **设置资源限制**：防止资源耗尽
3. **配置健康检查**：确保应用健康状态可见
4. **使用标签**：方便管理和筛选
5. **日志输出到 stdout**：方便日志收集

## 下一步

- [Deployment - 无状态应用部署](./02-deployment.md)



