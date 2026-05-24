# 📐 Workflow Spec 与 Template 类型详解

> 这一篇是 Argo Workflow 的"字段大全"。读完之后，你看任何线上 Workflow yaml 都不应该再有"这个字段是啥"的疑问。

## 1. Workflow 顶层结构

一个完整的 Workflow 顶层结构如下：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: my-wf-
  namespace: argo
  labels: {}
  annotations: {}
spec:
  # ===== 1. 入口与模板 =====
  entrypoint: main
  templates: []

  # ===== 2. 输入参数与制品 =====
  arguments:
    parameters: []
    artifacts: []

  # ===== 3. 运行时控制 =====
  serviceAccountName: argo-workflow-sa
  activeDeadlineSeconds: 3600          # 整个 wf 超时
  ttlStrategy:                         # 跑完多久清理
    secondsAfterCompletion: 86400
  podGC:                               # Pod 清理
    strategy: OnWorkflowSuccess
  parallelism: 5                       # 全局并行度上限
  suspend: false                       # 是否一开始就挂起

  # ===== 4. Pod 通用配置 =====
  podMetadata:
    labels: {}
    annotations: {}
  nodeSelector: {}
  tolerations: []
  affinity: {}
  imagePullSecrets: []

  # ===== 5. 卷与制品仓库 =====
  volumes: []
  volumeClaimTemplates: []
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: default-v1

  # ===== 6. 钩子 =====
  onExit: cleanup                      # 退出处理器（成功/失败都跑）
  hooks: {}

status: {}                             # 由控制器维护，不要手写
```

写 Workflow 时**不需要全填**，绝大多数字段有默认值。下面挑你最常用的字段单独说。

## 2. 必填三件套：entrypoint + templates + 至少一个 template

最小可用的 Workflow 是：

```yaml
spec:
  entrypoint: hello
  templates:
    - name: hello
      container:
        image: alpine
        command: [echo, hi]
```

`entrypoint` 指定从哪个 template 开始执行，`templates` 是一组模板定义。

## 3. 6 种 Template 类型逐个讲

> Template 是 Argo 的核心抽象。理解 6 种类型，你写任何流水线都不慌。

### 3.1 container：跑一个容器（最常用）

```yaml
- name: build
  container:
    image: golang:1.22
    command: [sh, -c]
    args: ["go build -o app ./cmd/app"]
    workingDir: /workspace
    env:
      - name: GOPROXY
        value: https://goproxy.cn
    resources:
      requests: {cpu: 500m, memory: 512Mi}
      limits:   {cpu: "2",  memory: 2Gi}
```

字段几乎和 Pod 的 `containers[0]` 一样。一个 Pod 一个 container template。

### 3.2 script：直接写代码段

适合写小段 shell / python 不想做镜像的场景：

```yaml
- name: gen-id
  script:
    image: python:3.11
    command: [python]
    source: |
      import uuid
      print(uuid.uuid4(), end='')
```

`source` 内容会被存进 `/argosrc/script` 然后执行。脚本的标准输出会自动作为 `result` 输出（见参数篇）。

### 3.3 resource：直接操作 K8s 资源

让 Argo 帮你 `kubectl apply / patch / delete`：

```yaml
- name: create-cm
  resource:
    action: create
    manifest: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        generateName: result-
      data:
        value: hello
    successCondition: status.phase == Succeeded   # 可选，等待资源达到某状态
    failureCondition: status.phase == Failed
```

适合：
- 跑一个原生 Job 然后等它结束
- 提交一个 SparkApplication / TFJob 然后等它跑完
- 配合 `successCondition` 实现"等待外部资源就绪"

### 3.4 suspend：暂停步骤

让流水线在这里暂停，直到：
- 人工 `argo resume`
- 或到达 `duration` 自动恢复

```yaml
- name: wait-approval
  suspend: {}                  # 等人工恢复

