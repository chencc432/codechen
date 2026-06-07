# 🔔 Argo Events：事件监听与自动触发

## 1. 从门铃系统说起

想象你住在一栋智能公寓里：

- **门铃**（EventSource）：有人按门铃、有快递到、有外卖送来
- **智能管家**（Sensor）：收到门铃信号后判断——是快递就开柜子、是外卖就通知你、是陌生人就不理
- **执行设备**（Trigger）：开柜子、发通知、开灯、开门

Argo Events 就是 Kubernetes 里的这套"智能门铃系统"——监听各种事件源，根据条件触发各种动作。

## 2. 一句话定义

> Argo Events 是 Kubernetes 原生的事件驱动自动化框架——监听外部/内部事件源，根据过滤条件触发 Workflow 提交、K8s 资源创建、HTTP 调用等动作。

## 3. 它解决什么问题

| 没有 Argo Events 时 | 有了 Argo Events 后 |
|---------------------|---------------------|
| 代码 push 后手动触发 CI | Git push → 自动触发构建 Workflow |
| 文件上传到 S3 后手动跑处理 | S3 新文件 → 自动触发数据处理 |
| 定时任务只能写简单 CronJob | Cron → 触发复杂 DAG Workflow |
| Kafka 消息来了要写消费者代码 | Kafka 消息 → 直接触发 K8s Job |
| 想 Webhook 触发部署要自己写 API | Webhook → 直接 submit Workflow |

## 4. 三个核心概念

```text
┌─────────────────────────────────────────────────────────────────┐
│                     Argo Events 架构                              │
│                                                                 │
│  ┌─────────────┐       ┌──────────┐       ┌─────────────────┐  │
│  │ EventSource │──────▶│  EventBus│──────▶│     Sensor      │  │
│  │   "耳朵"    │       │  "神经"  │       │    "大脑"       │  │
│  │ 监听外部事件│       │ 事件传输 │       │ 过滤+决策+触发  │  │
│  └─────────────┘       └──────────┘       └────────┬────────┘  │
│                                                     │           │
│       GitHub Webhook ─┐                             ▼           │
│       Kafka 消息    ──┤                    ┌─────────────────┐  │
│       S3 文件上传   ──┤                    │    Trigger      │  │
│       Cron 定时     ──┤                    │    "手脚"       │  │
│       NATS 消息     ──┤                    │ 创建 Workflow   │  │
│       Webhook       ──┘                    │ 创建 K8s 资源  │  │
│                                            │ 调用 HTTP API   │  │
│                                            └─────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.1 EventSource（事件源）—— "耳朵"

定义"从哪里监听事件"。支持 20+ 种事件源：

| 类型 | 说明 | 典型用途 |
|------|------|---------|
| **webhook** | 监听 HTTP 请求 | 通用 Webhook |
| **github** | GitHub 事件（push/PR/issue...） | CI 触发 |
| **gitlab** | GitLab 事件 | CI 触发 |
| **calendar** | 定时/Cron 触发 | 定时任务 |
| **kafka** | Kafka 消息 | 数据管道触发 |
| **nats** | NATS 消息 | 微服务间事件 |
| **sns/sqs** | AWS SNS/SQS | 云事件 |
| **minio/s3** | 对象存储文件变化 | 数据上传触发处理 |
| **redis** | Redis Pub/Sub | 实时事件 |
| **resource** | K8s 资源变化 | Pod 状态变化触发 |
| **amqp** | RabbitMQ 消息 | 消息队列 |
| **slack** | Slack 命令 | ChatOps |

### 4.2 Sensor（传感器）—— "大脑"

定义"收到事件后该怎么做"。核心包含：

- **dependencies**：依赖哪些事件（可以等多个事件都到了才触发）
- **triggers**：执行什么动作

### 4.3 EventBus（事件总线）—— "神经系统"

EventSource 和 Sensor 之间的消息传输通道。底层实现：

| 类型 | 说明 | 适用 |
|------|------|------|
| **NATS Streaming** | 默认推荐 | 通用场景 |
| **JetStream** | NATS 新一代 | 推荐生产用 |
| **Kafka** | 已有 Kafka 集群 | 大规模 |

## 5. 安装 Argo Events

```bash
# 创建命名空间
kubectl create namespace argo-events

