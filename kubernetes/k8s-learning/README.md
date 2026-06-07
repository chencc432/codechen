# 🚀 Kubernetes 完整学习指南

> 面向初学者、进阶学习者和实际使用者的一套 Kubernetes 学习文档。  
> 目标不是只会背概念，而是建立体系化认知，能看懂 YAML、能排查问题、能逐步走向实战。

## 这套文档适合谁

如果你属于下面任意一种情况，这套文档都会比较适合：

- 刚接触 Kubernetes，想知道这些资源到底是怎么串起来的
- 会写一点 YAML，但经常不确定字段为什么这么写
- 能把应用跑起来，但遇到 `Pending`、`CrashLoopBackOff`、服务不通时不会排查
- 想从使用 Kubernetes 继续深入到调度、网络、安全、client-go、控制器开发

## 你会学到什么

这套文档会覆盖 Kubernetes 学习最核心的 6 个层面：

1. **基础认知**：Kubernetes 是什么，为什么需要它，架构如何分层
2. **资源模型**：Pod、Deployment、Service、ConfigMap、Volume、Namespace 等核心资源怎么理解
3. **日常操作**：如何用 `kubectl`、如何写 YAML、如何做常见运维
4. **排障方法**：出现问题时应该先查哪里，再查哪里
5. **进阶原理**：网络、存储、调度、安全、Ingress
6. **编程扩展**：client-go、Informer、自定义控制器
7. **平台扩展**：CRD、自定义控制器、Operator、资源设计与版本演进
8. **存储运维**：卷、存储类、CSI、扩容、迁移、备份与恢复
9. **工作流编排**：Argo Workflow（DAG/Steps、参数与制品、CronWorkflow、生产实践）
10. **控制器深度**：Informer 内部、Workqueue、Reconcile、Leader Election、Finalizer 等
11. **网络工程**：企业网络架构全景、VPC 与内网互通、公网暴露、办公网与堡垒机、网络转发与排障

## 推荐学习方式

这套文档不是单纯按目录从上看到下，而是建议你按下面顺序学习：

### 路线 1：零基础入门

适合第一次系统接触 Kubernetes 的同学：

1. [环境搭建指南](./00-setup/environment.md)
2. [Kubernetes 概述与架构](./01-basics/01-overview.md)
3. [核心组件详解](./01-basics/02-components.md)
4. [核心概念与术语](./01-basics/03-concepts.md)
5. [Pod - 最小调度单元](./02-resources/01-pod.md)
6. [Deployment - 无状态应用部署](./02-resources/02-deployment.md)
7. [Service - 服务发现与负载均衡](./02-resources/03-service.md)
8. [kubectl 命令完全手册](./03-practice/01-kubectl-commands.md)
9. [YAML 编写规范与技巧](./03-practice/02-yaml-guide.md)

### 路线 2：工作实战导向

适合已经会基本使用，希望快速提升排障和运维能力的同学：

1. [kubectl 命令完全手册](./03-practice/01-kubectl-commands.md)
2. [常见运维操作指南](./03-practice/03-operations.md)
3. [故障排查与调试](./03-practice/04-troubleshooting.md)
4. [Kubernetes 运维指令大全](./09-ops-command-handbook/README.md)
5. [Service - 服务发现与负载均衡](./02-resources/03-service.md)
6. [Kubernetes 网络模型](./04-advanced/01-networking.md)
7. [调度机制与策略](./04-advanced/03-scheduling.md)
8. [安全与权限控制](./04-advanced/04-security.md)

### 路线 3：开发者进阶

适合想写自动化工具、Operator 或控制器的同学：

1. [client-go 入门](./05-client-go/01-introduction.md)
2. [客户端配置与连接](./05-client-go/02-client-setup.md)
3. [资源的 CRUD 操作](./05-client-go/03-crud-operations.md)
4. [Informer 机制详解](./05-client-go/04-informer.md)
5. [实战项目：自定义控制器](./05-client-go/05-controller-demo.md)
6. [Kubernetes 自定义资源专题](./07-custom-resources/README.md)
7. [控制器深度专题](./11-controller-deep-dive/README.md)

### 路线 5：工作流与 Pipeline

适合需要在 Kubernetes 上跑 CI/CD、ML Pipeline、批处理流水线的同学：

