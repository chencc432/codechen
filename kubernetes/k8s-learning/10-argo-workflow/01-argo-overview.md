# 🌊 Argo Workflow 概述与核心概念

## 1. 一句话定义

> Argo Workflow 是一个 Kubernetes 原生的工作流引擎，把"一组按顺序或按依赖关系执行的任务"用 CRD 的方式定义出来，由控制器编排成一组 Pod 跑完。

它本质上是：

- 一组 CRD（`Workflow`、`WorkflowTemplate`、`CronWorkflow` 等）
- 一个控制器（`workflow-controller`）
- 一个可选的 API Server / UI（`argo-server`）

## 2. 它解决了什么问题

Kubernetes 原生提供了 `Job` 和 `CronJob`，但它们只能解决"单个任务跑一次或跑多次"，解决不了：

- 多个任务**有顺序、有依赖**：A 跑完才能跑 B，B 和 C 可以并行，最后跑 D
- 任务之间需要**传递数据**：A 算出来一个文件，B 接着用
- 任务编排里要**带条件/循环/重试**：根据 A 的结果决定要不要跑 B
- 整个流水线要**可观测**：能看到 DAG、每一步在哪、卡在哪
- 同一套流水线要**复用**：模板化，传不同参数跑不同任务

这些都是 Argo Workflow 的目标场景。

## 3. 典型使用场景

| 场景 | 怎么用 Argo |
|------|-------------|
| **CI/CD** | 拉代码 → 构建镜像 → 推到仓库 → 部署，全在 K8s 上跑 |
| **机器学习训练** | 数据预处理 → 训练 → 评估 → 上线，多步骤 + GPU 调度 |
| **数据处理 / ETL** | 抽数 → 清洗 → 聚合 → 入库，按 DAG 执行 |
| **批量推理** | 把一批数据切片并行推理（fan-out / fan-in） |
| **替代 CronJob** | 用 CronWorkflow 调度多步骤任务 |
| **基础设施流水线** | terraform plan/apply、备份恢复、巡检 |

## 4. 几个最核心的概念

理解下面这 5 个概念，你就理解了 Argo 的 80%：

### 4.1 Workflow

一次具体的"工作流执行实例"。每提交一个 `Workflow`，都会被创建为集群里一个 CR，触发控制器去拉起 Pod。

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: alpine:3.18
        command: [echo, "hello argo"]
```

### 4.2 Template（模板）

一个"步骤"的定义。每个 step 必然属于某个 Template。Template 有多种类型：
- `container`：跑一个容器（最常用）
- `script`：直接写一段 shell/python，会自动包成容器
- `dag`：定义一组节点 + 依赖关系
- `steps`：定义按顺序执行的步骤组
- `resource`：直接 create/patch/delete 一个 K8s 资源
- `suspend`：暂停（等待人工或定时唤醒）

> 注意：`Workflow` 本身不直接"做事"，做事的是它里面的 `Template`，并通过 `entrypoint` 指定从哪个模板开始。

### 4.3 DAG 与 Steps

两种编排方式：

- **Steps**：列表式，每一项是一组并行步骤；每一组完成才进入下一组
- **DAG**：图结构，每个节点用 `dependencies` 声明依赖谁

```text
Steps: [[A], [B, C], [D]]
        ↓ 依次跑：A 跑完 → B/C 并行 → 都跑完 → D

DAG:    A → B
        A → C
        B → D
        C → D
```

DAG 更灵活，复杂工作流推荐 DAG；简单线性流程用 Steps 更直观。

### 4.4 Parameters 与 Artifacts

任务之间传递东西，分两种：

- **Parameters（参数）**：传字符串/小段文本，比如版本号、URL
- **Artifacts（制品）**：传文件，比如训练好的模型、构建好的二进制

Parameters 走 etcd，Artifacts 走对象存储（S3、OSS、MinIO 等）。

### 4.5 WorkflowTemplate 与 CronWorkflow

- `WorkflowTemplate`：把 Workflow 定义保存成模板，多次复用、参数化提交
- `CronWorkflow`：定时触发 Workflow，类似 CronJob 但支持完整 DAG

## 5. Argo 与其他工具对比

| 工具 | 定位 | 关键差异 |
|------|------|----------|
| **K8s Job/CronJob** | 单任务/定时 | 没有依赖、没有 DAG、不能传文件 |
| **Argo Workflow** | 通用云原生工作流 | DAG、artifact、UI 都有，自由度最高 |
| **Tekton** | 偏 CI/CD | 概念更聚焦构建发布；DAG 能力较 Argo 弱 |
| **Airflow** | 数据工程 | Python DSL；不是云原生（虽然能跑在 K8s 上） |
| **Argo Events** | 事件驱动 | 监听事件触发 Workflow，常和 Argo Workflow 搭配 |
| **Kubeflow Pipelines** | ML Pipeline | 底层就是 Argo Workflow + Python SDK |

如果你是用来做 AI 训练/推理 Pipeline，常见的栈是：

```text
Kubeflow Pipelines (Python SDK 写 pipeline)
       ↓ 编译
Argo Workflow YAML
       ↓ 提交
workflow-controller 创建 Pod 执行
```

## 6. 用一张图理解整体流程

```text
┌──────────────┐   submit   ┌─────────────┐
│  argo CLI /  ├───────────▶│  argo-server│
│  Argo UI     │            │  (REST API) │
└──────────────┘            └─────┬───────┘
                                  │ create CR
                                  ▼
                       ┌────────────────────┐
                       │  Workflow CR (etcd)│
                       └────────┬───────────┘
                                │ watch
                                ▼
                  ┌──────────────────────────┐
                  │   workflow-controller    │
                  │ (DAG 解析 / Pod 编排)    │
                  └─────────┬────────────────┘
                            │ create
                            ▼
                  ┌──────────────────────────┐
                  │  Pods（每一步一个）      │
                  │  + Artifacts (S3/MinIO)  │
                  └──────────────────────────┘
```

## 7. 你需要建立的核心心智模型

读后面所有篇章前，请先把下面 4 件事记牢：

1. **Workflow 是 CRD**：你提交它就是创建一个 K8s 资源，控制器才是真正干活的
2. **每一步基本就是一个 Pod**：Argo 不会"自己跑业务代码"，它负责按依赖关系拉起 Pod
3. **Template 是骨架，参数让它千变万化**：模板抽象 + 参数化是 Argo 的精髓
4. **数据传递分两类**：参数走 K8s 对象，文件走对象存储

带着这 4 个心智模型，再去看后面的字段就不会迷路。

## 下一步

接下来推荐先动手：

- 装环境：[安装与快速上手](./02-installation-and-quickstart.md)
- 看完装好后再读：[Workflow Spec 与 Template 类型详解](./03-templates-and-spec.md)
