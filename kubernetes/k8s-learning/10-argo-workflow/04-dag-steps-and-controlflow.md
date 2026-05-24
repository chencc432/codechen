# 🔀 DAG、Steps 与控制流

> 工作流的灵魂在于"怎么编排"。这一篇专门讲 Argo 的编排能力：DAG vs Steps、条件、循环、重试、超时、错误处理、退出处理器。

## 1. DAG vs Steps：再细说一次怎么选

| 维度 | Steps | DAG |
|------|-------|-----|
| 写法 | 二维列表（`[[a],[b,c]]`） | 节点 + `dependencies` |
| 心智模型 | 阶段式 | 图 |
| 灵活度 | 低 | 高 |
| 跳过/条件 | 支持 | 支持 |
| fan-out / fan-in | 别扭 | 自然 |
| 复杂依赖 | 难表达 | 容易表达 |
| 调试可视化 | UI 也能看，但图不直观 | UI 上画的图最清晰 |

**经验法则**：> 5 个步骤、有跨阶段依赖时，闭眼用 DAG。

## 2. 条件执行：when

DAG 和 Steps 都支持 `when`：

```yaml
- name: deploy-prod
  dependencies: [test]
  template: deploy
  when: "{{workflow.parameters.env}} == prod"
```

支持的语法（基于 [govaluate](https://github.com/Knetic/govaluate)）：

```text
==  !=  <  >  <=  >=  &&  ||  !
+ - * / %  in
```

例子：

```yaml
when: "{{tasks.A.status}} == Succeeded && {{workflow.parameters.skipTest}} != true"
```

**注意：when 不为真时，Argo 会把这个节点标为 Skipped，对它的依赖关系不会报错**——也就是说后面 `dependencies: [deploy-prod]` 的节点也会跟着 Skip 或视情况执行（受 `depends` 表达式控制）。

## 3. 高级依赖：depends（DAG 专用）

`dependencies` 是简单的"全部成功才跑"，`depends` 可以写表达式：

```yaml
tasks:
  - name: A
    template: foo
  - name: B
    template: bar
  - name: C
    depends: "A.Succeeded || B.Succeeded"   # 任一成功就跑
    template: merge
  - name: D
    depends: "(A.Succeeded && B.Failed) || A.Errored"
    template: handle-failure
```

支持的状态：

```text
Succeeded   成功
Failed      失败
Errored     运行出错（如 Pod 异常）
Skipped     被跳过
Omitted     被忽略
Daemoned    Daemon 模式运行中
AnySucceeded / AllFailed   作用于循环展开的多个子任务
```

`depends` 比 `dependencies` 强很多，建议线上 Workflow 默认用 `depends`。

## 4. 循环：withItems / withParam / withSequence

让一个任务"展开"成多个并行子任务。

### 4.1 withItems：固定列表

```yaml
- name: ping
  template: ping-tpl
  withItems:
    - {host: a.com, port: 80}
    - {host: b.com, port: 443}
    - {host: c.com, port: 22}
  arguments:
    parameters:
      - name: target
        value: "{{item.host}}:{{item.port}}"
```

`{{item.host}}` 取出元素的字段。如果元素是简单字符串，直接 `{{item}}`。

### 4.2 withParam：从前一步动态生成

```yaml
- name: split
  template: split-tpl                # 输出一个 JSON 数组到 result
- name: process
  dependencies: [split]
  template: process-tpl
  withParam: "{{tasks.split.outputs.result}}"
  arguments:
    parameters: [{name: chunk, value: "{{item}}"}]
```

典型用法：A 算出来要处理的 N 份数据，B 并行处理 N 份。**这是 fan-out / fan-in 的标准写法。**

### 4.3 withSequence：数字序列

```yaml
withSequence:
  start: 0
  end: 9        # 包含 0~9，10 个
  format: "%03d"
arguments:
  parameters: [{name: idx, value: "{{item}}"}]
```

## 5. 重试：retryStrategy

Workflow 级别和 Template 级别都能配：

```yaml
- name: flaky
  retryStrategy:
    limit: 5                         # 最多重试 5 次
    retryPolicy: "OnError"           # OnFailure / OnError / Always / OnTransientError
    backoff:
      duration: "30s"                # 第一次等待
      factor: 2                      # 指数退避因子
      maxDuration: "10m"             # 最大间隔
    affinity:
      nodeAntiAffinity: {}           # 重试时换一个节点（避坏机器）
  container:
    image: my-image
    command: [./flaky-job.sh]
```

| retryPolicy | 触发条件 |
|-------------|----------|
| `OnFailure` | 业务返回非 0 |
| `OnError` | Pod 启动失败 / 调度失败等 |
| `Always` | 不管哪种失败都重试 |
| `OnTransientError` | 系统瞬时错误（推荐生产用） |

**`{{retries}}` 变量**可以在容器里拿到当前重试次数，用于幂等处理或换 seed。

## 6. 超时：activeDeadlineSeconds

```yaml
spec:
  activeDeadlineSeconds: 1800        # 整个 wf
  templates:
    - name: long-task
      activeDeadlineSeconds: 600     # 这一步
      container: {...}
```

超时后 Argo 会让对应 Pod / Workflow 标记失败，并触发 `onExit`。

> 注意区别：`activeDeadlineSeconds` 是 Pod / Workflow 级超时，**不是任务等待超时**。如果你想"超过 X 分钟没排到 Pod 就失败"，目前要靠 `parallelism` + `podPriority` + 监控配合。

## 7. 错误处理：continueOn

希望某一步失败后流水线继续往下走：

```yaml
- name: optional-test
  template: run-test
  continueOn:
    failed: true
    error: false
```

后续节点用 `depends` 显式表达："不管 optional-test 怎样都跑"：

```yaml
- name: report
  depends: "optional-test.Succeeded || optional-test.Failed"
```

## 8. 退出处理器：onExit

**onExit 是 Argo 最被低估的能力之一。**它会在整个 Workflow 结束时触发，不论成败。

```yaml
spec:
  onExit: cleanup
  templates:
    - name: cleanup
      steps:
        - - name: notify
            template: send-slack
        - - name: gc
            template: delete-temp-resources
```

在 onExit 模板里，可以用：

```text
{{workflow.status}}      Succeeded / Failed / Error
{{workflow.failures}}    失败的节点列表
{{workflow.duration}}    总时长
```

典型用途：发通知、清理临时资源、上报监控指标。

## 9. Hooks：节点级钩子

template 级别可以挂 hooks，类似生命周期钩子：

```yaml
- name: deploy
  hooks:
    failure:
      template: rollback
      expression: status == "Failed"
    running:
      template: notify-start
      expression: status == "Running"
  container: {...}
```

适合"某一步进入某状态就触发某动作"，比 `onExit` 粒度更细。

## 10. 组合一个真实控制流：fan-out + retry + onExit

```yaml
spec:
  entrypoint: main
  onExit: report
  templates:
    - name: main
      dag:
        tasks:
          - name: split
            template: split

          - name: process
            depends: "split.Succeeded"
            template: do-one
            withParam: "{{tasks.split.outputs.result}}"
            arguments:
              parameters: [{name: chunk, value: "{{item}}"}]

          - name: merge
            depends: "process"           # 等所有展开任务（fan-in）
            template: merge

    - name: split
      script:
        image: python:3.11
        command: [python]
        source: |
          import json
          print(json.dumps(["a", "b", "c", "d"]), end='')

    - name: do-one
      retryStrategy:
        limit: 3
        retryPolicy: OnTransientError
        backoff: {duration: "10s", factor: 2}
      inputs:
        parameters: [{name: chunk}]
      container:
        image: my-worker
        command: [./run.sh, "{{inputs.parameters.chunk}}"]

    - name: merge
      container:
        image: alpine
        command: [echo, "all chunks done"]

    - name: report
      container:
        image: curlimages/curl
        command: [sh, -c]
        args: ["curl -X POST hooks.example.com -d 'status={{workflow.status}}'"]
```

读懂这一段，你就掌握了 Argo 80% 的编排技巧。

## 11. 编排时的几个常见坑

| 坑 | 现象 | 解决 |
|----|------|------|
| `dependencies` 写错名字 | 依赖节点找不到，wf 直接 Failed | 仔细核对，名字大小写敏感 |
| `withParam` 入参不是合法 JSON 数组 | 展开失败 | 上一步输出必须是 `["a","b"]` 这种合法 JSON |
| `when` 表达式里把数字当字符串比 | 总是 false | 数字别加引号；字符串记得用 `==` 不是 `=` |
| `onExit` 用了 `inputs.parameters` | 拿不到值 | onExit 模板里只能用 `{{workflow.*}}` 变量 |
| 重试 + 输出参数冲突 | 第二次重试拿到的是上一次的输出 | 记得在容器里覆盖输出文件 |

## 下一步

控制流讲完了，剩下最重要的是数据传递：

- [参数、制品与数据传递](./05-parameters-and-artifacts.md)