1. [Argo Workflow 概述与核心概念](./10-argo-workflow/01-argo-overview.md)
2. [安装与快速上手](./10-argo-workflow/02-installation-and-quickstart.md)
3. [Workflow Spec 与 Template 类型详解](./10-argo-workflow/03-templates-and-spec.md)
4. [DAG、Steps 与控制流](./10-argo-workflow/04-dag-steps-and-controlflow.md)
5. [参数、制品与数据传递](./10-argo-workflow/05-parameters-and-artifacts.md)
6. [WorkflowTemplate 与 CronWorkflow](./10-argo-workflow/06-workflowtemplate-and-cron.md)
7. [架构与控制器原理](./10-argo-workflow/07-architecture-and-controller.md)
8. [生产实践与故障排查](./10-argo-workflow/08-production-and-troubleshooting.md)

### 路线 6：GitOps 与渐进发布

适合需要搭建完整 GitOps + CI/CD + 渐进发布体系的同学：

1. [Argo 全家桶全景概览](./13-argo-suite/01-argo-ecosystem-overview.md)
2. [Argo CD 核心概念与原理](./13-argo-suite/02-argocd-core-concepts.md)
3. [Argo CD 安装与快速上手](./13-argo-suite/03-argocd-installation-and-quickstart.md)
4. [Argo CD 进阶](./13-argo-suite/04-argocd-advanced.md)
5. [Argo Rollouts：金丝雀与蓝绿部署](./13-argo-suite/05-argo-rollouts-core.md)
6. [Argo Events：事件监听与自动触发](./13-argo-suite/06-argo-events-core.md)
7. [全家桶联动实战](./13-argo-suite/07-argo-suite-integration.md)
8. [生产最佳实践与避坑](./13-argo-suite/08-production-best-practices.md)

### 路线 4：平台与存储专题

适合希望把 Kubernetes 用到“平台能力”和“生产存储运维”层面的同学：

1. [存储系统详解](./04-advanced/02-storage.md)
2. [调度机制与策略](./04-advanced/03-scheduling.md)
3. [Kubernetes 自定义资源专题](./07-custom-resources/README.md)
4. [Kubernetes 存储与运维专题](./08-storage-operations/README.md)

## 文档结构总览

### 第一部分：基础理论篇

这部分解决的问题是："Kubernetes 到底是什么，它的基本构成是什么，为什么这些资源会这样设计？"

1. [Kubernetes 概述与架构](./01-basics/01-overview.md)
2. [核心组件详解](./01-basics/02-components.md)
3. [核心概念与术语](./01-basics/03-concepts.md)

### 第二部分：核心资源详解

这部分解决的问题是："日常最常见的资源对象应该怎么理解、怎么配置、怎么排查？"

1. [Pod - 最小调度单元](./02-resources/01-pod.md)
2. [Deployment - 无状态应用部署](./02-resources/02-deployment.md)
3. [Service - 服务发现与负载均衡](./02-resources/03-service.md)
4. [ConfigMap 与 Secret](./02-resources/04-configmap-secret.md)
5. [Volume 与持久化存储](./02-resources/05-volume.md)
6. [Namespace - 资源隔离](./02-resources/06-namespace.md)
7. [StatefulSet - 有状态应用](./02-resources/07-statefulset.md)
8. [DaemonSet 与 Job](./02-resources/08-daemonset-job.md)

### 第三部分：实战操作篇

这部分解决的问题是："工作里真正怎么查、怎么改、怎么调试、怎么运维？"

1. [kubectl 命令完全手册](./03-practice/01-kubectl-commands.md)
2. [YAML 编写规范与技巧](./03-practice/02-yaml-guide.md)
3. [常见运维操作指南](./03-practice/03-operations.md)
4. [故障排查与调试](./03-practice/04-troubleshooting.md)

### 第四部分：进阶主题

这部分解决的问题是："Kubernetes 底层能力和高级机制到底怎么运转？"

1. [Kubernetes 网络模型](./04-advanced/01-networking.md)
2. [存储系统详解](./04-advanced/02-storage.md)
3. [调度机制与策略](./04-advanced/03-scheduling.md)
4. [安全与权限控制](./04-advanced/04-security.md)
5. [Ingress 与流量管理](./04-advanced/05-ingress.md)

### 第五部分：client-go 编程

这部分解决的问题是："如果不用 kubectl，而是自己写程序访问 Kubernetes，该怎么做？"

1. [client-go 入门](./05-client-go/01-introduction.md)
2. [客户端配置与连接](./05-client-go/02-client-setup.md)
3. [资源的 CRUD 操作](./05-client-go/03-crud-operations.md)
4. [Informer 机制详解](./05-client-go/04-informer.md)
5. [实战项目：自定义控制器](./05-client-go/05-controller-demo.md)

