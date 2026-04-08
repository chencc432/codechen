# 🧭 Kubernetes 调度机制与策略

## 为什么调度值得单独学

很多人刚接触 Kubernetes 时，会觉得 Pod 提交以后系统“自己会找地方跑起来”。这句话不算错，但太粗了。

更准确地说，Kubernetes 调度器在回答一个很具体的问题：

> 这个 Pod 应该被放到哪一个节点，才能同时满足约束、资源、策略和整体集群效率？

调度问题一旦理解不清，后面几类现象都会很难排查：

- Pod 一直 `Pending`
- 明明有节点，但就是不调度
- Pod 总跑到“不想让它去”的机器上
- 同类副本都扎堆在同一台机器
- GPU、SSD、数据库等专用节点被普通业务抢占

所以调度不是边缘知识，而是 Kubernetes 的核心运行机制之一。

## 先建立一个整体心智模型

调度器工作时，可以把它想成 3 步：

1. **先排除不能放的节点**
2. **再给剩下的节点打分**
3. **最后选择最合适的节点**

也就是：

```text
待调度 Pod
  -> 过滤（哪些节点绝对不行）
  -> 打分（哪些节点更合适）
  -> 绑定（最终选中一个节点）
```

## 1. 调度器到底看什么

调度器通常会综合考虑以下因素：

- 节点剩余 CPU、内存是否足够
- Pod 的 `requests` 是否满足
- `nodeSelector` 是否匹配
- `nodeAffinity` 是否满足
- Pod 亲和性 / 反亲和性是否满足
- 是否存在污点以及 Pod 是否容忍
- 某些端口、卷、拓扑约束是否冲突
- 高可用、负载均衡、镜像本地性等偏好

这意味着：

- 调度不是只看资源
- 也不是只看标签
- 而是“硬约束 + 软偏好 + 集群当前状态”的综合结果

## 2. requests 为什么会影响调度

很多人知道 `resources.requests`，但不知道它对调度器有多重要。

最关键的一句是：

> 调度器主要看 `requests` 来判断节点是否“装得下”这个 Pod。

例如：

```yaml
resources:
  requests:
    cpu: "500m"
    memory: "512Mi"
  limits:
    cpu: "1"
    memory: "1Gi"
```

在调度阶段，更关键的是：

- 这个节点能不能至少提供 `500m CPU + 512Mi 内存`

所以常见现象是：

- 节点总资源很多
- 但由于可分配资源不足
- Pod 仍然会一直 `Pending`

## 3. 最基础的调度控制：nodeSelector

`nodeSelector` 是最简单的节点约束方式，本质上就是：

> “Pod 只能调度到带某些标签的节点上。”

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  nodeSelector:
    disktype: ssd
  containers:
  - name: nginx
    image: nginx
```

### 3.1 适用场景

- 指定 SSD 节点
- 指定 GPU 节点
- 指定 Linux/Windows 节点
- 指定某类业务专用节点

### 3.2 优缺点

优点：

- 简单
- 易读
- 易上手

缺点：

- 只能做精确匹配
- 不能表达“最好而不是必须”
- 复杂场景扩展性差

## 4. 更灵活的方式：Node Affinity

`nodeAffinity` 可以理解为 `nodeSelector` 的增强版。

它支持两种模式：

- **硬性要求**：必须满足
- **软性偏好**：最好满足

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: affinity-demo
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: disktype
            operator: In
            values:
            - ssd
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        preference:
          matchExpressions:
          - key: zone
            operator: In
            values:
            - zone-a
  containers:
  - name: nginx
    image: nginx
```

### 4.1 最值得记住的区别

- `required...`：不满足就不能调度
- `preferred...`：满足更好，不满足也可能调度

### 4.2 常见操作符

| 操作符 | 含义 |
|--------|------|
| `In` | 在给定列表中 |
| `NotIn` | 不在给定列表中 |
| `Exists` | 标签存在 |
| `DoesNotExist` | 标签不存在 |
| `Gt` | 大于 |
| `Lt` | 小于 |

## 5. Pod 亲和性与反亲和性

如果说 `nodeAffinity` 是“Pod 想靠近什么样的节点”，那么 `podAffinity` / `podAntiAffinity` 更像是：

- 想靠近哪些 Pod
- 想远离哪些 Pod

### 5.1 Pod Affinity

适合“放近一点”的场景：

- Web 和缓存放同一节点，减少网络延迟
- 某些强耦合服务希望在同一可用区

### 5.2 Pod Anti-Affinity

适合“打散部署”的场景：

- 多副本不要集中在同一节点
- 多可用区分布，提高可用性

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
spec:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: web
          topologyKey: kubernetes.io/hostname
  containers:
  - name: nginx
    image: nginx
