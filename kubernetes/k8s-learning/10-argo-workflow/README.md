# 🌊 Argo Workflow 专题

> 这是一个面向"用 Kubernetes 跑流水线、跑训练任务、跑批处理"的同学的 Argo Workflow 系统讲解。
> 目标不是只让你会写一个 hello world Workflow，而是让你能写出可维护、可调试、可上生产的工作流。

## 这个专题适合谁

- 已经熟悉 Kubernetes 基础（Pod、Job、CRD、控制器思想）
- 有以下任意一种使用诉求：
  - 替代 Jenkins / GitLab CI 做云原生 CI/CD
  - 跑机器学习训练、推理、数据处理 Pipeline
  - 把多个步骤串成 DAG（有向无环图）批量跑批
  - 做定时调度（替代或补充 CronJob）
- 想搞清楚 Argo 控制器是怎么把 Workflow 翻译成一堆 Pod 的

## 你会学到什么

这个专题会从 4 个层面讲清楚 Argo Workflow：

1. **是什么、为什么**：Argo Workflow 解决了原生 Job/CronJob 解决不了的什么问题
2. **怎么写**：Workflow Spec、Template 类型、DAG/Steps、参数、制品、控制流
3. **怎么管**：WorkflowTemplate、ClusterWorkflowTemplate、CronWorkflow、归档
4. **怎么扛生产**：架构原理、控制器机制、故障排查、性能与稳定性

## 学习路线

### 路线 1：先跑起来，再深入

适合第一次接触 Argo 的同学：

1. [Argo Workflow 概述与核心概念](./01-argo-overview.md)
2. [安装与快速上手](./02-installation-and-quickstart.md)
3. [Workflow Spec 与 Template 类型详解](./03-templates-and-spec.md)
4. [DAG、Steps 与控制流](./04-dag-steps-and-controlflow.md)

### 路线 2：写真实业务流水线

适合需要落地业务 Pipeline 的同学：

1. [参数、制品与数据传递](./05-parameters-and-artifacts.md)
2. [WorkflowTemplate、ClusterWorkflowTemplate 与 CronWorkflow](./06-workflowtemplate-and-cron.md)
3. [生产实践与故障排查](./08-production-and-troubleshooting.md)

### 路线 3：理解原理 / 想做平台化

适合做内部 AI 平台、调度平台、CI 平台的同学：

1. [Argo 架构与控制器原理](./07-architecture-and-controller.md)
2. [生产实践与故障排查](./08-production-and-troubleshooting.md)

## 文档列表

| 序号 | 标题 | 主要内容 |
|------|------|----------|
| 01 | [概述与核心概念](./01-argo-overview.md) | Argo 是什么、解决什么问题、与 Job/Tekton 的关系 |
| 02 | [安装与快速上手](./02-installation-and-quickstart.md) | 安装、UI、CLI、第一个 Workflow |
| 03 | [Workflow Spec 与 Template 类型](./03-templates-and-spec.md) | Spec 字段、6 种 Template 类型逐个讲 |
| 04 | [DAG、Steps 与控制流](./04-dag-steps-and-controlflow.md) | DAG / Steps 选择、条件、循环、重试、超时、退出处理器 |
| 05 | [参数、制品与数据传递](./05-parameters-and-artifacts.md) | parameters、artifacts、output、值传递规律、S3/MinIO 配置 |
| 06 | [WorkflowTemplate 与 CronWorkflow](./06-workflowtemplate-and-cron.md) | 复用、模板引用、定时调度 |
| 07 | [架构与控制器原理](./07-architecture-and-controller.md) | argo-server、workflow-controller、Pod 编排原理 |
| 08 | [生产实践与故障排查](./08-production-and-troubleshooting.md) | 资源、归档、限流、Pod 卡死、UI 黑屏等典型问题 |

## 推荐配套阅读

- [Kubernetes Job / CronJob](../02-resources/08-daemonset-job.md)（原生批处理对象）
- [自定义资源专题](../07-custom-resources/README.md)（理解 Argo 是怎么用 CRD + Controller 实现的）
- [控制器深度专题](../11-controller-deep-dive/README.md)（Argo 的控制器套路与本系列一致）

## 阅读建议

读这一系列时，最有效的方式是：

1. **先跑通**：装一个 minikube/kind，把 hello world、DAG、参数传递依次跑通
2. **再读字段**：读 Spec 详解时，对着 yaml 改一改、提交一次 Workflow，看变化
3. **再读架构**：等你写过 5~10 个 Workflow 后再读架构与控制器原理，能对上号
4. **最后读生产**：等你真的把它接到业务里，再回头看生产实践与排障

不要一次性把所有篇都看完再上手，那样很容易看完什么都没记住。
