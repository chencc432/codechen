# 📈 项目三：监控告警系统

## 项目目标

这个项目的目标是带你理解，在 Kubernetes 中如何构建一套基础的监控与告警体系，而不是只知道“装个 Prometheus 就结束了”。

你需要理解的是：

- 为什么监控在集群里是必需品，而不是可选项
- 指标从应用到监控系统是怎么流动的
- Service、ServiceMonitor、Exporter、Alertmanager 这些角色分别在干什么
- 什么时候看指标，什么时候看日志，什么时候看事件

## 监控系统要解决什么问题

一套监控系统通常至少要回答下面这些问题：

- 集群资源是否健康
- 节点是否异常
- Pod 是否频繁重启
- 接口延迟是否升高
- 错误率是否变高
- 存储、网络、CPU、内存是否接近瓶颈
- 出现异常时应该通知谁

如果这些问题答不上来，线上问题往往只能“等用户报错”。

## 先建立整体认知

在 Kubernetes 里，监控一般可以拆成 4 层：

1. **指标暴露层**：应用或组件暴露 metrics
2. **指标采集层**：Prometheus 周期性抓取指标
3. **展示分析层**：Grafana 等工具展示仪表盘
4. **告警通知层**：Alertmanager 根据规则发送告警

可以把它想成一条链：

```text
应用 / 节点 / 集群组件
  -> 暴露指标
  -> Prometheus 抓取
  -> 存储与查询
  -> Grafana 展示
  -> Alertmanager 告警
```

## 典型架构图

```text
┌───────────────────────────────────────────────────────────────┐
│                      Kubernetes Cluster                       │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│   App Pods         Node Exporter         kube-state-metrics   │
│      │                  │                        │             │
│      └──────────┬───────┴──────────┬────────────┘             │
│                 ▼                  ▼                          │
│             ┌────────────────────────────┐                    │
│             │         Prometheus         │                    │
│             └──────────────┬─────────────┘                    │
│                            ▼                                  │
│                      ┌────────────┐                           │
│                      │ Alertmanager│                          │
│                      └────────────┘                           │
│                            │                                  │
│                            ▼                                  │
│                     飞书 / 邮件 / Webhook                      │
│                                                               │
│             ┌────────────────────────────┐                    │
│             │          Grafana           │                    │
│             └────────────────────────────┘                    │
└───────────────────────────────────────────────────────────────┘
```

## 会用到哪些组件

| 组件 | 作用 |
|------|------|
| `Prometheus` | 抓取、存储、查询指标 |
| `Grafana` | 展示仪表盘 |
| `Alertmanager` | 聚合和发送告警 |
| `node-exporter` | 暴露节点级指标 |
| `kube-state-metrics` | 暴露 Kubernetes 对象状态指标 |
| 应用自身 metrics 接口 | 暴露业务指标、应用指标 |

## 推荐学习重点

做这个项目时，建议不要只关注怎么安装，更要理解：

1. 指标和日志的分工有什么不同
2. 为什么一个应用需要主动暴露 metrics
3. 为什么仅靠 `kubectl top` 不够
4. 告警规则应该怎么避免“全是噪音”

## 指标、日志、事件的区别

这是监控体系里非常值得先讲清楚的一件事。

| 类型 | 适合回答什么问题 |
|------|------------------|
| 指标（Metrics） | 系统现在是否健康，趋势是否异常 |
| 日志（Logs） | 某次具体错误发生了什么 |
| 事件（Events） | Kubernetes 最近对资源做了什么 |

实战里常见组合是：

- 先看指标，发现哪一层异常
- 再看事件，确认 K8s 最近是否有调度、驱逐、重启等动作
- 最后看日志，定位具体错误细节

## 最小监控体系应该包含什么

如果你要做一个“够用”的基础监控系统，建议至少覆盖：

### 1. 集群资源

- 节点 CPU / 内存 / 磁盘
- Pod 重启次数
- 容器资源消耗
- 节点是否 Ready

### 2. Kubernetes 对象状态

- Deployment 可用副本数
- Pod Pending 数量
- PVC 状态
- Job 是否失败

### 3. 业务指标