### 第六部分：实战项目

这部分解决的问题是："如何把前面学过的对象、命令、排障方法真正串起来？"

1. [项目一：部署微服务应用](./06-projects/01-microservice-deploy/README.md)
2. [项目二：日志收集系统](./06-projects/02-logging-system/README.md)
3. [项目三：监控告警系统](./06-projects/03-monitoring/README.md)

### 第七部分：自定义资源专题

这部分解决的问题是："如果 Kubernetes 没有我想要的资源类型，我该如何把自己的领域对象扩展成原生 API，并配合控制器实现自动化能力？"

1. [Kubernetes 自定义资源专题总览](./07-custom-resources/README.md)
2. [CRD 基础与 API 扩展](./07-custom-resources/01-crd-basics.md)
3. [控制器、Reconcile 与 Operator 模式](./07-custom-resources/02-controller-and-operator.md)
4. [CRD 设计、版本演进与最佳实践](./07-custom-resources/03-crd-design-and-versioning.md)
5. [真实案例拆解：从领域模型到 CRD 设计](./07-custom-resources/04-real-world-crd-case-study.md)
6. [Kubebuilder 工作流：从脚手架到控制器骨架](./07-custom-resources/05-kubebuilder-workflow.md)
7. [从 0 到 1 设计一个完整 Operator 的实战样例](./07-custom-resources/06-operator-end-to-end-example.md)
8. [Operator 测试、调试与发布](./07-custom-resources/07-operator-testing-debugging-release.md)

### 第八部分：存储与运维专题

这部分解决的问题是："如何从生产视角理解 Kubernetes 存储体系，以及如何围绕卷、PVC、StorageClass、CSI 做扩容、迁移、备份和排障？"

1. [Kubernetes 存储与运维专题总览](./08-storage-operations/README.md)
2. [存储基础与架构认知](./08-storage-operations/01-storage-foundations.md)
3. [StorageClass、CSI 与动态供给](./08-storage-operations/02-storageclass-and-csi.md)
4. [存储运维手册：扩容、迁移、备份、排障](./08-storage-operations/03-storage-operations-runbook.md)
5. [存储方案对比：本地卷、NFS、云盘、分布式存储](./08-storage-operations/04-storage-solution-comparison.md)
6. [备份恢复与演练手册](./08-storage-operations/05-backup-recovery-drills.md)
7. [数据库、中间件、日志系统三类场景的存储选型与运维差异](./08-storage-operations/06-storage-by-workload-scenarios.md)
8. [PVC/PV/CSI 故障案例库与排障剧本](./08-storage-operations/07-pvc-pv-csi-failure-cases.md)

### 第九部分：运维指令专题

这部分解决的问题是："如果我站在运维、SRE、值班排障的角度，Kubernetes 常用命令到底该怎么学、怎么查、怎么组合、怎么降低误操作风险？"

1. [Kubernetes 运维指令大全](./09-ops-command-handbook/README.md)
2. [集群连接、上下文切换与资源巡检](./09-ops-command-handbook/01-cluster-context-and-inspection.md)
3. [工作负载发布、变更、回滚与扩缩容](./09-ops-command-handbook/02-workload-release-and-change.md)
4. [日志、调试、容器排障与应急定位](./09-ops-command-handbook/03-logs-debug-and-troubleshooting.md)
5. [节点维护、调度控制与存储相关命令](./09-ops-command-handbook/04-node-scheduling-and-storage.md)
6. [高频运维命令组合与值班速查清单](./09-ops-command-handbook/05-practical-command-combinations.md)

### 第十部分：Argo Workflow 专题

这部分解决的问题是："如何在 Kubernetes 上跑出一条可维护、可调试、可上生产的工作流？覆盖 CI/CD、ML Pipeline、批处理等真实场景。"

1. [Argo Workflow 专题总览](./10-argo-workflow/README.md)
2. [概述与核心概念](./10-argo-workflow/01-argo-overview.md)
3. [安装与快速上手](./10-argo-workflow/02-installation-and-quickstart.md)
4. [Workflow Spec 与 Template 类型详解](./10-argo-workflow/03-templates-and-spec.md)
5. [DAG、Steps 与控制流](./10-argo-workflow/04-dag-steps-and-controlflow.md)
6. [参数、制品与数据传递](./10-argo-workflow/05-parameters-and-artifacts.md)
7. [WorkflowTemplate、ClusterWorkflowTemplate 与 CronWorkflow](./10-argo-workflow/06-workflowtemplate-and-cron.md)
8. [架构与控制器原理](./10-argo-workflow/07-architecture-and-controller.md)
9. [生产实践与故障排查](./10-argo-workflow/08-production-and-troubleshooting.md)