# 安装 Argo Events
kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml
kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install-validating-webhook.yaml

# 验证
kubectl get pods -n argo-events
# controller-manager-xxx    Running
# events-webhook-xxx        Running
```

### 创建 EventBus（必须先有总线）

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
  namespace: argo-events
spec:
  nats:
    native:
      replicas: 3               # 3 副本保证高可用
      auth: token               # 或 none（学习环境可以不加认证）
```

```bash
kubectl apply -f eventbus.yaml -n argo-events

# 验证 EventBus Pod 起来了
kubectl get pods -n argo-events -l eventbus-name=default
# eventbus-default-stan-0   Running
# eventbus-default-stan-1   Running
# eventbus-default-stan-2   Running
```

## 6. 实战一：Webhook 触发 Workflow

最简单的例子——收到 HTTP 请求就跑一个 Workflow。

### Step 1: 创建 EventSource

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook-source
  namespace: argo-events
spec:
  service:
    ports:
      - port: 12000
        targetPort: 12000
  webhook:
    build-trigger:              # 事件名称
      port: "12000"
      endpoint: /build          # 监听路径
      method: POST
```

```bash
kubectl apply -f eventsource.yaml -n argo-events

# 验证：会创建一个 Pod + Service
kubectl get pods -n argo-events -l eventsource-name=webhook-source
kubectl get svc -n argo-events -l eventsource-name=webhook-source
```

### Step 2: 创建 Sensor + Trigger

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook-sensor
  namespace: argo-events
spec:
  dependencies:
    - name: build-dep
      eventSourceName: webhook-source
      eventName: build-trigger
  triggers:
    - template:
        name: build-workflow-trigger
        argoWorkflow:
          operation: submit
          source:
            resource:
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: webhook-build-
              spec:
                entrypoint: main
                templates:
                  - name: main
                    container:
                      image: alpine:3.18
                      command: [sh, -c]
                      args: ["echo 'Build triggered by webhook!' && sleep 5 && echo 'Done'"]
```

```bash
kubectl apply -f sensor.yaml -n argo-events
```

### Step 3: 测试触发

```bash
# 端口转发到 EventSource 的 Service
kubectl port-forward svc/webhook-source-eventsource-svc -n argo-events 12000:12000

# 发送请求
curl -X POST http://localhost:12000/build -H "Content-Type: application/json" -d '{"repo": "my-app"}'

# 查看是否创建了 Workflow
kubectl get wf -n argo-events
# webhook-build-xxxxx   Succeeded
```

## 7. 实战二：GitHub Push 触发 CI

```yaml
# EventSource：监听 GitHub Push
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: github-source
  namespace: argo-events
spec:
  service:
    ports:
      - port: 12000
        targetPort: 12000
  github:
    my-app-push:
      repositories:
        - owner: your-org
          names:
            - my-app
      webhook:
        endpoint: /github/push
        port: "12000"
        method: POST
      events:
        - push
        - pull_request
      apiToken:
        name: github-token-secret       # K8s Secret 里存的 GitHub Token
        key: token
      webhookSecret:
        name: github-webhook-secret
        key: secret
```

```yaml
# Sensor：Push 到 main 分支时触发构建
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: ci-sensor
  namespace: argo-events
spec:
  dependencies:
    - name: github-push
      eventSourceName: github-source
      eventName: my-app-push
      filters:
        data:
          - path: body.ref
            type: string
            value:
              - "refs/heads/main"        # 只有 push 到 main 才触发
  triggers:
    - template:
        name: ci-trigger
        argoWorkflow:
          operation: submit
          source:
            resource:
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: ci-
              spec:
                entrypoint: build
                arguments:
                  parameters:
                    - name: repo-url
                    - name: commit-sha
                templates:
                  - name: build
                    inputs:
                      parameters:
                        - name: repo-url
                        - name: commit-sha
                    container:
                      image: docker:24-dind
                      command: [sh, -c]
                      args:
                        - |
                          echo "Building repo: {{inputs.parameters.repo-url}}"
                          echo "Commit: {{inputs.parameters.commit-sha}}"
          parameters:
            # 从事件 payload 中提取参数注入 Workflow
            - src:
                dependencyName: github-push
                dataKey: body.repository.clone_url
              dest: spec.arguments.parameters.0.value
            - src:
                dependencyName: github-push
                dataKey: body.after
              dest: spec.arguments.parameters.1.value
```