```

### 5.3 topologyKey 是什么

它定义了“靠近”或“分开”的粒度。

常见值：

- `kubernetes.io/hostname`：节点级别
- `topology.kubernetes.io/zone`：可用区级别
- `topology.kubernetes.io/region`：区域级别

最常用、也最容易理解的是：

- 同一节点：`kubernetes.io/hostname`

## 6. 污点与容忍度

这是调度里最常被误解的一组概念。

### 6.1 污点（Taint）

污点加在节点上，表达的是：

> “这台节点不欢迎普通 Pod。”

```bash
kubectl taint nodes node1 gpu=true:NoSchedule
```

### 6.2 容忍度（Toleration）

容忍度写在 Pod 上，表达的是：

> “我可以接受这种污点，不要把我拦在门外。”

```yaml
tolerations:
- key: "gpu"
  operator: "Equal"
  value: "true"
  effect: "NoSchedule"
```

### 6.3 最常见的误区

`toleration` 只是“允许进入”，不是“强制去那里”。

如果你希望 Pod 一定跑到 GPU 节点，通常要组合：

- 节点打污点
- Pod 配容忍度
- 再加 `nodeSelector` 或 `nodeAffinity`

## 7. 调度优先级和抢占

在集群资源紧张时，并不是所有 Pod 地位都一样。

Kubernetes 可以通过优先级类（PriorityClass）表达：

- 哪些 Pod 更重要
- 资源不够时谁优先被调度
- 极端情况下谁可以抢占低优先级 Pod 的资源

### 7.1 什么时候会用到

- 核心系统组件优先级高于普通业务
- 生产业务高于测试业务
- 监控、网关、关键控制面组件要优先活下来

### 7.2 抢占需要谨慎

抢占可以解决高优先级业务上不去的问题，但也可能带来：

- 低优先级业务被驱逐
- 突发连锁影响
- 调度行为更难预测

所以生产里通常需要明确优先级设计，而不是随意设置。

## 8. 常见调度场景

### 8.1 GPU 节点

目标：

- 普通 Pod 不要进 GPU 节点
- 需要 GPU 的业务才能进去

常见做法：

1. 给节点打标签：`accelerator=nvidia`
2. 给节点打污点：`gpu=true:NoSchedule`
3. GPU Pod 添加：
   - `tolerations`
   - `nodeSelector` 或 `nodeAffinity`

### 8.2 高可用多副本

目标：

- 多个副本不要集中到一个节点

常见做法：

- 使用 `podAntiAffinity`
- 或者结合拓扑分布策略

### 8.3 数据库与状态服务

目标：

- 固定在某类高性能节点
- 与普通业务隔离

常见做法：

- 节点标签
- 节点污点
- 严格的 `requiredDuringScheduling...`

## 9. Pod 一直 Pending 时怎么查

这是最常见的调度排障问题。

### 9.1 第一优先级命令

```bash
kubectl describe pod <pod-name>
```

重点看：

- `Events`
- `Warning`
- `FailedScheduling`

### 9.2 常见失败原因

| 现象 | 常见原因 |
|------|----------|
| `0/3 nodes are available: insufficient cpu` | 节点 CPU 不足 |
| `node(s) didn't match node selector` | `nodeSelector` 不匹配 |
| `node(s) didn't match Pod's node affinity` | 节点亲和性过严 |
| `node(s) had taint ... that the pod didn't tolerate` | 没有容忍度 |
| PVC 相关错误 | 存储未准备好 |

### 9.3 排查顺序建议

1. `kubectl describe pod`
2. `kubectl get nodes --show-labels`
3. 检查资源请求是否过大
4. 检查 `nodeSelector` / `affinity`
5. 检查 `tolerations`
6. 如果用了卷，再检查 PVC / PV

## 10. 一个综合调度示例

下面这个例子同时表达了：

- 必须是 SSD 节点
- 优先选择 `zone-a`
- 能容忍专用节点污点

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: scheduling-complete-demo
spec:
  nodeSelector:
    disktype: ssd
  tolerations:
  - key: "dedicated"
    operator: "Equal"
    value: "backend"
    effect: "NoSchedule"
  affinity:
    nodeAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 80
        preference:
          matchExpressions:
          - key: topology.kubernetes.io/zone
            operator: In
            values:
            - zone-a
  containers:
  - name: app
    image: nginx
    resources:
      requests:
        cpu: "250m"
        memory: "256Mi"
```

## 11. 最佳实践

- 先从简单规则开始，不要一上来写过度复杂的亲和性
- 节点标签命名要稳定、统一、可预期
- 专用节点建议同时使用“标签 + 污点 + 容忍度”
- 副本型工作负载优先考虑打散策略，提升可用性
- 调度问题优先看 `describe pod` 的事件，而不是只盯着 YAML
- `requests` 要真实，过大和过小都会让调度质量变差

## 12. 一页总结

- 调度器核心任务是“为 Pod 选择最合适的节点”
- `requests` 是调度阶段最重要的资源依据
- `nodeSelector` 简单直接，`nodeAffinity` 更灵活
- `podAntiAffinity` 很适合做高可用打散
- 污点是节点的拒绝策略，容忍度是 Pod 的准入许可
- `Pending` 不等于系统坏了，通常是调度条件不满足
- 排查调度问题最重要的入口是 `kubectl describe pod`

## 下一步

- [安全与权限控制](./04-security.md)
- [核心概念与术语](../01-basics/03-concepts.md)
