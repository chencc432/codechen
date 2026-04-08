# 🚀 项目一：部署微服务应用

## 项目目标

这个项目的目标是把前面学过的 Kubernetes 核心对象真正串起来，完成一个“小而完整”的微服务应用部署链路。

你会接触到的典型角色包括：

- 前端服务（Frontend）
- 后端 API 服务（Backend）
- 数据库（MySQL）
- 缓存（Redis）
- 可选的入口层（Ingress）

更重要的是，你会在这个项目里真正体会到：

- `Deployment`、`Service`、`ConfigMap`、`Secret`、`Volume` 为什么要一起出现
- 为什么应用不是“跑起来就结束”，而是要考虑配置、依赖、可访问性和排障
- 一个典型业务系统在 Kubernetes 中是如何拆层部署的

## 建议你在这个项目里重点理解什么

做这个项目时，建议你不要只关注命令能否执行成功，而是优先关注下面这些问题：

1. 前端、后端、数据库、缓存为什么要拆成不同资源
2. 哪些服务应该使用 `Deployment`，哪些更适合配持久卷
3. Service 在这里到底解决了什么问题
4. 当后端访问 MySQL 或 Redis 失败时，应该先查什么

## 架构图

```text
                    Internet
                        │
                        ▼
                   ┌─────────┐
                   │ Ingress │
                   └────┬────┘
                        │
                        ▼
            ┌───────────────────────┐
            │    Frontend Service   │
            │      (Nginx + SPA)    │
            └───────────┬───────────┘
                        │
                        ▼
            ┌───────────────────────┐
            │    Backend Service    │
            │      (API Server)     │
            └───────────┬───────────┘
                   ┌────┴────┐
                   │         │
                   ▼         ▼
           ┌───────────┐ ┌───────────┐
           │   MySQL   │ │   Redis   │
           └───────────┘ └───────────┘
```

## 这套架构各层分别负责什么

### Frontend

通常负责：

- 提供浏览器访问界面
- 处理静态资源
- 调用 Backend API

在 Kubernetes 里一般会使用：

- `Deployment`
- `Service`
- 如果要对外暴露，再配 `Ingress`

### Backend

通常负责：

- 提供 API
- 读取配置
- 连接 MySQL 和 Redis
- 实现业务逻辑

在 Kubernetes 里一般会使用：

- `Deployment`
- `Service`
- `ConfigMap`
- `Secret`

### MySQL

通常负责：

- 业务数据持久化
- 提供结构化数据存储

在 Kubernetes 里至少要考虑：

- 持久化卷
- 密码和账号管理
- 是否需要稳定网络标识

### Redis

通常负责：

- 缓存
- 会话
- 临时状态

在 Kubernetes 里需要思考：

- 是否只做开发测试缓存
- 是否需要持久化
- 是否可以接受重建丢失

## 推荐的目录结构

如果你要把这个项目继续补造成完整示例，建议采用类似结构：

```text
01-microservice-deploy/
├── README.md
├── namespace.yaml
├── mysql/
│   ├── secret.yaml
│   ├── pvc.yaml
│   ├── deployment.yaml
│   └── service.yaml
├── redis/
│   ├── deployment.yaml
│   └── service.yaml
├── backend/
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── deployment.yaml
│   └── service.yaml
├── frontend/
│   ├── configmap.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── ingress.yaml
```

## 推荐部署顺序

部署微服务系统时，不建议一股脑全部 `apply`，而是按依赖顺序部署：

### 1. 创建命名空间

```bash
kubectl apply -f namespace.yaml
```

作用：

- 把项目资源隔离到一个独立环境里
- 后续查询、删除、排障更清晰

### 2. 部署 MySQL

```bash
kubectl apply -f mysql/
```

部署数据库时要优先检查：

- 密码是否通过 `Secret` 提供
- 是否配置了持久卷
- Pod 是否 Ready
- Service 是否创建成功

### 3. 部署 Redis

```bash
kubectl apply -f redis/
```

Redis 常见学习重点：

