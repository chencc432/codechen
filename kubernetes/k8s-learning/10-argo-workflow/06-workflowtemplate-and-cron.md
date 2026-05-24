# 🧩 WorkflowTemplate、ClusterWorkflowTemplate 与 CronWorkflow

> 这一篇讲 Argo 的"模板复用"与"定时调度"两块能力——这是 Argo 从"能用"走向"平台化"的关键。

## 1. 为什么需要这些资源

如果只用 `Workflow` 这一个对象，会出现三个问题：

1. **重复**：每次提交都要把整套 Pipeline yaml 复制一遍
2. **难以治理**：业务方各写各的，安全、资源、SA 都不可控
3. **没有定时**：CronJob 跑不了 DAG，但 Argo 的 `Workflow` 又不能"自己定时"

`WorkflowTemplate` / `ClusterWorkflowTemplate` 解决"复用与治理"，`CronWorkflow` 解决"定时调度"。

## 2. WorkflowTemplate：命名空间级模板

`WorkflowTemplate` 长得几乎和 `Workflow` 一模一样，只是 kind 不同 + 不会自动跑：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: build-image
  namespace: argo
spec:
  entrypoint: main
  arguments:
    parameters:
      - name: repo
      - name: ref
        value: main
  templates:
    - name: main
      inputs:
        parameters: [{name: repo}, {name: ref}]
      container:
        image: gcr.io/kaniko-project/executor
        args: ["--context=git://{{inputs.parameters.repo}}#{{inputs.parameters.ref}}",
               "--destination=registry.local/app:{{inputs.parameters.ref}}"]
```

部署模板：

```bash
kubectl apply -n argo -f build-image-tpl.yaml
```

提交一次执行：

```bash
# 用模板生成一次 Workflow
argo submit --from workflowtemplate/build-image \
  -n argo \
  -p repo=github.com/acme/app \
  -p ref=v1.0.0
```

也可以用 `kubectl create -f` 配合下面的引用方式提交：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: build-
spec:
  workflowTemplateRef:
    name: build-image
  arguments:
    parameters:
      - name: repo
        value: github.com/acme/app
      - name: ref
        value: v1.0.0
```

> **`workflowTemplateRef` 是引用模式**：`spec` 里其它字段几乎都不需要写，运行时会用模板的 spec。这是平台化最干净的提交方式。

## 3. ClusterWorkflowTemplate：集群级模板

跨命名空间复用就用 `ClusterWorkflowTemplate`：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: standard-build
spec:
  entrypoint: main
  templates: [...]
```

引用：

```yaml
spec:
  workflowTemplateRef:
    name: standard-build
    clusterScope: true
```

适合：

- 平台方提供"标准化构建/部署模板"
- 公司级共享流水线（如统一安全扫描、统一镜像构建）

## 4. 复用 template：templateRef

除了引用整个 WorkflowTemplate 来跑，还可以**只引用 WorkflowTemplate 里的某个 template**。这是更细粒度的复用：

```yaml
# 模板：build-lib
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: build-lib
spec:
  templates:
    - name: kaniko-build
      inputs:
        parameters: [{name: image}, {name: context}]
      container:
        image: gcr.io/kaniko-project/executor
        args: ["--context={{inputs.parameters.context}}",
               "--destination={{inputs.parameters.image}}"]
```

调用：

```yaml
- name: build
  templateRef:
    name: build-lib            # 同 namespace
    template: kaniko-build
    # clusterScope: true       # 跨命名空间时打开
  arguments:
    parameters:
      - {name: image,   value: registry.local/app:1.0}
      - {name: context, value: /workspace}
```

> 实战推荐：把"基础动作"（kaniko 构建、kubectl 部署、helm upgrade、slack 通知）做成一组 ClusterWorkflowTemplate；业务方写自己的 Workflow 时只组合调用，不重复造轮子。

## 5. CronWorkflow：定时跑 Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: nightly-report
  namespace: argo
spec:
  schedule: "0 2 * * *"                       # 每天 02:00
  timezone: "Asia/Shanghai"
  concurrencyPolicy: "Forbid"                 # 禁止并发
  startingDeadlineSeconds: 300                # 错过 5 分钟内还能补跑
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 5
  workflowSpec:
    entrypoint: main
    templates:
      - name: main
        container:
          image: my-report:latest
          command: [./gen-report.sh]
```