### 第十一部分：控制器深度专题

这部分解决的问题是："client-go 的 Informer 内部到底怎么转？Workqueue 是怎么去重和退避的？Reconcile 函数怎么写才幂等？Finalizer / Leader Election / status / Conditions 在生产里到底怎么落地？"

1. [控制器深度专题总览](./11-controller-deep-dive/README.md)
2. [控制器模式与声明式 API 的本质](./11-controller-deep-dive/01-controller-pattern.md)
3. [Informer 内部机制详解](./11-controller-deep-dive/02-informer-internals.md)
4. [WorkQueue 与限速器](./11-controller-deep-dive/03-workqueue.md)
5. [Reconcile 函数的设计要点](./11-controller-deep-dive/04-reconcile-design.md)
6. [Leader Election 与 Finalizer](./11-controller-deep-dive/05-leader-election-and-finalizers.md)
7. [Status 子资源与 Conditions 设计](./11-controller-deep-dive/06-status-and-conditions.md)
8. [controller-runtime 与原生 client-go 对比](./11-controller-deep-dive/07-controller-runtime-vs-clientgo.md)
9. [最佳实践与常见坑](./11-controller-deep-dive/08-best-practices-and-pitfalls.md)

### 第十三部分：Argo 全家桶专题

这部分解决的问题是："如何用 Argo CD、Argo Workflows、Argo Rollouts、Argo Events 四件套搭建完整的 GitOps + CI/CD + 渐进发布 + 事件驱动体系？"

1. [Argo 全家桶专题总览](./13-argo-suite/README.md)
2. [全家桶全景概览](./13-argo-suite/01-argo-ecosystem-overview.md)
3. [Argo CD 核心概念与原理](./13-argo-suite/02-argocd-core-concepts.md)
4. [Argo CD 安装与快速上手](./13-argo-suite/03-argocd-installation-and-quickstart.md)
5. [Argo CD 进阶：多集群、App of Apps、Secrets](./13-argo-suite/04-argocd-advanced.md)
6. [Argo Rollouts：金丝雀与蓝绿部署](./13-argo-suite/05-argo-rollouts-core.md)
7. [Argo Events：事件监听与自动触发](./13-argo-suite/06-argo-events-core.md)
8. [全家桶联动实战](./13-argo-suite/07-argo-suite-integration.md)
9. [生产最佳实践与避坑](./13-argo-suite/08-production-best-practices.md)

### 第十二部分：网络工程专题

这部分解决的问题是："K8s 集群在真实生产环境中是如何与内网、公网、办公网、堡垒机等网络互通的？iptables、SSH 隧道、端口转发、NAT 这些技术在网络链路中各自扮演什么角色？"

1. [网络工程专题总览](./12-network-engineering/README.md)
2. [企业网络架构全景](./12-network-engineering/01-network-architecture-overview.md)
3. [集群内部网络深度解析](./12-network-engineering/02-cluster-internal-network.md)
4. [内网与 VPC 网络](./12-network-engineering/03-vpc-and-intranet.md)
5. [公网访问与暴露](./12-network-engineering/04-public-network-access.md)
6. [办公网络与堡垒机](./12-network-engineering/05-office-and-bastion-network.md)
7. [网络转发与代理](./12-network-engineering/06-network-forwarding.md)
8. [多网络互通全景方案](./12-network-engineering/07-network-interconnection.md)
9. [网络排障实战](./12-network-engineering/08-network-troubleshooting.md)

## 当前文档完善状态

为了让你对这套文档的成熟度有预期，下面给出当前状态说明：

| 状态 | 含义 |
|------|------|
| `已完善` | 内容相对完整，适合直接阅读 |
| `持续增强` | 已可阅读，但还会继续细化、补例子、补排障 |
| `已补齐骨架` | 已有较完整导读和主体结构，后续会继续补细节 |