**效果**：Push 到 main 分支 → Argo Events 收到 GitHub Webhook → 自动提交一个带参数的 Workflow。

## 8. 实战三：Cron 定时触发

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: calendar-source
  namespace: argo-events
spec:
  calendar:
    daily-etl:
      schedule: "0 2 * * *"           # 每天凌晨 2 点
      timezone: "Asia/Shanghai"
---
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: etl-sensor
  namespace: argo-events
spec:
  dependencies:
    - name: daily-trigger
      eventSourceName: calendar-source
      eventName: daily-etl
  triggers:
    - template:
        name: etl-workflow
        argoWorkflow:
          operation: submit
          source:
            resource:
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                generateName: daily-etl-
              spec:
                entrypoint: etl-pipeline
                templates:
                  - name: etl-pipeline
                    dag:
                      tasks:
                        - name: extract
                          template: run-step
                          arguments:
                            parameters: [{name: step, value: "extract"}]
                        - name: transform
                          dependencies: [extract]
                          template: run-step
                          arguments:
                            parameters: [{name: step, value: "transform"}]
                        - name: load
                          dependencies: [transform]
                          template: run-step
                          arguments:
                            parameters: [{name: step, value: "load"}]
                  - name: run-step
                    inputs:
                      parameters: [{name: step}]
                    container:
                      image: my-etl:latest
                      command: [python, run.py]
                      args: ["--step={{inputs.parameters.step}}"]
```

## 9. 事件过滤（Filters）

Sensor 支持多种过滤条件，避免不相关的事件触发动作：

### 数据过滤（Data Filter）

```yaml
dependencies:
  - name: github-push
    eventSourceName: github-source
    eventName: my-app-push
    filters:
      data:
        # 只有 push 到 main 分支
        - path: body.ref
          type: string
          value: ["refs/heads/main"]
        # 且修改了 src/ 目录下的文件
        - path: body.commits.#.modified.#(% "src/*")
          type: string
          value: [""]
```

### 时间过滤（Time Filter）

```yaml
filters:
  time:
    start: "08:00:00"
    stop: "18:00:00"             # 只在工作时间触发
```

### 表达式过滤（Expr Filter）

```yaml
filters:
  exprs:
    - expr: source == "production" && severity >= 3
      fields:
        - name: source
          path: body.source
        - name: severity
          path: body.severity
```

## 10. 多事件依赖（AND / OR）

Sensor 可以等多个事件**都到了**才触发（AND 语义）：

```yaml
spec:
  dependencies:
    - name: code-push
      eventSourceName: github-source
      eventName: my-app-push
    - name: tests-passed
      eventSourceName: webhook-source
      eventName: test-result
  triggers:
    - template:
        name: deploy-trigger
        conditions: "code-push && tests-passed"    # 两个事件都到了才触发
        # ...
```

也支持 OR：

```yaml
        conditions: "code-push || manual-trigger"  # 任一事件即触发
```

## 11. 参数传递：从事件到 Trigger

Argo Events 最强大的能力之一——把事件 payload 里的数据提取出来注入到 Trigger 的资源里：

```yaml
triggers:
  - template:
      name: deploy
      argoWorkflow:
        operation: submit
        source:
          resource:
            apiVersion: argoproj.io/v1alpha1
            kind: Workflow
            metadata:
              generateName: deploy-
            spec:
              arguments:
                parameters:
                  - name: image-tag       # 占位符
                  - name: environment
              # ...
      parameters:
        # 从事件数据中提取 image tag
        - src:
            dependencyName: code-push
            dataKey: body.after           # Git commit SHA
          dest: spec.arguments.parameters.0.value
        # 从事件 header 中提取
        - src:
            dependencyName: code-push
            dataKey: body.repository.default_branch
          dest: spec.arguments.parameters.1.value