- 请求量
- 错误率
- 响应时间
- 队列堆积
- 缓存命中率

## 推荐的目录结构

如果你后续要把这个项目做成实际可部署示例，建议用类似结构：

```text
03-monitoring/
├── README.md
├── namespace.yaml
├── prometheus/
│   ├── configmap.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── pvc.yaml
├── grafana/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── ingress.yaml
├── alertmanager/
│   ├── configmap.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── exporters/
    ├── node-exporter-daemonset.yaml
    └── kube-state-metrics.yaml
```

## 推荐部署顺序

1. 创建命名空间
2. 部署 Prometheus
3. 部署节点和集群指标采集组件
4. 部署 Grafana
5. 配置基础仪表盘
6. 部署 Alertmanager
7. 配置告警规则和通知渠道

## 一个最小 metrics 暴露示例

假设你的应用自己提供 `/metrics` 接口：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: app-metrics
  labels:
    app: myapp
spec:
  selector:
    app: myapp
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
```

这里的关键不是“这个 YAML 会不会背”，而是你要知道：

- Prometheus 必须先能访问到一个稳定地址
- 应用也必须真实暴露 metrics 接口
- 如果接口不存在，监控系统是抓不到业务指标的

## 为什么 node-exporter 和 kube-state-metrics 经常一起出现

因为它们关注的是两类完全不同的信息：

### node-exporter

更关注节点本身：

- CPU
- 内存
- 文件系统
- 网络
- 系统负载

### kube-state-metrics

更关注 Kubernetes 对象状态：

- Deployment 副本数
- Pod 状态
- Job 状态
- HPA 状态

简单说：

- `node-exporter` 回答“机器怎么样”
- `kube-state-metrics` 回答“Kubernetes 资源对象怎么样”

## 告警应该怎么理解

告警系统不是“只要有指标就告警”，而是：

> 当关键指标持续异常，并且达到了需要人处理的阈值时，系统应该主动通知。

一个好的告警系统至少要做到：

- 能及时发现问题
- 不制造大量无意义告警
- 能指出大概是哪一层异常
- 能让值班人员快速接手处理

## 常见告警分类

### 平台层

- 节点 NotReady
- 磁盘空间不足
- kubelet 异常
- 控制面组件不可用

### 容器层

- Pod 重启过多
- OOMKilled
- 容器 CPU / 内存持续过高

### 业务层

- 接口错误率升高
- 响应时间飙升
- 请求成功率下降
- 数据同步延迟变大

## 常见排查问题

### 问题 1：Prometheus 抓不到目标

优先检查：

- Service 是否存在
- Pod 是否 Ready
- metrics 端口是否写对
- 应用是否真的暴露了 `/metrics`
- 网络是否可达

### 问题 2：Grafana 打不开图表

优先检查：

- 数据源是否配置正确
- Prometheus 查询是否有结果
- 时间范围是否合理
- 指标名是否写对

### 问题 3：告警太多

优先检查：

- 阈值是否过于敏感
- 是否缺少持续时间窗口
- 是否把瞬时抖动也当成异常
- 是否缺少降噪和分组

### 问题 4：有问题但没告警

优先检查：

- 规则是否加载成功
- 指标是否真的存在
- 告警是否被静默
- 通知通道是否可用

## 最佳实践

- 平台指标、应用指标、业务指标要分层看待
- 告警规则尽量围绕“用户影响”和“处理必要性”设计
- 不要只做仪表盘，不做告警
- 也不要只做告警，不做定位上下文
- 仪表盘应能支持从“集群 -> 节点 -> Pod -> 应用”逐层下钻
- 核心监控组件本身也要被监控

## 做完这个项目后你应该掌握什么

完成这个项目后，至少应该能回答这些问题：

- 指标、日志、事件有什么区别
- 为什么一个集群通常不止一种 exporter
- Prometheus、Grafana、Alertmanager 分别负责什么
- 监控为什么必须与告警、排障一起设计

## 下一步

- [项目二：日志收集系统](../02-logging-system/README.md)
- [故障排查与调试](../../03-practice/04-troubleshooting.md)
- [安全与权限控制](../../04-advanced/04-security.md)