| 模块 | 当前状态 | 说明 |
|------|----------|------|
| 基础理论篇 | `持续增强` | 已开始朝更细、更友好的讲解方式统一 |
| 核心资源篇 | `持续增强` | 已有主体内容，后续会继续补字段解释、最佳实践、排障 |
| 实战操作篇 | `持续增强` | 命令较全，后续会强化场景化使用方式 |
| 进阶主题 | `已补齐骨架` | 已补全缺失章节入口，后续继续做深讲 |
| client-go | `持续增强` | 基础到控制器链路已存在，后续补更多原理和实战说明 |
| 实战项目 | `已补齐骨架` | 已开始补项目导读、步骤、验证与排障 |
| 自定义资源专题 | `已补齐骨架` | 已新增完整专题结构，覆盖 CRD、控制器、版本设计 |
| 存储与运维专题 | `已补齐骨架` | 已新增专题结构，覆盖存储体系与生产运维动作 |
| 运维指令专题 | `已补齐骨架` | 已新增运维命令专题，覆盖巡检、发布、排障、节点、存储和值班速查 |
| Argo Workflow 专题 | `已完善` | 新增完整专题，覆盖概念、Spec、DAG、参数制品、模板复用、原理、生产 |
| 控制器深度专题 | `已完善` | 新增完整专题，覆盖 Informer 内部、Workqueue、Reconcile、Leader、Finalizer、cr vs client-go |
| 网络工程专题 | `已完善` | 新增完整专题，覆盖企业网络架构、VPC/内网/公网/办公网/堡垒机互通、网络转发与排障 |
| Argo 全家桶专题 | `已完善` | 新增完整专题，覆盖 Argo CD/Rollouts/Events 核心概念、安装实战、全家桶联动、生产最佳实践 |

## 阅读时建议重点关注的东西

很多人读 Kubernetes 文档时容易陷入两个误区：

- 只记命令，不理解资源关系
- 只看 YAML，不理解字段为什么这样设计

所以建议你读每一篇时都优先关注下面 4 件事：

1. 这个对象解决什么问题
2. 它和哪些对象有关系
3. 最关键的字段是哪几个
4. 出问题时先去哪里查

如果你能用这 4 个问题去读文档，理解速度会快很多。

## 学习节奏建议

如果你希望在 4 周内完成一轮系统学习，可以参考这个节奏：

```text
第 1 周：环境搭建 + 基础理论 + 核心概念
第 2 周：Pod / Deployment / Service / 配置与存储
第 3 周：kubectl / YAML / 运维 / 故障排查
第 4 周：网络 / 调度 / 安全 / client-go / 项目实战
第 5 周：CRD / Operator / 存储运维专题
第 6 周：Argo Workflow / 控制器深度专题
第 7 周：网络工程专题（企业网络架构、VPC、公网暴露、堡垒机、网络转发与排障）
第 8 周：Argo 全家桶专题（Argo CD、Argo Rollouts、Argo Events、全家桶联动）
```

如果你时间更碎片化，也可以按主题拆开学：

- 今天只学一个对象，比如 `Pod`
- 明天只练一个能力，比如 `kubectl logs / describe / exec`
- 后天只做一个排障专题，比如服务不通时如何逐层定位

这种方式更适合工作中边学边用。

## 环境准备

- 推荐使用 [Minikube](https://minikube.sigs.k8s.io/) 或 [Kind](https://kind.sigs.k8s.io/) 搭建本地学习环境
- 如果你已经在云上使用托管集群，也可以直接在测试环境练习
- 环境细节可以看 [环境搭建指南](./00-setup/environment.md)

## 配套使用建议

建议把这套文档与下面几种方法配合使用：

- 一边读，一边用 `kubectl get/describe/logs` 实际观察资源变化
- 一边学对象，一边自己写最小 YAML
- 遇到不懂的字段时，先看本仓库文档，再交叉参考 Kubernetes 官方文档
- 练习时优先关注结果为什么发生，而不是只关注命令能不能执行

## 官方文档

- Kubernetes 官方文档：[https://kubernetes.io/docs/](https://kubernetes.io/docs/)
- Kubernetes 官方任务示例：[https://kubernetes.io/docs/tasks/](https://kubernetes.io/docs/tasks/)
- Kubernetes API 参考：[https://kubernetes.io/docs/reference/generated/kubernetes-api/](https://kubernetes.io/docs/reference/generated/kubernetes-api/)

---

如果你准备开始，最推荐的起点是：

1. 先完成 [环境搭建指南](./00-setup/environment.md)
2. 然后阅读 [Kubernetes 概述与架构](./01-basics/01-overview.md)
3. 接着进入 [核心概念与术语](./01-basics/03-concepts.md)

这样你会更容易把后面的所有资源、命令和实战内容串起来。