- Service 名称是否与后端配置一致
- 是否需要持久化
- 只是测试缓存还是实际业务依赖

### 4. 部署 Backend

```bash
kubectl apply -f backend/
```

部署后端时重点关注：

- 环境变量是否正确
- 是否能解析 MySQL / Redis 的 Service 名称
- 探针是否通过
- 日志里是否有连接失败信息

### 5. 部署 Frontend

```bash
kubectl apply -f frontend/
```

前端最常见的问题往往不是 Pod 起不来，而是：

- 后端地址配置不对
- 静态资源路径有误
- Ingress 路由不匹配

### 6. 配置 Ingress（可选）

```bash
kubectl apply -f ingress.yaml
```

如果你希望从浏览器以域名或统一入口访问系统，这一步很有价值。

## 部署完成后应该怎么验证

### 基础验证

```bash
# 查看所有资源
kubectl get all -n microservice

# 检查 Pod 状态
kubectl get pods -n microservice

# 查看 Service
kubectl get svc -n microservice
```

### 前端访问验证

```bash
kubectl port-forward svc/frontend -n microservice 8080:80
curl http://localhost:8080
```

### 后端连通验证

可以进一步检查：

```bash
kubectl logs -n microservice deploy/backend
kubectl describe pod -n microservice <backend-pod-name>
```

重点看：

- 是否成功连接 MySQL
- 是否成功连接 Redis
- 是否存在配置缺失、探针失败、DNS 解析失败

## 一条非常实用的排障顺序

如果项目部署后不能正常工作，建议按这个顺序查：

1. Pod 是否全部处于 `Running` / `Ready`
2. Service 是否都创建成功
3. 后端是否能访问数据库和缓存
4. 前端是否能访问后端
5. Ingress 是否正确转发流量

不要一开始就钻进应用代码，先把基础设施链路查清楚。

## 典型问题清单

### 问题 1：MySQL 一直起不来

优先检查：

- PVC 是否绑定成功
- 初始化环境变量是否正确
- Secret 是否存在
- 镜像启动参数是否正确

### 问题 2：Backend 起了，但接口报错

优先检查：

- MySQL / Redis Service 名称是否写对
- 端口是否匹配
- ConfigMap / Secret 是否挂载或注入成功
- 应用日志中是否有连接超时或认证失败

### 问题 3：Frontend 页面打不开

优先检查：

- Frontend Service 是否可达
- Ingress 是否生效
- 前端静态资源是否正确构建
- 前端调用的 API 地址是否写错

### 问题 4：服务名能解析，但请求超时

优先检查：

- 后端容器是否真正监听目标端口
- `targetPort` 是否正确
- Pod 是否 `Ready`
- 是否有网络策略阻断

## 你会在这个项目里自然复习到哪些知识

完成这个项目时，你实际上会把前面几类文档都串起来：

- `Namespace`：资源隔离
- `Deployment`：无状态服务部署
- `Service`：服务发现
- `ConfigMap / Secret`：配置与敏感信息
- `Volume / PVC`：数据库持久化
- `Ingress`：外部流量入口
- `kubectl logs / describe / port-forward`：排障与验证

## 最佳实践

- 先把内部链路打通，再考虑外部暴露
- 业务配置放 `ConfigMap`，敏感信息放 `Secret`
- 数据库类组件优先考虑持久化
- 先用 `port-forward` 验证服务可用性，再做 Ingress
- 每部署一层就做一次验证，不要全部部署完才统一排查

## 清理

```bash
kubectl delete namespace microservice
```

## 做完这个项目后你应该掌握什么

完成后，你至少应该能清楚说出：

- 一个简单微服务系统在 Kubernetes 中通常怎么拆层
- 业务流量和依赖服务是如何通过 Service 连接起来的
- 哪些组件更适合无状态部署，哪些必须考虑持久化
- 出问题时应该如何按“前端 -> 后端 -> 数据库/缓存”逐层排查

## 下一步

- [项目二：日志收集系统](../02-logging-system/README.md)
- [项目三：监控告警系统](../03-monitoring/README.md)