- name: wait-30s
  suspend:
    duration: "30s"            # 定时恢复
```

典型应用：发布到 prod 前等审批；上一步训练完等数据落盘。

### 3.5 dag：定义有向无环图

```yaml
- name: pipeline
  dag:
    tasks:
      - name: build
        template: build-tpl
      - name: test
        dependencies: [build]
        template: test-tpl
      - name: deploy-staging
        dependencies: [test]
        template: deploy-tpl
        arguments:
          parameters: [{name: env, value: staging}]
      - name: deploy-prod
        dependencies: [deploy-staging]
        template: deploy-tpl
        when: "{{workflow.parameters.release}} == true"
        arguments:
          parameters: [{name: env, value: prod}]
```

DAG 的 task 关键字段：
- `dependencies`：依赖的前置任务
- `template`：调用哪个模板
- `arguments`：给被调用的模板传参
- `when`：条件执行
- `withItems` / `withParam` / `withSequence`：循环
- `continueOn`：失败/Error 时是否继续后续

### 3.6 steps：按顺序的步骤组

```yaml
- name: pipeline
  steps:
    - - name: build              # 注意是双横线（一个组开头）
        template: build-tpl
    - - name: test-unit          # 这一组里两个并行
        template: unit-tpl
      - name: test-lint
        template: lint-tpl
    - - name: deploy
        template: deploy-tpl
```

`steps` 是 `[[step1], [step2a, step2b], [step3]]` 的二维结构：
- 外层是顺序：上一组完成才进入下一组
- 内层是并行：同一组里并行执行

DAG vs Steps 怎么选？

| 场景 | 推荐 |
|------|------|
| 简单线性流水线 / 严格分阶段 | Steps |
| 多分支、多依赖、复杂依赖关系 | DAG |
| 想做 fan-out / fan-in | DAG |
| 跑 ML pipeline | DAG |

## 4. 输入与输出（inputs / outputs）

每个 template 都可以声明输入和输出。

```yaml
- name: process
  inputs:
    parameters:
      - name: lang
        default: zh
    artifacts:
      - name: src
        path: /input/data.txt
  container:
    image: alpine
    command: [sh, -c]
    args: ["cat /input/data.txt > /tmp/out.txt"]
  outputs:
    parameters:
      - name: line-count
        valueFrom:
          path: /tmp/lines.txt
    artifacts:
      - name: result
        path: /tmp/out.txt
