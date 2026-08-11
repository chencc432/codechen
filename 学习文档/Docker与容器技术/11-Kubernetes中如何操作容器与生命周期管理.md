# 11 — Kubernetes 中如何操作容器与生命周期管理

> Docker 擅长“把一个容器跑起来”，Kubernetes 擅长“把一组容器长期稳定地跑下去”。

---

## 一、先建立认知：K8s 不是替代 Docker，而是“容器操作系统”

很多同学学到 K8s 时会有一个疑问：  
我已经会 `docker run` 了，为什么还要学 Kubernetes？

答案很简单：**当容器数量从 1 个变成 1000 个时，手工操作会失控**。

```
单机阶段（Docker）：
  你手动 run / stop / restart 容器

集群阶段（Kubernetes）：
  你声明“我要 3 个副本、滚动更新、自动恢复”
  系统自动持续对齐到这个目标状态
```

### 1.1 Docker 与 K8s 的职责边界

| 维度 | 普通容器（Docker 直管） | Kubernetes |
|------|-------------------------|------------|
| 管理对象 | 单机容器 | 集群中的 Pod/工作负载 |
| 操作方式 | 命令式（你一步步执行） | 声明式（你描述目标状态） |
| 故障恢复 | 手工重启 | 控制器自动拉起新实例 |
| 发布策略 | 手工替换容器 | 滚动更新、回滚、分批发布 |
| 扩缩容 | 手工改数量 | 自动扩缩容（HPA） |
| 适用场景 | 本地开发、简单单机服务 | 生产级多节点服务治理 |

---

## 二、K8s 里“操作容器”，为什么总是先操作 Pod？

在 Docker 里你直接操作容器；在 K8s 里你通常不直接管“裸容器”，而是通过 **Pod**。

### 2.1 Pod 是最小调度单元

- 一个 Pod 可以有一个或多个容器（主业务容器 + sidecar）
- Pod 内容器共享网络命名空间（同一个 IP）
- Pod 内容器可以共享 Volume

> 记忆法：Docker 的最小单位是 Container；K8s 的最小单位是 Pod（容器是 Pod 的组成部分）。

### 2.2 常见资源关系图

```
Deployment
   └── ReplicaSet
         └── Pod
               ├── Container A（主应用）
               └── Container B（日志/代理 sidecar）
```

你日常最常操作的是：

- **Pod**：看运行状态、进容器、看日志
- **Deployment**：改镜像、滚动更新、扩缩容
- **Service**：稳定访问入口

---

## 三、K8s 中如何“操作容器”：高频命令实战

下面这组命令可以覆盖 80% 的日常排查与运维动作。

### 3.1 查看和筛选

```bash
# 查看所有 Pod（含节点与IP）
kubectl get pods -o wide

# 查看 Deployment
kubectl get deploy

# 持续观察状态变化
kubectl get pods -w

# 看某个 Pod 的详细事件（调度失败、拉镜像失败等）
kubectl describe pod myapp-7f4b6d7c8b-abcde
```

### 3.2 日志与进入容器

```bash
# 查看 Pod 日志
kubectl logs myapp-7f4b6d7c8b-abcde

# 多容器 Pod 需指定容器名
kubectl logs myapp-7f4b6d7c8b-abcde -c app

# 实时跟踪日志
kubectl logs -f myapp-7f4b6d7c8b-abcde

# 进入容器
kubectl exec -it myapp-7f4b6d7c8b-abcde -- sh
```

### 3.3 临时重启、扩缩容、更新镜像

```bash
# 触发 Deployment 滚动重启（常用于配置变更后重启）
kubectl rollout restart deployment myapp

# 手工扩容到 5 个副本
kubectl scale deployment myapp --replicas=5

# 更新镜像（触发滚动更新）
kubectl set image deployment/myapp app=myrepo/myapp:v2

# 查看发布进度
kubectl rollout status deployment myapp

# 回滚到上一个版本
kubectl rollout undo deployment myapp
```

### 3.4 删除行为差异（重点）

```bash
# 删除 Pod：控制器会很快补一个新 Pod（如果它属于 Deployment）
kubectl delete pod myapp-7f4b6d7c8b-abcde

# 删除 Deployment：控制器消失，副本整体下线
kubectl delete deployment myapp
```

**关键区别**：在 K8s 中，很多对象是“可再生的”。你删掉某个 Pod，系统可能会自动再创建一个，这是正常行为。

---

## 四、容器生命周期在 K8s 中到底怎么走？

### 4.1 Pod 生命周期主流程

```
Pending  ->  Running  ->  Succeeded / Failed
              │
              └-> CrashLoopBackOff（反复崩溃重启）
```

- **Pending**：还在调度或拉镜像
- **Running**：Pod 已运行，至少一个容器在工作
- **Succeeded**：任务成功结束（常见于 Job）
- **Failed**：任务失败结束
- **CrashLoopBackOff**：容器反复启动失败，进入退避重试

### 4.2 容器生命周期钩子（Lifecycle Hooks）

K8s 提供两个非常实用的钩子：

- `postStart`：容器启动后执行
- `preStop`：容器终止前执行（优雅下线关键）

示例：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: lifecycle-demo
spec:
  containers:
    - name: app
      image: nginx:1.25
      lifecycle:
        postStart:
          exec:
            command: ["sh", "-c", "echo started > /tmp/state.txt"]
        preStop:
          exec:
            command: ["sh", "-c", "sleep 5"]