```

**常用 dataKey 路径**：

| 事件类型 | 常用路径 | 含义 |
|---------|---------|------|
| GitHub push | `body.after` | commit SHA |
| GitHub push | `body.ref` | 分支（refs/heads/main） |
| GitHub push | `body.repository.clone_url` | 仓库地址 |
| GitHub PR | `body.pull_request.number` | PR 号 |
| Webhook | `body.xxx` | 自定义 payload 的字段 |
| Calendar | `body.metadata.time` | 触发时间 |

## 12. 架构与运行原理

```text
┌─────────────────────────────────────────────┐
│              EventSource Pod                 │
│                                             │
│  监听外部事件（HTTP Server / MQ Consumer）    │
│  收到事件 → 包装成 CloudEvent 格式            │
│  发送到 EventBus                             │
└────────────────────┬────────────────────────┘
                     │ CloudEvent
                     ▼
┌─────────────────────────────────────────────┐
│              EventBus (NATS)                 │
│                                             │
│  持久化存储事件                               │
│  保证至少一次投递                             │
└────────────────────┬────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│              Sensor Pod                      │
│                                             │
│  订阅 EventBus 上的事件                       │
│  根据 filters 过滤                           │
│  满足 dependencies 条件后                     │
│  提取参数 → 渲染 Trigger Template             │
│  执行 Trigger（submit Workflow / create 资源）│
└─────────────────────────────────────────────┘
```

**关键点**：
- EventSource Pod 是"监听者"，一个 EventSource 资源对应一个 Pod
- Sensor Pod 是"决策者"，一个 Sensor 资源对应一个 Pod
- EventBus 是中间人，解耦 Source 和 Sensor
- 所有事件格式统一为 [CloudEvents](https://cloudevents.io/) 规范

## 13. 常见问题排查

| 现象 | 排查方向 |
|------|---------|
| EventSource Pod 起不来 | `kubectl logs` 看报错；通常是端口冲突或凭据错误 |
| 事件发了但没触发 | 检查 Sensor filters 是否匹配；`kubectl logs` sensor pod |
| Workflow 没被创建 | Sensor 的 ServiceAccount 是否有权限 create Workflow |
| EventBus Pod CrashLoop | NATS 配置问题；检查存储卷/内存 |
| GitHub Webhook 收不到 | endpoint 是否公网可达？Secret 是否匹配？ |

**调试技巧**：

```bash
# 看 EventSource 日志
kubectl logs -n argo-events -l eventsource-name=<name> -f

# 看 Sensor 日志
kubectl logs -n argo-events -l sensor-name=<name> -f

# 看 EventBus 状态
kubectl get eventbus -n argo-events
```

## 14. RBAC 配置

Sensor 需要有权限创建 Workflow 或其他 K8s 资源：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: argo-events-sa
  namespace: argo-events
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: argo-events-role
rules:
  - apiGroups: ["argoproj.io"]
    resources: ["workflows", "workflowtemplates"]
    verbs: ["create", "get", "list"]
  - apiGroups: [""]
    resources: ["pods", "configmaps"]
    verbs: ["create", "get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argo-events-binding
subjects:
  - kind: ServiceAccount
    name: argo-events-sa
    namespace: argo-events
roleRef:
  kind: ClusterRole
  name: argo-events-role
  apiGroup: rbac.authorization.k8s.io
```

在 Sensor 里指定这个 ServiceAccount：

```yaml
spec:
  template:
    serviceAccountName: argo-events-sa
```

## 下一步

三个独立工具都了解后，看它们怎么串起来：

→ [全家桶联动实战](./07-argo-suite-integration.md)