`spec.workflowSpec` 内容就是一个完整 Workflow 的 spec，CronWorkflow 控制器到点会创建一个 Workflow。

### 5.1 关键字段速览

| 字段 | 含义 |
|------|------|
| `schedule` | 标准 cron 表达式 |
| `timezone` | 时区，**强烈建议显式写**，否则用 controller 容器的时区 |
| `concurrencyPolicy` | `Allow` / `Forbid` / `Replace` |
| `startingDeadlineSeconds` | 错过窗口多少秒内可以补跑（默认不补） |
| `suspend` | 临时停掉调度（不删 cron） |
| `workflowMetadata` | 给生成的 wf 打 label/annotation |

### 5.2 CronWorkflow 引用 WorkflowTemplate

CronWorkflow 也可以走 `workflowTemplateRef`，最干净：

```yaml
spec:
  schedule: "0 * * * *"
  workflowSpec:
    workflowTemplateRef:
      name: hourly-cleanup
```

### 5.3 与原生 CronJob 的区别

| 维度 | CronJob | CronWorkflow |
|------|---------|--------------|
| 单步 vs 多步 | 单 Job | 完整 DAG |
| 失败重试 | Pod restart | Argo retryStrategy（更可控） |
| 历史可视化 | kubectl get jobs | Argo UI 一目了然 |
| 资源/产物 | 自己管 | artifact 体系直接复用 |
| 时区 | K8s 1.27+ 才好 | 早就支持 |

> 如果你已经在用 Argo，定时任务一律用 CronWorkflow，运维上明显省事。

## 6. WorkflowEventBinding 与提交方式（进阶）

除了 CLI / API 提交，可以用 `WorkflowEventBinding` 把外部事件映射成提交动作（webhook 回调）：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowEventBinding
metadata:
  name: gitlab-push
spec:
  event:
    selector: payload.event_type == "push"
  submit:
    workflowTemplateRef:
      name: build-image
    arguments:
      parameters:
        - name: repo
          valueFrom:
            event: payload.repository.url
```

外部系统给 `argo-server` 的 `/api/v1/events` 推送事件，argo-server 根据 `selector` 匹配，提交对应 Workflow。

> 如果做更复杂的事件源（Kafka / NATS / 文件系统等），用配套的 **Argo Events** 项目（Sensor + EventSource），而不是只用 WorkflowEventBinding。

## 7. 治理建议（写给平台方）

1. **业务命名空间不允许直接写完整 Workflow，只允许引用 ClusterWorkflowTemplate**——通过 OPA/Kyverno 策略校验
2. **ServiceAccount 收紧**：默认 SA 只能创建 wf，wf 内部 Pod 用专门的 `argo-workflow-sa`
3. **default 资源（resources）必须写**：通过 `workflowDefaults`（controller config）兜底
4. **TTL + PodGC 必须开**：避免历史 wf 把 etcd / Pod 撑爆
5. **统一对外只暴露 WorkflowTemplate / CronWorkflow**：升级、加 hook、加监控都集中可控

`workflowDefaults` 在 controller 的 `workflow-controller-configmap` 里配：

```yaml
data:
  workflowDefaults: |
    spec:
      ttlStrategy:
        secondsAfterCompletion: 86400
      podGC:
        strategy: OnWorkflowSuccess
      activeDeadlineSeconds: 7200
      serviceAccountName: argo-workflow-sa
```

这样任何人提交的 Workflow，没写这些字段都会自动套上默认值。

## 下一步

模板与定时讲完了，最后两篇是原理与生产：

- [架构与控制器原理](./07-architecture-and-controller.md)
- [生产实践与故障排查](./08-production-and-troubleshooting.md)