```

读图：
- `inputs.parameters.lang` 在容器里作为变量 `{{inputs.parameters.lang}}`
- `inputs.artifacts.src` 会自动从对象存储下载到 `/input/data.txt`
- `outputs.parameters.line-count` 从容器的某个文件里读
- `outputs.artifacts.result` 自动从容器路径上传到对象存储

> 详细传递规则放在 [参数与制品](./05-parameters-and-artifacts.md) 单独讲。

## 5. 运行时控制：你迟早会用上的 8 个字段

| 字段 | 作用 | 例子 |
|------|------|------|
| `serviceAccountName` | 业务 Pod 的 SA | `argo-workflow-sa` |
| `activeDeadlineSeconds` | 整个 Workflow 超时（秒） | `3600` |
| `ttlStrategy.secondsAfterCompletion` | 跑完多久自动清理 | `86400` |
| `podGC.strategy` | Pod 清理策略 | `OnPodCompletion` / `OnWorkflowSuccess` |
| `parallelism` | 全局并行 Pod 上限 | `10` |
| `nodeSelector` | 调度到带标签的节点 | `accelerator: nvidia` |
| `tolerations` | 容忍污点 | GPU 池常用 |
| `imagePullSecrets` | 私有镜像凭证 | `[{name: docker-secret}]` |

template 级别也能写这些字段，**template 级别会覆盖 workflow 级别**。

## 6. 模板里的常用变量（把字段串起来的胶水）

写 Workflow 时你会大量用到 `{{...}}` 占位符，下面是高频几个：

```text
{{workflow.name}}                  当前 Workflow 名
{{workflow.namespace}}             命名空间
{{workflow.uid}}                   UID
{{workflow.parameters.x}}          顶层参数 x
{{inputs.parameters.x}}            模板入参 x
{{inputs.artifacts.y.path}}        制品 y 在容器内的路径
{{outputs.parameters.x}}           模板输出 x（用于父级 dag 引用）
{{steps.<name>.outputs.parameters.x}}     steps 模式下取上一步输出
{{tasks.<name>.outputs.parameters.x}}     dag 模式下取上一节点输出
{{item}} / {{item.key}}            withItems / withParam 循环变量
{{retries}}                        当前重试次数
```

记住一个区分：**steps 用 `steps.X`，dag 用 `tasks.X`**，写错了 controller 会拿不到值。

## 7. 一个写得"舒服"的真实例子

下面这个例子综合用了 dag、参数、制品、退出处理器、超时与清理：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: build-deploy-
spec:
  entrypoint: pipeline
  serviceAccountName: argo-workflow-sa
  activeDeadlineSeconds: 1800
  ttlStrategy:
    secondsAfterCompletion: 86400
  podGC:
    strategy: OnWorkflowSuccess
  arguments:
    parameters:
      - name: repo
        value: github.com/acme/app
      - name: ref
        value: main
  onExit: notify
  templates:
    - name: pipeline
      dag:
        tasks:
          - name: clone
            template: git-clone
            arguments:
              parameters: [{name: repo, value: "{{workflow.parameters.repo}}"},
                           {name: ref,  value: "{{workflow.parameters.ref}}"}]
          - name: build
            dependencies: [clone]
            template: build
            arguments:
              artifacts: [{name: src, from: "{{tasks.clone.outputs.artifacts.src}}"}]
          - name: deploy
            dependencies: [build]
            template: deploy
            when: "{{tasks.build.outputs.parameters.image}} != ''"
            arguments:
              parameters: [{name: image, value: "{{tasks.build.outputs.parameters.image}}"}]

    - name: git-clone
      inputs:
        parameters: [{name: repo}, {name: ref}]
      container:
        image: alpine/git:2.43
        command: [sh, -c]
        args: ["git clone https://{{inputs.parameters.repo}} /workspace && cd /workspace && git checkout {{inputs.parameters.ref}}"]
      outputs:
        artifacts:
          - name: src
            path: /workspace

    - name: build
      inputs:
        artifacts: [{name: src, path: /workspace}]
      container:
        image: gcr.io/kaniko-project/executor:latest
        args: ["--context=/workspace", "--destination=registry.local/app:{{workflow.uid}}"]
      outputs:
        parameters:
          - name: image
            value: "registry.local/app:{{workflow.uid}}"

    - name: deploy
      inputs:
        parameters: [{name: image}]
      container:
        image: bitnami/kubectl:1.30
        command: [sh, -c]
        args: ["kubectl set image deploy/app app={{inputs.parameters.image}}"]

    - name: notify
      container:
        image: curlimages/curl
        command: [sh, -c]
        args: ["curl -X POST hooks.example.com -d 'wf={{workflow.name}}, status={{workflow.status}}'"]
```

读这个例子时，建议你按下面顺序读：

1. 先读 `entrypoint: pipeline`，定位入口
2. 再读 `pipeline` 的 dag，看 4 个节点的顺序
3. 然后逐一读 4 个底层模板的 `inputs / outputs`
4. 最后读全局字段（onExit、ttlStrategy）

读完你应该能回答：clone 的产物怎么传给 build？build 的镜像怎么传给 deploy？deploy 失败之后 notify 还会不会跑？（答：onExit 不论成功失败都会跑，所以会 notify。）

## 下一步

下一篇深入讲编排：

- [DAG、Steps 与控制流](./04-dag-steps-and-controlflow.md)