```

`preStop + terminationGracePeriodSeconds` 常用于：

- 从负载均衡摘流
- 等待请求处理完成
- 刷盘/上报状态后再退出

### 4.3 优雅终止流程（生产必会）

K8s 终止 Pod 的典型流程：

1. Pod 被标记 `Terminating`
2. 执行 `preStop` 钩子
3. 发送 `SIGTERM` 给容器主进程
4. 等待宽限期（`terminationGracePeriodSeconds`）
5. 超时仍未退出则 `SIGKILL`

> 如果应用不处理 `SIGTERM`，发布或缩容时就容易出现请求中断。

---

## 五、K8s 如何“管理”容器生命周期：控制器才是核心

Docker 偏“你来管容器”；K8s 偏“控制器来持续纠偏”。

### 5.1 Deployment：无状态应用生命周期管家

它负责：

- 保证副本数达标（少了就补）
- 滚动更新（逐步替换旧 Pod）
- 发布失败可回滚

示例（含探针与滚动策略）：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      terminationGracePeriodSeconds: 30
      containers:
        - name: app
          image: nginx:1.25
          ports:
            - containerPort: 80
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

### 5.2 三种探针：生命周期管理的“自动化开关”

| 探针 | 作用 | 失败后果 |
|------|------|----------|
| livenessProbe | 判断容器是否“活着” | 失败会重启容器 |
| readinessProbe | 判断是否可接流量 | 失败会从 Service 端点摘除 |
| startupProbe | 给慢启动应用缓冲期 | 启动探针没过前不做 liveness 判定 |

**对比 Docker**：Docker 有 `HEALTHCHECK`，但不会像 K8s 一样与服务流量路由、滚动发布、自动恢复形成完整闭环。

### 5.3 不同工作负载的生命周期管理对象

| 资源 | 适用场景 | 生命周期特征 |
|------|----------|--------------|
| Deployment | 无状态服务 | 持续运行、可滚动更新 |
| StatefulSet | 有状态服务（DB、队列） | 稳定网络标识、顺序发布 |
| DaemonSet | 每节点一个 Pod | 跟随节点生命周期 |
| Job/CronJob | 批处理/定时任务 | 运行完成即结束 |

---

## 六、普通容器 vs K8s 容器生命周期：一张表看懂

| 生命周期动作 | 普通容器（Docker） | K8s（推荐做法） |
|--------------|--------------------|-----------------|
| 创建 | `docker run` | `kubectl apply -f deploy.yaml` |
| 观测状态 | `docker ps` | `kubectl get pods/deploy` |
| 查详情 | `docker inspect` | `kubectl describe` |
| 看日志 | `docker logs` | `kubectl logs` |
| 进入容器 | `docker exec` | `kubectl exec` |
| 重启 | `docker restart` | `rollout restart` 或让探针自动恢复 |
| 扩容 | 手工多开容器 | `kubectl scale` / HPA |
| 更新版本 | 停旧开新（常中断） | 滚动更新（尽量无感） |
| 故障自愈 | 依赖 restart policy | 控制器 + 探针 +调度自动恢复 |
| 节点故障 | 需人工迁移 | 调度到其他健康节点 |

---

## 七、一个完整例子：从“跑起来”到“可持续运行”

### 7.1 在 Docker 里你可能这样做

```bash
docker run -d --name web -p 80:80 nginx:1.25
```

问题是：

- 容器挂了怎么办？（你可能很晚才发现）
- 要升级版本怎么办？（容易中断）
- 要扩容怎么办？（命令可重复但难统一管理）

### 7.2 在 K8s 里你这样做

1. 写好 Deployment + Service YAML
2. `kubectl apply -f` 提交目标状态
3. 用 `rollout status` 观察发布
4. 用探针和控制器确保自愈
5. 需要扩缩容时改副本数或启用 HPA

这背后体现的是思维转换：

```
命令式思维：我现在执行什么命令？
声明式思维：我希望系统长期保持什么状态？
```

---

## 八、实战建议：如何把“容器思维”升级到“K8s 思维”

1. **先稳住基础**：镜像、容器、网络、存储这些 Docker 能力仍是 K8s 前提  
2. **不要执着于“单 Pod 调试”**：K8s 的核心是控制器与期望状态  
3. **优先掌握发布安全**：滚动更新、回滚、readiness 探针  
4. **把优雅终止当成必修课**：`SIGTERM`、`preStop`、宽限期  
5. **学会看事件流**：很多问题不在日志，在 `kubectl describe` 的 Events  
6. **从 Deployment 入门，再学 StatefulSet/Job**：按场景逐步升级

---

## 九、本章小结

```
✅ K8s 操作容器的入口是 Pod，但管理核心是控制器（Deployment 等）
✅ 容器生命周期在 K8s 中不仅是启动/停止，还包含探针、自愈、滚动发布、回滚
✅ preStop + SIGTERM + 宽限期是优雅下线的关键机制
✅ 对比普通容器，K8s 把“手工运维”升级成“声明式自动化运维”
✅ 从 docker run 到 kubectl apply，本质是从命令式走向目标状态管理
```

---

> 下一步建议：结合你现有 `kubernetes/k8s-learning` 目录，把本章里的 Deployment 与探针示例实际跑一遍，体感会立刻提升。
